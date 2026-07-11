package game

import "testing"

// Tier-2 fidelity golden test (deterministic — no RNG, no game-code change).
//
// These tests pin the deterministic portions of the regen functions (hit_gain, mana_gain,
// move_gain) from src/limits.c. We test the position/level multipliers and class modifiers
// that don't involve RNG. Anything that rolls dice or depends on equipment is excluded.
//
// Ground truth: src/limits.c hit_gain(), mana_gain(), move_gain()

// ---------------------------------------------------------------------------
// hit_gain deterministic portions
// ---------------------------------------------------------------------------

// TestHitGainNPC_Golden tests the NPC branch of hit_gain from src/limits.c lines 133-137.
// NPC regen: level < 23 → 2.5*level, level >= 23 → 4.0*level
func TestHitGainNPC_Golden(t *testing.T) {
	tests := []struct {
		level int
		want  int
	}{
		{1, 2},    // 2.5*1 = 2.5 → int truncation in C gives 2
		{10, 25},  // 2.5*10 = 25
		{22, 55},  // 2.5*22 = 55
		{23, 92},  // 4*23 = 92
		{30, 120}, // 4*30 = 120
		{40, 160}, // 4*40 = 160
	}
	for _, tt := range tests {
		m := &MobInstance{Level: tt.level}
		got := MobHitGain(m)
		if got != tt.want {
			t.Errorf("MobHitGain(level=%d) = %d, want %d (per src/limits.c)", tt.level, got, tt.want)
		}
	}
}

// TestHitGain_BaseGain tests the base gain for PCs: 20 (src/limits.c line 140).
// We test a standing, non-veteran, non-poisoned, not hungry player in a non-regen room.
// This requires a minimal Player setup.
func TestHitGain_BaseGain(t *testing.T) {
	p := &Player{
		Class:    ClassWarrior,
		Level:    10,
		Position: PosStanding,
		Conditions: [3]int{
			CondFull:   20,
			CondThirst: 20,
			CondDrunk:  0,
		},
	}
	w := &World{}
	// No room flags, no affects, no equipment — just the base.
	got := w.HitGain(p)
	// Base 20, no modifiers: standing (no position bonus), no class penalty for warrior.
	// From C: gain = 20; warrior has no hit regen penalty (only mage/cleric do).
	want := 20
	if got != want {
		t.Errorf("HitGain(warrior L10 standing) = %d, want %d (per src/limits.c)", got, want)
	}
}

// TestHitGain_PositionMultipliers tests the position-based multipliers from src/limits.c lines 149-167.
// Sleeping: +50% (gain += gain>>1), Resting: +25% (gain += gain>>2), Sitting: +12.5% (gain += gain>>3)
func TestHitGain_PositionMultipliers(t *testing.T) {
	base := 20
	tests := []struct {
		pos  int
		want int
	}{
		{PosStanding, base},               // no multiplier
		{PosFighting, base},               // no multiplier (same as standing)
		{PosSitting, base + (base >> 3)},  // 20 + 2 = 22
		{PosResting, base + (base >> 2)},  // 20 + 5 = 25
		{PosSleeping, base + (base >> 1)}, // 20 + 10 = 30
	}
	for _, tt := range tests {
		p := &Player{
			Class:    ClassWarrior,
			Level:    10,
			Position: tt.pos,
			Conditions: [3]int{
				CondFull:   20,
				CondThirst: 20,
				CondDrunk:  0,
			},
		}
		w := &World{}
		got := w.HitGain(p)
		if got != tt.want {
			t.Errorf("HitGain(pos=%d) = %d, want %d (per src/limits.c position multipliers)", tt.pos, got, tt.want)
		}
	}
}

// TestHitGain_MageClericPenalty tests the mage/cleric HP regen penalty from src/limits.c lines 171-173.
// Mage and cleric: gain >>= 1 (halved).
func TestHitGain_MageClericPenalty(t *testing.T) {
	for _, class := range []int{ClassMageUser, ClassCleric} {
		p := &Player{
			Class:    class,
			Level:    10,
			Position: PosStanding,
			Conditions: [3]int{
				CondFull:   20,
				CondThirst: 20,
				CondDrunk:  0,
			},
		}
		w := &World{}
		got := w.HitGain(p)
		// Base 20, halved = 10
		want := 10
		if got != want {
			t.Errorf("HitGain(class=%d, standing) = %d, want %d (mage/cleric halved)", class, got, want)
		}
	}
}

// TestHitGain_PoisonPenalty tests the poison HP regen penalty from src/limits.c lines 176-178.
// Poisoned: gain >>= 2 (quartered).
func TestHitGain_PoisonPenalty(t *testing.T) {
	p := &Player{
		Class:    ClassWarrior,
		Level:    10,
		Position: PosStanding,
		Affects:  1 << AffPoison,
		Conditions: [3]int{
			CondFull:   20,
			CondThirst: 20,
			CondDrunk:  0,
		},
	}
	w := &World{}
	got := w.HitGain(p)
	// Base 20, quartered by poison = 5
	want := 5
	if got != want {
		t.Errorf("HitGain(poisoned) = %d, want %d (per src/limits.c poison penalty)", got, want)
	}
}

// ---------------------------------------------------------------------------
// mana_gain deterministic portions
// ---------------------------------------------------------------------------

// TestManaGainNPC_Golden tests the NPC branch of mana_gain from src/limits.c line 62.
// NPC mana regen: gain = GET_LEVEL(ch)
func TestManaGainNPC_Golden(t *testing.T) {
	m := &MobInstance{Level: 25}
	got := ManaGainNPC(m)
	if got != 25 {
		t.Errorf("ManaGainNPC(level=25) = %d, want 25 (per src/limits.c)", got)
	}
}

// TestManaGain_BaseGain tests the base gain for PCs: 14 (src/limits.c line 65).
func TestManaGain_BaseGain(t *testing.T) {
	p := &Player{
		Class:    ClassWarrior,
		Level:    10,
		Position: PosStanding,
		Conditions: [3]int{
			CondFull:   20,
			CondThirst: 20,
			CondDrunk:  0,
		},
	}
	w := &World{}
	got := w.ManaGain(p)
	// Base 14, no position multiplier (standing), no class multiplier (warrior).
	want := 14
	if got != want {
		t.Errorf("ManaGain(warrior L10 standing) = %d, want %d (per src/limits.c)", got, want)
	}
}

// TestManaGain_PositionMultipliers tests the position-based multipliers from src/limits.c lines 72-85.
// Sleeping: ×2 (gain <<= 1), Resting: +50% (gain += gain>>1), Sitting: +25% (gain += gain>>2)
func TestManaGain_PositionMultipliers(t *testing.T) {
	base := 14
	tests := []struct {
		pos  int
		want int
	}{
		{PosStanding, base},              // no multiplier
		{PosSitting, base + (base >> 2)}, // 14 + 3 = 17
		{PosResting, base + (base >> 1)}, // 14 + 7 = 21
		{PosSleeping, base << 1},         // 28
	}
	for _, tt := range tests {
		p := &Player{
			Class:    ClassWarrior,
			Level:    10,
			Position: tt.pos,
			Conditions: [3]int{
				CondFull:   20,
				CondThirst: 20,
				CondDrunk:  0,
			},
		}
		w := &World{}
		got := w.ManaGain(p)
		if got != tt.want {
			t.Errorf("ManaGain(pos=%d) = %d, want %d (per src/limits.c position multipliers)", tt.pos, got, tt.want)
		}
	}
}

// TestManaGain_CasterMultiplier tests the caster mana regen bonus from src/limits.c lines 97-104.
// Mage/Cleric/Magus/Avatar/Mystic: gain <<= 1 (doubled).
// Psionic/Ninja: gain += gain>>2 (+25%).
func TestManaGain_CasterMultiplier(t *testing.T) {
	doubled := []int{ClassMageUser, ClassCleric, ClassMagus, ClassAvatar, ClassMystic}
	for _, class := range doubled {
		p := &Player{
			Class:    class,
			Level:    10,
			Position: PosStanding,
			Conditions: [3]int{
				CondFull:   20,
				CondThirst: 20,
				CondDrunk:  0,
			},
		}
		w := &World{}
		got := w.ManaGain(p)
		// Base 14, doubled = 28
		want := 28
		if got != want {
			t.Errorf("ManaGain(class=%d, standing) = %d, want %d (caster doubled)", class, got, want)
		}
	}

	plusQuarter := []int{ClassPsionic, ClassNinja}
	for _, class := range plusQuarter {
		p := &Player{
			Class:    class,
			Level:    10,
			Position: PosStanding,
			Conditions: [3]int{
				CondFull:   20,
				CondThirst: 20,
				CondDrunk:  0,
			},
		}
		w := &World{}
		got := w.ManaGain(p)
		// Base 14, +25% = 14 + 3 = 17
		want := 17
		if got != want {
			t.Errorf("ManaGain(class=%d, standing) = %d, want %d (psionic/ninja +25%%)", class, got, want)
		}
	}
}

// TestManaGain_PoisonPenalty tests the poison mana regen penalty from src/limits.c lines 108-110.
// Poisoned: gain >>= 2 (quartered).
func TestManaGain_PoisonPenalty(t *testing.T) {
	p := &Player{
		Class:    ClassWarrior,
		Level:    10,
		Position: PosStanding,
		Affects:  1 << AffPoison,
		Conditions: [3]int{
			CondFull:   20,
			CondThirst: 20,
			CondDrunk:  0,
		},
	}
	w := &World{}
	got := w.ManaGain(p)
	// Base 14, quartered by poison = 3
	want := 3
	if got != want {
		t.Errorf("ManaGain(poisoned) = %d, want %d (per src/limits.c poison penalty)", got, want)
	}
}

// ---------------------------------------------------------------------------
// move_gain deterministic portions
// ---------------------------------------------------------------------------

// TestMoveGainNPC_Golden tests the NPC branch of move_gain from src/limits.c line 200.
// NPC move regen: gain = GET_LEVEL(ch)
func TestMoveGainNPC_Golden(t *testing.T) {
	m := &MobInstance{Level: 15}
	got := MoveGainNPC(m)
	if got != 15 {
		t.Errorf("MoveGainNPC(level=15) = %d, want 15 (per src/limits.c)", got)
	}
}

// TestMoveGain_BaseGain tests the base gain for PCs: 20 (src/limits.c line 203).
func TestMoveGain_BaseGain(t *testing.T) {
	p := &Player{
		Class:    ClassWarrior,
		Level:    10,
		Position: PosStanding,
		Conditions: [3]int{
			CondFull:   20,
			CondThirst: 20,
			CondDrunk:  0,
		},
	}
	w := &World{}
	got := w.MoveGain(p)
	// Base 20, no position multiplier (standing).
	want := 20
	if got != want {
		t.Errorf("MoveGain(warrior L10 standing) = %d, want %d (per src/limits.c)", got, want)
	}
}

// TestMoveGain_PositionMultipliers tests the position-based multipliers from src/limits.c lines 217-235.
// Sleeping: +50% (gain += gain>>1), Resting: +25% (gain += gain>>2), Sitting: +12.5% (gain += gain>>3)
func TestMoveGain_PositionMultipliers(t *testing.T) {
	base := 20
	tests := []struct {
		pos  int
		want int
	}{
		{PosStanding, base},               // no multiplier
		{PosSitting, base + (base >> 3)},  // 20 + 2 = 22
		{PosResting, base + (base >> 2)},  // 20 + 5 = 25
		{PosSleeping, base + (base >> 1)}, // 20 + 10 = 30
	}
	for _, tt := range tests {
		p := &Player{
			Class:    ClassWarrior,
			Level:    10,
			Position: tt.pos,
			Conditions: [3]int{
				CondFull:   20,
				CondThirst: 20,
				CondDrunk:  0,
			},
		}
		w := &World{}
		got := w.MoveGain(p)
		if got != tt.want {
			t.Errorf("MoveGain(pos=%d) = %d, want %d (per src/limits.c position multipliers)", tt.pos, got, tt.want)
		}
	}
}

// TestMoveGain_PoisonPenalty tests the poison move regen penalty from src/limits.c lines 239-241.
// Poisoned: gain >>= 2 (quartered).
func TestMoveGain_PoisonPenalty(t *testing.T) {
	p := &Player{
		Class:    ClassWarrior,
		Level:    10,
		Position: PosStanding,
		Affects:  1 << AffPoison,
		Conditions: [3]int{
			CondFull:   20,
			CondThirst: 20,
			CondDrunk:  0,
		},
	}
	w := &World{}
	got := w.MoveGain(p)
	// Base 20, quartered by poison = 5
	want := 5
	if got != want {
		t.Errorf("MoveGain(poisoned) = %d, want %d (per src/limits.c poison penalty)", got, want)
	}
}

// TestMoveGain_HungerPenalty tests the hunger/thirst move regen penalty from src/limits.c lines 244-246.
// Hungry or thirsty: gain >>= 2 (quartered).
func TestMoveGain_HungerPenalty(t *testing.T) {
	p := &Player{
		Class:    ClassWarrior,
		Level:    10,
		Position: PosStanding,
		Conditions: [3]int{
			CondFull:   0, // hungry
			CondThirst: 20,
			CondDrunk:  0,
		},
	}
	w := &World{}
	got := w.MoveGain(p)
	// Base 20, quartered by hunger = 5
	want := 5
	if got != want {
		t.Errorf("MoveGain(hungry) = %d, want %d (per src/limits.c hunger penalty)", got, want)
	}
}
