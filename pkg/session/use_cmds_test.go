package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// makeWandPrototype builds a minimal wand prototype with the given current
// charges. CircleMUD wand value layout: [0]=level, [1]=max charges,
// [2]=current charges, [3]=spell number.
func makeWandPrototype(currentCharges int) *parser.Obj {
	return &parser.Obj{
		VNum:      99001,
		Keywords:  "test wand",
		ShortDesc: "a test wand",
		LongDesc:  "A test wand lies here.\n",
		TypeFlag:  game.ITEM_WAND,
		WearFlags: [4]int{(1 << 0) | (1 << 14)}, // TAKE + HOLD
		Values:    [4]int{1, currentCharges, currentCharges, 0},
		Weight:    1,
		Cost:      1,
	}
}

// TestZap_InstanceCharges_Isolated verifies DP-1110: decrementing a wand's
// charges via zap mutates only the zapped instance, leaving the shared
// prototype and any other instances unchanged.
func TestZap_InstanceCharges_Isolated(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Zapper", 1001, true)

	// Create two instances from the same prototype.
	proto := makeWandPrototype(5)
	wandA := game.NewObjectInstance(proto, -1)
	wandB := game.NewObjectInstance(proto, -1)

	// Equip wandA in the hold slot; leave wandB in inventory.
	s.player.Equipment = game.NewEquipment()
	s.player.Inventory = game.NewInventory()
	if err := s.player.Equipment.Equip(wandA, s.player.Inventory); err != nil {
		t.Fatalf("equip wandA: %v", err)
	}
	if err := s.player.Inventory.AddItem(wandB); err != nil {
		t.Fatalf("add wandB to inventory: %v", err)
	}

	// Sanity: both instances start with the prototype's charge count.
	if got := wandA.GetValue(2); got != 5 {
		t.Fatalf("wandA pre-zap charges = %d, want 5", got)
	}
	if got := wandB.GetValue(2); got != 5 {
		t.Fatalf("wandB pre-zap charges = %d, want 5", got)
	}

	// Zap self — target resolution works without a second entity in the room.
	if err := cmdZap(s, []string{"self"}); err != nil {
		t.Fatalf("cmdZap: %v", err)
	}

	// The zapped instance lost one charge.
	if got := wandA.GetValue(2); got != 4 {
		t.Errorf("wandA post-zap charges = %d, want 4", got)
	}

	// The other instance and the shared prototype are untouched.
	if got := wandB.GetValue(2); got != 5 {
		t.Errorf("wandB post-zap charges = %d, want 5 (instance isolation broken)", got)
	}
	if got := proto.Values[2]; got != 5 {
		t.Errorf("prototype post-zap charges = %d, want 5 (prototype corruption)", got)
	}
}

// TestZap_MessagesMatchC asserts the player-facing strings that C prints for
// a successful wand zap and for an exhausted wand. These are the C messages
// from src/spell_parser.c (mag_objectmagic) used by the original `use` path.
func TestZap_MessagesMatchC(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Zapper", 1001, true)

	proto := makeWandPrototype(1)
	proto.Values[3] = 3 // harmless spell so the success message path runs
	wand := game.NewObjectInstance(proto, -1)
	s.player.Equipment = game.NewEquipment()
	s.player.Inventory = game.NewInventory()
	if err := s.player.Equipment.Equip(wand, s.player.Inventory); err != nil {
		t.Fatalf("equip wand: %v", err)
	}

	// Successful zap at self.
	if err := cmdZap(s, []string{"self"}); err != nil {
		t.Fatalf("cmdZap: %v", err)
	}

	got := drainSendChannel(t, s)
	want := "Your a test wand bathes you in a blinding glow!"
	if !strings.Contains(got, want) {
		t.Errorf("successful zap message did not contain C string %q; got:\n%s", want, got)
	}

	// Second zap should now be out of charges and print C's powerless message.
	if err := cmdZap(s, []string{"self"}); err != nil {
		t.Fatalf("cmdZap second: %v", err)
	}
	got = drainSendChannel(t, s)
	want = "It seems powerless."
	if !strings.Contains(got, want) {
		t.Errorf("depleted zap message did not contain C string %q; got:\n%s", want, got)
	}
}

// TestZap_BroadcastToRoom_SubstitutesActorNameAndPronoun verifies that the
// TO_ROOM broadcasts are pre-substituted: broadcastToRoom does not perform
// C-style $-substitution, so $n/$m must be resolved before calling it.
func TestZap_BroadcastToRoom_SubstitutesActorNameAndPronoun(t *testing.T) {
	m := makeTestManager(t)

	actor := makeTestSession(t, m, "Zapper", 1001, true)
	observer := makeTestSession(t, m, "Watcher", 1001, true)

	// Register both sessions so BroadcastToRoom can deliver to the observer.
	m.mu.Lock()
	m.sessions[actor.player.Name] = actor
	m.sessions[observer.player.Name] = observer
	m.mu.Unlock()

	proto := makeWandPrototype(1)
	proto.Values[3] = 3 // harmless spell so the success path runs
	wand := game.NewObjectInstance(proto, -1)
	actor.player.Equipment = game.NewEquipment()
	actor.player.Inventory = game.NewInventory()
	if err := actor.player.Equipment.Equip(wand, actor.player.Inventory); err != nil {
		t.Fatalf("equip wand: %v", err)
	}

	if err := cmdZap(actor, []string{"self"}); err != nil {
		t.Fatalf("cmdZap: %v", err)
	}

	got := drainSendChannel(t, observer)

	// Must contain the actor's actual name, not the literal token $n.
	if strings.Contains(got, "$n") {
		t.Errorf("room observer saw literal $n in broadcast; got:\n%s", got)
	}
	// Must contain the actor's objective pronoun, not the literal token $m.
	if strings.Contains(got, "$m") {
		t.Errorf("room observer saw literal $m in broadcast; got:\n%s", got)
	}
	want := "Zapper's a test wand bathes him in a blinding glow!"
	if !strings.Contains(got, want) {
		t.Errorf("room broadcast did not contain expected text %q; got:\n%s", want, got)
	}
}

// drainSendChannel returns all text accumulated on the session's send channel.
func drainSendChannel(t *testing.T, s *Session) string {
	t.Helper()
	var out strings.Builder
	for {
		select {
		case msg := <-s.send:
			out.Write(msg)
		default:
			return out.String()
		}
	}
}
