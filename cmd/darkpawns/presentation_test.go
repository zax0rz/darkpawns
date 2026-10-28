package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/spf13/cobra"
	"github.com/zax0rz/darkpawns/internal/portable"
)

func cliInstance(t *testing.T, port int) string {
	t.Helper()
	p := t.TempDir()
	b, err := json.Marshal(portable.Config{Format: 1, HTTPPort: port, JWTSecret: strings.Repeat("00", 32)})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(p, "instance.json"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	canonical, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func TestNonTerminalWatchAndPlainStatusKeepLegacyBytes(t *testing.T) {
	p := cliInstance(t, 45318)
	for _, flags := range [][]string{{}, {"--watch"}, {"--watch", "--plain"}, {"--plain"}} {
		cmd := command()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetIn(strings.NewReader(""))
		cmd.SetArgs(append([]string{"status", "--instance", p}, flags...))
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		want := "stopped\nInstance: " + p + "\n"
		if out.String() != want {
			t.Fatalf("flags %v changed bytes: %q", flags, out.String())
		}
		if richOutput(cmd) {
			t.Fatal("buffer was treated as a terminal")
		}
	}
	cmd := command()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"status", "--watch", "--json", "--instance", p})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	want, err := json.Marshal(portable.RuntimeStatus{State: "stopped", Instance: p})
	if err != nil {
		t.Fatal(err)
	}
	if out.String() != string(want)+"\n" {
		t.Fatalf("JSON changed: %s", out.String())
	}
}

func TestNonTerminalUpdateKeepsLegacyPrefixAndError(t *testing.T) {
	p := cliInstance(t, 45318)
	root := &cobra.Command{SilenceUsage: true, SilenceErrors: true}
	root.PersistentFlags().Bool("plain", false, "")
	root.SetIn(strings.NewReader("n\n"))
	var out bytes.Buffer
	root.SetOut(&out)
	addUpdate(root, &p, func() (string, error) { return filepath.Join(t.TempDir(), "missing-release"), nil })
	root.SetArgs([]string{"update", "--check"})
	if err := root.Execute(); err == nil {
		t.Fatal("invalid release accepted")
	}
	if out.String() != "Checking and staging the release. Your world keeps running during verification.\n" {
		t.Fatalf("plain output changed: %q", out.String())
	}
}

func TestTerminalTextStripsLogAndPathControls(t *testing.T) {
	raw := "hello\x1b]52;c;ZXhmaWw=\x07\x1b[31m red\x1b[0m\r\b\nnext"
	got := terminalText(raw)
	if got != "hello red\nnext" {
		t.Fatalf("unsafe or altered text: %q", got)
	}
}

func TestWatchResizeNavigationAndUnknownData(t *testing.T) {
	for _, width := range []int{40, 80, 120} {
		m := newWatch(context.Background(), nil)
		model, _ := m.Update(tea.WindowSizeMsg{Width: width, Height: 24})
		m = model.(watchModel)
		model, _ = m.Update(watchLoaded(cliSnapshot{status: portable.RuntimeStatus{State: "running", Instance: "/a/long/path"}, version: "v0.1.7", uptime: "2h", onlineKnown: true, online: []onlinePlayer{{"One", 1, 100}, {"Two", 2, 200}, {"Three", 3, 300}}, logs: strings.Repeat("log line\n", 40), errors: []string{"ERROR: a long failure message"}}))
		m = model.(watchModel)
		view := m.View()
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > width {
				t.Fatalf("width %d overflows: %q\nVIEW: %q", width, line, view)
			}
		}
		if len(strings.Split(view, "\n")) > 24 {
			t.Fatalf("footer falls below terminal: %d lines at width %d", len(strings.Split(view, "\n")), width)
		}
		model, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = model.(watchModel)
		if m.follow {
			t.Fatal("scroll did not pause follow")
		}
		model, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		m = model.(watchModel)
		if !m.follow {
			t.Fatal("follow toggle failed")
		}
		model, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
		m = model.(watchModel)
		if !m.playerFocus {
			t.Fatal("player focus unreachable")
		}
	}
	s := cliSnapshot{status: portable.RuntimeStatus{State: "running"}}
	if strings.Contains(onlineSummary(s), "players: 0") {
		t.Fatal("unknown online count presented as zero")
	}
}

func TestLocalAdminSessionAndRedirectRefusal(t *testing.T) {
	var tokenSeen bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/login" {
			http.SetCookie(w, &http.Cookie{Name: "admin_token", Value: "fixture-token", Secure: true, HttpOnly: true})
			_, _ = io.WriteString(w, `{"role":"admin"}`)
			return
		}
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "https://example.invalid", http.StatusFound)
			return
		}
		tokenSeen = r.Header.Get("Authorization") == "Bearer fixture-token" && r.Header.Get("X-Requested-With") != ""
		_, _ = io.WriteString(w, `[]`)
	}))
	defer server.Close()
	port, err := strconv.Atoi(strings.TrimPrefix(server.URL, "http://127.0.0.1:"))
	if err != nil {
		t.Fatal(err)
	}
	p := cliInstance(t, port)
	client, err := localAdmin(p, portable.RuntimeStatus{URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	defer client.client.CloseIdleConnections()
	if err := client.request(context.Background(), "POST", "/admin/login", strings.NewReader(`{}`), nil); err != nil {
		t.Fatal(err)
	}
	var players []onlinePlayer
	if err := client.request(context.Background(), "GET", "/admin/players", nil, &players); err != nil {
		t.Fatal(err)
	}
	if !tokenSeen {
		t.Fatal("session/header missing")
	}
	if err := client.request(context.Background(), "GET", "/redirect", nil, nil); err == nil {
		t.Fatal("redirect followed")
	}
	if _, err := localAdmin(p, portable.RuntimeStatus{URL: "http://example.invalid:" + fmt.Sprint(port)}); err == nil {
		t.Fatal("remote credential destination accepted")
	}
}

func TestProgressOnlyCompletesConfirmedStages(t *testing.T) {
	o := newUpdateObserver(t.TempDir(), 100)
	for _, p := range o.phases() {
		if p.complete {
			t.Fatal("invented completed stage")
		}
	}
	work := filepath.Join(o.instance, ".updates", "fixture")
	if err := os.MkdirAll(work, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "download"), make([]byte, 40), 0o600); err != nil {
		t.Fatal(err)
	}
	if p := o.phases()[0]; p.fraction != 0.4 || p.complete {
		t.Fatalf("not measured: %+v", p)
	}
	o.prepared.Store(true)
	p := o.phases()
	if !p[0].complete || !p[1].complete || !p[2].complete || !p[3].skip {
		t.Fatalf("wrong phase transition: %+v", p)
	}
	o.applying.Store(true)
	o.applied.Store(true)
	if p := o.phases()[3]; p.complete || !p.skip {
		t.Fatal("invented health success on stopped world")
	}
	o.wasRunning.Store(true)
	if !o.phases()[3].complete {
		t.Fatal("missed confirmed health gate")
	}
}

func TestWaitCancellationKeepsWorkingUntilResult(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	sp := newWatch(ctx, nil).spinner
	model := waitModel{spinner: sp, cancel: cancel, done: done}
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !result.(waitModel).cancelling {
		t.Fatal("cancel not shown")
	}
	if ctx.Err() == nil {
		t.Fatal("operation not cancelled")
	}
	if result.(waitModel).finished {
		t.Fatal("UI ended before recovery result")
	}
	close(done)
}

func TestTailIsBoundedAndSanitized(t *testing.T) {
	p := filepath.Join(t.TempDir(), "launcher.log")
	if err := os.WriteFile(p, []byte(strings.Repeat("old\n", 20000)+"ERROR final\x1b[31m\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	tail, err := readTail(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(tail) > 64*1024 || !strings.HasSuffix(tail, "ERROR final\n") || strings.Contains(tail, "\x1b") {
		t.Fatal("bad bounded tail")
	}
}

func TestPlainFlagKeepsAnchorOwnedCommandsInAnchor(t *testing.T) {
	instance := desktopTestInstance(t)
	if err := os.MkdirAll(filepath.Join(instance, ".updates"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(instance, ".updates", "active.json"), []byte("broken pointer"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"setup", "init", "_open"} {
		if forwarded, err := forwardActive(command(), []string{"--plain", name}); forwarded || err != nil {
			t.Fatalf("%s escaped anchor: %v %v", name, forwarded, err)
		}
	}
	if _, err := forwardActive(command(), []string{"--plain", "status"}); err == nil {
		t.Fatal("negative control: status did not inspect pointer")
	}
}

func TestWaitDoesNotReturnBeforeCancelledWorkFinishes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := command()
	cmd.SetContext(ctx)
	cmd.SetIn(strings.NewReader(""))
	var out bytes.Buffer
	cmd.SetOut(&out)
	started := make(chan struct{})
	cancelled := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		_, err := runWait(cmd, "Cancellation rehearsal", nil, func(ctx context.Context) (struct{}, error) {
			close(started)
			<-ctx.Done()
			close(cancelled)
			<-release
			return struct{}{}, ctx.Err()
		})
		finished <- err
	}()
	<-started
	cancel()
	<-cancelled
	select {
	case err := <-finished:
		t.Fatalf("returned before recovery ended: %v", err)
	default:
	}
	close(release)
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("lost cancellation: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("terminal did not finish after work returned")
	}
}

func TestUpdateConfirmationCopy(t *testing.T) {
	const closing = "\nThe new release is downloaded and verified before your world stops.\nIf it fails its health check, the current release comes back automatically."
	for _, tc := range []struct {
		name  string
		names []string
		want  string
	}{
		{"empty", nil, "No one is online."},
		{"one", []string{"Aiko"}, "1 player online: Aiko. They'll be disconnected; the character is saved first."},
		{"two", []string{"Aiko", "Frontline"}, "2 players online: Aiko, Frontline. They'll be disconnected; characters are saved first."},
		{"five", []string{"A", "B", "C", "D", "E"}, "5 players online: A, B, C, D, E. They'll be disconnected; characters are saved first."},
		{"six", []string{"A", "B", "C", "D", "E", "F"}, "6 players online: A, B, C, D, E and 1 more. They'll be disconnected; characters are saved first."},
		{"seven", []string{"A", "B", "C", "D", "E", "F", "G"}, "7 players online: A, B, C, D, E and 2 more. They'll be disconnected; characters are saved first."},
		{"terminal controls", []string{"\x1b[31mAiko\x1b[0m"}, "1 player online: Aiko. They'll be disconnected; the character is saved first."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := cliSnapshot{onlineKnown: true}
			for _, name := range tc.names {
				s.online = append(s.online, onlinePlayer{Name: name})
			}
			if got := updateConfirmation(s); got != tc.want+closing {
				t.Fatalf("confirmation = %q, want %q", got, tc.want+closing)
			}
		})
	}
	s := cliSnapshot{playerError: "Sign-in skipped."}
	want := "Couldn't check who's online. Sign-in skipped. Anyone online will be disconnected; characters are saved first." + closing
	if got := updateConfirmation(s); got != want {
		t.Fatalf("unknown players confirmation = %q, want %q", got, want)
	}
}
