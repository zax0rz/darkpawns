package spells

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestCallMagicItemCastIgnoresFightingPosition pins DP-1481 shape 4. C's
// call_magic() applies only the sitting refusal (src/spell_parser.c:431-434), so
// a fighting caster still zaps a wand, reads a scroll or quaffs a potion even
// when the spell's own minimum position is higher; that minimum position is
// do_cast's command gate alone (src/spell_parser.c:855-870). Before this fix the
// command gate leaked into every item cast, so once the caster was fighting
// every later `use` answered "You can't concentrate enough!".
func TestCallMagicItemCastIgnoresFightingPosition(t *testing.T) {
	info := GetSpellInfo(SpellBless)
	if info == nil {
		t.Fatal("SPELL_BLESS has no SpellInfo")
	}
	if info.MinPosition <= PosFighting {
		t.Fatalf("SPELL_BLESS minimum position = %v, want above POS_FIGHTING (%v)", info.MinPosition, PosFighting)
	}

	caster := &mockSpellsChar{name: "Caster", level: 31, position: int(PosFighting)}
	victim := &mockSpellsChar{name: "Victim", level: 10, position: int(PosStanding)}

	if !CallMagic(caster, victim, nil, SpellBless, 31, CastWand, nil) {
		t.Fatal("a fighting caster's wand cast was refused")
	}
	if len(victim.activeAffects) == 0 {
		t.Error("the wand's spell did not reach the victim")
	}
}

// TestCallMagicStillRefusesSittingCaster is the other half of the same C gate:
// call_magic's own sitting refusal must survive (src/spell_parser.c:431-434).
func TestCallMagicStillRefusesSittingCaster(t *testing.T) {
	caster := &mockSpellsChar{name: "Caster", level: 31, position: int(PosSitting)}
	victim := &mockSpellsChar{name: "Victim", level: 10, position: int(PosStanding)}

	if CallMagic(caster, victim, nil, SpellBless, 31, CastWand, nil) {
		t.Error("a sitting caster's wand cast was allowed")
	}
	if len(victim.activeAffects) != 0 {
		t.Error("the sitting caster's spell still reached the victim")
	}
}

// noMagicTestWorld serves one room for call_magic's ROOM_NOMAGIC gate.
type noMagicTestWorld struct{ room *parser.Room }

func (w noMagicTestWorld) GetRoomInWorld(vnum int) *parser.Room { return w.room }

// TestCallMagicRefusesSpokenCastInNoMagicRoom pins C's ROOM_NOMAGIC gate
// (src/spell_parser.c:414-428): a spell cast from a NOMAGIC room is refused for
// mortal casters with a magic-specific caster line ("Your magic fizzles out and
// dies.") or the psionic/mystic power line ("Your will fades, disturbed by an
// unseen force.") and no spell routine runs. The Go port had read ROOM_NOMAGIC
// as bit 20 (RoomNoMagic=20) instead of bit 7, so the gate never fired and the
// spell ran. Reverting either the constant or the message wording fails this
// test.
func TestCallMagicRefusesSpokenCastInNoMagicRoom(t *testing.T) {
	// RoomHasFlagBit words are decimal; bit 7 is 1<<7 = 128 in word 0.
	room := &parser.Room{Flags: []string{"128", "0", "0", "0"}}
	world := noMagicTestWorld{room: room}

	caster := &mockSpellsChar{name: "Caster", level: 20, class: 0, roomVNum: 8004, position: int(PosStanding)}
	victim := &mockSpellsChar{name: "Victim", level: 10, position: int(PosStanding)}
	if CallMagic(caster, victim, nil, SpellBless, 20, CastSpell, world) {
		t.Fatal("a mortal's spoken cast in a NOMAGIC room was allowed")
	}
	if got := caster.messages; len(got) == 0 || got[0] != "Your magic fizzles out and dies.\r\n" {
		t.Fatalf("caster message = %#v, want the C magic-fizzles line", got)
	}
	if len(victim.activeAffects) != 0 {
		t.Error("the refused NOMAGIC cast still reached the victim")
	}

	psi := &mockSpellsChar{name: "Psi", level: 20, class: 9, roomVNum: 8004, position: int(PosStanding)}
	psiTarget := &mockSpellsChar{name: "Target", level: 10, position: int(PosStanding)}
	if CallMagic(psi, psiTarget, nil, SpellBless, 20, CastSpell, world) {
		t.Fatal("a psionic's will in a NOMAGIC room was allowed")
	}
	if got := psi.messages; len(got) == 0 || got[0] != "Your will fades, disturbed by an unseen force.\r\n" {
		t.Fatalf("psionic message = %#v, want the C will-fades line", got)
	}
	if len(psiTarget.activeAffects) != 0 {
		t.Error("the refused NOMAGIC will still reached the target")
	}
}

// TestCallMagicExemptsImmortalsFromNoMagic pins C's level gate on the same
// branch (src/spell_parser.c:415): an immortal casts normally in a NOMAGIC
// room.
func TestCallMagicExemptsImmortalsFromNoMagic(t *testing.T) {
	room := &parser.Room{Flags: []string{"128", "0", "0", "0"}}
	world := noMagicTestWorld{room: room}

	caster := &mockSpellsChar{name: "God", level: 41, class: 0, roomVNum: 8004, position: int(PosStanding)}
	victim := &mockSpellsChar{name: "Victim", level: 10, position: int(PosStanding)}
	if !CallMagic(caster, victim, nil, SpellBless, 41, CastSpell, world) {
		t.Fatal("an immortal's cast in a NOMAGIC room was refused")
	}
	if len(victim.activeAffects) == 0 {
		t.Error("the immortal's NOMAGIC-room cast did not reach the victim")
	}
}
