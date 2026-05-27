package session

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/game/systems"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// doorSendTimeout is how long we wait for a message to appear on the session send channel.
const doorSendTimeout = 100 * time.Millisecond

// readDoorMessage reads one JSON message from the session's send channel.
// Returns the message text from EventData, or empty string on timeout.
func readDoorMessage(t *testing.T, s *Session) string {
	t.Helper()
	select {
	case raw := <-s.send:
		var msg ServerMessage
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("failed to unmarshal door message: %v", err)
		}
		if msg.Type != MsgEvent {
			t.Fatalf("expected Event message, got %s", msg.Type)
		}
		data, ok := msg.Data.(map[string]interface{})
		if !ok {
			t.Fatalf("unexpected data type %T", msg.Data)
		}
		text, _ := data["text"].(string)
		return text
	case <-time.After(doorSendTimeout):
		return ""
	}
}

// makeTestKey creates a key item prototype with the given VNum.
// TypeFlag 6 = ITEM_KEY.
func makeTestKey(vnum int) *game.ObjectInstance {
	return &game.ObjectInstance{
		VNum: vnum,
		Prototype: &parser.Obj{
			VNum:     vnum,
			TypeFlag: 6,
			Keywords: "key",
		},
	}
}

// makeTestLockpick creates a lockpick item with VNum 8027.
func makeTestLockpick() *game.ObjectInstance {
	return &game.ObjectInstance{
		VNum: 8027,
		Prototype: &parser.Obj{
			VNum:     8027,
			Keywords: "lockpick lock pick",
		},
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorOpen — player opens a closed door
// ---------------------------------------------------------------------------

func TestDoDoorOpen(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Locked:    false,
	}
	dm.AddDoor(door)

	s.doDoorOpen(door, 1001, "north")

	if door.Closed {
		t.Error("door should be open after doDoorOpen")
	}

	msg := readDoorMessage(t, s)
	if msg != "You open the door." {
		t.Errorf("expected 'You open the door.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorOpen_AlreadyOpen — door is already open
// ---------------------------------------------------------------------------

func TestDoDoorOpen_AlreadyOpen(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    false,
		Locked:    false,
	}
	dm.AddDoor(door)

	s.doDoorOpen(door, 1001, "north")

	if door.Closed {
		t.Error("door should remain open")
	}

	msg := readDoorMessage(t, s)
	if msg != "It's already open." {
		t.Errorf("expected 'It\\'s already open.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorOpen_Locked — door is locked, cannot open
// ---------------------------------------------------------------------------

func TestDoDoorOpen_Locked(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Locked:    true,
	}
	dm.AddDoor(door)

	s.doDoorOpen(door, 1001, "north")

	if !door.Closed {
		t.Error("locked door should remain closed")
	}
	if !door.Locked {
		t.Error("locked door should remain locked")
	}

	msg := readDoorMessage(t, s)
	if msg != "It's locked." {
		t.Errorf("expected 'It\\'s locked.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorClose — player closes an open door
// ---------------------------------------------------------------------------

func TestDoDoorClose(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    false,
		Locked:    false,
	}
	dm.AddDoor(door)

	s.doDoorClose(door, 1001, "north")

	if !door.Closed {
		t.Error("door should be closed after doDoorClose")
	}

	msg := readDoorMessage(t, s)
	if msg != "You close the door." {
		t.Errorf("expected 'You close the door.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorClose_AlreadyClosed — door is already closed
// ---------------------------------------------------------------------------

func TestDoDoorClose_AlreadyClosed(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
	}
	dm.AddDoor(door)

	s.doDoorClose(door, 1001, "north")

	if !door.Closed {
		t.Error("door should stay closed")
	}

	msg := readDoorMessage(t, s)
	if msg != "It's already closed." {
		t.Errorf("expected 'It\\'s already closed.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorUnlock — player unlocks a locked door with the correct key
// ---------------------------------------------------------------------------

func TestDoDoorUnlock(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// Give player the key
	err := s.player.Inventory.AddItem(makeTestKey(500))
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Locked:    true,
		KeyVNum:   500,
	}
	dm.AddDoor(door)

	s.doDoorUnlock(door, 1001, "north")

	if door.Locked {
		t.Error("door should be unlocked after doDoorUnlock")
	}

	msg := readDoorMessage(t, s)
	if msg != "You unlock the door." {
		t.Errorf("expected 'You unlock the door.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorUnlock_NoKey — player tries to unlock without the key
// ---------------------------------------------------------------------------

func TestDoDoorUnlock_NoKey(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// No key in inventory

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Locked:    true,
		KeyVNum:   500,
	}
	dm.AddDoor(door)

	s.doDoorUnlock(door, 1001, "north")

	if !door.Locked {
		t.Error("door should remain locked without key")
	}

	msg := readDoorMessage(t, s)
	if msg != "You don't have the right key." {
		t.Errorf("expected 'You don\\'t have the right key.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorUnlock_AlreadyUnlocked — door is already unlocked
// ---------------------------------------------------------------------------

func TestDoDoorUnlock_AlreadyUnlocked(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Locked:    false,
	}
	dm.AddDoor(door)

	s.doDoorUnlock(door, 1001, "north")

	msg := readDoorMessage(t, s)
	if msg != "It's already unlocked." {
		t.Errorf("expected 'It\\'s already unlocked.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorUnlock_NotClosed — door must be closed to unlock
// ---------------------------------------------------------------------------

func TestDoDoorUnlock_NotClosed(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// Player has a key (any key will do — but doDoorUnlock checks Closed first)
	err := s.player.Inventory.AddItem(makeTestKey(500))
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    false,
		Locked:    true,
	}
	dm.AddDoor(door)

	s.doDoorUnlock(door, 1001, "north")

	msg := readDoorMessage(t, s)
	if msg != "You must close it first." {
		t.Errorf("expected 'You must close it first.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorLock — player locks a closed, unlocked door
// ---------------------------------------------------------------------------

func TestDoDoorLock(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	err := s.player.Inventory.AddItem(makeTestKey(500))
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Locked:    false,
		KeyVNum:   500,
	}
	dm.AddDoor(door)

	s.doDoorLock(door, 1001, "north")

	if !door.Locked {
		t.Error("door should be locked after doDoorLock")
	}

	msg := readDoorMessage(t, s)
	if msg != "You lock the door." {
		t.Errorf("expected 'You lock the door.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorLock_AlreadyLocked — door is already locked
// ---------------------------------------------------------------------------

func TestDoDoorLock_AlreadyLocked(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Locked:    true,
	}
	dm.AddDoor(door)

	s.doDoorLock(door, 1001, "north")

	msg := readDoorMessage(t, s)
	if msg != "It's already locked." {
		t.Errorf("expected 'It\\'s already locked.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorLock_NotClosed — door must be closed to lock
// ---------------------------------------------------------------------------

func TestDoDoorLock_NotClosed(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    false,
		Locked:    false,
	}
	dm.AddDoor(door)

	s.doDoorLock(door, 1001, "north")

	msg := readDoorMessage(t, s)
	if msg != "You must close it first." {
		t.Errorf("expected 'You must close it first.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorLock_NoKey — player has no key for the lock
// ---------------------------------------------------------------------------

func TestDoDoorLock_NoKey(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Locked:    false,
		KeyVNum:   500,
	}
	dm.AddDoor(door)

	s.doDoorLock(door, 1001, "north")

	if door.Locked {
		t.Error("door should not be locked without key")
	}

	msg := readDoorMessage(t, s)
	if msg != "You don't have the right key." {
		t.Errorf("expected 'You don\\'t have the right key.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorPick — player picks the lock successfully
// ---------------------------------------------------------------------------

func TestDoDoorPick(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// Give player lockpicks
	err := s.player.Inventory.AddItem(makeTestLockpick())
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	// Set pick lock skill
	s.player.SetSkill(game.SkillPickLock, 80)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:   1001,
		ToRoom:     1002,
		Direction:  "north",
		Closed:     true,
		Locked:     true,
		Pickproof:  false,
		Difficulty: 50,
	}
	dm.AddDoor(door)

	s.doDoorPick(door, 1001, "north")

	if door.Locked {
		t.Error("door should be unlocked after successful pick")
	}

	msg := readDoorMessage(t, s)
	if msg != "You pick the lock." {
		t.Errorf("expected 'You pick the lock.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorPick_NoLockpick — player has no lockpick tool
// ---------------------------------------------------------------------------

func TestDoDoorPick_NoLockpick(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// No lockpick in inventory
	s.player.SetSkill(game.SkillPickLock, 80)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Locked:    true,
		Pickproof: false,
	}
	dm.AddDoor(door)

	s.doDoorPick(door, 1001, "north")

	if !door.Locked {
		t.Error("door should remain locked without lockpick")
	}

	msg := readDoorMessage(t, s)
	if msg != "You don't have any lockpicks." {
		t.Errorf("expected 'You don\\'t have any lockpicks.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorPick_NoSkill — player has no pick lock skill
// ---------------------------------------------------------------------------

func TestDoDoorPick_NoSkill(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	err := s.player.Inventory.AddItem(makeTestLockpick())
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	// No skill set — GetSkill returns 0

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Locked:    true,
		Pickproof: false,
	}
	dm.AddDoor(door)

	s.doDoorPick(door, 1001, "north")

	msg := readDoorMessage(t, s)
	if msg != "You have no idea how to pick locks." {
		t.Errorf("expected 'You have no idea how to pick locks.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorPick_Pickproof — door is pickproof
// ---------------------------------------------------------------------------

func TestDoDoorPick_Pickproof(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	err := s.player.Inventory.AddItem(makeTestLockpick())
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}
	s.player.SetSkill(game.SkillPickLock, 80)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:   1001,
		ToRoom:     1002,
		Direction:  "north",
		Closed:     true,
		Locked:     true,
		Pickproof:  true,
		Difficulty: 50,
	}
	dm.AddDoor(door)

	s.doDoorPick(door, 1001, "north")

	if !door.Locked {
		t.Error("pickproof door should remain locked")
	}

	msg := readDoorMessage(t, s)
	if msg != "This lock is too complex to pick." {
		t.Errorf("expected 'This lock is too complex to pick.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorPick_NotLocked — door is not locked
// ---------------------------------------------------------------------------

func TestDoDoorPick_NotLocked(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Locked:    false,
	}
	dm.AddDoor(door)

	s.doDoorPick(door, 1001, "north")

	msg := readDoorMessage(t, s)
	if msg != "It's not locked." {
		t.Errorf("expected 'It\\'s not locked.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorBash — player bashes down a door
// ---------------------------------------------------------------------------

func TestDoDoorBash(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Strong", 1001, true)

	// High strength to destroy the door in one bash
	// str = 50 + Strength/2; damage = str/10
	// Need damage >= 100 to destroy (Hp=100), so Strength >= 1900
	s.player.Strength = 2500

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Bashable:  true,
		Hp:        100,
		MaxHp:     100,
	}
	dm.AddDoor(door)

	s.doDoorBash(door, 1001, "north")

	if door.Closed {
		t.Error("door should be open after bash success")
	}
	if door.Locked {
		t.Error("door should be unlocked after bash success")
	}
	if door.Hp != 0 {
		t.Errorf("door HP should be 0 after destruction, got %d", door.Hp)
	}

	msg := readDoorMessage(t, s)
	if msg != "You bash the door down!" {
		t.Errorf("expected 'You bash the door down!', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorBash_NotDestroyed — bash damages but does not destroy
// ---------------------------------------------------------------------------

func TestDoDoorBash_NotDestroyed(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Weak", 1001, true)

	// Low strength only does 50/10 = 5 damage
	s.player.Strength = 50

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Bashable:  true,
		Hp:        100,
		MaxHp:     100,
	}
	dm.AddDoor(door)

	s.doDoorBash(door, 1001, "north")

	if !door.Closed {
		t.Error("door should remain closed after partial bash")
	}
	if door.Hp >= 100 {
		t.Errorf("door HP should have decreased, got %d", door.Hp)
	}

	msg := readDoorMessage(t, s)
	if msg != "You bash the door. It looks damaged." {
		t.Errorf("expected 'You bash the door. It looks damaged.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorBash_NotBashable — door cannot be bashed
// ---------------------------------------------------------------------------

func TestDoDoorBash_NotBashable(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Bashable:  false,
	}
	dm.AddDoor(door)

	s.doDoorBash(door, 1001, "north")

	msg := readDoorMessage(t, s)
	if msg != "This door cannot be bashed." {
		t.Errorf("expected 'This door cannot be bashed.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorBash_AlreadyOpen — door is already open
// ---------------------------------------------------------------------------

func TestDoDoorBash_AlreadyOpen(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    false,
	}
	dm.AddDoor(door)

	s.doDoorBash(door, 1001, "north")

	msg := readDoorMessage(t, s)
	if msg != "It's already open." {
		t.Errorf("expected 'It\\'s already open.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoDoorBash_AlreadyDestroyed — door HP is already 0
// ---------------------------------------------------------------------------

func TestDoDoorBash_AlreadyDestroyed(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	dm := getDoorManager(s)
	door := &systems.Door{
		FromRoom:  1001,
		ToRoom:    1002,
		Direction: "north",
		Closed:    true,
		Bashable:  true,
		Hp:        0,
		MaxHp:     100,
	}
	dm.AddDoor(door)

	s.doDoorBash(door, 1001, "north")

	msg := readDoorMessage(t, s)
	if msg != "The door has already been destroyed." {
		t.Errorf("expected 'The door has already been destroyed.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestFindKeyForDoor — player has the matching key
// ---------------------------------------------------------------------------

func TestFindKeyForDoor(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	err := s.player.Inventory.AddItem(makeTestKey(500))
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	door := &systems.Door{
		KeyVNum: 500,
	}

	result := s.findKeyForDoor(door)
	if result != 500 {
		t.Errorf("findKeyForDoor = %d, want 500", result)
	}
}

// ---------------------------------------------------------------------------
// TestFindKeyForDoor_NoKey — player does not have the matching key
// ---------------------------------------------------------------------------

func TestFindKeyForDoor_NoKey(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// No items in inventory at all — playerHasKey returns false

	door := &systems.Door{
		KeyVNum: 500,
	}

	result := s.findKeyForDoor(door)
	if result != -1 {
		t.Errorf("findKeyForDoor = %d, want -1", result)
	}
}

// ---------------------------------------------------------------------------
// TestFindKeyForDoor_WrongKey — player has a different key
// ---------------------------------------------------------------------------

func TestFindKeyForDoor_WrongKey(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	err := s.player.Inventory.AddItem(makeTestKey(501))
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	door := &systems.Door{
		KeyVNum: 500,
	}

	result := s.findKeyForDoor(door)
	if result != -1 {
		t.Errorf("findKeyForDoor = %d, want -1", result)
	}
}

// ---------------------------------------------------------------------------
// TestFindKeyForDoor_AnyKey — door with no specific key uses any ITEM_KEY
// ---------------------------------------------------------------------------

func TestFindKeyForDoor_AnyKey(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// Door has no specific key required (KeyVNum == -1)
	// Player has a generic key item (type 6)
	err := s.player.Inventory.AddItem(makeTestKey(999))
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	door := &systems.Door{
		KeyVNum: -1,
	}

	result := s.findKeyForDoor(door)
	if result != 999 {
		t.Errorf("findKeyForDoor = %d, want 999 (first ITEM_KEY)", result)
	}
}

// ---------------------------------------------------------------------------
// TestFindKeyForDoor_AnyKey_NoItems — no items, -1 for any-key door
// ---------------------------------------------------------------------------

func TestFindKeyForDoor_AnyKey_NoItems(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	door := &systems.Door{
		KeyVNum: -1,
	}

	result := s.findKeyForDoor(door)
	if result != -1 {
		t.Errorf("findKeyForDoor = %d, want -1", result)
	}
}

// ---------------------------------------------------------------------------
// TestFindItemByVNum — player has an item with the given VNum
// ---------------------------------------------------------------------------

func TestFindItemByVNum(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	err := s.player.Inventory.AddItem(&game.ObjectInstance{VNum: 8027})
	if err != nil {
		t.Fatalf("AddItem failed: %v", err)
	}

	if !s.findItemByVNum(8027) {
		t.Error("findItemByVNum(8027) should return true")
	}
}

// ---------------------------------------------------------------------------
// TestFindItemByVNum_NotFound — player does not have the item
// ---------------------------------------------------------------------------

func TestFindItemByVNum_NotFound(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	if s.findItemByVNum(9999) {
		t.Error("findItemByVNum(9999) should return false")
	}
}

// ---------------------------------------------------------------------------
// TestDoGenDoor_NoArgs — each subcommand with no arguments
// ---------------------------------------------------------------------------

func TestDoGenDoor_NoArgs(t *testing.T) {
	tests := []struct {
		name    string
		subcmd  int
		wantMsg string
	}{
		{"open", doorSCMDOpen, "Open what? (Try: open door north)"},
		{"close", doorSCMDClose, "Close what? (Try: close door north)"},
		{"unlock", doorSCMDUnlock, "Unlock what? (Try: unlock door north)"},
		{"lock", doorSCMDLock, "Lock what? (Try: lock door north)"},
		{"pick", doorSCMDPick, "Pick what? (Try: pick door north)"},
		{"bash", doorSCMDBash, "Bash what? (Try: bash door north)"},
	}

	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.doGenDoor(tt.subcmd, []string{})

			msg := readDoorMessage(t, s)
			if msg != tt.wantMsg {
				t.Errorf("expected %q, got %q", tt.wantMsg, msg)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestDoGenDoor_NoDoor — direction given but no door exists
// ---------------------------------------------------------------------------

func TestDoGenDoor_NoDoor(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	// No door added — dm.GetDoor will not find one
	s.doGenDoor(doorSCMDOpen, []string{"north"})

	msg := readDoorMessage(t, s)
	if msg != "There is no door north of here." {
		t.Errorf("expected 'There is no door north of here.', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestDoGenDoor_BadDirection — invalid direction string
// ---------------------------------------------------------------------------

func TestDoGenDoor_BadDirection(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Alice", 1001, true)

	s.doGenDoor(doorSCMDOpen, []string{"xyzzy"})

	msg := readDoorMessage(t, s)
	if msg != "open what door?" {
		t.Errorf("expected 'open what door?', got %q", msg)
	}
}

// ---------------------------------------------------------------------------
// TestCmdDoorName — command name for each subcommand index
// ---------------------------------------------------------------------------

func TestCmdDoorName(t *testing.T) {
	tests := []struct {
		subcmd int
		want   string
	}{
		{doorSCMDOpen, "open"},
		{doorSCMDClose, "close"},
		{doorSCMDUnlock, "unlock"},
		{doorSCMDLock, "lock"},
		{doorSCMDPick, "pick"},
		{doorSCMDBash, "bash"},
		{999, "do"},
	}

	for _, tt := range tests {
		got := cmdDoorName(tt.subcmd)
		if got != tt.want {
			t.Errorf("cmdDoorName(%d) = %q, want %q", tt.subcmd, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// TestGetOppositeDirection — direction reversal
// ---------------------------------------------------------------------------

func TestGetOppositeDirection(t *testing.T) {
	tests := []struct {
		dir  string
		want string
	}{
		{"north", "south"},
		{"south", "north"},
		{"east", "west"},
		{"west", "east"},
		{"up", "down"},
		{"down", "up"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		got := getOppositeDirection(tt.dir)
		if got != tt.want {
			t.Errorf("getOppositeDirection(%q) = %q, want %q", tt.dir, got, tt.want)
		}
	}
}
