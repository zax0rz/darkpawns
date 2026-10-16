package spells

import (
	"errors"
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestMagAffectsIndoorFlamestrikeRefusalOutputMatchC(t *testing.T) {
	previousSaveRoll := magAffectsSaveRoll
	magAffectsSaveRoll = func(interface{}, int) bool { return false }
	defer func() { magAffectsSaveRoll = previousSaveRoll }()

	caster := newAffectMale("Mage", 8162)
	victim := newAffectMale("Target", 8162)
	world := &mockAreaWorld{
		room:  &parser.Room{Flags: []string{"8"}, Sector: 0},
		chars: []interface{}{caster, victim},
	}

	MagAffects(30, caster, victim, SpellFlameStrike, int(SaveSpell), world)

	if !reflect.DeepEqual(caster.messages, []string{"You can only do this outdoors!\r\n"}) {
		t.Errorf("caster bytes = %q, want C indoor refusal", caster.messages)
	}
	if len(victim.affects) != 0 || len(victim.messages) != 0 {
		t.Errorf("indoor refusal applied effect or sent extra output: affects=%d messages=%q", len(victim.affects), victim.messages)
	}
}

func TestMagAreasEarthquakeCasterAndRoomOutputMatchC(t *testing.T) {
	dprng.ResetStream(1)
	defer dprng.ResetStream(1)

	caster := newAffectMale("Mage", 8162)
	witness := newAffectMale("Witness", 8162)
	world := newAffectMsgWorld(caster, witness)

	MagAreas(20, caster, SpellEarthquake, int(SaveSpell), world)

	if !reflect.DeepEqual(caster.messages, []string{"You gesture and the earth begins to shake all around you!\r\n"}) {
		t.Errorf("caster bytes = %q, want C earthquake act", caster.messages)
	}
	if !reflect.DeepEqual(witness.messages, []string{"Mage gracefully gestures and the earth begins to shake violently!\r\n"}) {
		t.Errorf("room bytes = %q, want C earthquake room act", witness.messages)
	}
}

type spellObjectActCall struct {
	message string
	toRoom  bool
}

type spellObjectActWorld struct {
	calls []spellObjectActCall
}

func (w *spellObjectActWorld) SpellObjectAct(_, _ interface{}, message string, toRoom bool) {
	w.calls = append(w.calls, spellObjectActCall{message: message, toRoom: toRoom})
}

func TestMagAlterObjsCOutputCallsMatchC(t *testing.T) {
	caster := &mockSpellsChar{name: "Mage", level: 20, class: 0}
	obj := &alterObjFixture{weight: 1}
	world := &spellObjectActWorld{}

	MagAlterObjs(20, caster, obj, SpellBless, world)

	want := []spellObjectActCall{
		{message: "$p glows briefly with a ethereal light.", toRoom: false},
		{message: "$p glows briefly with a ethereal light.", toRoom: true},
	}
	if !reflect.DeepEqual(world.calls, want) {
		t.Errorf("object act calls = %#v, want %#v", world.calls, want)
	}
	if len(caster.messages) != 0 {
		t.Errorf("caster received fallback output despite SpellObjectAct: %q", caster.messages)
	}
}

type failedObjectSpawnWorld struct{}

func (failedObjectSpawnWorld) SpawnObjectToPlayerInventoryI(int, interface{}) (interface{}, error) {
	return nil, errors.New("missing prototype")
}

func TestMagCreationsMissingPrototypeErrorOutputMatchC(t *testing.T) {
	caster := &mockSpellsChar{name: "Mage", level: 20, roomVNum: 8162}

	MagCreations(20, caster, SpellCreateFood, failedObjectSpawnWorld{})

	if !reflect.DeepEqual(caster.messages, []string{"I seem to have goofed.\r\n"}) {
		t.Errorf("caster bytes = %q, want C missing-prototype error", caster.messages)
	}
}
