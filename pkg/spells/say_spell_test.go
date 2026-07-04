package spells

import (
	"strings"
	"testing"
)

// mockSayWorld implements the roomIterable interface used by sendToRoom.
type mockSayWorld struct {
	players []interface{}
	mobs    []interface{}
}

func (w *mockSayWorld) ForEachPlayerInRoomInterface(roomVNum int, fn func(interface{})) {
	for _, p := range w.players {
		fn(p)
	}
}

func (w *mockSayWorld) ForEachMobInRoomInterface(roomVNum int, fn func(interface{})) {
	for _, m := range w.mobs {
		fn(m)
	}
}

func newSayChar(name string, class int, room int) *mockSpellsChar {
	return &mockSpellsChar{
		name:     name,
		class:    class,
		roomVNum: room,
	}
}

func TestSaySpell_RoomMessageNamesCaster(t *testing.T) {
	caster := newSayChar("Alice", 0, 100)
	observer := newSayChar("Bob", 0, 100)
	world := &mockSayWorld{players: []interface{}{caster, observer}}

	SaySpell(caster, 1, nil, nil, world)

	if len(observer.messages) == 0 {
		t.Fatal("observer received no message")
	}
	msg := observer.messages[0]
	if !strings.Contains(msg, "Alice") {
		t.Errorf("room message should name caster Alice, got: %s", msg)
	}
	if !strings.Contains(msg, "armor") {
		t.Errorf("room message should show real spell name to same-class observer, got: %s", msg)
	}
}

func TestSaySpell_RoomMessageObfuscatesForDifferentClass(t *testing.T) {
	caster := newSayChar("Alice", 0, 100)
	observer := newSayChar("Bob", 9, 100)
	world := &mockSayWorld{players: []interface{}{caster, observer}}

	SaySpell(caster, 1, nil, nil, world)

	if len(observer.messages) == 0 {
		t.Fatal("observer received no message")
	}
	msg := observer.messages[0]
	if !strings.Contains(msg, "Alice") {
		t.Errorf("room message should name caster Alice, got: %s", msg)
	}
	if strings.Contains(msg, "armor") {
		t.Errorf("room message should obfuscate spell for different-class observer, got: %s", msg)
	}
}

func TestSaySpell_TargetMessageDeliveredToTarget(t *testing.T) {
	caster := newSayChar("Alice", 0, 100)
	target := newSayChar("Bob", 0, 100)
	world := &mockSayWorld{players: []interface{}{caster, target}}

	SaySpell(caster, 1, target, nil, world)

	foundTargetMsg := false
	for _, msg := range target.messages {
		if strings.Contains(msg, "stares at you") {
			foundTargetMsg = true
			break
		}
	}
	if !foundTargetMsg {
		t.Errorf("target should receive a 'stares at you' message, got: %v", target.messages)
	}
}

func TestSaySpell_TargetMessageNotDeliveredToCaster(t *testing.T) {
	caster := newSayChar("Alice", 0, 100)
	target := newSayChar("Bob", 0, 100)
	world := &mockSayWorld{players: []interface{}{caster, target}}

	SaySpell(caster, 1, target, nil, world)

	for _, msg := range caster.messages {
		if strings.Contains(msg, "stares at you") {
			t.Errorf("caster should not receive target-specific message, got: %s", msg)
		}
	}
}
