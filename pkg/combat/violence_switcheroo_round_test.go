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

// TestRoundRetargetKeepsRemainingAttacks pins C's mid-round retarget
// (src/fight.c:2017-2024): a switcheroo that picks a DIFFERENT opponent
// (src/fight.c:1413-1440) does not end the round — the remaining attacks
// swing at the new FIGHTING. The dragon redirects from the tank to the rogue
// on the first swing; the round's later swings must land on the rogue.
func TestRoundRetargetKeepsRemainingAttacks(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	dragon := &mockCombatant{name: "dragon", npc: true, room: 20, level: 30, hp: 200, maxHP: 200, position: PosFighting}
	tank := &mockCombatant{name: "Tank", room: 20, level: 20, hp: 500, maxHP: 500, position: PosFighting}
	rogue := &mockCombatant{name: "Rogue", room: 20, level: 20, hp: 500, maxHP: 500, position: PosFighting}

	ce := NewCombatEngine()
	swings := map[string]int{}
	ce.MessageFunc = func(_, defender Combatant, _, _ int) bool {
		swings[defender.GetName()]++
		return true
	}
	ce.SetCallbacks(&GameCallbacks{
		GetRoomCombatants: func(room int) []Combatant {
			// The scan sees the tank first; the scripted zero picks the tank's
			// OWN switcheroo roll... order the rogue FIRST so the redirect
			// lands on the rogue (fight.c iterates the room's people list).
			return []Combatant{rogue, dragon, tank}
		},
	})
	if err := ce.StartCombat(dragon, tank); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}
	rogue.SetFightingBody(dragon)

	// Leading zeros: attacks bonus (0,900), first swing's to-hit, first
	// switcheroo scan roll (0,80 -> the redirect fires to the rogue). Then
	// 1s: to-hits that land and scans that decline, so the round's later
	// swings stay on the rogue.
	roller := &countingThenOneRoller{lead: []int{0, 0, 0}}
	WithRoller(roller, func() {
		ce.processCombatPair(ce.combatPairs[CombatPairKey{Attacker: dragon, Target: tank}])
	})

	if dragon.GetFightingBody() != rogue {
		t.Fatalf("dragon fighting = %v, want the rogue after the redirect", dragon.GetFightingBody())
	}
	// A level-30 NPC round is 4 attacks (+1 from the zero bonus draw). The
	// redirect consumed the first; the remaining swings belong to the rogue.
	if swings["Rogue"] < 3 {
		t.Fatalf("rogue took %d swings after the redirect, want >= 3: the retarget ended the round", swings["Rogue"])
	}
	if swings["Tank"] > 2 {
		t.Fatalf("tank took %d swings, want only the pre-redirect ones", swings["Tank"])
	}
}
