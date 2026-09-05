package spells

import (
	"reflect"
	"testing"
)

func TestMagAffectsSleepOutlawGate(t *testing.T) {
	caster := newAffectMale("Castero", 8162)
	victim := newAffectMale("Barnaby", 8162)
	observer := newAffectMale("Observe", 8162)
	world := newAffectMsgWorld(caster, victim, observer)

	MagAffects(20, caster, victim, SpellSleep, int(SaveSpell), world)

	wantCaster := []string{
		"You attempt the spell without the components...\r\n",
		"Your spell fails to affect them because you are not an Outlaw!\r\n",
	}
	wantVictim := []string{
		"Castero tried to cast a spell on you but failed because Castero is not an Outlaw!\r\n",
	}
	if !reflect.DeepEqual(caster.messages, wantCaster) {
		t.Errorf("outlaw caster messages = %q, want %q", caster.messages, wantCaster)
	}
	if !reflect.DeepEqual(victim.messages, wantVictim) {
		t.Errorf("outlaw victim messages = %q, want %q", victim.messages, wantVictim)
	}
	if len(victim.affects) != 0 {
		t.Fatalf("outlaw gate applied %d affects", len(victim.affects))
	}
}

func TestMagAffectsSleepLevelWindowGate(t *testing.T) {
	caster := newAffectMale("Castero", 8162)
	caster.level = 20
	caster.affectedBits = 1 // PLR_OUTLAW
	victim := newAffectMale("Barnaby", 8162)
	victim.level = 24
	observer := newAffectMale("Observe", 8162)
	world := newAffectMsgWorld(caster, victim, observer)

	MagAffects(20, caster, victim, SpellSleep, int(SaveSpell), world)

	if !reflect.DeepEqual(observer.messages, []string{"Barnaby shakes his head wearily, but then snaps out of it!\r\n"}) {
		t.Errorf("level-window room messages = %q", observer.messages)
	}
	wantCaster := []string{
		"You attempt the spell without the components...\r\n",
		"Barnaby shakes his head wearily, but then snaps out of it!\r\n",
	}
	if !reflect.DeepEqual(caster.messages, wantCaster) {
		t.Errorf("level-window caster messages = %q", caster.messages)
	}
	if len(victim.messages) != 0 || len(victim.affects) != 0 {
		t.Errorf("level-window victim state messages=%q affects=%d", victim.messages, len(victim.affects))
	}
}
