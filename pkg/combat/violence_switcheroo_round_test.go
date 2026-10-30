package combat

import "testing"

// countingThenOneRoller returns the scripted leading values, then 1 forever.
type countingThenOneRoller struct {
	lead  []int
	calls int
}

func (r *countingThenOneRoller) Number(_, _ int) int {
	i := r.calls
	r.calls++
	if i < len(r.lead) {
		return r.lead[i]
	}
	return 1
}

func (r *countingThenOneRoller) Dice(_, _ int) int { return 1 }
func (r *countingThenOneRoller) IntN(int) int      { return 0 }

// TestRoundContinuesAfterSameTargetSwitcheroo pins C's attack loop shape:
// perform_violence's while loop never exits on damage()'s return value —
// every iteration re-reads FIGHTING(ch) (src/fight.c:2017-2024). A switcheroo
// whose scan picks the current defender (a duel: vict is whoever FIGHTINGs
// ch, src/fight.c:1413-1440) re-enrolls the same target and the round keeps
// its remaining attacks. Breaking on the redirect's return ended the whole
// round after one swing, which the teleport_victim NPC-refusal vehicle
// exposed as a draw-stream split on its second violence round.
func TestRoundContinuesAfterSameTargetSwitcheroo(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	dragon := &mockCombatant{name: "dragon", npc: true, room: 20, level: 30, hp: 200, maxHP: 200, position: PosFighting}
	tank := &mockCombatant{name: "Tank", room: 20, level: 20, hp: 100, maxHP: 100, position: PosFighting}

	ce := NewCombatEngine()
	swingCount := 0
	ce.MessageFunc = func(_, _ Combatant, _, _ int) bool {
		swingCount++
		return true
	}
	ce.SetCallbacks(&GameCallbacks{
		GetRoomCombatants: func(room int) []Combatant {
			return []Combatant{dragon, tank}
		},
	})
	if err := ce.StartCombat(dragon, tank); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	// Leading zeros: the attacks bonus (0,900), the first swing's to-hit,
	// and the first switcheroo scan (0,80) — the redirect fires on swing one
	// and re-enrolls the same defender. Everything after is 1: to-hits that
	// land and switcheroo scans that decline.
	roller := &countingThenOneRoller{lead: []int{0, 0, 0}}
	WithRoller(roller, func() {
		ce.processCombatPair(ce.combatPairs[CombatPairKey{Attacker: dragon, Target: tank}])
	})

	// A level-30 NPC round is 4 attacks (+1 from the zero bonus draw). The
	// first iteration produced two swing messages (the refused original plus
	// the redirect's replacement); the remaining iterations one each — six
	// total. The pre-fix break produced exactly the first two.
	if swingCount < 5 {
		t.Fatalf("round produced %d swing messages, want >= 5: the switcheroo ended the round early", swingCount)
	}
	if dragon.GetFightingBody() != tank {
		t.Fatalf("duel switcheroo lost its target: fighting=%v", dragon.GetFightingBody())
	}
}
