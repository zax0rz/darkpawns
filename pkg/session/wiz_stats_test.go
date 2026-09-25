package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// TestStatPlayer_DefaultPositionStanding is the DP-1332 regression: C's
// do_stat_character prints "Default position: Standing" for all characters
// (using mob_specials.default_pos, which clear_char initializes to POS_STANDING
// even for players). The port was using the player's current position instead.
func TestStatPlayer_DefaultPositionStanding(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)
	s.player.SetPosition(combat.PosResting)

	s.sendStatPlayerReport(s.player, 1001, true, false)

	var texts []string
	for {
		select {
		case msg := <-s.send:
			if IsInputMarkFrame(msg) {
				continue
			}
			texts = append(texts, string(msg))
		default:
			goto drain
		}
	}
drain:

	found := false
	for _, raw := range texts {
		if strings.Contains(raw, "Default position:") {
			found = true
			if strings.Contains(raw, "Default position: Resting") {
				t.Errorf("stat used current position for Default position; want Standing (C's clear_char default).\n  raw=%s", raw)
			}
			if !strings.Contains(raw, "Default position: Standing") {
				t.Errorf("Default position line missing 'Standing':\n  raw=%s", raw)
			}
			break
		}
	}
	if !found {
		t.Fatal("stat output did not contain a 'Default position:' line")
	}
}
