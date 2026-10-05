package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCombatBodySessionCleanup(t *testing.T) {
	for _, kind := range []string{"disconnect", "peace", "remove"} {
		t.Run(kind, func(t *testing.T) {
			m, wizard, peer, _, first, second := combatDescriptorFixture(t)
			// The descriptor fixture lends Wizard an NPC; ordinary cleanup here owns
			// the actual PC, while unexpected disconnect retains the world bodies.
			m.mu.Lock()
			wizard.isSwitched = false
			wizard.switchedMob = nil
			m.mu.Unlock()
			for _, pair := range [][2]combat.Combatant{{first, wizard.player}, {second, peer.player}} {
				if err := m.combatEngine.StartCombat(pair[0], pair[1]); err != nil {
					t.Fatal(err)
				}
			}
			switch kind {
			case "disconnect":
				if !m.HandleTransportDisconnect(wizard) {
					t.Fatal("playing body not retained")
				}
				if first.GetFightingBody() != wizard.player || wizard.player.GetFightingBody() != first {
					t.Fatal("disconnect stopped retained fight")
				}
			case "peace":
				wizard.player.SetLevel(game.LVL_IMPL)
				if err := cmdDark(wizard, nil); err != nil {
					t.Fatal(err)
				}
				for _, b := range []combat.Combatant{first, second, wizard.player, peer.player} {
					if b.GetFightingBody() != nil || m.combatEngine.IsFighting(b) {
						t.Fatal("peace left body fighting")
					}
				}
			case "remove":
				m.UnregisterSession(wizard)
				if first.GetFightingBody() != nil || wizard.player.GetFightingBody() != nil {
					t.Fatal("cleanup retained stale fight")
				}
				if second.GetFightingBody() != peer.player || peer.player.GetFightingBody() != second {
					t.Fatal("cleanup erased duplicate fight")
				}
			}
		})
	}
}

func TestCombatBodySwitchedDisconnect(t *testing.T) {
	m, s, h, original := switchedPCFixture(t)
	body := h.player
	// These two concrete PCs have independent fights; lost transport detaches
	// the descriptor without retiring either the original or borrowed body.
	original.SetFightingBody(body)
	body.SetFightingBody(original)
	if !m.HandleTransportDisconnect(s) {
		t.Fatal("not retained")
	}
	if original.GetFightingBody() != body || body.GetFightingBody() != original {
		t.Fatal("disconnect retired a switched identity")
	}
}
