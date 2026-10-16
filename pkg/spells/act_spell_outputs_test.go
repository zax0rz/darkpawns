package spells

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

type enchantOutputObject struct {
	identifyOutputObject
	flags [4]int
}

func (o *enchantOutputObject) SetAffectsOverride(affects []parser.ObjAffect) {
	o.affects = affects
}

func (o *enchantOutputObject) SetExtraFlag(word, bit int) {
	if word >= 0 && word < len(o.flags) && bit >= 0 && bit < 32 {
		o.flags[word] |= 1 << bit
	}
}

func (o *enchantOutputObject) HasExtraFlag(word, bit int) bool {
	return word >= 0 && word < len(o.flags) && bit >= 0 && bit < 32 && o.flags[word]&(1<<bit) != 0
}

type alignedSpellProbe struct {
	*mockSpellsChar
	alignment int
}

func (p *alignedSpellProbe) GetAlignment() int { return p.alignment }

func TestCastEnchantOutputBranches(t *testing.T) {
	for _, tc := range []struct {
		name      string
		spell     int
		objType   int
		shortDesc string
		alignment int
		color     string
	}{
		{name: "weapon good", spell: SpellEnchantWeapon, objType: 5, shortDesc: "a dagger", alignment: 1000, color: "blue"},
		{name: "weapon evil", spell: SpellEnchantWeapon, objType: 5, shortDesc: "a dagger", alignment: -1000, color: "red"},
		{name: "weapon neutral", spell: SpellEnchantWeapon, objType: 5, shortDesc: "a dagger", color: "yellow"},
		{name: "armor good", spell: SpellEnchantArmor, objType: 9, shortDesc: "a breast plate", alignment: 1000, color: "blue"},
		{name: "armor evil", spell: SpellEnchantArmor, objType: 9, shortDesc: "a breast plate", alignment: -1000, color: "red"},
		{name: "armor neutral", spell: SpellEnchantArmor, objType: 9, shortDesc: "a breast plate", color: "yellow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := &alignedSpellProbe{
				mockSpellsChar: &mockSpellsChar{name: "Enchanter", level: 30},
				alignment:      tc.alignment,
			}
			obj := &enchantOutputObject{identifyOutputObject: identifyOutputObject{typeFlag: tc.objType, short: tc.shortDesc}}
			ExecuteManualSpell(tc.spell, caster.level, caster, nil, obj, "", nil)

			want := tc.shortDesc + " glows " + tc.color + ".\r\n"
			if got := strings.Join(caster.messages, ""); got != want {
				t.Fatalf("enchant output = %q, want exact C bytes %q", got, want)
			}
			if !obj.HasExtraFlag(0, 6) {
				t.Fatal("successful enchant did not set ITEM_MAGIC")
			}
		})
	}

	t.Run("weapon refusal", func(t *testing.T) {
		caster := &alignedSpellProbe{mockSpellsChar: &mockSpellsChar{name: "Enchanter", level: 30}}
		obj := &enchantOutputObject{identifyOutputObject: identifyOutputObject{typeFlag: 9, short: "a breast plate"}}
		ExecuteManualSpell(SpellEnchantWeapon, 30, caster, nil, obj, "", nil)
		if got, want := strings.Join(caster.messages, ""), "Nothing seems to happen.\r\n"; got != want {
			t.Fatalf("enchant weapon refusal = %q, want exact C bytes %q", got, want)
		}
	})
}

type zenSpellOutputWorld struct {
	actor         interface{}
	format        string
	hideInvisible bool
	calls         int
}

func (w *zenSpellOutputWorld) SpellActToRoom(actor, victim interface{}, format string, hideInvisible bool) bool {
	w.actor = actor
	w.format = format
	w.hideInvisible = hideInvisible
	w.calls++
	return true
}

func TestCastZenOutputAndMeditationState(t *testing.T) {
	caster := &mockSpellsChar{
		name: "Zenmind", level: 10, position: int(PosStanding), hp: 20, maxHP: 60,
	}
	world := &zenSpellOutputWorld{}

	ExecuteManualSpell(SpellZen, caster.level, caster, caster, nil, "", world)

	const want = "You begin to meditate deeply, focusing your thoughts only on healing.\r\n"
	if got := strings.Join(caster.messages, ""); got != want {
		t.Fatalf("zen caster output = %q, want exact C bytes %q", got, want)
	}
	if world.calls != 1 || world.actor != caster || world.format != "$n sinks into a deep, meditative sleep." || world.hideInvisible {
		t.Fatalf("zen room act = {calls:%d actor:%v format:%q hideInvisible:%t}, want one visible TO_ROOM act", world.calls, world.actor, world.format, world.hideInvisible)
	}
	if caster.position != int(PosStunned) || caster.hp != 40 {
		t.Fatalf("zen state = {position:%d hp:%d}, want stunned with 40 HP", caster.position, caster.hp)
	}
}

type cocOutputObject struct {
	vnum  int
	timer int
}

func (o *cocOutputObject) SetTimer(timer int) { o.timer = timer }

type cocOutputWorld struct {
	object        *cocOutputObject
	room          int
	actor         interface{}
	format        string
	hideInvisible bool
	calls         int
}

func (w *cocOutputWorld) SpawnObjectToRoomForSpell(vnum, room int) (interface{}, error) {
	w.object = &cocOutputObject{vnum: vnum}
	w.room = room
	return w.object, nil
}

func (w *cocOutputWorld) SpellActToRoom(actor, victim interface{}, format string, hideInvisible bool) bool {
	w.actor = actor
	w.format = format
	w.hideInvisible = hideInvisible
	w.calls++
	return true
}

func TestCastCoCOutputAndRoomAudience(t *testing.T) {
	caster := &mockSpellsChar{name: "Circle", level: 30, roomVNum: 8105}
	world := &cocOutputWorld{}

	ExecuteManualSpell(SpellCoC, caster.level, caster, nil, nil, "", world)

	if got, want := strings.Join(caster.messages, ""), "You draw a magic circle on the ground.\r\n"; got != want {
		t.Fatalf("COC caster output = %q, want exact C bytes %q", got, want)
	}
	if world.object == nil || world.object.vnum != CocVnum || world.room != caster.roomVNum {
		t.Fatalf("COC object placement = {object:%+v room:%d}, want vnum %d in room %d", world.object, world.room, CocVnum, caster.roomVNum)
	}
	if world.object.timer < caster.level/2-2 || world.object.timer > caster.level/2+1 {
		t.Fatalf("COC object timer = %d, want level/2 plus C's -2..1 draw", world.object.timer)
	}
	if world.calls != 1 || world.actor != caster || world.format != "$n draws a magic circle on the ground." || world.hideInvisible {
		t.Fatalf("COC room act = {calls:%d actor:%v format:%q hideInvisible:%t}, want one visible TO_ROOM act", world.calls, world.actor, world.format, world.hideInvisible)
	}
}

type mentalLapseProbe struct {
	mockSpellsChar
	hunting string
	cleared bool
}

func (p *mentalLapseProbe) GetHunting() string { return p.hunting }
func (p *mentalLapseProbe) ClearHunting()      { p.hunting, p.cleared = "", true }

func TestCastMentalLapseOutputBranches(t *testing.T) {
	for _, tc := range []struct {
		name    string
		level   int
		sex     int
		hunting string
		want    string
		cleared bool
	}{
		{name: "male already hunting caster", level: 30, sex: 0, hunting: "Caster", want: "You sense his intentions even from where you are, and change his mind.\r\n", cleared: true},
		{name: "female redirects other target", level: 30, sex: 1, hunting: "Other", want: "You send out your mental voice...\r\nYou sense her intentions even from where you are, and change her mind.\r\n", cleared: true},
		{name: "neutral redirects other target", level: 31, sex: 2, hunting: "Other", want: "You send out your mental voice...\r\nYou sense its intentions even from where you are, and change its mind.\r\n", cleared: true},
		{name: "below level gate recoils", level: 29, sex: 0, hunting: "Other", want: "Your psionic energy recoils!\r\n"},
		{name: "no hunting target recoils", level: 30, sex: 0, want: "Your psionic energy recoils!\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := &mockSpellsChar{name: "Caster", level: tc.level}
			victim := &mentalLapseProbe{
				mockSpellsChar: mockSpellsChar{name: "Hunted", npc: true, sex: tc.sex},
				hunting:        tc.hunting,
			}
			ExecuteManualSpell(SpellMentalLapse, tc.level, caster, victim, nil, "", nil)
			if got := strings.Join(caster.messages, ""); got != tc.want {
				t.Fatalf("mental lapse output = %q, want exact C bytes %q", got, tc.want)
			}
			if victim.cleared != tc.cleared {
				t.Fatalf("hunting cleared=%t, want %t", victim.cleared, tc.cleared)
			}
		})
	}
}
