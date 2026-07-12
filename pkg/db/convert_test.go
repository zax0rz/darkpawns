package db

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestPlayerToRecordAndBack(t *testing.T) {
	// Create mock world with object prototypes
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Spawn Room", Zone: 1},
		},
		Objs: []parser.Obj{
			{
				VNum:      10,
				ShortDesc: "a shiny gold coin",
				Keywords:  "coin gold shiny",
			},
			{
				VNum:      20,
				ShortDesc: "a broad sword",
				Keywords:  "sword broad",
			},
		},
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { world.StopAITicker() })

	// Set up a player
	p := game.NewPlayer(123, "ConvertTestPlayer", 1001)
	p.SetLevel(15)
	p.SetExp(25000)
	p.SetHP(85)
	p.SetMaxHP(100)
	p.SetMove(90)
	p.SetMaxMove(100)
	p.Gold = 500
	p.BankGold = 1000
	p.Stats.Str = 18
	p.Stats.StrAdd = 50
	p.Stats.Int = 14
	p.Stats.Wis = 12
	p.Stats.Dex = 15
	p.Stats.Con = 13
	p.Stats.Cha = 11
	p.Hunger = 20
	p.Thirst = 24
	p.Drunk = 0

	// Add inventory item
	proto10, ok := world.GetObjPrototype(10)
	if !ok {
		t.Fatal("expected prototype 10")
	}
	item10 := game.NewObjectInstance(proto10, -1)
	_ = p.Inventory.AddItem(item10)

	// Equip item
	proto20, ok := world.GetObjPrototype(20)
	if !ok {
		t.Fatal("expected prototype 20")
	}
	item20 := game.NewObjectInstance(proto20, -1)
	// Equip in slot Wield
	slot := game.SlotWield
	p.Equipment.Slots[slot] = item20

	// Convert Player to Record
	rec, err := PlayerToRecord(p, nil)
	if err != nil {
		t.Fatalf("PlayerToRecord failed: %v", err)
	}

	// Verify record values
	if rec.ID != p.ID {
		t.Errorf("Record ID = %d, want %d", rec.ID, p.ID)
	}
	if rec.Name != p.Name {
		t.Errorf("Record Name = %q, want %q", rec.Name, p.Name)
	}
	if rec.Level != p.Level {
		t.Errorf("Record Level = %d, want %d", rec.Level, p.Level)
	}
	if rec.Exp != p.Exp {
		t.Errorf("Record Exp = %d, want %d", rec.Exp, p.Exp)
	}
	if rec.StatStr != p.Stats.Str {
		t.Errorf("Record Str = %d, want %d", rec.StatStr, p.Stats.Str)
	}

	// Restore Player from Record
	restored, err := RecordToPlayer(rec, world)
	if err != nil {
		t.Fatalf("RecordToPlayer failed: %v", err)
	}

	// Verify restored values
	if restored.ID != p.ID {
		t.Errorf("Restored ID = %d, want %d", restored.ID, p.ID)
	}
	if restored.Name != p.Name {
		t.Errorf("Restored Name = %q, want %q", restored.Name, p.Name)
	}
	if restored.Level != p.Level {
		t.Errorf("Restored Level = %d, want %d", restored.Level, p.Level)
	}
	if restored.GetRoom() != p.GetRoom() {
		t.Errorf("Restored Room = %d, want %d", restored.GetRoom(), p.GetRoom())
	}
	if restored.Stats.Str != p.Stats.Str {
		t.Errorf("Restored Stats.Str = %d, want %d", restored.Stats.Str, p.Stats.Str)
	}
	if restored.Inventory.MaxWeight != 280 {
		t.Errorf("Restored MaxWeight = %d, want 280", restored.Inventory.MaxWeight)
	}
	if restored.Inventory.Capacity != 19 {
		t.Errorf("Restored Capacity = %d, want 19", restored.Inventory.Capacity)
	}

	// Verify inventory restored
	invItems := restored.Inventory.FindItems("")
	if len(invItems) != 1 {
		t.Fatalf("expected 1 restored inventory item, got %d", len(invItems))
	}
	if invItems[0].GetVNum() != 10 {
		t.Errorf("restored item VNum = %d, want 10", invItems[0].GetVNum())
	}

	// Verify equipment restored
	eqItem := restored.Equipment.Slots[slot]
	if eqItem == nil {
		t.Fatal("expected equipped item in slot Wield")
	}
	if eqItem.GetVNum() != 20 {
		t.Errorf("restored equipped VNum = %d, want 20", eqItem.GetVNum())
	}
}

// TestRecordToPlayer_OverCapacityInventoryNotDropped guards against silent item
// loss on load: a saved inventory larger than the current capacity must be fully
// restored, not truncated. Before the fix, RecordToPlayer discarded the
// ErrInventoryFull from AddItem and dropped the overflow.
func TestRecordToPlayer_OverCapacityInventoryNotDropped(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Spawn Room", Zone: 1}},
		Objs:  []parser.Obj{{VNum: 10, ShortDesc: "a gold coin", Keywords: "coin gold"}},
	}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { world.StopAITicker() })

	proto, ok := world.GetObjPrototype(10)
	if !ok {
		t.Fatal("expected prototype 10")
	}

	p := game.NewPlayer(321, "PackRat", 1001)
	const itemCount = 30 // exceeds the default capacity of 20
	p.Inventory.Capacity = 5
	for i := 0; i < itemCount; i++ {
		p.Inventory.RestoreItem(game.NewObjectInstance(proto, -1))
	}
	if got := len(p.Inventory.FindItems("")); got != itemCount {
		t.Fatalf("setup: inventory has %d items, want %d", got, itemCount)
	}

	rec, err := PlayerToRecord(p, nil)
	if err != nil {
		t.Fatalf("PlayerToRecord failed: %v", err)
	}
	restored, err := RecordToPlayer(rec, world)
	if err != nil {
		t.Fatalf("RecordToPlayer failed: %v", err)
	}

	if got := len(restored.Inventory.FindItems("")); got != itemCount {
		t.Errorf("restored inventory has %d items, want %d (items dropped on load)", got, itemCount)
	}
}
