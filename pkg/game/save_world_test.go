package game

import (
	"encoding/json"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestSerializeWorldEmpty verifies that SerializeWorld produces valid JSON
// for a fresh world with no dynamic state.
func TestSerializeWorldEmpty(t *testing.T) {
	pw := &parser.World{
		Rooms: []parser.Room{{VNum: 100, Exits: make(map[string]parser.Exit)}},
		Mobs:  []parser.Mob{},
		Objs:  []parser.Obj{},
		Zones: []parser.Zone{},
	}
	w, err := NewWorld(pw)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	data, err := SerializeWorld(w)
	if err != nil {
		t.Fatalf("SerializeWorld: %v", err)
	}

	var sd saveWorldData
	if err := json.Unmarshal([]byte(data), &sd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if sd.NextMobID != 1 {
		t.Errorf("NextMobID = %d, want 1", sd.NextMobID)
	}
	if sd.NextObjID != 1 {
		t.Errorf("NextObjID = %d, want 1", sd.NextObjID)
	}
	if len(sd.DoorStates) != 0 {
		t.Errorf("DoorStates = %d entries, want 0", len(sd.DoorStates))
	}
	if len(sd.Mobs) != 0 {
		t.Errorf("Mobs = %d entries, want 0", len(sd.Mobs))
	}
	if len(sd.RoomItems) != 0 {
		t.Errorf("RoomItems = %d entries, want 0", len(sd.RoomItems))
	}
}

// TestSerializeWorldDoorStates verifies that door states are serialized correctly.
func TestSerializeWorldDoorStates(t *testing.T) {
	pw := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 100,
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 101, DoorState: 2}, // locked
					"east":  {Direction: "east", ToRoom: 102, DoorState: 0},  // open — should NOT be saved
				},
			},
			{
				VNum:  101,
				Exits: map[string]parser.Exit{},
			},
		},
		Mobs:  []parser.Mob{},
		Objs:  []parser.Obj{},
		Zones: []parser.Zone{},
	}
	w, err := NewWorld(pw)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	data, err := SerializeWorld(w)
	if err != nil {
		t.Fatalf("SerializeWorld: %v", err)
	}

	var sd saveWorldData
	if err := json.Unmarshal([]byte(data), &sd); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(sd.DoorStates) != 1 {
		t.Fatalf("DoorStates has %d rooms, want 1", len(sd.DoorStates))
	}
	dirMap, ok := sd.DoorStates[100]
	if !ok {
		t.Fatal("room 100 not in DoorStates")
	}
	if dirMap["north"] != 2 {
		t.Errorf("north door = %d, want 2 (locked)", dirMap["north"])
	}
	if _, exists := dirMap["east"]; exists {
		t.Error("east door should not be saved (state=0)")
	}
}

// TestDeserializeWorldDoorStates verifies that door states are restored correctly.
func TestDeserializeWorldDoorStates(t *testing.T) {
	pw := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 100,
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 101, DoorState: 0}, // starts open
				},
			},
			{
				VNum:  101,
				Exits: map[string]parser.Exit{},
			},
		},
		Mobs:  []parser.Mob{},
		Objs:  []parser.Obj{},
		Zones: []parser.Zone{},
	}
	w, err := NewWorld(pw)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	// Simulate saved state with north door locked
	saved := saveWorldData{
		NextMobID:  1,
		NextObjID:  1,
		DoorStates: map[int]map[string]int{
			100: {"north": 2},
		},
	}
	raw, _ := json.Marshal(saved)

	if err := DeserializeWorld(string(raw), w); err != nil {
		t.Fatalf("DeserializeWorld: %v", err)
	}

	w.mu.RLock()
	exit := w.rooms[100].Exits["north"]
	w.mu.RUnlock()
	if exit.DoorState != 2 {
		t.Errorf("north door state = %d, want 2 (locked)", exit.DoorState)
	}
}

// TestDeserializeWorldNextIDs verifies that nextID counters are restored
// and never decrease.
func TestDeserializeWorldNextIDs(t *testing.T) {
	pw := &parser.World{
		Rooms: []parser.Room{{VNum: 1, Exits: map[string]parser.Exit{}}},
		Mobs:  []parser.Mob{},
		Objs:  []parser.Obj{},
		Zones: []parser.Zone{},
	}
	w, _ := NewWorld(pw)

	// Set nextIDs higher via deserialization
	saved := saveWorldData{NextMobID: 50, NextObjID: 100}
	raw, _ := json.Marshal(saved)
	if err := DeserializeWorld(string(raw), w); err != nil {
		t.Fatalf("DeserializeWorld: %v", err)
	}

	w.mu.RLock()
	defer w.mu.RUnlock()
	if w.nextMobID != 50 {
		t.Errorf("nextMobID = %d, want 50", w.nextMobID)
	}
	if w.nextObjID != 100 {
		t.Errorf("nextObjID = %d, want 100", w.nextObjID)
	}
}

// TestDeserializeWorldGossip verifies gossip history restoration.
func TestDeserializeWorldGossip(t *testing.T) {
	pw := &parser.World{
		Rooms: []parser.Room{{VNum: 1, Exits: map[string]parser.Exit{}}},
		Mobs:  []parser.Mob{},
		Objs:  []parser.Obj{},
		Zones: []parser.Zone{},
	}
	w, _ := NewWorld(pw)

	saved := saveWorldData{
		Gossip: []saveGossipEntry{
			{Name: "Zax", Message: "hello world", Invis: 0},
			{Name: "Brenda", Message: "systems online", Invis: 1},
		},
	}
	raw, _ := json.Marshal(saved)
	if err := DeserializeWorld(string(raw), w); err != nil {
		t.Fatalf("DeserializeWorld: %v", err)
	}

	w.gossipMu.RLock()
	defer w.gossipMu.RUnlock()
	if len(w.gossipHistory) != 2 {
		t.Fatalf("gossipHistory len = %d, want 2", len(w.gossipHistory))
	}
	if w.gossipHistory[0].Name != "Zax" {
		t.Errorf("gossip[0].Name = %q, want %q", w.gossipHistory[0].Name, "Zax")
	}
	if w.gossipHistory[1].Message != "systems online" {
		t.Errorf("gossip[1].Message = %q, want %q", w.gossipHistory[1].Message, "systems online")
	}
}

// TestSerializeWorldRoundTrip verifies a full serialize → deserialize cycle
// preserves all dynamic state.
func TestSerializeWorldRoundTrip(t *testing.T) {
	pw := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 100,
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 101, DoorState: 1}, // closed
				},
			},
			{
				VNum:  101,
				Exits: map[string]parser.Exit{},
			},
		},
		Mobs: []parser.Mob{
			{VNum: 3000, ShortDesc: "a guard"},
		},
		Objs: []parser.Obj{
			{VNum: 5000, ShortDesc: "a sword"},
		},
		Zones: []parser.Zone{},
	}
	w, err := NewWorld(pw)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	// Spawn a mob and modify its state
	mob, err := w.SpawnMob(3000, 100)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.mu.Lock()
	mob.RoomVNum = 101 // moved from 100 to 101
	mob.CurrentHP = 42
	mob.MaxHP = 80
	mob.mu.Unlock()

	// Add gossip
	w.gossipMu.Lock()
	w.gossipHistory = []gossipEntry{
		{Name: "Test", Message: "hello", Invis: 0},
	}
	w.gossipMu.Unlock()

	// Serialize
	data, err := SerializeWorld(w)
	if err != nil {
		t.Fatalf("SerializeWorld: %v", err)
	}

	// Create a fresh world with same static data
	w2, _ := NewWorld(pw)
	// Simulate zone reset spawning the mob
	_, _ = w2.SpawnMob(3000, 100) // spawns in default room 100

	// Deserialize into the new world
	if err := DeserializeWorld(data, w2); err != nil {
		t.Fatalf("DeserializeWorld: %v", err)
	}

	// Verify door state
	w2.mu.RLock()
	exit := w2.rooms[100].Exits["north"]
	w2.mu.RUnlock()
	if exit.DoorState != 1 {
		t.Errorf("north door state = %d, want 1 (closed)", exit.DoorState)
	}

	// Verify mob was repositioned
	w2.mu.RLock()
	var foundMob *MobInstance
	for _, m := range w2.activeMobs {
		foundMob = m
		break
	}
	w2.mu.RUnlock()
	if foundMob == nil {
		t.Fatal("no active mobs after deserialization")
	}
	foundMob.mu.RLock()
	roomVNum := foundMob.RoomVNum
	hp := foundMob.CurrentHP
	maxHP := foundMob.MaxHP
	foundMob.mu.RUnlock()
	if roomVNum != 101 {
		t.Errorf("mob RoomVNum = %d, want 101", roomVNum)
	}
	if hp != 42 {
		t.Errorf("mob CurrentHP = %d, want 42", hp)
	}
	if maxHP != 80 {
		t.Errorf("mob MaxHP = %d, want 80", maxHP)
	}

	// Verify gossip
	w2.gossipMu.RLock()
	if len(w2.gossipHistory) != 1 || w2.gossipHistory[0].Message != "hello" {
		t.Errorf("gossip not restored: %+v", w2.gossipHistory)
	}
	w2.gossipMu.RUnlock()
}
