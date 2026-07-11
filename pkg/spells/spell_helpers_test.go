package spells

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

type poserMock struct {
	position int
}

func (m *poserMock) GetPosition() int { return m.position }
func (m *poserMock) SetPosition(int)  {}

// ---------------------------------------------------------------------------
// Mock types for helper-function tests
// ---------------------------------------------------------------------------

type hpMock struct {
	hp int
}

func (m *hpMock) GetHP() int  { return m.hp }
func (m *hpMock) SetHP(v int) { m.hp = v }

type alignMock struct {
	alignment int
}

func (m *alignMock) GetAlignment() int { return m.alignment }

type npcMock struct{}

func (m *npcMock) IsNPC() bool { return true }

type roomMock struct {
	vnum int
}

func (m *roomMock) GetRoomVNum() int { return m.vnum }

type zoneWorldMock struct {
	roomZone int // zone returned by GetRoomZone for the caster's room
	zone     int // zone used by ForEachPlayerInZoneInterface
	players  []interface{}
}

func (w *zoneWorldMock) GetRoomZone(vnum int) int { return w.roomZone }
func (w *zoneWorldMock) ForEachPlayerInZoneInterface(zone int, fn func(interface{})) {
	if zone != w.zone {
		return
	}
	for _, p := range w.players {
		fn(p)
	}
}

type msgMock struct {
	messages []string
}

func (m *msgMock) SendMessage(msg string) { m.messages = append(m.messages, msg) }

// ---------------------------------------------------------------------------
// damage_spells.go helpers
// ---------------------------------------------------------------------------

func TestIsNPC(t *testing.T) {
	if !isNPC(&npcMock{}) {
		t.Error("expected IsNPC() == true")
	}
	if isNPC(&hpMock{}) {
		t.Error("expected isNPC to return false for non-npc")
	}
}

func TestIsEvil(t *testing.T) {
	if !isEvil(&alignMock{alignment: -1}) {
		t.Error("expected alignment -1 to be evil")
	}
	if isEvil(&alignMock{alignment: 0}) {
		t.Error("expected alignment 0 to not be evil")
	}
	if isEvil(&hpMock{}) {
		t.Error("expected non-aligner to not be evil")
	}
}

func TestIsGood(t *testing.T) {
	if !isGood(&alignMock{alignment: 1}) {
		t.Error("expected alignment 1 to be good")
	}
	if isGood(&alignMock{alignment: 0}) {
		t.Error("expected alignment 0 to not be good")
	}
	if isGood(&hpMock{}) {
		t.Error("expected non-aligner to not be good")
	}
}

func TestGetHP(t *testing.T) {
	if got := getHP(&hpMock{hp: 42}); got != 42 {
		t.Errorf("getHP = %d, want 42", got)
	}
	if got := getHP("not an hper"); got != 0 {
		t.Errorf("getHP = %d, want 0", got)
	}
}

func TestSendToZone(t *testing.T) {
	caster := &roomMock{vnum: 1001}
	receiver := &msgMock{}
	world := &zoneWorldMock{
		roomZone: 5,
		zone:     5,
		players:  []interface{}{receiver, &hpMock{}},
	}

	sendToZone("hello zone", caster, world)
	if len(receiver.messages) != 1 || receiver.messages[0] != "hello zone" {
		t.Errorf("expected zone message, got %v", receiver.messages)
	}

	// Wrong room zone should not deliver.
	receiver.messages = nil
	wrongZoneWorld := &zoneWorldMock{
		roomZone: 5,
		zone:     99,
		players:  []interface{}{receiver},
	}
	sendToZone("hello zone", caster, wrongZoneWorld)
	if len(receiver.messages) != 0 {
		t.Errorf("expected no message for wrong zone, got %v", receiver.messages)
	}
}

func TestRandBool(t *testing.T) {
	if !randBool(1) {
		t.Error("randBool(1) should always be true")
	}
	if !randBool(0) {
		t.Error("randBool(0) should return true for denom <= 1")
	}
	// denom > 1 is probabilistic; just verify it runs without panic.
	_ = randBool(100)
}

func TestConsumeReagent(t *testing.T) {
	reagent := &mockItem{shortDesc: "spider leg"}
	inv := &mockInventory{items: []*mockItem{reagent}}
	caster := &mockCaster{inv: inv}

	if !consumeReagent(caster, 1, 10, "spider leg", "The leg twitches.", "A leg twitches.") {
		t.Error("expected consumeReagent to return true when reagent found")
	}
	if len(inv.items) != 0 {
		t.Errorf("expected reagent consumed, got %d items", len(inv.items))
	}

	// Missing reagent.
	caster2 := &mockCaster{inv: &mockInventory{}}
	if consumeReagent(caster2, 1, 10, "spider leg", "", "") {
		t.Error("expected consumeReagent to return false when reagent missing")
	}
}

// ---------------------------------------------------------------------------
// say_spell.go helpers
// ---------------------------------------------------------------------------

func TestSameRoom(t *testing.T) {
	if !sameRoom(&roomMock{vnum: 1001}, &roomMock{vnum: 1001}) {
		t.Error("expected same room")
	}
	if sameRoom(&roomMock{vnum: 1001}, &roomMock{vnum: 1002}) {
		t.Error("expected different rooms")
	}
	if sameRoom(&hpMock{}, &roomMock{vnum: 1001}) {
		t.Error("expected false when one lacks room")
	}
}

func TestSameRoomWithObj(t *testing.T) {
	// Current implementation is a stub that always returns true.
	if !sameRoomWithObj(nil, nil) {
		t.Error("sameRoomWithObj stub should return true")
	}
}

func TestSendAct(t *testing.T) {
	caster := &msgMock{}
	victim := &msgMock{}

	sendAct("to caster", caster, nil, nil, nil)
	if len(caster.messages) != 1 || caster.messages[0] != "to caster" {
		t.Errorf("expected message to caster, got %v", caster.messages)
	}

	sendAct("to victim", caster, nil, victim, nil)
	if len(victim.messages) != 1 || victim.messages[0] != "to victim" {
		t.Errorf("expected message to victim, got %v", victim.messages)
	}
}

// ---------------------------------------------------------------------------
// spells.go Cast
// ---------------------------------------------------------------------------

func TestCast_UnknownSpellReturnsFalse(t *testing.T) {
	caster := &msgMock{}
	if got := CallMagic(caster, nil, nil, -9999, 10, CastSpell, nil); got {
		t.Error("expected CallMagic with unknown spell to return false")
	}
}

type spawnWorldMock struct {
	spawnedVNum int
	spawnRoom   int
}

func (w *spawnWorldMock) SpawnObject(vnum, roomVNum int) (interface{}, error) {
	w.spawnedVNum = vnum
	w.spawnRoom = roomVNum
	return &mockItem{shortDesc: "a magic mushroom"}, nil
}

func TestMagCreations_CreateFood(t *testing.T) {
	caster := &roomMock{vnum: 1001}
	world := &spawnWorldMock{}

	MagCreations(10, caster, SpellCreateFood, world)

	if world.spawnedVNum != 8062 {
		t.Errorf("expected spawn vnum 8062, got %d", world.spawnedVNum)
	}
	if world.spawnRoom != 1001 {
		t.Errorf("expected spawn room 1001, got %d", world.spawnRoom)
	}
}

func TestMagCreations_NilCaster(t *testing.T) {
	world := &spawnWorldMock{}
	MagCreations(10, nil, SpellCreateFood, world)
	if world.spawnedVNum != 0 {
		t.Error("expected no spawn with nil caster")
	}
}

func TestCast_RoutesToCallMagic(t *testing.T) {
	// Cast is a thin wrapper around CallMagic. Use an unregistered spell so
	// CallMagic returns immediately without side effects.
	caster := &msgMock{}
	Cast(caster, nil, -9999, 10, nil)
	// If it does not panic and does not send any message, the wrapper is sane.
	if len(caster.messages) != 0 {
		t.Errorf("Cast with unknown spell should not send messages, got %v", caster.messages)
	}
}

func TestCheckPosition(t *testing.T) {
	si := &SpellInfo{MinPosition: PosStanding}

	t.Run("dead", func(t *testing.T) {
		caster := &msgMock{}
		if checkPosition(&poserMock{position: combat.PosDead}, si) {
			t.Error("expected checkPosition to return false for dead caster")
		}
		if len(caster.messages) != 0 {
			t.Error("expected no message because mock lacks SendMessage")
		}
	})

	t.Run("position too low", func(t *testing.T) {
		if checkPosition(&poserMock{position: combat.PosSleeping}, si) {
			t.Error("expected checkPosition to return false when position too low")
		}
	})

	t.Run("position ok", func(t *testing.T) {
		if !checkPosition(&poserMock{position: combat.PosStanding}, si) {
			t.Error("expected checkPosition to return true when position ok")
		}
	})

	t.Run("non-poser", func(t *testing.T) {
		if checkPosition("not a poser", si) {
			t.Error("expected checkPosition to return false for non-poser")
		}
	})
}
