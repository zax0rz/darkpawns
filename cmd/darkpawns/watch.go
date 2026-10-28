package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
	"github.com/zax0rz/darkpawns/internal/portable"
)

type adminReader struct {
	client      *http.Client
	base, token string
}

func localAdmin(path string, status portable.RuntimeStatus) (*adminReader, error) {
	cfg, err := portable.Load(path)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(status.URL)
	if err != nil || u.Scheme != "http" || u.Hostname() != "127.0.0.1" || u.Port() != fmt.Sprint(cfg.HTTPPort) || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "/" && u.Path != "") {
		return nil, errors.New("instance did not report its configured loopback web address")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	return &adminReader{base: strings.TrimSuffix(status.URL, "/"), client: &http.Client{Timeout: 3 * time.Second, Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("local admin redirects are refused") }}}, nil
}

func (a *adminReader) request(ctx context.Context, method, path string, body io.Reader, value any) error {
	req, err := http.NewRequestWithContext(ctx, method, a.base+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("X-Requested-With", "darkpawns-launcher")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}
	res, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("admin read returned HTTP %d", res.StatusCode)
	}
	if path == "/admin/login" {
		for _, cookie := range res.Cookies() {
			if cookie.Name == "admin_token" {
				a.token = cookie.Value
			}
		}
		if a.token == "" {
			return errors.New("admin sign-in returned no session")
		}
	}
	if value != nil {
		return json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(value)
	}
	return nil
}

func signInAdmin(cmd *cobra.Command, path string, status portable.RuntimeStatus) (*adminReader, error) {
	if status.State != "running" {
		return nil, nil
	}
	reader, err := localAdmin(path, status)
	if err != nil {
		return nil, err
	}
	approved, err := confirmAction(cmd, "Read live player details?", "Sign in to this instance with an immortal account. Your sign-in lasts only until you close this view. Cancel to continue with process status and local logs.")
	if err != nil || !approved {
		return nil, err
	}
	var name, password string
	form := huh.NewForm(huh.NewGroup(huh.NewInput().Title("Immortal character").Value(&name), huh.NewInput().Title("Password").EchoMode(huh.EchoModePassword).Value(&password))).WithTheme(setupTheme()).WithInput(cmd.InOrStdin()).WithOutput(cmd.OutOrStdout())
	if err := form.RunWithContext(cmd.Context()); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, nil
		}
		return nil, err
	}
	data, err := json.Marshal(struct {
		Name     string `json:"player_name"`
		Password string `json:"password"`
	}{name, password})
	password = ""
	if err != nil {
		return nil, err
	}
	err = reader.request(cmd.Context(), http.MethodPost, "/admin/login", bytes.NewReader(data), nil)
	clear(data)
	if err != nil {
		return nil, fmt.Errorf("sign-in failed: %w", err)
	}
	return reader, nil
}

type (
	onlinePlayer struct {
		Name  string `json:"name"`
		Level int    `json:"level"`
		Room  int    `json:"room"`
	}
	cliSnapshot struct {
		status          portable.RuntimeStatus
		version, uptime string
		online          []onlinePlayer
		onlineKnown     bool
		playerError     string
		logs            string
		errors          []string
		err             error
		at              time.Time
		fixture         string
	}
)

func readTail(path string) (string, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	start := max(int64(0), info.Size()-64*1024)
	if _, err = f.Seek(start, io.SeekStart); err != nil {
		return "", err
	}
	data, err := io.ReadAll(io.LimitReader(f, 64*1024))
	if err != nil {
		return "", err
	}
	if start > 0 {
		if idx := bytes.IndexByte(data, '\n'); idx >= 0 {
			data = data[idx+1:]
		}
	}
	return terminalText(string(data)), nil
}

func readCLISnapshot(ctx context.Context, path, base string, reader *adminReader) cliSnapshot {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	s := cliSnapshot{at: time.Now()}
	s.status, s.err = portable.Status(ctx, path)
	release := s.status.Bundle
	if release == "" {
		release = base
	}
	if manifest, err := portable.ReadManifest(release); err == nil {
		s.version = manifest.Version
	}
	if s.status.State == "stopped" {
		s.onlineKnown = true
		s.uptime = "Stopped"
	} else if reader != nil && s.err == nil {
		var server struct {
			Uptime string `json:"uptime"`
		}
		err := reader.request(ctx, "GET", "/admin/server", nil, &server)
		if err == nil {
			s.uptime = server.Uptime
		}
		if err = errors.Join(err, reader.request(ctx, "GET", "/admin/players", nil, &s.online)); err != nil {
			s.playerError = err.Error()
		} else {
			s.onlineKnown = true
		}
	}
	if reader == nil && s.status.State != "stopped" {
		s.playerError = "Sign-in skipped; live player details unavailable."
	}
	var logErr error
	s.logs, logErr = readTail(filepath.Join(path, "lib", "logs", "launcher.log"))
	if logErr != nil {
		if os.IsNotExist(logErr) {
			s.logs = "No launcher log yet."
		} else {
			s.logs = "Launcher log unavailable: " + terminalText(logErr.Error())
		}
	}
	for _, line := range strings.Split(s.logs, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "ERROR ") || strings.Contains(line, "level=ERROR") || strings.Contains(line, `"level":"ERROR"`) {
			s.errors = append(s.errors, line)
		}
	}
	if len(s.errors) > 3 {
		s.errors = s.errors[len(s.errors)-3:]
	}
	return s
}

func onlineSummary(s cliSnapshot) string {
	if !s.onlineKnown {
		return "Couldn't check who's online. " + s.playerError
	}
	if len(s.online) == 0 {
		return "No one is online."
	}
	names := make([]string, 0, min(5, len(s.online)))
	for _, p := range s.online[:min(5, len(s.online))] {
		names = append(names, terminalText(p.Name))
	}
	shown := strings.Join(names, ", ")
	if rest := len(s.online) - len(names); rest > 0 {
		shown += fmt.Sprintf(" and %d more", rest)
	}
	plural := "players"
	if len(s.online) == 1 {
		plural = "player"
	}
	return fmt.Sprintf("%d %s online: %s", len(s.online), plural, shown)
}

func onlineImpact(s cliSnapshot) string {
	if s.onlineKnown && len(s.online) == 0 {
		return onlineSummary(s)
	}
	if !s.onlineKnown {
		return onlineSummary(s) + " Anyone online will be disconnected; characters are saved first."
	}
	saved := "characters are"
	if len(s.online) == 1 {
		saved = "the character is"
	}
	return onlineSummary(s) + ". They'll be disconnected; " + saved + " saved first."
}

type (
	watchTick   time.Time
	watchLoaded cliSnapshot
	watchKeys   struct{ Quit, Refresh, Follow, Scroll, Help, Focus key.Binding }
)

func (k watchKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Quit, k.Refresh, k.Scroll, k.Follow, k.Focus, k.Help}
}
func (k watchKeys) FullHelp() [][]key.Binding { return [][]key.Binding{k.ShortHelp()} }
func newWatchKeys() watchKeys {
	return watchKeys{key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")), key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")), key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "follow log")), key.NewBinding(key.WithKeys("up", "down", "pgup", "pgdown"), key.WithHelp("↑/↓", "scroll log")), key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")), key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "players/log"))}
}

type watchModel struct {
	load            func(context.Context) cliSnapshot
	ctx             context.Context
	data            cliSnapshot
	viewport        viewport.Model
	players         table.Model
	spinner         spinner.Model
	help            help.Model
	keys            watchKeys
	width, height   int
	loading, follow bool
	playerFocus     bool
}

func newWatch(ctx context.Context, load func(context.Context) cliSnapshot) watchModel {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = cliRule
	t := table.New(table.WithColumns([]table.Column{{Title: "Player", Width: 24}, {Title: "Level", Width: 8}, {Title: "Room", Width: 10}}), table.WithHeight(4))
	styles := table.DefaultStyles()
	styles.Header = lipgloss.NewStyle().Bold(true)
	styles.Selected = lipgloss.NewStyle().Foreground(setupAccent).Bold(true)
	t.SetStyles(styles)
	footer := newCLIHelp()
	return watchModel{ctx: ctx, load: load, viewport: viewport.New(76, 8), players: t, spinner: sp, help: footer, keys: newWatchKeys(), width: 80, height: 32, loading: true, follow: true}
}
func (m watchModel) fetch() tea.Cmd { return func() tea.Msg { return watchLoaded(m.load(m.ctx)) } }
func (m watchModel) Init() tea.Cmd  { return tea.Batch(m.fetch(), m.spinner.Tick, watchTickCommand()) }

func watchTickCommand() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return watchTick(t) })
}

func (m watchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = max(20, msg.Width)
		m.height = max(10, msg.Height)
		m.viewport.Width = max(16, m.width-4)
		m.viewport.Height = max(2, m.height-22)
		m.help.Width = m.width - 4
		m.players.SetWidth(m.width - 4)
		m.players.SetColumns([]table.Column{{Title: "Player", Width: max(6, m.width-26)}, {Title: "Level", Width: 6}, {Title: "Room", Width: 8}})
	case watchLoaded:
		m.loading = false
		m.data = cliSnapshot(msg)
		m.viewport.SetContent(m.data.logs)
		if m.follow {
			m.viewport.GotoBottom()
		}
		rows := []table.Row{}
		for _, p := range m.data.online {
			rows = append(rows, table.Row{terminalText(p.Name), fmt.Sprint(p.Level), fmt.Sprint(p.Room)})
		}
		m.players.SetRows(rows)
		return m, nil
	case watchTick:
		if !m.loading {
			m.loading = true
			return m, tea.Batch(m.fetch(), watchTickCommand())
		}
		return m, watchTickCommand()
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Refresh):
			if !m.loading {
				m.loading = true
				return m, m.fetch()
			}
		case key.Matches(msg, m.keys.Follow):
			m.follow = !m.follow
			if m.follow {
				m.viewport.GotoBottom()
			}
		case key.Matches(msg, m.keys.Focus):
			m.playerFocus = !m.playerFocus
			m.players.Focus()
			if !m.playerFocus {
				m.players.Blur()
			}
		case key.Matches(msg, m.keys.Help):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(msg, m.keys.Scroll):
			m.follow = false
		}
	}
	var cmd, spin tea.Cmd
	if m.playerFocus {
		m.players, cmd = m.players.Update(msg)
	} else {
		m.viewport, cmd = m.viewport.Update(msg)
	}
	m.spinner, spin = m.spinner.Update(msg)
	return m, tea.Batch(cmd, spin)
}

func (m watchModel) View() string {
	w := max(16, m.width-4)
	rule := cliRule.Render(strings.Repeat("─", w))
	heading := "Dark Pawns · live status"
	if m.data.fixture != "" {
		heading += " · " + m.data.fixture
	}
	state := terminalText(m.data.status.State)
	if state == "" {
		state = "Loading"
	}
	if m.loading {
		state = m.spinner.View() + " " + state
	}
	version := m.data.version
	if version == "" {
		version = "Unavailable"
	}
	uptime := m.data.uptime
	if uptime == "" {
		uptime = "Unavailable"
	}
	if m.help.ShowAll {
		text := cliTitle.Render(heading) + "\n\nKeyboard help\n\n"
		for _, k := range m.keys.ShortHelp() {
			text += k.Help().Key + "  " + k.Help().Desc + "\n"
		}
		return lipgloss.NewStyle().Width(w).Padding(1, 2).Render(text + "\n? return to status")
	}
	body := cliTitle.Render(heading) + "\n" + ansi.Truncate(terminalText(m.data.status.Instance), w, "…") + "\n" + rule + "\n" + lipgloss.NewStyle().Width(w).Render(fmt.Sprintf("%s    Uptime %s    Release %s", state, terminalText(uptime), terminalText(version))) + "\n"
	if m.data.err != nil {
		body += cliRule.Render(ansi.Truncate("Refresh failed: "+terminalText(m.data.err.Error()), w, "…")) + "\n"
	}
	if m.data.onlineKnown {
		body += fmt.Sprintf("\nPlayers online · %d\n", len(m.data.online))
		if len(m.data.online) > 0 {
			players := m.players
			players.SetHeight(min(4, max(1, m.height/8)))
			body += players.View() + "\n"
		}
	} else {
		body += "\n" + lipgloss.NewStyle().Width(w).Render(onlineSummary(m.data)) + "\n"
	}
	body += "\n" + lipgloss.NewStyle().Bold(true).Render("Recent errors") + "\n"
	if len(m.data.errors) == 0 {
		body += "No errors in the recent logs.\n"
	} else {
		body += lipgloss.NewStyle().Width(w).Foreground(setupAccent).Render(ansi.Truncate(m.data.errors[len(m.data.errors)-1], w, "…")) + "\n"
	}
	mode := "paused"
	if m.follow {
		mode = "following"
	}
	focus := "log"
	if m.playerFocus {
		focus = "players"
	}
	body += "\n" + lipgloss.NewStyle().Bold(true).Render("Launcher log · "+mode+" · focus "+focus) + "\n"
	helpView := m.help.View(m.keys)
	if m.width < 80 {
		helpView = m.help.ShortHelpView([]key.Binding{m.keys.Quit, m.keys.Refresh, m.keys.Help})
	}
	footer := "\n" + rule + "\n" + lipgloss.NewStyle().Width(w).Render(helpView)
	vp := m.viewport
	vp.Height = max(1, m.height-lipgloss.Height(body)-lipgloss.Height(footer)-2)
	if m.follow {
		vp.GotoBottom()
	}
	body += vp.View() + footer
	return lipgloss.NewStyle().Padding(1, 2).Render(body)
}

func watchStatus(cmd *cobra.Command, path, base string) error {
	initial, err := portable.Status(cmd.Context(), path)
	if err != nil {
		return err
	}
	reader, err := signInAdmin(cmd, path, initial)
	if err != nil {
		return err
	}
	if reader != nil {
		defer reader.client.CloseIdleConnections()
	}
	_, err = tea.NewProgram(newWatch(cmd.Context(), func(ctx context.Context) cliSnapshot { return readCLISnapshot(ctx, path, base, reader) }), tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout()), tea.WithAltScreen(), tea.WithContext(cmd.Context())).Run()
	return err
}
