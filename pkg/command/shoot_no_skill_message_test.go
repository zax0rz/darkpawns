package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func TestShootNoSkillMessagePath(t *testing.T) {
	s, _, v, _, wire := shootOutcomeFixture(t, false, 200)
	old := combat.GetCallbacks()
	t.Cleanup(func() { combat.SetCallbacks(old) })
	cb := s.world.WireCombatCallbacks()
	cb.SkillMessage = func(int, combat.Combatant, combat.Combatant, int, int) bool {
		t.Fatal("shoot called skill_message")
		return false
	}
	combat.SetCallbacks(cb)
	dprng.ResetStream(71)
	fireOutcome(t, s, v)
	if v.GetFighting() != "" || s.player.GetFighting() != "" {
		t.Fatal("shoot entered generic damage combat")
	}
	if strings.Contains(strings.Join(s.getMessages(), "")+wire["Victim"], "HURT") {
		t.Fatal("shoot used damage messages")
	}
}
