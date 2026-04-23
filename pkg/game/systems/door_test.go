package systems

import (
	"testing"
)

func TestNewDoor(t *testing.T) {
	tests := []struct {
		name           string
		fromRoom       int
		toRoom         int
		direction      string
		doorState      int
		keyVNum        int
		expectedClosed bool
		expectedLocked bool
	}{
		{
			name:           "open door",
			fromRoom:       100,
			toRoom:         101,
			direction:      "north",
			doorState:      0,
			keyVNum:        -1,
			expectedClosed: false,
			expectedLocked: false,
		},
		{
			name:           "closed door",
			fromRoom:       100,
			toRoom:         101,
			direction:      "south",
			doorState:      1,
			keyVNum:        -1,
			expectedClosed: true,
			expectedLocked: false,
		},
		{
			name:           "locked door",
			fromRoom:       100,
			toRoom:         101,
			direction:      "east",
			doorState:      2,
			keyVNum:        500,
			expectedClosed: true,
			expectedLocked: true,
		},
		{
			name:           "invalid door state defaults to open",
			fromRoom:       100,
			toRoom:         101,
			direction:      "west",
			doorState:      99,
			keyVNum:        -1,
			expectedClosed: false,
			expectedLocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			door := NewDoor(tt.fromRoom, tt.toRoom, tt.direction, tt.doorState, tt.keyVNum)

			if door.FromRoom != tt.fromRoom {
				t.Errorf("FromRoom = %d, want %d", door.FromRoom, tt.fromRoom)
			}
			if door.ToRoom != tt.toRoom {
				t.Errorf("ToRoom = %d, want %d", door.ToRoom, tt.toRoom)
			}
			if door.Direction != tt.direction {
				t.Errorf("Direction = %s, want %s", door.Direction, tt.direction)
			}
			if door.KeyVNum != tt.keyVNum {
				t.Errorf("KeyVNum = %d, want %d", door.KeyVNum, tt.keyVNum)
			}
			if door.Closed != tt.expectedClosed {
				t.Errorf("Closed = %v, want %v", door.Closed, tt.expectedClosed)
			}
			if door.Locked != tt.expectedLocked {
				t.Errorf("Locked = %v, want %v", door.Locked, tt.expectedLocked)
			}
			if door.Hp != 100 {
				t.Errorf("Hp = %d, want 100", door.Hp)
			}
			if door.MaxHp != 100 {
				t.Errorf("MaxHp = %d, want 100", door.MaxHp)
			}
			if door.Difficulty != 50 {
				t.Errorf("Difficulty = %d, want 50", door.Difficulty)
			}
		})
	}
}

func TestDoor_IsPassable(t *testing.T) {
	tests := []struct {
		name     string
		closed   bool
		locked   bool
		expected bool
	}{
		{"open door", false, false, true},
		{"closed door", true, false, false},
		{"locked door", true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			door := &Door{
				Closed: tt.closed,
				Locked: tt.locked,
			}

			result := door.IsPassable()
			if result != tt.expected {
				t.Errorf("IsPassable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDoor_Open(t *testing.T) {
	tests := []struct {
		name     string
		closed   bool
		locked   bool
		success  bool
		expected string
	}{
		{"already open", false, false, false, "It's already open."},
		{"closed", true, false, true, "You open the door."},
		{"locked", true, true, false, "It's locked."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			door := &Door{
				Closed: tt.closed,
				Locked: tt.locked,
			}

			success, msg := door.Open()
			if success != tt.success {
				t.Errorf("Open() success = %v, want %v", success, tt.success)
			}
			if msg != tt.expected {
				t.Errorf("Open() msg = %q, want %q", msg, tt.expected)
			}
			if success && door.Closed {
				t.Error("Open() should set Closed to false on success")
			}
		})
	}
}

func TestDoor_Close(t *testing.T) {
	tests := []struct {
		name     string
		closed   bool
		success  bool
		expected string
	}{
		{"already closed", true, false, "It's already closed."},
		{"open", false, true, "You close the door."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			door := &Door{
				Closed: tt.closed,
			}

			success, msg := door.Close()
			if success != tt.success {
				t.Errorf("Close() success = %v, want %v", success, tt.success)
			}
			if msg != tt.expected {
				t.Errorf("Close() msg = %q, want %q", msg, tt.expected)
			}
			if success && !door.Closed {
				t.Error("Close() should set Closed to true on success")
			}
		})
	}
}

func TestDoor_Lock(t *testing.T) {
	tests := []struct {
		name     string
		closed   bool
		locked   bool
		keyVNum  int
		useKey   int
		success  bool
		expected string
	}{
		{"already locked", true, true, 500, 500, false, "It's already locked."},
		{"not closed", false, false, 500, 500, false, "You must close it first."},
		{"wrong key", true, false, 500, 501, false, "You don't have the right key."},
		{"correct key", true, false, 500, 500, true, "You lock the door."},
		{"no key required", true, false, -1, 500, true, "You lock the door."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			door := &Door{
				Closed:  tt.closed,
				Locked:  tt.locked,
				KeyVNum: tt.keyVNum,
			}

			success, msg := door.Lock(tt.useKey)
			if success != tt.success {
				t.Errorf("Lock() success = %v, want %v", success, tt.success)
			}
			if msg != tt.expected {
				t.Errorf("Lock() msg = %q, want %q", msg, tt.expected)
			}
			if success && !door.Locked {
				t.Error("Lock() should set Locked to true on success")
			}
		})
	}
}

func TestDoor_Unlock(t *testing.T) {
	tests := []struct {
		name     string
		locked   bool
		keyVNum  int
		useKey   int
		success  bool
		expected string
	}{
		{"already unlocked", false, 500, 500, false, "It's already unlocked."},
		{"wrong key", true, 500, 501, false, "You don't have the right key."},
		{"correct key", true, 500, 500, true, "You unlock the door."},
		{"no key required", true, -1, 500, true, "You unlock the door."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			door := &Door{
				Locked:  tt.locked,
				KeyVNum: tt.keyVNum,
			}

			success, msg := door.Unlock(tt.useKey)
			if success != tt.success {
				t.Errorf("Unlock() success = %v, want %v", success, tt.success)
			}
			if msg != tt.expected {
				t.Errorf("Unlock() msg = %q, want %q", msg, tt.expected)
			}
			if success && door.Locked {
				t.Error("Unlock() should set Locked to false on success")
			}
		})
	}
}

func TestDoor_Pick(t *testing.T) {
	tests := []struct {
		name       string
		locked     bool
		pickproof  bool
		difficulty int
		skill      int
		success    bool
		expected   string
	}{
		{"not locked", false, false, 50, 100, false, "It's not locked."},
		{"pickproof", true, true, 50, 100, false, "This lock is too complex to pick."},
		{"skill too low", true, false, 80, 50, false, "You fail to pick the lock."},
		{"skill sufficient", true, false, 50, 80, true, "You pick the lock."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			door := &Door{
				Locked:     tt.locked,
				Pickproof:  tt.pickproof,
				Difficulty: tt.difficulty,
			}

			success, msg := door.Pick(tt.skill)
			if success != tt.success {
				t.Errorf("Pick() success = %v, want %v", success, tt.success)
			}
			if msg != tt.expected {
				t.Errorf("Pick() msg = %q, want %q", msg, tt.expected)
			}
			if success && door.Locked {
				t.Error("Pick() should set Locked to false on success")
			}
		})
	}
}

func TestDoor_Bash(t *testing.T) {
	tests := []struct {
		name     string
		closed   bool
		bashable bool
		hp       int
		strength int
		success  bool
		expectHp int
	}{
		{"already open", false, true, 100, 50, false, 100},
		{"not bashable", true, false, 100, 50, false, 100},
		{"bash with damage", true, true, 100, 50, false, 95}, // 50/10 = 5 damage
		{"bash destroyed", true, true, 5, 50, true, 0},       // 50/10 = 5 damage, destroys door
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			door := &Door{
				Closed:   tt.closed,
				Bashable: tt.bashable,
				Hp:       tt.hp,
				MaxHp:    tt.hp,
			}

			success, _ := door.Bash(tt.strength)
			if success != tt.success {
				t.Errorf("Bash() success = %v, want %v", success, tt.success)
			}
			if door.Hp != tt.expectHp {
				t.Errorf("Bash() Hp = %d, want %d", door.Hp, tt.expectHp)
			}
			if success && door.Closed {
				t.Error("Bash() should set Closed to false when door is destroyed")
			}
			if success && door.Locked {
				t.Error("Bash() should set Locked to false when door is destroyed")
			}
		})
	}
}

func TestDoor_GetStatus(t *testing.T) {
	tests := []struct {
		name     string
		hidden   bool
		closed   bool
		locked   bool
		expected string
	}{
		{"hidden", true, false, false, "hidden"},
		{"open", false, false, false, "open"},
		{"closed", false, true, false, "closed"},
		{"locked", false, true, true, "closed and locked"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			door := &Door{
				Hidden: tt.hidden,
				Closed: tt.closed,
				Locked: tt.locked,
			}

			result := door.GetStatus()
			if result != tt.expected {
				t.Errorf("GetStatus() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDoorManager_AddGetDoor(t *testing.T) {
	dm := NewDoorManager()

	door := &Door{
		FromRoom:  100,
		ToRoom:    101,
		Direction: "north",
	}

	dm.AddDoor(door)

	// Test GetDoor
	retrieved, ok := dm.GetDoor(100, "north")
	if !ok {
		t.Fatal("GetDoor() should find the door")
	}
	if retrieved != door {
		t.Error("GetDoor() should return the same door")
	}

	// Test GetDoorBetween
	retrieved2, ok := dm.GetDoorBetween(100, 101)
	if !ok {
		t.Fatal("GetDoorBetween() should find the door")
	}
	if retrieved2 != door {
		t.Error("GetDoorBetween() should return the same door")
	}

	// Test reverse direction
	retrieved3, ok := dm.GetDoorBetween(101, 100)
	if !ok {
		t.Fatal("GetDoorBetween() should find the door in reverse")
	}
	if retrieved3 != door {
		t.Error("GetDoorBetween() should return the same door in reverse")
	}

	// Test non-existent door
	_, ok = dm.GetDoor(100, "south")
	if ok {
		t.Error("GetDoor() should not find non-existent door")
	}

	_, ok = dm.GetDoorBetween(100, 102)
	if ok {
		t.Error("GetDoorBetween() should not find non-existent connection")
	}
}

func TestDoorManager_RemoveDoor(t *testing.T) {
	dm := NewDoorManager()

	door := &Door{
		FromRoom:  100,
		ToRoom:    101,
		Direction: "north",
	}

	dm.AddDoor(door)

	// Verify door exists
	_, ok := dm.GetDoor(100, "north")
	if !ok {
		t.Fatal("Door should exist before removal")
	}

	// Remove door
	dm.RemoveDoor(100, "north")

	// Verify door is gone
	_, ok = dm.GetDoor(100, "north")
	if ok {
		t.Error("Door should not exist after removal")
	}
}

func TestDoorManager_GetDoorsInRoom(t *testing.T) {
	dm := NewDoorManager()

	doors := []*Door{
		{FromRoom: 100, ToRoom: 101, Direction: "north"},
		{FromRoom: 100, ToRoom: 102, Direction: "east"},
		{FromRoom: 101, ToRoom: 103, Direction: "south"},
	}

	for _, door := range doors {
		dm.AddDoor(door)
	}

	// Get doors in room 100
	room100Doors := dm.GetDoorsInRoom(100)
	if len(room100Doors) != 2 {
		t.Errorf("GetDoorsInRoom(100) = %d doors, want 2", len(room100Doors))
	}

	// Get doors in room 101
	room101Doors := dm.GetDoorsInRoom(101)
	if len(room101Doors) != 1 {
		t.Errorf("GetDoorsInRoom(101) = %d doors, want 1", len(room101Doors))
	}

	// Get doors in room 103 (should have none)
	room103Doors := dm.GetDoorsInRoom(103)
	if len(room103Doors) != 0 {
		t.Errorf("GetDoorsInRoom(103) = %d doors, want 0", len(room103Doors))
	}
}

func TestDoorManager_CanPass(t *testing.T) {
	dm := NewDoorManager()

	// Add an open door
	openDoor := &Door{
		FromRoom:  100,
		ToRoom:    101,
		Direction: "north",
		Closed:    false,
		Hidden:    false,
	}
	dm.AddDoor(openDoor)

	// Add a closed door
	closedDoor := &Door{
		FromRoom:  100,
		ToRoom:    102,
		Direction: "east",
		Closed:    true,
		Hidden:    false,
	}
	dm.AddDoor(closedDoor)

	// Add a locked door
	lockedDoor := &Door{
		FromRoom:  100,
		ToRoom:    103,
		Direction: "south",
		Closed:    true,
		Locked:    true,
		Hidden:    false,
	}
	dm.AddDoor(lockedDoor)

	// Add a hidden door
	hiddenDoor := &Door{
		FromRoom:  100,
		ToRoom:    104,
		Direction: "west",
		Closed:    false,
		Hidden:    true,
	}
	dm.AddDoor(hiddenDoor)

	tests := []struct {
		name     string
		room     int
		dir      string
		success  bool
		expected string
	}{
		{"open door", 100, "north", true, ""},
		{"closed door", 100, "east", false, "The door is closed."},
		{"locked door", 100, "south", false, "The door is locked."},
		{"hidden door", 100, "west", false, "There is no door there."},
		{"non-existent door", 100, "up", false, "There is no door there."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success, msg := dm.CanPass(tt.room, tt.dir)
			if success != tt.success {
				t.Errorf("CanPass() success = %v, want %v", success, tt.success)
			}
			if msg != tt.expected {
				t.Errorf("CanPass() msg = %q, want %q", msg, tt.expected)
			}
		})
	}
}

func TestDoorManager_Count(t *testing.T) {
	dm := NewDoorManager()

	if dm.Count() != 0 {
		t.Errorf("Count() = %d, want 0", dm.Count())
	}

	// Add some doors
	doors := []*Door{
		{FromRoom: 100, ToRoom: 101, Direction: "north"},
		{FromRoom: 100, ToRoom: 102, Direction: "east"},
		{FromRoom: 101, ToRoom: 103, Direction: "south"},
	}

	for _, door := range doors {
		dm.AddDoor(door)
	}

	if dm.Count() != 3 {
		t.Errorf("Count() = %d, want 3", dm.Count())
	}
}
