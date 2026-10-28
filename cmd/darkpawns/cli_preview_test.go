//go:build clipreview

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/zax0rz/darkpawns/internal/portable"
)

// TestCLIPreview is an opt-in terminal rehearsal. Every fixture is explicitly
// synthetic; no server, instance, credential or update is mutated.
func TestCLIPreview(t *testing.T) {
	mode := os.Getenv("DP_CLI_PREVIEW")
	if mode == "" {
		t.Skip("opt-in VHS rehearsal")
	}
	cmd := command()
	cmd.SetIn(os.Stdin)
	cmd.SetOut(os.Stdout)
	cmd.SetContext(context.Background())
	switch mode {
	case "watch":
		load := func(context.Context) cliSnapshot {
			return cliSnapshot{fixture: "SYNTHETIC REHEARSAL", status: portable.RuntimeStatus{State: "running", Instance: "/examples/wyldlands"}, version: "v0.1.7", uptime: "3h 42m", onlineKnown: true, online: []onlinePlayer{{"Frontline", 40, 8004}, {"Aiko", 20, 8021}, {"Adventurer", 8, 8100}}, logs: strings.Join([]string{"INFO  18:01:02 World ready; 91 zones loaded", "INFO  18:01:14 Aiko has connected", "WARN  18:02:00 Synthetic disk retry recovered", "ERROR 18:02:12 Synthetic script failure: guard.lua:18", "INFO  18:02:19 Zone 80 reset complete", "INFO  18:02:25 Adventurer has connected"}, "\n"), errors: []string{"ERROR 18:02:12 Synthetic script failure: guard.lua:18"}, at: time.Now()}
		}
		_, err := tea.NewProgram(newWatch(context.Background(), load), tea.WithInput(os.Stdin), tea.WithOutput(os.Stdout), tea.WithAltScreen()).Run()
		if err != nil {
			t.Fatal(err)
		}
	case "progress":
		var step atomic.Int64
		phases := func() []phaseView {
			n := step.Load()
			names := []string{"Download", "Verify", "Stage", "Health gate"}
			out := make([]phaseView, 4)
			for i, name := range names {
				out[i] = phaseView{name: name, detail: "Synthetic UI rehearsal; no update is running", known: true, complete: n > int64(i*10+10)}
				if n >= int64(i*10) && n < int64(i*10+10) {
					out[i].fraction = float64(n-int64(i*10)) / 10
				}
			}
			return out
		}
		_, err := runWait(cmd, "Synthetic update rehearsal", phases, func(ctx context.Context) (struct{}, error) {
			for i := 0; i <= 42; i++ {
				select {
				case <-ctx.Done():
					return struct{}{}, ctx.Err()
				case <-time.After(120 * time.Millisecond):
					step.Store(int64(i))
				}
			}
			return struct{}{}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	case "confirm":
		_, err := confirmAction(cmd, "Restart this synthetic instance?", "Online players: 3\nFrontline, Aiko, Adventurer\nThis rehearsal never starts or stops a world.")
		if err != nil {
			t.Fatal(err)
		}
	case "instances":
		_, err := pickInstance(cmd, []list.Item{instanceItem{"/examples/wyldlands"}, instanceItem{"/examples/staging"}, instanceItem{"/examples/test-world"}})
		if err != nil && err != errCancelled {
			t.Fatal(err)
		}
	case "check":
		if err := checklist(cmd.OutOrStdout(), "Synthetic release check", []portable.Check{{Name: "Release v0.1.6 → v0.1.7", OK: true, Detail: "Rehearsal only; no release verified"}, {Name: "Active world", OK: true, Detail: "No world was stopped"}}, cliWidth(cmd)); err != nil {
			t.Fatal(err)
		}
		fallthrough
	case "notes":
		if err := showReleaseNotes(cmd, releaseNotes{Tag: "v0.1.7", Body: "# v0.1.7 · synthetic release notes\n\n## Operator tools\n\n- Live world status and searchable instances.\n- Readable diagnostics and confirmed update stages.\n\n**Your world stays running during preparation.**\n"}, "v0.1.7"); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal(fmt.Sprintf("unknown rehearsal %q", mode))
	}
}
