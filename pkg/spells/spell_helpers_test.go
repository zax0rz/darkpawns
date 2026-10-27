package spells

import (
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

type poserMock struct {
	position int
}

func (m *poserMock) GetPosition() int { return m.position }
func (m *poserMock) SetPosition(int)  {}

type positionedMsgMock struct {
	position int
	messages []string
}

func (m *positionedMsgMock) GetPosition() int { return m.position }
func (m *positionedMsgMock) SendMessage(msg string) {
	m.messages = append(m.messages, msg)
}

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

type spellObjectMock struct {
	room    int
	carried interface{}
}

func (o *spellObjectMock) InRoom(room int) bool { return o.room == room }
func (o *spellObjectMock) CarriedBy(actor interface{}) bool {
	return o.carried == actor
}

type zoneWorldMock struct {
	roomVNum int
	message  string
	calls    int
}

func (w *zoneWorldMock) SendToZone(roomVNum int, message string) {
	w.roomVNum = roomVNum
	w.message = message
	w.calls++
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
	world := &zoneWorldMock{}

	sendToZone("hello zone", caster, world)
	if world.calls != 1 || world.roomVNum != caster.vnum || world.message != "hello zone" {
		t.Fatalf("SendToZone call = (%d, %d, %q), want (1, %d, %q)", world.calls, world.roomVNum, world.message, caster.vnum, "hello zone")
	}

	// A live world without C's zone-send operation must not fall back to a
	// broader broadcast that would include the caster's room.
	sendToZone("hello zone", caster, struct{}{})
	if world.calls != 1 {
		t.Errorf("unsupported world changed recorded calls: %d", world.calls)
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
	caster := &roomMock{vnum: 1001}
	if !sameRoomWithObj(caster, &spellObjectMock{room: 1001}) {
		t.Error("object in caster's room should use object phrase")
	}
	if !sameRoomWithObj(caster, &spellObjectMock{carried: caster}) {
		t.Error("object carried by caster should use object phrase")
	}
	if sameRoomWithObj(caster, &spellObjectMock{room: 1002}) {
		t.Error("object in another room should use generic phrase")
	}
	if sameRoomWithObj(caster, &spellObjectMock{}) {
		t.Error("object neither in room nor carried should use generic phrase")
	}
	if sameRoomWithObj(nil, &spellObjectMock{room: 1001}) || sameRoomWithObj(caster, nil) {
		t.Error("nil caster and object should use generic phrase")
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
	for _, spellNum := range []int{-9999, 0, topSpellDefine + 1} {
		if got := CallMagic(caster, nil, nil, spellNum, 10, CastSpell, nil); got {
			t.Errorf("CallMagic with out-of-range spell %d returned true", spellNum)
		}
	}
}

func TestCallMagicInactiveInRangeSpellIsSuccessfulNoop(t *testing.T) {
	if GetSpellInfo(SpellDrowning) != nil {
		t.Fatalf("SpellDrowning unexpectedly has an active SpellInfo entry")
	}

	t.Run("standing continues item sequence", func(t *testing.T) {
		caster := &positionedMsgMock{position: int(PosStanding)}
		if !CallMagic(caster, caster, nil, SpellDrowning, 20, CastPotion, nil) {
			t.Fatal("in-range inactive spell should return success so later item slots run")
		}
		if len(caster.messages) != 0 {
			t.Fatalf("inactive spell sent messages: %q", caster.messages)
		}
	})

	t.Run("sitting still uses call_magic gate", func(t *testing.T) {
		caster := &positionedMsgMock{position: int(PosSitting)}
		if CallMagic(caster, caster, nil, SpellDrowning, 20, CastPotion, nil) {
			t.Fatal("sitting caster should fail before the inactive spell no-op")
		}
		if got, want := caster.messages, []string{"You cannot do this sitting!\r\n"}; !reflect.DeepEqual(got, want) {
			t.Fatalf("sitting message = %q, want %q", got, want)
		}
	})
}

type spawnWorldMock struct {
	spawnedVNum int
	player      interface{}
	players     []interface{}
}

func (w *spawnWorldMock) SpawnObjectToPlayerInventoryI(vnum int, player interface{}) (interface{}, error) {
	w.spawnedVNum = vnum
	w.player = player
	return &mockItem{shortDesc: "a Magic Mushroom"}, nil
}

func (w *spawnWorldMock) ForEachPlayerInRoomInterface(room int, fn func(interface{})) {
	for _, player := range w.players {
		fn(player)
	}
}

func (w *spawnWorldMock) ForEachMobInRoomInterface(int, func(interface{})) {}

func TestMagCreations_CreateFood(t *testing.T) {
	caster := newAffectMale("Castero", 1001)
	observer := newAffectMale("Observe", 1001)
	world := &spawnWorldMock{players: []interface{}{caster, observer}}

	MagCreations(10, caster, SpellCreateFood, world)

	if world.spawnedVNum != 8062 {
		t.Errorf("expected spawn vnum 8062, got %d", world.spawnedVNum)
	}
	if world.player != caster {
		t.Errorf("inventory placement player = %v, want caster", world.player)
	}
	if got, want := caster.messages, []string{"You create a Magic Mushroom.\r\n"}; !reflect.DeepEqual(got, want) {
		t.Errorf("caster creation message = %q, want %q", got, want)
	}
	if got, want := observer.messages, []string{"Castero creates a Magic Mushroom.\r\n"}; !reflect.DeepEqual(got, want) {
		t.Errorf("observer creation message = %q, want %q", got, want)
	}
}

func TestMagCreationsCreateFoodRoomFallbackUsesActGate(t *testing.T) {
	caster := newAffectMale("Castero", 1001)
	sleeping := newAffectMale("Sleeper", 1001)
	sleeping.position = int(PosSleeping)
	writer := newAffectMale("Writer", 1001)
	writer.affectedBits = 1 << 4 // PLR_WRITING (src/structs.h:225).
	awake := newAffectMale("Awake", 1001)
	world := &spawnWorldMock{players: []interface{}{caster, sleeping, writer, awake}}

	MagCreations(10, caster, SpellCreateFood, world)

	if len(sleeping.messages) != 0 {
		t.Fatalf("sleeping observer received Create Food room act %q", sleeping.messages)
	}
	if len(writer.messages) != 0 {
		t.Fatalf("writing observer received Create Food room act %q", writer.messages)
	}
	if got, want := awake.messages, []string{"Castero creates a Magic Mushroom.\r\n"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("awake observer output = %q, want %q", got, want)
	}
}

type createFoodActWorld struct {
	*spawnWorldMock
	actCalled bool
}

func (w *createFoodActWorld) SpellCreatedObjectAct(interface{}, interface{}) bool {
	w.actCalled = true
	return true
}

func TestMagCreationsCreateFoodUsesLiveActAdapter(t *testing.T) {
	caster := newAffectMale("Castero", 1001)
	world := &createFoodActWorld{spawnWorldMock: &spawnWorldMock{players: []interface{}{caster}}}

	MagCreations(10, caster, SpellCreateFood, world)

	if !world.actCalled {
		t.Fatal("Create Food did not route its room output through SpellCreatedObjectAct")
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
