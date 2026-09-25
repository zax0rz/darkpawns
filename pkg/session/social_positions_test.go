package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// A mobile's socials (game.NPCCommand) are gated by the same C
// minimum_position as a player's, which command_gates.tsv holds.
func TestSocialMinPositionsMatchCommandGates(t *testing.T) {
	checked := 0
	for name := range game.Socials {
		gate, ok := commandGates[name]
		if !ok || !strings.HasPrefix(gate.Source, "C ") {
			continue // not a C command: a mobile's "Huh?!?" reaches no one
		}
		position, ok := game.SocialMinPosition(name)
		if !ok && name == "roll" {
			continue // C's "roll" row is do_roll, not do_action (interpreter.c:664)
		}
		if !ok {
			t.Errorf("social %q has a C gate (position %d) but no NPC minimum position", name, gate.MinPosition)
			continue
		}
		if position != gate.MinPosition {
			t.Errorf("social %q: NPC minimum position %d, C gate %d", name, position, gate.MinPosition)
		}
		checked++
	}
	if checked < 150 {
		t.Fatalf("checked only %d socials against command_gates.tsv", checked)
	}
}
