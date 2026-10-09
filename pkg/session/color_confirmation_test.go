package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// src/act.informative.c:2494-2495 evaluates the macros after setting flags.
func TestColorConfirmationNewLevelANSI(t *testing.T) {
	for _, old := range []int{0, 3} {
		for level, name := range []string{"off", "sparse", "normal", "complete"} {
			t.Run(name+map[int]string{0: "-from-off", 3: "-from-complete"}[old], func(t *testing.T) {
				s := makeTestSession(t, makeTestManager(t), "Viewer", 1001, true)
				s.player.SetPlrFlag(game.PrfColor1, old&1 != 0)
				s.player.SetPlrFlag(game.PrfColor2, old&2 != 0)
				if err := cmdColor(s, []string{name}); err != nil {
					t.Fatal(err)
				}
				flags := s.player.GetFlags()
				if (flags&(1<<uint(game.PrfColor1)) != 0) != (level&1 != 0) || (flags&(1<<uint(game.PrfColor2)) != 0) != (level&2 != 0) {
					t.Fatal("new color flags incorrect")
				}
				msg := drainMsg(t, s)
				raw, ok := RenderTerminalFrame(msg)
				red := ""
				if level > 0 {
					red = "\x1b[31m"
				}
				want := "Your " + red + "color\x1b[0m is now " + name + ".\r\n"
				if !ok || raw.Text != want {
					t.Fatalf("confirmation=%q want=%q", raw.Text, want)
				}
				expanded, ok := s.RenderTerminalFrame(msg)
				if !ok || expanded.Text != want {
					t.Fatalf("terminal changed raw ANSI: %q want=%q", expanded.Text, want)
				}
			})
		}
	}
}
