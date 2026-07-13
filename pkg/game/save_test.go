package game

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestPlayerSerializationRoundTrip(t *testing.T) {
	player := NewPlayer(99, "SaveTestPlayer", 1001)
	player.SetLevel(10)
	player.SetExp(5000)
	player.BankGold = 2500
	player.Title = "the Test Hero"
	player.Description = "A heroic look."
	player.SetCondition(CondFull, 12)
	player.SetCondition(CondThirst, 18)
	player.SetCondition(CondDrunk, 2)
	player.Stats.Str = 18
	player.Stats.StrAdd = 100
	player.Stats.Dex = 16

	// Round-trip serialize
	serialized, err := SerializePlayer(player)
	if err != nil {
		t.Fatalf("SerializePlayer failed: %v", err)
	}

	deserialized, err := DeserializePlayer(serialized)
	if err != nil {
		t.Fatalf("DeserializePlayer failed: %v", err)
	}

	// Verify fields
	if deserialized.ID != player.ID {
		t.Errorf("ID = %d, want %d", deserialized.ID, player.ID)
	}
	if deserialized.Name != player.Name {
		t.Errorf("Name = %q, want %q", deserialized.Name, player.Name)
	}
	if deserialized.GetLevel() != player.GetLevel() {
		t.Errorf("Level = %d, want %d", deserialized.GetLevel(), player.GetLevel())
	}
	if deserialized.GetExp() != player.GetExp() {
		t.Errorf("Exp = %d, want %d", deserialized.GetExp(), player.GetExp())
	}
	if deserialized.BankGold != player.BankGold {
		t.Errorf("BankGold = %d, want %d", deserialized.BankGold, player.BankGold)
	}
	if deserialized.Title != player.Title {
		t.Errorf("Title = %q, want %q", deserialized.Title, player.Title)
	}
	if deserialized.Description != player.Description {
		t.Errorf("Description = %q, want %q", deserialized.Description, player.Description)
	}
	if deserialized.Hunger != player.Hunger {
		t.Errorf("Hunger = %d, want %d", deserialized.Hunger, player.Hunger)
	}
	if deserialized.Thirst != player.Thirst {
		t.Errorf("Thirst = %d, want %d", deserialized.Thirst, player.Thirst)
	}
	if deserialized.Drunk != player.Drunk {
		t.Errorf("Drunk = %d, want %d", deserialized.Drunk, player.Drunk)
	}
	if deserialized.Inventory.MaxWeight != 480 {
		t.Errorf("MaxWeight = %d, want 480", deserialized.Inventory.MaxWeight)
	}
	if deserialized.Inventory.Capacity != 18 {
		t.Errorf("Capacity = %d, want 18", deserialized.Inventory.Capacity)
	}
}

func TestSaveLoadPlayerDisk(t *testing.T) {
	// Create uniquely named test player
	testName := "Test_Save_Disk_Plr_99"
	player := NewPlayer(999, testName, 1001)
	player.SetLevel(5)
	player.SetHP(80)
	player.SetMaxHP(100)
	player.BankGold = 1234

	// Save player
	err := SavePlayer(player)
	if err != nil {
		t.Fatalf("SavePlayer failed: %v", err)
	}

	t.Cleanup(func() {
		_ = DeletePlayer(testName)
	})

	// Verify save exists
	if !PlayerSaveExists(testName) {
		t.Errorf("PlayerSaveExists(%q) should be true", testName)
	}

	// Load player
	loaded, err := LoadPlayer(testName)
	if err != nil {
		t.Fatalf("LoadPlayer failed: %v", err)
	}

	// Verify fields
	if loaded.Name != player.Name {
		t.Errorf("Loaded Name = %q, want %q", loaded.Name, player.Name)
	}
	if loaded.GetLevel() != player.GetLevel() {
		t.Errorf("Loaded Level = %d, want %d", loaded.GetLevel(), player.GetLevel())
	}
	if loaded.GetHP() != player.GetHP() {
		t.Errorf("Loaded HP = %d, want %d", loaded.GetHP(), player.GetHP())
	}
	if loaded.GetMaxHP() != player.GetMaxHP() {
		t.Errorf("Loaded MaxHP = %d, want %d", loaded.GetMaxHP(), player.GetMaxHP())
	}
	if loaded.BankGold != player.BankGold {
		t.Errorf("Loaded BankGold = %d, want %d", loaded.BankGold, player.BankGold)
	}

	// Delete player
	err = DeletePlayer(testName)
	if err != nil {
		t.Fatalf("DeletePlayer failed: %v", err)
	}

	// Verify save no longer exists
	if PlayerSaveExists(testName) {
		t.Errorf("PlayerSaveExists(%q) should be false after DeletePlayer", testName)
	}
}

func TestSaveLoadPlayerGoldAndBank(t *testing.T) {
	testName := "Test_GoldBank_Disk_Plr_99"
	player := NewPlayer(888, testName, 1002)
	player.SetLevel(12)
	player.SetGold(500)
	player.BankGold = 3000

	err := SavePlayer(player)
	if err != nil {
		t.Fatalf("SavePlayer failed: %v", err)
	}

	t.Cleanup(func() {
		_ = DeletePlayer(testName)
	})

	loaded, err := LoadPlayer(testName)
	if err != nil {
		t.Fatalf("LoadPlayer failed: %v", err)
	}

	if loaded.GetGold() != 500 {
		t.Errorf("Loaded Gold = %d, want 500", loaded.GetGold())
	}
	if loaded.BankGold != 3000 {
		t.Errorf("Loaded BankGold = %d, want 3000", loaded.BankGold)
	}
}

func TestSaveLoadPlayer_ConditionsPersistAfterGain(t *testing.T) {
	testName := "Test_Conditions_Persist_Plr_99"
	player := NewPlayer(777, testName, 1001)
	player.SetCondition(CondFull, 10)
	player.SetCondition(CondThirst, 15)
	player.SetCondition(CondDrunk, 0)

	// Mutate conditions the way consumables do (via GainCondition).
	GainCondition(player, CondFull, 5)
	GainCondition(player, CondThirst, -3)
	GainCondition(player, CondDrunk, 2)

	// Legacy fields should already be in sync, but assert it explicitly.
	if player.Hunger != player.Conditions[CondFull] {
		t.Fatalf("pre-save Hunger %d out of sync with Conditions[Full] %d", player.Hunger, player.Conditions[CondFull])
	}

	err := SavePlayer(player)
	if err != nil {
		t.Fatalf("SavePlayer failed: %v", err)
	}
	t.Cleanup(func() {
		_ = DeletePlayer(testName)
	})

	loaded, err := LoadPlayer(testName)
	if err != nil {
		t.Fatalf("LoadPlayer failed: %v", err)
	}

	if loaded.GetCondition(CondFull) != 15 {
		t.Errorf("Full condition = %d, want 15", loaded.GetCondition(CondFull))
	}
	if loaded.GetCondition(CondThirst) != 12 {
		t.Errorf("Thirst condition = %d, want 12", loaded.GetCondition(CondThirst))
	}
	if loaded.GetCondition(CondDrunk) != 2 {
		t.Errorf("Drunk condition = %d, want 2", loaded.GetCondition(CondDrunk))
	}
	// Ensure legacy fields also reflect the persisted values.
	if loaded.Hunger != 15 {
		t.Errorf("Hunger = %d, want 15", loaded.Hunger)
	}
	if loaded.Thirst != 12 {
		t.Errorf("Thirst = %d, want 12", loaded.Thirst)
	}
	if loaded.Drunk != 2 {
		t.Errorf("Drunk = %d, want 2", loaded.Drunk)
	}
}

func TestWorldSerializationRoundTrip(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1001, Name: "Room A", Zone: 1,
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 1002, DoorState: 1}, // doorClosed
				},
			},
			{
				VNum: 1002, Name: "Room B", Zone: 1,
				Exits: map[string]parser.Exit{
					"south": {Direction: "south", ToRoom: 1001, DoorState: 0},
				},
			},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	w.nextMobID = 100
	w.nextObjID = 200

	// Set gossip entry
	w.gossipHistory = append(w.gossipHistory, gossipEntry{
		Name:    "GossipTest",
		Message: "Hello World",
		Invis:   0,
	})

	// Round-trip serialize
	serialized, err := SerializeWorld(w)
	if err != nil {
		t.Fatalf("SerializeWorld failed: %v", err)
	}

	// Restore onto a fresh world with same schema
	w2, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w2.StopAITicker() })

	err = DeserializeWorld(serialized, w2)
	if err != nil {
		t.Fatalf("DeserializeWorld failed: %v", err)
	}

	if w2.nextMobID != w.nextMobID {
		t.Errorf("nextMobID = %d, want %d", w2.nextMobID, w.nextMobID)
	}
	if w2.nextObjID != w.nextObjID {
		t.Errorf("nextObjID = %d, want %d", w2.nextObjID, w.nextObjID)
	}

	// Verify door state
	room1001, ok := w2.rooms[1001]
	if !ok {
		t.Fatal("room 1001 missing in deserialized world")
	}
	exitNorth := room1001.Exits["north"]
	if exitNorth.DoorState != 1 {
		t.Errorf("door state north = %d, want 1", exitNorth.DoorState)
	}

	// Verify gossip
	if len(w2.gossipHistory) != 1 {
		t.Errorf("gossipCount = %d, want 1", len(w2.gossipHistory))
	} else {
		entry := w2.gossipHistory[0]
		if entry.Name != "GossipTest" || entry.Message != "Hello World" {
			t.Errorf("unexpected gossip entry: %+v", entry)
		}
	}
}

func TestSaveLoadWorldDisk(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 5001, Name: "Test Room", Zone: 5,
				Exits: map[string]parser.Exit{
					"up": {Direction: "up", ToRoom: 5002, DoorState: 2}, // doorLocked
				},
			},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	w.nextMobID = 50
	w.nextObjID = 60

	// Save world to disk (uses worldStateFile constant path)
	err = SaveWorld(w)
	if err != nil {
		t.Fatalf("SaveWorld failed: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Remove("./data/world_state.json")
	})

	// Load onto a fresh world
	w2, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w2.StopAITicker() })

	err = LoadWorld(w2)
	if err != nil {
		t.Fatalf("LoadWorld failed: %v", err)
	}

	if w2.nextMobID != w.nextMobID {
		t.Errorf("Loaded world nextMobID = %d, want %d", w2.nextMobID, w.nextMobID)
	}
	if w2.nextObjID != w.nextObjID {
		t.Errorf("Loaded world nextObjID = %d, want %d", w2.nextObjID, w.nextObjID)
	}

	// Verify door state
	room, ok := w2.rooms[5001]
	if !ok {
		t.Fatal("room 5001 missing after LoadWorld")
	}
	exitUp := room.Exits["up"]
	if exitUp.DoorState != 2 {
		t.Errorf("Loaded world door state = %d, want 2", exitUp.DoorState)
	}
}

func TestSavePlayer_IncludesVersion(t *testing.T) {
	player := NewPlayer(77, "VersionTestPlayer", 1001)
	player.SetLevel(5)

	data := playerToSaveData(player)
	if data.SaveVersion != CurrentSaveVersion {
		t.Errorf("SaveVersion = %d, want %d", data.SaveVersion, CurrentSaveVersion)
	}

	// Also verify through serialization
	serialized, err := SerializePlayer(player)
	if err != nil {
		t.Fatalf("SerializePlayer failed: %v", err)
	}

	var decoded savePlayerData
	if err := jsonUnmarshal([]byte(serialized), &decoded); err != nil {
		t.Fatalf("unmarshal serialized data: %v", err)
	}
	if decoded.SaveVersion != CurrentSaveVersion {
		t.Errorf("SaveVersion after round-trip = %d, want %d", decoded.SaveVersion, CurrentSaveVersion)
	}
}

// jsonUnmarshal is a test helper that calls json.Unmarshal.
func jsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func TestLoadPlayer_OldSave_NoVersion_Warns(t *testing.T) {
	// Simulate an old save file without save_version field.
	oldJSON := `{
		"id": 88,
		"name": "OldSaveGuy",
		"level": 10
	}`

	loaded, err := DeserializePlayer(oldJSON)
	if err != nil {
		t.Fatalf("DeserializePlayer should succeed for old format: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded player should not be nil")
	}
	if loaded.GetLevel() != 10 {
		t.Errorf("Level = %d, want 10", loaded.GetLevel())
	}
}

func TestLoadPlayer_FutureVersion_Warns(t *testing.T) {
	// Simulate a save file from a future version.
	futureJSON := `{
		"save_version": 99,
		"id": 42,
		"name": "FutureGuy",
		"level": 20
	}`

	loaded, err := DeserializePlayer(futureJSON)
	if err != nil {
		t.Fatalf("DeserializePlayer should still succeed for future version: %v", err)
	}
	if loaded == nil {
		t.Fatal("loaded player should not be nil")
	}
	if loaded.GetLevel() != 20 {
		t.Errorf("Level = %d, want 20", loaded.GetLevel())
	}
}

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"Alice", "Alice"},
		{"Alice-Bob", "Alice-Bob"},
		{"Alice_Bob", "Alice_Bob"},
		{"Alice/Bob", "AliceBob"},
		{"Alice\\Bob", "AliceBob"},
		{"Alice!@#$%^&*()", "Alice"},
		{"..Alice..", "Alice"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
