package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestMlistRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["mlist"]
	if !ok {
		t.Fatal("mlist command has no C gate")
	}
	if gate.MinLevel != LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("mlist gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_IMMORT, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("mlist")
	if !ok {
		t.Fatal("mlist command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("mlist registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestMlistAtoiMatchesCDecimalPrefix(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int
	}{
		{input: "", want: 0},
		{input: "abc", want: 0},
		{input: "183abc", want: 183},
		{input: "+183", want: 183},
		{input: "-1", want: -1},
	} {
		if got := mlistAtoi(tc.input); got != tc.want {
			t.Errorf("mlistAtoi(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
