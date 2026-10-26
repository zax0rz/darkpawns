package game

// Regen gear against C's formulas. mana_gain adds an APPLY_MANA_REGEN
// modifier when (modifier > 0 && POS_SLEEPING) || modifier < 0
// (src/limits.c:89-95); hit_gain and move_gain add ANY modifier but only
// inside their POS_SLEEPING cases (limits.c:152-158 after the ×1.5,
// :224-230 before it). All three read the object INSTANCE's affected[],
// not the prototype's.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newRegenPlayer(class int, pos int) *Player {
	return &Player{
		Class:    class,
		Level:    10,
		Position: pos,
		Conditions: [3]int{
			CondFull:   20,
			CondThirst: 20,
			CondDrunk:  0,
		},
	}
}

// equipRegenItem puts an object with the given affects into an equipment
// slot. With protoAffects set the affects live on the prototype; otherwise
// they are an instance-only override (C's GET_EQ(ch,i)->affected[j] after a
// spell or Lua mutation, with an unmodified prototype).
func equipRegenItem(t *testing.T, p *Player, slot EquipmentSlot, protoAffects []parser.ObjAffect, instanceAffects []parser.ObjAffect) {
	t.Helper()
	obj := NewObjectInstance(&parser.Obj{
		VNum: 9301, Keywords: "band", ShortDesc: "a regen band",
		WearFlags: [4]int{1<<0 | 1<<14}, // TAKE | HOLD — any slot works via SetSlot
		Affects:   protoAffects,
	}, -1)
	if instanceAffects != nil {
		obj.AffectsOverride = instanceAffects
	}
	if p.Equipment == nil {
		p.Equipment = NewEquipment()
	}
	if err := p.Equipment.SetSlot(slot, obj); err != nil {
		t.Fatalf("SetSlot %v: %v", slot, err)
	}
}

// TestManaGainRegenGearMatchesC pins limits.c:89-95 for a cleric (×2 class
// multiplier, limits.c:97-98), fed and unaffect­ed, in a plain room: a
// positive APPLY_MANA_REGEN counts ONLY while sleeping; a negative one
// counts in every position.
func TestManaGainRegenGearMatchesC(t *testing.T) {
	w := &World{}
	cases := []struct {
		name        string
		proto       []parser.ObjAffect
		instance    []parser.ObjAffect
		pos         int
		extraSecond bool // equip a second item (−4) alongside
		want        int
	}{
		{name: "+10 sleeping", proto: []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: 10}}, pos: PosSleeping, want: (28 + 10) * 2},
		{name: "+10 resting", proto: []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: 10}}, pos: PosResting, want: (21 + 0) * 2},
		{name: "+10 standing", proto: []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: 10}}, pos: PosStanding, want: (14 + 0) * 2},
		{name: "+10 sitting", proto: []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: 10}}, pos: PosSitting, want: (17 + 0) * 2},
		{name: "-10 sleeping", proto: []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: -10}}, pos: PosSleeping, want: (28 - 10) * 2},
		{name: "-10 resting", proto: []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: -10}}, pos: PosResting, want: (21 - 10) * 2},
		{name: "-10 standing", proto: []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: -10}}, pos: PosStanding, want: (14 - 10) * 2},
		{name: "+10 and -4 sleeping", proto: []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: 10}}, pos: PosSleeping, extraSecond: true, want: (28 + 10 - 4) * 2},
		{name: "+10 and -4 standing", proto: []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: 10}}, pos: PosStanding, extraSecond: true, want: (14 - 4) * 2},
		// Instance-only affect, clean prototype: C reads the instance's
		// affected[] (limits.c:83-95), so it counts exactly like the
		// prototype case.
		{name: "instance-only +10 sleeping", instance: []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: 10}}, pos: PosSleeping, want: (28 + 10) * 2},
		{name: "instance-only +10 standing", instance: []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: 10}}, pos: PosStanding, want: (14 + 0) * 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := newRegenPlayer(ClassCleric, tc.pos)
			equipRegenItem(t, p, SlotHold, tc.proto, tc.instance)
			if tc.extraSecond {
				equipRegenItem(t, p, SlotBody, []parser.ObjAffect{{Location: ApplyManaRegen, Modifier: -4}}, nil)
			}
			if got := w.ManaGain(p); got != tc.want {
				t.Fatalf("ManaGain(cleric %s) = %d, want %d (limits.c:89-95)", tc.name, got, tc.want)
			}
		})
	}
}

// TestHitAndMoveRegenGearMatchC guards hit_gain and move_gain while the mana
// polarity is fixed: their equipment loops sit INSIDE the POS_SLEEPING case
// (limits.c:152-158, 224-230) — hit adds items after the ×1.5, move adds
// them before — and no modifier of either sign applies while awake.
func TestHitAndMoveRegenGearMatchC(t *testing.T) {
	w := &World{}

	p := newRegenPlayer(ClassWarrior, PosSleeping)
	equipRegenItem(t, p, SlotHold, []parser.ObjAffect{{Location: ApplyHitRegen, Modifier: 10}}, nil)
	if got := w.HitGain(p); got != 40 { // 20 +10 (×1.5) = 30, +10 = 40 (limits.c:152-158)
		t.Fatalf("HitGain(sleeping warrior, +10) = %d, want 40", got)
	}
	p.Position = PosStanding
	if got := w.HitGain(p); got != 20 {
		t.Fatalf("HitGain(standing warrior, +10) = %d, want 20 (hit gear applies only while sleeping)", got)
	}
	equipRegenItem(t, p, SlotBody, []parser.ObjAffect{{Location: ApplyHitRegen, Modifier: -5}}, nil)
	if got := w.HitGain(p); got != 20 {
		t.Fatalf("HitGain(standing warrior, +10/-5) = %d, want 20 (no hit gear applies while awake, either sign)", got)
	}

	p = newRegenPlayer(ClassWarrior, PosSleeping)
	equipRegenItem(t, p, SlotHold, []parser.ObjAffect{{Location: ApplyMoveRegen, Modifier: 10}}, nil)
	if got := w.MoveGain(p); got != 45 { // 20 +10 = 30, then ×1.5 = 45 (limits.c:224-230)
		t.Fatalf("MoveGain(sleeping warrior, +10) = %d, want 45", got)
	}
	p.Position = PosStanding
	if got := w.MoveGain(p); got != 20 {
		t.Fatalf("MoveGain(standing warrior, +10) = %d, want 20 (move gear applies only while sleeping)", got)
	}
}
