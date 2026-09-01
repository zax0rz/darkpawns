package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestLinesRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["lines"]
	if !ok {
		t.Fatal("lines command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosDead {
		t.Fatalf("lines gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosDead)
	}
}

func TestParseLinesSizeMirrorsCAtoi(t *testing.T) {
	tests := map[string]int{
		"foo":   0,
		"25abc": 25,
		"-8":    -8,
		"+8":    8,
		"007":   7,
	}
	for input, want := range tests {
		if got := parseLinesSize(input); got != want {
			t.Errorf("parseLinesSize(%q) = %d, want %d", input, got, want)
		}
	}
}
