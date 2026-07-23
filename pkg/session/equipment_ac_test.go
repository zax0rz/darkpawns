package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestCalculateAC_BaseOnly(t *testing.T) {
	p := game.NewPlayer(1, "TestPlayer", 1001)
	// Default AC is 10 in NewPlayer
	if got := CalculateAC(p); got != 10 {
		t.Errorf("CalculateAC with no equipment = %d, want 10", got)
	}
}

func TestCalculateAC_WithArmor(t *testing.T) {
	p := game.NewPlayer(1, "TestPlayer", 1001)

	// Equip a piece of armor: ITEM_ARMOR (type 9), Values[0] = AC bonus
	armorProto := &parser.Obj{
		VNum:      100,
		Keywords:  "leather_armor",
		ShortDesc: "leather armor",
		TypeFlag:  9,                       // ITEM_ARMOR
		Values:    [4]int{5, 0, 0, 0},      // AC bonus = 5
		WearFlags: [4]int{1 << 3, 0, 0, 0}, // ITEM_WEAR_BODY
	}
	armor := game.NewObjectInstance(armorProto, 1001)

	// Equip the armor on the body
	if err := p.Equipment.Equip(armor, p.Inventory); err != nil {
		t.Fatalf("Equip: %v", err)
	}

	// AC = base(10) - armor_bonus(5) = 5
	if got := CalculateAC(p); got != 5 {
		t.Errorf("CalculateAC with armor = %d, want 5", got)
	}
}

func TestCalculateAC_WithMultipleArmor(t *testing.T) {
	p := game.NewPlayer(1, "TestPlayer", 1001)

	armorProto := &parser.Obj{
		VNum:      100,
		Keywords:  "chainmail",
		ShortDesc: "chainmail",
		TypeFlag:  9, // ITEM_ARMOR
		Values:    [4]int{7, 0, 0, 0},
		WearFlags: [4]int{1 << 3, 0, 0, 0}, // ITEM_WEAR_BODY
	}
	armor := game.NewObjectInstance(armorProto, 1001)
	if err := p.Equipment.Equip(armor, p.Inventory); err != nil {
		t.Fatalf("Equip chainmail: %v", err)
	}

	helmProto := &parser.Obj{
		VNum:      101,
		Keywords:  "helmet",
		ShortDesc: "iron helmet",
		TypeFlag:  9,
		Values:    [4]int{3, 0, 0, 0},
		WearFlags: [4]int{1 << 4, 0, 0, 0}, // ITEM_WEAR_HEAD
	}
	helm := game.NewObjectInstance(helmProto, 1001)
	if err := p.Equipment.Equip(helm, p.Inventory); err != nil {
		t.Fatalf("Equip helmet: %v", err)
	}

	// AC = base(10) - chainmail(7) - helmet(3) = 0
	if got := CalculateAC(p); got != 0 {
		t.Errorf("CalculateAC with multiple armor = %d, want 0", got)
	}
}

func TestCalculateAC_WithAffects(t *testing.T) {
	p := game.NewPlayer(1, "TestPlayer", 1001)
	p.AC = 10

	// Add an active affect that modifies AC (ApplyAC)
	p.ActiveAffects = []*engine.Affect{
		{Location: engine.ApplyAC, Magnitude: -5, Duration: 10, Source: "armor spell"},
	}

	// AC = base(10) + affect(-5) = 5
	if got := CalculateAC(p); got != 5 {
		t.Errorf("CalculateAC with affect = %d, want 5", got)
	}
}

func TestCalculateAC_WithArmorAndAffects(t *testing.T) {
	p := game.NewPlayer(1, "TestPlayer", 1001)

	armorProto := &parser.Obj{
		VNum:      100,
		Keywords:  "plate_mail",
		ShortDesc: "plate mail",
		TypeFlag:  9,
		Values:    [4]int{10, 0, 0, 0},
		WearFlags: [4]int{1 << 3, 0, 0, 0}, // ITEM_WEAR_BODY
	}
	armor := game.NewObjectInstance(armorProto, 1001)
	if err := p.Equipment.Equip(armor, p.Inventory); err != nil {
		t.Fatalf("Equip: %v", err)
	}

	p.ActiveAffects = []*engine.Affect{
		{Location: engine.ApplyAC, Magnitude: 3, Duration: 5, Source: "curse"},
	}

	// AC = base(10) - armor(10) + affect(3) = 3
	if got := CalculateAC(p); got != 3 {
		t.Errorf("CalculateAC with armor and affect = %d, want 3", got)
	}
}

func TestGetACString(t *testing.T) {
	p := game.NewPlayer(1, "TestPlayer", 1001)
	// Default AC = 10
	want := "Armor Class: 10"
	if got := GetACString(p); got != want {
		t.Errorf("GetACString = %q, want %q", got, want)
	}
}

	// DP-1198: "You are not wearing anything." / "Armor Class:" DOCUMENT current
	// (invented) display output — C do_equipment (act.informative.c:1474) says
	// "You are using:" + per-slot lines. Update expectations with the fix.
func TestGetEquipmentString_Empty(t *testing.T) {
	p := game.NewPlayer(1, "TestPlayer", 1001)
	want := "You are not wearing anything."
	if got := GetEquipmentString(p); got != want {
		t.Errorf("GetEquipmentString(empty) = %q, want %q", got, want)
	}
}

func TestGetEquipmentString_WithEquipment(t *testing.T) {
	p := game.NewPlayer(1, "TestPlayer", 1001)

	armorProto := &parser.Obj{
		VNum:      100,
		Keywords:  "leather_armor",
		ShortDesc: "leather armor",
		TypeFlag:  9,
		Values:    [4]int{5, 0, 0, 0},
		WearFlags: [4]int{1 << 3, 0, 0, 0}, // ITEM_WEAR_BODY
	}
	armor := game.NewObjectInstance(armorProto, 1001)
	if err := p.Equipment.Equip(armor, p.Inventory); err != nil {
		t.Fatalf("Equip: %v", err)
	}

	got := GetEquipmentString(p)
	if !strings.Contains(got, "leather armor") {
		t.Errorf("GetEquipmentString should contain item desc, got %q", got)
	}
	if !strings.Contains(got, "<") || !strings.Contains(got, ">") {
		t.Errorf("GetEquipmentString should contain slot brackets, got %q", got)
	}
}

func TestFormatEquipmentDisplay(t *testing.T) {
	p := game.NewPlayer(1, "TestPlayer", 1001)

	// No equipment
	got := FormatEquipmentDisplay(p)
	if !strings.Contains(got, "You are not wearing anything.") {
		t.Errorf("FormatEquipmentDisplay should contain empty message, got %q", got)
	}
	if !strings.Contains(got, "Armor Class: 10") {
		t.Errorf("FormatEquipmentDisplay should contain AC, got %q", got)
	}
}
