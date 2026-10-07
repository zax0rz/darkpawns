package session

import (
	"os"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestReportMudlogRawBytesAndOrder(t *testing.T) {
	for _, command := range []string{"bug", "typo", "idea", "todo"} {
		for _, blocked := range []bool{false, true} {
			t.Run(command+map[bool]string{false: "/success", true: "/open-failure"}[blocked], func(t *testing.T) {
				t.Chdir(t.TempDir())
				m := makeTestManager(t)
				actor := makeCommandTestSession(t, m, "Reporter", 40, 1001)
				actor.player.SetInvisLevel(40)
				watch := makeCommandTestSession(t, m, "Watcher", 31, 1001)
				watch.player.SetPlrFlag(game.PrfLog1, true)
				watch.player.SetPlrFlag(game.PrfLog2, true)
				below := makeCommandTestSession(t, m, "Below", 30, 1001)
				below.player.SetPlrFlag(game.PrfLog1, true)
				below.player.SetPlrFlag(game.PrfLog2, true)
				normal := makeCommandTestSession(t, m, "Normal", 40, 1001)
				normal.player.SetPlrFlag(game.PrfLog2, true)
				for _, s := range []*Session{actor, watch, below, normal} {
					registerTestSession(t, m, s, s.playerName)
				}
				if blocked {
					if err := os.WriteFile("misc", []byte("obstruction"), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				file := captureMudlogFile(t)
				sink := m.world.MessageSink
				calls := 0
				m.world.MessageSink = func(name string, b []byte) {
					if name == "Watcher" {
						calls++
						if got := strings.Join(drainSessionText(t, actor), ""); got != "" {
							t.Errorf("report acknowledgement precedes log: %q", got)
						}
						if !blocked {
							if _, err := os.Stat("misc"); !os.IsNotExist(err) {
								t.Error("producer must precede directory/file write")
							}
						}
					}
					sink(name, b)
				}
				if err := executeCommandRaw(actor, command, []string{"two", "words", "$$"}, false, " \ttwo  words $$  "); err != nil {
					t.Fatal(err)
				}
				want := "[ Reporter " + command + ": two  words $$   ]\r\n"
				if got := strings.Join(drainSessionText(t, watch), ""); got != want {
					t.Fatalf("report log = %q, want %q", got, want)
				}
				if calls != 1 {
					t.Fatalf("producer count=%d, want 1", calls)
				}
				for _, s := range []*Session{below, normal} {
					if got := strings.Join(drainSessionText(t, s), ""); got != "" {
						t.Errorf("recipient gate leaked: %q", got)
					}
				}
				if strings.Contains(file.String(), "Reporter "+command+":") {
					t.Fatal("report producer must be file FALSE")
				}
				wantAck := "Okay.  Thanks!\r\n"
				if blocked {
					wantAck = "Could not open the file.  Sorry.\r\n"
				}
				if got := strings.Join(drainSessionText(t, actor), ""); got != wantAck {
					t.Errorf("report acknowledgement=%q, want %q", got, wantAck)
				}
			})
		}
	}
}

func TestReportMudlogTokenizedDollars(t *testing.T) {
	t.Chdir(t.TempDir())
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Reporter", 40, 1001)
	watch := makeCommandTestSession(t, m, "Watcher", 31, 1001)
	watch.player.SetPlrFlag(game.PrfLog1, true)
	watch.player.SetPlrFlag(game.PrfLog2, true)
	for _, s := range []*Session{actor, watch} {
		registerTestSession(t, m, s, s.playerName)
	}
	for _, command := range []string{"bug", "typo", "idea", "todo"} {
		if err := ExecuteCommand(actor, command, []string{"literal", "$$"}); err != nil {
			t.Fatal(err)
		}
		if got, want := strings.Join(drainSessionText(t, watch), ""), "[ Reporter "+command+": literal $$ ]\r\n"; got != want {
			t.Fatalf("tokenized %s = %q, want %q", command, got, want)
		}
		drainSessionText(t, actor)
	}
}
