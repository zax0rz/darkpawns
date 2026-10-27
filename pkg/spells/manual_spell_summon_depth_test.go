package spells

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestSummonOutlawResistanceDispatchMatchesCNotice(t *testing.T) {
	circle := &summonTestObject{vnum: CocVnum, timer: 2}
	caster := &mockSpellsChar{
		name: "Mage", level: 30, roomVNum: 1001,
		flags: uint64(1) << prfSummonableBit,
	}
	victim := &mockSpellsChar{
		name: "Target", level: 5, roomVNum: 1002,
		flags: 1, // PLR_OUTLAW is bit 0 in src/structs.h.
	}
	world := &summonTestWorld{
		items: []interface{}{circle},
		rooms: map[int]*parser.Room{
			1001: {VNum: 1001, Name: "Summon Test Room", Exits: map[string]parser.Exit{"north": {Direction: "north", ToRoom: 1002}}},
		},
	}

	oldResistanceRoll := magSummonOutlawResistanceRoll
	magSummonOutlawResistanceRoll = func() int { return 65 }
	t.Cleanup(func() { magSummonOutlawResistanceRoll = oldResistanceRoll })
	oldSaveRoll := magSummonSaveRoll
	magSummonSaveRoll = func(interface{}, int) bool {
		t.Fatal("the outlaw resistance must return before the save")
		return false
	}
	t.Cleanup(func() { magSummonSaveRoll = oldSaveRoll })

	ExecuteManualSpell(SpellSummon, 30, caster, victim, nil, "", world)

	if got, want := caster.messages, []string{"You failed.\r\n"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("caster output = %#v, want %#v", got, want)
	}
	if got, want := victim.messages, []string{"Mage just tried to summon you to: Summon Test Room.\r\nHe failed.\r\n"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("victim output = %#v, want %#v", got, want)
	}
	if circle.timer != 1 {
		t.Fatalf("circle timer = %d, want one consumed tick", circle.timer)
	}
}

type summonBackfireWorld struct {
	summonTestWorld
	players   map[int][]interface{}
	transfers []int
	looks     []interface{}
	hits      [][2]combat.Combatant
}

func (w *summonBackfireWorld) ForEachPlayerInRoomInterface(room int, fn func(interface{})) {
	for _, player := range w.players[room] {
		fn(player)
	}
}

func (w *summonBackfireWorld) ForEachMobInRoomInterface(int, func(interface{})) {}

func (w *summonBackfireWorld) PlayerTransfer(actor interface{}, room int) error {
	switch p := actor.(type) {
	case *mockSpellsChar:
		p.roomVNum = room
	case *breathCombatant:
		p.roomVNum = room
	}
	w.transfers = append(w.transfers, room)
	return nil
}

func (w *summonBackfireWorld) CharacterTransferForSpell(actor interface{}, room int) error {
	return w.PlayerTransfer(actor, room)
}

func TestSummonSuccessTransfersPlayerBeforeArrivalAct(t *testing.T) {
	circle := &summonTestObject{vnum: CocVnum, timer: 2}
	caster := &mockSpellsChar{name: "Mage", level: combat.LVL_IMMORT, roomVNum: 1001, position: int(PosStanding)}
	victim := &mockSpellsChar{name: "Target", level: 1, roomVNum: 1002, position: int(PosStanding)}
	originPeer := &mockSpellsChar{name: "Originpeer", roomVNum: 1001, position: int(PosStanding)}
	targetPeer := &mockSpellsChar{name: "Targetpeer", roomVNum: 1002, position: int(PosStanding)}
	world := &summonBackfireWorld{
		summonTestWorld: summonTestWorld{
			items: []interface{}{circle},
			rooms: map[int]*parser.Room{
				1001: {VNum: 1001, Name: "Caster Room", Exits: map[string]parser.Exit{"north": {Direction: "north", ToRoom: 1002}}},
				1002: {VNum: 1002, Name: "Victim Room"},
			},
		},
		players: map[int][]interface{}{
			1001: {caster, originPeer},
			1002: {victim, targetPeer},
		},
	}

	oldSaveRoll := magSummonSaveRoll
	magSummonSaveRoll = func(interface{}, int) bool { return false }
	t.Cleanup(func() { magSummonSaveRoll = oldSaveRoll })

	ExecuteManualSpell(SpellSummon, combat.LVL_IMMORT, caster, victim, nil, "", world)

	if victim.GetRoomVNum() != 1001 {
		t.Fatalf("summoned target room = %d, want caster room 1001", victim.GetRoomVNum())
	}
	if got, want := world.transfers, []int{1001}; !reflect.DeepEqual(got, want) {
		t.Fatalf("summon transfers = %#v, want %#v", got, want)
	}
	if got := strings.Join(targetPeer.messages, ""); !strings.Contains(got, "Target disappears suddenly.") {
		t.Fatalf("target-room peer messages = %q, want the departure act", got)
	}
	if got := strings.Join(originPeer.messages, ""); !strings.Contains(got, "Target arrives suddenly.") {
		t.Fatalf("caster-room peer messages = %q, want the arrival act", got)
	}
	if got, want := victim.messages, []string{"Mage has summoned you!\r\n"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("target messages = %#v, want %#v", got, want)
	}
}

func (w *summonBackfireWorld) MobTransfer(actor interface{}, room int) error {
	return w.PlayerTransfer(actor, room)
}

func (w *summonBackfireWorld) LookAtRoomForSpell(actor interface{}) {
	w.looks = append(w.looks, actor)
}

func (w *summonBackfireWorld) MobFirstHit(attacker, defender combat.Combatant) {
	w.hits = append(w.hits, [2]combat.Combatant{attacker, defender})
}

func TestSummonBackfireDispatchesCOutputsAndNPCRetaliation(t *testing.T) {
	circle := &summonTestObject{vnum: CocVnum, timer: 2}
	caster := &breathCombatant{mockSpellsChar: mockSpellsChar{
		name: "Mage", level: 30, roomVNum: 1001, position: int(PosStanding),
		hp: 100, maxHP: 100,
	}}
	victim := &breathCombatant{mockSpellsChar: mockSpellsChar{
		name: "Guard", npc: true, level: 10, roomVNum: 1002, position: int(PosStanding),
		hp: 100, maxHP: 100, mobFlags: uint64(1) << mobNoCharmBit,
	}}
	oldPeer := &mockSpellsChar{name: "Old room watcher", roomVNum: 1001, position: int(PosStanding)}
	newPeer := &mockSpellsChar{name: "New room watcher", roomVNum: 1002, position: int(PosStanding)}
	world := &summonBackfireWorld{
		summonTestWorld: summonTestWorld{
			items: []interface{}{circle},
			rooms: map[int]*parser.Room{
				1001: {VNum: 1001, Name: "Caster Room", Exits: map[string]parser.Exit{"north": {Direction: "north", ToRoom: 1002}}},
				1002: {VNum: 1002, Name: "Victim Room"},
			},
		},
		players: map[int][]interface{}{
			1001: {caster, oldPeer},
			1002: {newPeer},
		},
	}

	oldSaveRoll := magSummonSaveRoll
	magSummonSaveRoll = func(interface{}, int) bool { return false }
	t.Cleanup(func() { magSummonSaveRoll = oldSaveRoll })
	oldBackfireRoll := magSummonBackfireRoll
	magSummonBackfireRoll = func() int { return 0 }
	t.Cleanup(func() { magSummonBackfireRoll = oldBackfireRoll })

	ExecuteManualSpell(SpellSummon, 30, caster, victim, nil, "", world)

	if got, want := caster.messages, []string{"Your spell backfires!\r\n"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("caster output = %#v, want %#v", got, want)
	}
	if got, want := oldPeer.messages, []string{"Mage disappears suddenly.\r\n"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("old room output = %#v, want %#v", got, want)
	}
	if got, want := newPeer.messages, []string{"Mage arrives suddenly.\r\n"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("new room output = %#v, want %#v", got, want)
	}
	if got, want := world.transfers, []int{1002}; !reflect.DeepEqual(got, want) {
		t.Fatalf("caster transfer destinations = %#v, want %#v", got, want)
	}
	if len(world.looks) != 1 || world.looks[0] != caster {
		t.Fatalf("landing looks = %#v, want caster", world.looks)
	}
	if len(world.hits) != 1 || world.hits[0][0] != victim || world.hits[0][1] != caster {
		t.Fatalf("retaliation = %#v, want Guard -> Mage", world.hits)
	}
	if len(victim.messages) != 0 {
		t.Fatalf("victim messages = %#v, want none on backfire", victim.messages)
	}
}
