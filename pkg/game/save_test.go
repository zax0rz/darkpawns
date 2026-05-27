package game

import (
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
	player.Hunger = 12
	player.Thirst = 18
	player.Drunk = 2

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

func TestSavePlayerWithRent(t *testing.T) {
	testName := "Test_Rent_Disk_Plr_99"
	player := NewPlayer(888, testName, 1002)
	player.SetLevel(12)
	player.SetGold(500)
	player.BankGold = 3000

	err := SavePlayerWithRent(player, 2, 50) // RentRented=2, cost=50
	if err != nil {
		t.Fatalf("SavePlayerWithRent failed: %v", err)
	}

	t.Cleanup(func() {
		_ = DeletePlayer(testName)
	})

	// Load save data directly
	data, err := LoadSaveData(testName)
	if err != nil {
		t.Fatalf("LoadSaveData failed: %v", err)
	}

	if data.RentCode != 2 {
		t.Errorf("Loaded RentCode = %d, want 2", data.RentCode)
	}
	if data.NetCostPerDiem != 50 {
		t.Errorf("Loaded NetCostPerDiem = %d, want 50", data.NetCostPerDiem)
	}
	if data.SavedGold != 500 {
		t.Errorf("Loaded SavedGold = %d, want 500", data.SavedGold)
	}
	if data.SavedBankGold != 3000 {
		t.Errorf("Loaded SavedBankGold = %d, want 3000", data.SavedBankGold)
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
