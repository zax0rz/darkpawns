package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCalculateAC(t *testing.T) {
	t.Run("default player AC", func(t *testing.T) {
		p := game.NewPlayer(1, "Tester", 1001)
		got := CalculateAC(p)
		if got != p.AC {
			t.Errorf("CalculateAC with no equipment = %d, want %d", got, p.AC)
		}
	})

	t.Run("with armor equipped", func(t *testing.T) {
		p := game.NewPlayer(1, "Tester", 1001)
		p.Equipment = game.NewEquipment()
		p.Inventory = game.NewInventory()

		proto := &parser.Obj{
			VNum:      99001,
			Keywords:  "test armor",
			ShortDesc: "test armor",
			TypeFlag:  game.ITEM_ARMOR,
			WearFlags: [4]int{(1 << 0) | (1 << 3)}, // TAKE + BODY
			Values:    [4]int{7, 0, 0, 0},
			Weight:    5,
			Cost:      10,
		}
		armor := game.NewObjectInstance(proto, -1)
		if err := p.Equipment.Equip(armor, p.Inventory); err != nil {
			t.Fatalf("Equip: %v", err)
		}

		want := p.AC - 7
		if got := CalculateAC(p); got != want {
			t.Errorf("CalculateAC with armor = %d, want %d", got, want)
		}
	})
}

func TestGetEquipmentString(t *testing.T) {
	t.Run("no equipment", func(t *testing.T) {
		p := game.NewPlayer(1, "Tester", 1001)
		p.Equipment = game.NewEquipment()
		got := GetEquipmentString(p)
		if got != "You are not wearing anything." {
			t.Errorf("empty equipment = %q, want %q", got, "You are not wearing anything.")
		}
	})

	t.Run("with equipped item", func(t *testing.T) {
		p := game.NewPlayer(1, "Tester", 1001)
		p.Equipment = game.NewEquipment()
		p.Inventory = game.NewInventory()

		proto := &parser.Obj{
			VNum:      99001,
			Keywords:  "test armor",
			ShortDesc: "test armor",
			TypeFlag:  game.ITEM_ARMOR,
			WearFlags: [4]int{(1 << 0) | (1 << 3)}, // TAKE + BODY
			Values:    [4]int{7, 0, 0, 0},
			Weight:    5,
			Cost:      10,
		}
		armor := game.NewObjectInstance(proto, -1)
		if err := p.Equipment.Equip(armor, p.Inventory); err != nil {
			t.Fatalf("Equip: %v", err)
		}

		got := GetEquipmentString(p)
		want := "<body> [test armor]"
		if got != want {
			t.Errorf("GetEquipmentString = %q, want %q", got, want)
		}
	})
}

func TestGetACString(t *testing.T) {
	p := game.NewPlayer(1, "Tester", 1001)
	got := GetACString(p)
	want := "Armor Class: 10"
	if got != want {
		t.Errorf("GetACString = %q, want %q", got, want)
	}
}

func TestFormatEquipmentDisplay(t *testing.T) {
	t.Run("no equipment", func(t *testing.T) {
		p := game.NewPlayer(1, "Tester", 1001)
		p.Equipment = game.NewEquipment()
		got := FormatEquipmentDisplay(p)
		want := "You are not wearing anything.\nArmor Class: 10"
		if got != want {
			t.Errorf("FormatEquipmentDisplay = %q, want %q", got, want)
		}
	})
}
