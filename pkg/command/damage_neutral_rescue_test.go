package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSkillCommandNeutralRescueBeforeMessage(t *testing.T) {
	for _, path := range []string{"generic", "complete", "zero"} {
		t.Run(path, func(t *testing.T) {
			w, err := game.NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Neutral arena", Flags: []string{"neutral"}}, {VNum: 8004, Name: "Rescue destination"}}})
			if err != nil {
				t.Fatal(err)
			}
			w.StopAITicker()
			t.Cleanup(w.StopAITicker)
			ch := game.NewPlayer(1, "Caster", 1001)
			ch.SetLevel(30)
			vict := game.NewPlayer(2, "Victim", 1001)
			vict.SetLevel(15)
			vict.SetHP(1)
			for _, p := range []*game.Player{ch, vict} {
				if err := w.AddPlayer(p); err != nil {
					t.Fatal(err)
				}
			}
			ce := combat.NewCombatEngine()
			w.SetCombatEngine(ce)
			old := combat.GetCallbacks()
			t.Cleanup(func() { combat.SetCallbacks(old) })
			cb := w.WireCombatCallbacks()
			messages := 0
			cb.SkillMessage = func(int, combat.Combatant, combat.Combatant, int, int) bool { messages++; return true }
			ce.SetCallbacks(cb)
			sess := &killPayoutSession{player: ch, world: w, combatEngine: ce}
			res := game.SkillResult{Damage: 40, DamageSkill: game.SkillBash, SkillMsgType: game.SkillBashNum, StartCombat: true, TargetFalls: true, EffectsNeedDamage: true, WaitTarget: 2}
			if path == "complete" {
				res.DamageSkill = game.SkillDragonKick
				res.SkillMsgType = game.SkillDragonKickNum
				res.SkillMsgInDamage = true
			}
			if path == "zero" {
				res.Damage = 0
				vict.SetHP(0)
				vict.SetPosition(combat.PosStunned)
			}
			if err := sendSkillResult(sess, ch, vict, res); err != nil {
				t.Fatal(err)
			}
			if messages != 0 || vict.GetHP() != 1 || vict.GetRoom() != 8004 || vict.GetPosition() != combat.PosStanding || vict.GetWaitState() != 0 {
				t.Fatalf("messages=%d HP=%d room=%d pos=%d wait=%d", messages, vict.GetHP(), vict.GetRoom(), vict.GetPosition(), vict.GetWaitState())
			}
			// C does not enroll an already-stunned victim. Its one-way attacker
			// remains fighting until the next violence turn notices the room change
			// (src/fight.c:1498-1502; src/handler.c:512-518).
			if vict.GetFightingBody() != nil || path != "zero" && ch.GetFightingBody() != nil {
				t.Fatal("command re-enrolled rescued combatants")
			}
			if _, ok := ce.GetCombatTarget(ch); ok && path != "zero" {
				t.Fatal("command registered a post-rescue pair")
			}
		})
	}
}
