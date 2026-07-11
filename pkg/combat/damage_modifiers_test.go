package combat

import "testing"

// TestApplyDamageModifiers exercises the fight.c damage() modifier block
// (src/fight.c:1466-1483) now shared across melee, skill, and spell paths
// (DP-1025). Each case installs a controlled callback set so the affect and
// alignment lookups are deterministic, then restores the prior callbacks.
func TestApplyDamageModifiers(t *testing.T) {
	orig := callbacks
	t.Cleanup(func() { callbacks = orig })

	// A victim with the given affect bits set, plus an attacker alignment map.
	// affects: name -> set of AFF_* bit positions the character "has".
	// aligns:  name -> alignment score.
	// races:   name -> race id (for race-hate); hates: attacker -> []raceIDs.
	newCB := func(affects map[string]map[int]bool, aligns map[string]int,
		races map[string]int, hates map[string][]int,
	) *GameCallbacks {
		return &GameCallbacks{
			HasAffect: func(name string, aff int) bool {
				return affects[name] != nil && affects[name][aff]
			},
			GetAlignment: func(name string) int { return aligns[name] },
			GetRace:      func(name string) int { return races[name] },
			GetRaceHate: func(name string, index int) int {
				h := hates[name]
				if index < len(h) {
					return h[index]
				}
				return -1 // no hate in this slot
			},
		}
	}

	t.Run("sanctuary halves damage", func(t *testing.T) {
		callbacks = newCB(map[string]map[int]bool{
			"vic": {AFF_SANCTUARY: true},
		}, nil, nil, nil)
		ch := &mockCombatant{name: "att", level: 20}
		vic := &mockCombatant{name: "vic", level: 20, npc: true}
		if got := ApplyDamageModifiers(ch, vic, 100); got != 50 {
			t.Errorf("sanctuary: got %d, want 50", got)
		}
	})

	t.Run("protect evil reduces damage from evil attacker", func(t *testing.T) {
		callbacks = newCB(map[string]map[int]bool{
			"vic": {AFF_PROTECT_EVIL: true},
		}, map[string]int{"att": -500}, nil, nil)
		ch := &mockCombatant{name: "att", level: 20}
		vic := &mockCombatant{name: "vic", level: 40, npc: true} // level/4 = 10
		if got := ApplyDamageModifiers(ch, vic, 100); got != 90 {
			t.Errorf("protect-evil: got %d, want 90", got)
		}
		// A good/neutral attacker is unaffected by protect-evil.
		callbacks = newCB(map[string]map[int]bool{
			"vic": {AFF_PROTECT_EVIL: true},
		}, map[string]int{"att": 0}, nil, nil)
		if got := ApplyDamageModifiers(ch, vic, 100); got != 100 {
			t.Errorf("protect-evil vs neutral attacker: got %d, want 100", got)
		}
	})

	t.Run("protect good reduces damage from good attacker", func(t *testing.T) {
		callbacks = newCB(map[string]map[int]bool{
			"vic": {AFF_PROTECT_GOOD: true},
		}, map[string]int{"att": 500}, nil, nil)
		ch := &mockCombatant{name: "att", level: 20}
		vic := &mockCombatant{name: "vic", level: 40, npc: true}
		if got := ApplyDamageModifiers(ch, vic, 100); got != 90 {
			t.Errorf("protect-good: got %d, want 90", got)
		}
	})

	t.Run("sanctuary applies before protection (C order)", func(t *testing.T) {
		// C: dam /= 2 THEN dam -= level/4. 100 -> 50 -> 40.
		// Wrong order (subtract first) would give 100 -> 90 -> 45.
		callbacks = newCB(map[string]map[int]bool{
			"vic": {AFF_SANCTUARY: true, AFF_PROTECT_EVIL: true},
		}, map[string]int{"att": -500}, nil, nil)
		ch := &mockCombatant{name: "att", level: 20}
		vic := &mockCombatant{name: "vic", level: 40, npc: true}
		if got := ApplyDamageModifiers(ch, vic, 100); got != 40 {
			t.Errorf("sanctuary-then-protect: got %d, want 40", got)
		}
	})

	t.Run("race-hate adds attacker level per matching slot", func(t *testing.T) {
		// Victim race 3; attacker hates races {3, 3} in two slots -> +level twice.
		callbacks = newCB(nil, nil,
			map[string]int{"vic": 3},
			map[string][]int{"att": {3, 3, -1, -1, -1}})
		ch := &mockCombatant{name: "att", level: 15}
		vic := &mockCombatant{name: "vic", level: 20, npc: true}
		if got := ApplyDamageModifiers(ch, vic, 100); got != 130 {
			t.Errorf("race-hate x2: got %d, want 130", got)
		}
	})

	t.Run("immortal victim takes zero", func(t *testing.T) {
		callbacks = newCB(nil, nil, nil, nil)
		ch := &mockCombatant{name: "att", level: 20}
		vic := &mockCombatant{name: "vic", level: LVL_IMMORT, npc: false}
		if got := ApplyDamageModifiers(ch, vic, 100); got != 0 {
			t.Errorf("immortal victim: got %d, want 0", got)
		}
		// An NPC at immortal level is NOT immune (the guard is players only).
		vicNPC := &mockCombatant{name: "vic", level: LVL_IMMORT, npc: true}
		if got := ApplyDamageModifiers(ch, vicNPC, 100); got != 100 {
			t.Errorf("immortal-level NPC: got %d, want 100", got)
		}
	})

	t.Run("damage clamps to [0, 3000]", func(t *testing.T) {
		callbacks = newCB(nil, nil, nil, nil)
		ch := &mockCombatant{name: "att", level: 20}
		vic := &mockCombatant{name: "vic", level: 20, npc: true}
		if got := ApplyDamageModifiers(ch, vic, 999999); got != 3000 {
			t.Errorf("cap: got %d, want 3000", got)
		}
		// Protection over-reducing a small hit floors at 0, never negative.
		callbacks = newCB(map[string]map[int]bool{
			"vic": {AFF_PROTECT_EVIL: true},
		}, map[string]int{"att": -500}, nil, nil)
		vicBig := &mockCombatant{name: "vic", level: 100, npc: true} // level/4 = 25
		if got := ApplyDamageModifiers(ch, vicBig, 5); got != 0 {
			t.Errorf("floor: got %d, want 0", got)
		}
	})

	t.Run("nil attacker skips attacker-dependent mods but keeps sanctuary and cap", func(t *testing.T) {
		callbacks = newCB(map[string]map[int]bool{
			"vic": {AFF_SANCTUARY: true},
		}, nil, nil, nil)
		vic := &mockCombatant{name: "vic", level: 20, npc: true}
		if got := ApplyDamageModifiers(nil, vic, 100); got != 50 {
			t.Errorf("nil attacker + sanctuary: got %d, want 50", got)
		}
		if got := ApplyDamageModifiers(nil, vic, 999999); got != 3000 {
			t.Errorf("nil attacker + cap: got %d, want 3000", got)
		}
	})
}
