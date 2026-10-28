package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"
	"github.com/zax0rz/darkpawns/internal/portable"
)

var (
	errCancelled = errors.New("cancelled")
	cliTitle     = lipgloss.NewStyle().Bold(true).Foreground(setupAccent)
	cliRule      = lipgloss.NewStyle().Foreground(setupAccent)
)

func newCLIHelp() help.Model {
	footer := help.New()
	footer.Styles.ShortKey = lipgloss.NewStyle().Bold(true)
	footer.Styles.ShortDesc = lipgloss.NewStyle()
	footer.Styles.ShortSeparator = lipgloss.NewStyle()
	footer.Styles.Ellipsis = lipgloss.NewStyle()
	return footer
}

func plainRequested(cmd *cobra.Command) bool {
	local, _ := cmd.Flags().GetBool("plain")
	root, _ := cmd.Root().PersistentFlags().GetBool("plain")
	return local || root
}

func outputTerminal(cmd *cobra.Command) bool {
	f, ok := cmd.OutOrStdout().(*os.File)
	return ok && term.IsTerminal(f.Fd())
}

func richOutput(cmd *cobra.Command) bool {
	asJSON, _ := cmd.Flags().GetBool("json")
	return !asJSON && !plainRequested(cmd) && outputTerminal(cmd) && os.Getenv("TERM") != "dumb" && os.Getenv("NO_COLOR") == ""
}

func interactiveOutput(cmd *cobra.Command) bool {
	f, ok := cmd.InOrStdin().(*os.File)
	return richOutput(cmd) && ok && term.IsTerminal(f.Fd())
}

func cliWidth(cmd *cobra.Command) int {
	if f, ok := cmd.OutOrStdout().(*os.File); ok {
		if w, _, err := term.GetSize(f.Fd()); err == nil && w > 0 {
			return min(w, 100)
		}
	}
	return 80
}

// Never render control sequences from logs, paths, instance names or release text.
// The legacy plain path deliberately remains byte-for-byte unchanged.
func terminalText(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' {
			return r
		}
		if r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, ansi.Strip(s))
}

func checklist(out io.Writer, heading string, checks []portable.Check, width int) error {
	passed := 0
	for _, check := range checks {
		if check.OK {
			passed++
		}
	}
	if _, err := fmt.Fprintln(out, cliTitle.Render(heading), fmt.Sprintf("\n%d of %d checks passed\n", passed, len(checks))); err != nil {
		return err
	}
	for _, check := range checks {
		mark := "✓"
		if !check.OK {
			mark = "✗"
		}
		label := lipgloss.NewStyle().Bold(true)
		if !check.OK {
			label = label.Foreground(setupAccent)
		}
		if _, err := fmt.Fprintln(out, label.Render(mark+" "+terminalText(check.Name)), "\n"+lipgloss.NewStyle().Width(max(16, width-4)).PaddingLeft(2).Render(terminalText(check.Detail))); err != nil {
			return err
		}
	}
	return nil
}

type instanceItem struct{ path string }

func (i instanceItem) Title() string       { return terminalText(filepath.Base(i.path)) }
func (i instanceItem) Description() string { return terminalText(i.path) }
func (i instanceItem) FilterValue() string { return terminalText(i.path) }
func knownInstances(defaultPath string) []list.Item {
	paths := map[string]bool{}
	entries, err := os.ReadDir(filepath.Dir(defaultPath))
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			p := filepath.Join(filepath.Dir(defaultPath), entry.Name())
			if _, err := portable.Load(p); err == nil {
				paths[p] = true
			}
		}
	}
	order := []string{}
	for p := range paths {
		order = append(order, p)
	}
	sort.Strings(order)
	items := make([]list.Item, 0, len(order))
	for _, p := range order {
		items = append(items, instanceItem{p})
	}
	return items
}

type instancePicker struct {
	list      list.Model
	selected  string
	cancelled bool
}

func (m instancePicker) Init() tea.Cmd { return nil }
func (m instancePicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(max(20, msg.Width-4), max(8, msg.Height-3))
	case tea.KeyMsg:
		if m.list.FilterState() != list.Filtering {
			switch msg.String() {
			case "enter":
				if i, ok := m.list.SelectedItem().(instanceItem); ok {
					m.selected = i.path
					return m, tea.Quit
				}
			case "q", "esc", "ctrl+c":
				m.cancelled = true
				return m, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m instancePicker) View() string { return lipgloss.NewStyle().Padding(1, 2).Render(m.list.View()) }

func pickInstance(cmd *cobra.Command, items []list.Item) (string, error) {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.NormalTitle = lipgloss.NewStyle().PaddingLeft(2)
	delegate.Styles.NormalDesc = delegate.Styles.NormalTitle
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(setupAccent).BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).BorderForeground(setupAccent).PaddingLeft(1)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedTitle
	l := list.New(items, delegate, max(20, cliWidth(cmd)-4), 16)
	l.Title = "Choose an instance"
	l.Styles.Title = cliTitle
	l.Styles.FilterPrompt = cliRule
	l.Styles.FilterCursor = cliRule
	l.FilterInput.PromptStyle = cliRule
	l.FilterInput.Cursor.Style = cliRule
	l.Help = newCLIHelp()
	l.SetShowHelp(true)
	l.DisableQuitKeybindings()
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "choose")), key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "cancel"))}
	}
	model, err := tea.NewProgram(instancePicker{list: l}, tea.WithInput(cmd.InOrStdin()), tea.WithOutput(cmd.OutOrStdout()), tea.WithContext(cmd.Context())).Run()
	if err != nil {
		return "", err
	}
	result := model.(instancePicker)
	if result.selected == "" {
		return "", errCancelled
	}
	return result.selected, nil
}

func chooseInstance(cmd *cobra.Command, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	path, err := defaultInstance()
	if err != nil {
		return "", err
	}
	if interactiveOutput(cmd) {
		items := knownInstances(path)
		if len(items) > 1 {
			return pickInstance(cmd, items)
		}
	}
	return path, nil
}

func addInstances(root *cobra.Command) {
	root.AddCommand(&cobra.Command{Use: "instances", Short: "List or interactively filter instances in the default data folder", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		p, err := defaultInstance()
		if err != nil {
			return err
		}
		items := knownInstances(p)
		if !interactiveOutput(cmd) {
			for _, i := range items {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), i.(instanceItem).path); err != nil {
					return err
				}
			}
			return nil
		}
		if len(items) == 0 {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), "No instances found. Use darkpawns setup or pass --instance <path>.")
			return err
		}
		selected, err := pickInstance(cmd, items)
		if errors.Is(err, errCancelled) {
			return nil
		}
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(cmd.OutOrStdout(), selected)
		return err
	}})
}

func confirmAction(cmd *cobra.Command, title, description string) (bool, error) {
	approved := false
	err := huh.NewForm(huh.NewGroup(huh.NewConfirm().Title(title).Description(terminalText(description)).Affirmative("Continue").Negative("Cancel").Value(&approved))).WithTheme(setupTheme()).WithInput(cmd.InOrStdin()).WithOutput(cmd.OutOrStdout()).RunWithContext(cmd.Context())
	if errors.Is(err, huh.ErrUserAborted) {
		return false, nil
	}
	return approved, err
}
