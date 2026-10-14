package spells

import "testing"

type alterObjFixture struct {
	flags  int
	typ    int
	values [4]int
	weight int
}

func (o *alterObjFixture) ExtraFlagWord(word int) int {
	if word != 0 {
		return 0
	}
	return o.flags
}
func (o *alterObjFixture) SetExtraFlags(flags int)  { o.flags = flags }
func (o *alterObjFixture) GetObjType() int          { return o.typ }
func (o *alterObjFixture) GetObjVal(idx int) int    { return o.values[idx] }
func (o *alterObjFixture) SetObjVal(idx, value int) { o.values[idx] = value }
func (o *alterObjFixture) GetWeight() int           { return o.weight }
func (o *alterObjFixture) GetShortDesc() string     { return "a test object" }

// src/magic.c:1900-1949 uses the ITEM_* indices in src/structs.h:472-476.
func TestMagAlterObjsUsesCObjectFlagIndices(t *testing.T) {
	for _, tc := range []struct {
		name       string
		spell      int
		flags      int
		typ        int
		value2     int
		value3     int
		wantFlags  int
		wantValue2 int
		wantValue3 int
	}{
		{name: "bless", spell: SpellBless, flags: 1 << 0, wantFlags: (1 << 0) | (1 << 8)},
		{name: "curse weapon", spell: SpellCurse, typ: 3, value2: 2, wantFlags: 1 << 7, wantValue2: 1},
		{name: "invisible", spell: SpellInvisible, wantFlags: 1 << 5},
		{name: "remove curse weapon", spell: SpellRemoveCurse, flags: 1 << 7, typ: 3, value2: 1, wantValue2: 2},
		{name: "poison food", spell: SpellPoison, typ: 19, wantValue3: 1},
		{name: "remove poison food", spell: SpellRemovePoison, typ: 19, value3: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := &alterObjFixture{flags: tc.flags, typ: tc.typ, weight: 1}
			obj.values[2], obj.values[3] = tc.value2, tc.value3
			caster := &mockSpellsChar{level: 20}
			MagAlterObjs(20, caster, obj, tc.spell, nil)
			if obj.flags != tc.wantFlags {
				t.Errorf("extra flags=%#x want %#x", obj.flags, tc.wantFlags)
			}
			if obj.values[2] != tc.wantValue2 || obj.values[3] != tc.wantValue3 {
				t.Errorf("values[2:4]=%v want [%d %d]", obj.values[2:4], tc.wantValue2, tc.wantValue3)
			}
		})
	}
}

func TestMagAlterObjsUsesCNoEffectGates(t *testing.T) {
	for _, tc := range []struct {
		name   string
		spell  int
		flags  int
		typ    int
		value3 int
	}{
		{name: "blessed", spell: SpellBless, flags: 1 << 8},
		{name: "magic", spell: SpellBless, flags: 1 << 6},
		{name: "no invis", spell: SpellInvisible, flags: 1 << 4},
		{name: "not cursed", spell: SpellRemoveCurse},
		{name: "already poisoned food", spell: SpellPoison, typ: 19, value3: 1},
		{name: "not poisoned food", spell: SpellRemovePoison, typ: 19},
	} {
		t.Run(tc.name, func(t *testing.T) {
			obj := &alterObjFixture{flags: tc.flags, typ: tc.typ, weight: 1}
			obj.values[3] = tc.value3
			caster := &mockSpellsChar{level: 20}
			MagAlterObjs(20, caster, obj, tc.spell, nil)
			if obj.flags != tc.flags || obj.values[2] != 0 || obj.values[3] != tc.value3 {
				t.Errorf("no-effect call mutated object: flags=%#x values=%v", obj.flags, obj.values)
			}
			if len(caster.messages) != 1 || caster.messages[0] != "Nothing seems to happen.\r\n" {
				t.Errorf("no-effect bytes=%q", caster.messages)
			}
		})
	}
}
