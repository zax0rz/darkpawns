package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestWiznetRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("wiznet")
	if !ok {
		t.Fatal("wiznet command is not registered")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosDead {
		t.Fatalf("wiznet gate = level %d position %d, want level 0 position dead", entry.MinLevel, entry.MinPosition)
	}
}

func TestCmdWiznetEntryAllowsChosenMortal(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Chosenwiz", 1, 1001)
	m.sessions[actor.player.Name] = actor

	if err := cmdWiznetText(actor, "ordinary"); err != nil {
		t.Fatalf("ordinary wiznet: %v", err)
	}
	if got, want := readSessionText(t, actor), "Huh?!?"; got != want {
		t.Fatalf("unchosen mortal output = %q, want %q", got, want)
	}

	actor.player.SetPLRFlag(game.PlrChosen)
	if err := cmdWiznetText(actor, "chosen message"); err != nil {
		t.Fatalf("chosen wiznet: %v", err)
	}
	if got, want := readSessionText(t, actor), "Chosenwiz: chosen message\r\n"; got != want {
		t.Fatalf("chosen mortal output = %q, want %q", got, want)
	}
}

func TestWiznetHalfChopMatchesCWordBoundary(t *testing.T) {
	cases := []struct {
		input, first, remainder string
	}{
		{"  32  message  ", "32", "message  "},
		{"message", "message", ""},
		{"", "", ""},
	}
	for _, tc := range cases {
		first, remainder := wiznetHalfChop(tc.input)
		if first != tc.first || remainder != tc.remainder {
			t.Errorf("wiznetHalfChop(%q) = (%q, %q), want (%q, %q)", tc.input, first, remainder, tc.first, tc.remainder)
		}
	}
}
