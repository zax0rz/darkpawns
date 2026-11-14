package combat

// DP-1530-batch: fight.c attitude_loot's gossip quips (src/fight.c:1176-1245).
// The brag fires on a 1-in-21 mob-kill gate plus a 1-in-12 quip draw, so the
// individual quip strings are long-shot rolls the oracle census cannot target:
// they are pinned here with a scripted roller, one subtest per L1 site, and
// C's mob-victim skips (cases 7-11 return without speaking) are pinned too.

import (
	"testing"
)

func bragQuipCase(t *testing.T, name string, draw int, ch, victim *mockCombatant, want string) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		oldRoller := GetRoller()
		t.Cleanup(func() { SetRoller(oldRoller) })
		SetRoller(NewScriptedRoller([]int{draw}))
		got := pickBragMessage(ch, victim.GetName(), victim.IsNPC(), victim.GetSex(), victimEvilForBrag(victim))
		if got != want {
			t.Errorf("quip case %d = %q, want %q (src/fight.c)", draw, got, want)
		}
	})
}

func victimEvilForBrag(v *mockCombatant) bool {
	return cbGetAlignment(v) <= -350
}

func TestPickBragMessageQuipStrings(t *testing.T) {
	oldCB := GetCallbacks()
	t.Cleanup(func() { SetCallbacks(oldCB) })
	cb := &GameCallbacks{
		GetAlignment: func(c Combatant) int {
			if c.GetName() == "Evilbrag" {
				return -400
			}
			return 0
		},
		GetKills: func(c Combatant) int64 {
			return 7
		},
	}
	SetCallbacks(cb)

	ch := &mockCombatant{name: "Bragger", npc: true}
	victim := &mockCombatant{name: "Thug", npc: true, sex: 0}
	evilCh := &mockCombatant{name: "Evilbrag", npc: true}
	playerVictim := &mockCombatant{name: "Mortal", npc: false, sex: 0}

	// fight.c:1183 — case 2.
	bragQuipCase(t, "case2-tough-good-eq", 2, ch, victim, "Thug was tough, but had good eq...")
	// fight.c:1187 — case 3.
	bragQuipCase(t, "case3-easy-xp", 3, ch, victim, "Thug was easy xp.")
	// fight.c:1191 — case 4.
	bragQuipCase(t, "case4-muhahahaha", 4, ch, victim, "Muhahahaha... Thug is dead!")
	// fight.c:1196 — case 5, evil killer against a non-evil victim.
	bragQuipCase(t, "case5-evil-triumphs", 5, evilCh, victim,
		"Now you will see that evil will always triumph, because good is dumb.")
	// fight.c:1209 — case 7, player victim only.
	bragQuipCase(t, "case7-innocent-stare", 7, ch, playerVictim, "Oh, did that hurt, Mortal? *innocent stare*")
	// fight.c:1216 — case 8.
	bragQuipCase(t, "case8-newbie-training", 8, ch, playerVictim, "What the hell was Mortal doing out of newbie training?")
	// fight.c:1223 — case 9.
	bragQuipCase(t, "case9-fertilizer", 9, ch, playerVictim, "I think I finally found a use for that punk Mortal: fertilizer!")
	// fight.c:1230 — case 10.
	bragQuipCase(t, "case10-your-head", 10, ch, playerVictim, "Hrmm.. Is this your head, Mortal? *cackle*")
	// fight.c:1238 — case 11.
	bragQuipCase(t, "case11-suicide-or-fight", 11, ch, playerVictim, "Hey Mortal, was that suicide or did you try to fight back?")
}

// TestPickBragMessageMobVictimSkipsPlayerQuips pins the C mob-victim returns
// in cases 7-11 (src/fight.c:1207-1242): the mob-killer says nothing.
func TestPickBragMessageMobVictimSkipsPlayerQuips(t *testing.T) {
	oldRoller := GetRoller()
	t.Cleanup(func() { SetRoller(oldRoller) })
	ch := &mockCombatant{name: "Bragger", npc: true}
	victim := &mockCombatant{name: "Thug", npc: true, sex: 0}
	for _, draw := range []int{7, 8, 9, 10, 11} {
		SetRoller(NewScriptedRoller([]int{draw}))
		if got := pickBragMessage(ch, victim.GetName(), victim.IsNPC(), victim.GetSex(), false); got != "" {
			t.Errorf("case %d with a mob victim = %q, want empty (C returns before speaking)", draw, got)
		}
	}
}
