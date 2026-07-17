package game

import (
	"slices"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// mockCombatEngine is a stub CombatEngine for use in OnPlayerEnterRoom tests.
type mockCombatEngine struct {
	fighting map[string]bool
	started  []string // mob names that had StartCombat called
	startErr error
}

func newMockCE() *mockCombatEngine {
	return &mockCombatEngine{fighting: make(map[string]bool)}
}

func (m *mockCombatEngine) StartCombat(attacker, _ combat.Combatant) error {
	m.started = append(m.started, attacker.GetName())
	return m.startErr
}

func (m *mockCombatEngine) IsFighting(name string) bool { return m.fighting[name] }

func (m *mockCombatEngine) GetCombatTarget(name string) (combat.Combatant, bool) { return nil, false }

// newStartingItemsWorld builds a world that includes all starting item vnums
// so GiveStartingItems can fully execute for any class.
func newStartingItemsWorld(t *testing.T) (*World, *Player) {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Start Room", Zone: 1},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{
			{VNum: 8023, Keywords: "club", ShortDesc: "a club", LongDesc: "A wooden club."},
			{VNum: 8019, Keywords: "tunic", ShortDesc: "a tunic", LongDesc: "A plain tunic."},
			{VNum: 8036, Keywords: "dagger", ShortDesc: "a dagger", LongDesc: "A small dagger."},
			{VNum: 8037, Keywords: "sword", ShortDesc: "a small sword", LongDesc: "A small sword."},
			// pack (container) — TypeFlag 15 = ITEM_CONTAINER (src/structs.h:100)
			{VNum: 8038, Keywords: "pack", ShortDesc: "a pack", LongDesc: "A leather pack.", TypeFlag: 15, Values: [4]int{20, contCloseable | contClosed, -1, 0}},
			{VNum: 8010, Keywords: "bread", ShortDesc: "some bread", LongDesc: "A piece of bread."},
			{VNum: 8063, Keywords: "waterskin", ShortDesc: "a waterskin", LongDesc: "A full waterskin."},
			{VNum: 8027, Keywords: "lockpick", ShortDesc: "a set of lockpicks", LongDesc: "Lockpicks."},
			{VNum: 1239, Keywords: "obsidian", ShortDesc: "a piece of obsidian", LongDesc: "An obsidian gem."},
		},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	player := NewPlayer(1, "TestPlayer", 1001)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	return w, player
}

// ---------------------------------------------------------------------------
// GiveStartingItems tests
// ---------------------------------------------------------------------------

func TestGiveStartingItems_Warrior(t *testing.T) {
	w, player := newStartingItemsWorld(t)
	player.Class = ClassWarrior

	w.GiveStartingItems(player)

	// Warrior gets: small sword (8037) + tunic (8019) directly, then pack with bread+water
	invVNums := collectInventoryVNums(player)
	if want := []int{8038, 8019, 8037}; !slices.Equal(invVNums, want) {
		t.Errorf("warrior inventory order = %v, want C obj_to_char order %v", invVNums, want)
	}
	if !containsVNum(invVNums, 8037) {
		t.Error("warrior should receive small sword (vnum 8037)")
	}
	if !containsVNum(invVNums, 8019) {
		t.Error("warrior should receive tunic (vnum 8019)")
	}
	if !containsVNum(invVNums, 8038) {
		t.Error("warrior should receive pack (vnum 8038)")
	}
	if equipped := player.Equipment.GetEquippedItems(); len(equipped) != 0 {
		t.Fatalf("starter gear should be carried, not equipped: %+v", equipped)
	}
	for _, item := range player.Inventory.FindItems("") {
		if item.Location != LocInventoryPlayer(player.Name) {
			t.Errorf("starter item %d location = %+v, want player inventory", item.VNum, item.Location)
		}
	}
	// Pack should contain bread and waterskin
	pack := findItemByVNum(player, 8038)
	if pack == nil {
		t.Fatal("pack not found in inventory")
	}
	if pack.ID <= 0 || pack.GetValue(contFlags)&contClosed == 0 {
		t.Fatalf("starter pack id/flags = %d/%d, want registered and closed", pack.ID, pack.GetValue(contFlags))
	}
	packVNums := containerVNums(pack)
	if !containsVNum(packVNums, 8010) {
		t.Error("pack should contain bread (vnum 8010)")
	}
	if !containsVNum(packVNums, 8063) {
		t.Error("pack should contain waterskin (vnum 8063)")
	}
	for _, item := range pack.Contains {
		if item.ID <= 0 || item.Location != LocContainer(pack.ID) {
			t.Errorf("starter pack child %d has id/location %d/%+v", item.VNum, item.ID, item.Location)
		}
	}
}

func TestInventoryAddItemPrependsLikeObjToChar(t *testing.T) {
	inv := NewInventory()
	first := &ObjectInstance{VNum: 1}
	second := &ObjectInstance{VNum: 2}

	if err := inv.AddItem(first); err != nil {
		t.Fatalf("AddItem(first) failed: %v", err)
	}
	if err := inv.AddItem(second); err != nil {
		t.Fatalf("AddItem(second) failed: %v", err)
	}

	if got := []int{inv.Items[0].VNum, inv.Items[1].VNum}; !slices.Equal(got, []int{2, 1}) {
		t.Errorf("inventory order = %v, want newest-first [2 1]", got)
	}
}

func TestGiveStartingItems_Thief(t *testing.T) {
	w, player := newStartingItemsWorld(t)
	player.Class = ClassThief

	w.GiveStartingItems(player)

	invVNums := collectInventoryVNums(player)
	if !containsVNum(invVNums, 8036) {
		t.Error("thief should receive dagger (vnum 8036)")
	}
	if !containsVNum(invVNums, 8019) {
		t.Error("thief should receive tunic (vnum 8019)")
	}

	pack := findItemByVNum(player, 8038)
	if pack == nil {
		t.Fatal("thief should receive pack (vnum 8038)")
	}
	// Thief pack should contain lockpicks
	packVNums := containerVNums(pack)
	if !containsVNum(packVNums, 8027) {
		t.Error("thief pack should contain lockpicks (vnum 8027)")
	}
}

func TestGiveStartingItems_Mage(t *testing.T) {
	w, player := newStartingItemsWorld(t)
	player.Class = ClassMageUser

	w.GiveStartingItems(player)

	invVNums := collectInventoryVNums(player)
	if !containsVNum(invVNums, 8036) {
		t.Error("mage should receive dagger (vnum 8036)")
	}
	// Mage gets 2x obsidian — count them
	obsidianCount := 0
	for _, v := range invVNums {
		if v == 1239 {
			obsidianCount++
		}
	}
	if obsidianCount < 2 {
		t.Errorf("mage should receive 2 obsidian (vnum 1239), got %d", obsidianCount)
	}
}

func TestGiveStartingItems_DefaultClass(t *testing.T) {
	w, player := newStartingItemsWorld(t)
	player.Class = ClassCleric // falls to default branch

	w.GiveStartingItems(player)

	invVNums := collectInventoryVNums(player)
	if !containsVNum(invVNums, 8023) {
		t.Error("default class should receive club (vnum 8023)")
	}
}

// TestGiveStartingItems_EmptyWorld verifies no crash when item prototypes don't exist.
func TestGiveStartingItems_EmptyWorld(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Empty Room", Zone: 1}},
		Mobs:  []parser.Mob{},
		Objs:  []parser.Obj{},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	player := NewPlayer(1, "EmptyPlayer", 1001)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	// Must not panic — simply skips all items
	w.GiveStartingItems(player)

	if len(player.Inventory.Items) != 0 {
		t.Errorf("expected empty inventory when world has no items, got %d", len(player.Inventory.Items))
	}
}

// ---------------------------------------------------------------------------
// OnPlayerEnterRoom tests
// ---------------------------------------------------------------------------

// newAggroWorld creates a world with one room containing an aggressive mob.
func newAggroWorld(t *testing.T) (*World, *Player) {
	t.Helper()

	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Danger Room", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 2001, ShortDesc: "an orc", ActionFlags: []string{"aggressive"}},
		},
		Objs: []parser.Obj{},
	}

	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	player := NewPlayer(1, "TestPlayer", 1001)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	return w, player
}

// TestOnPlayerEnterRoom_NoMobs — room with no mobs returns false.
func TestOnPlayerEnterRoom_NoMobs(t *testing.T) {
	w, player := newTestWorld(t) // from object_movement_test.go — has room 1001, no mobs spawned

	ce := newMockCE()
	result := w.OnPlayerEnterRoom(player, 1001, ce)
	if result {
		t.Error("expected false when no mobs are in the room")
	}
	if len(ce.started) != 0 {
		t.Errorf("expected no combat started, got %d", len(ce.started))
	}
}

// TestOnPlayerEnterRoom_NonexistentRoom — non-existent room returns false without panic.
func TestOnPlayerEnterRoom_NonexistentRoom(t *testing.T) {
	w, player := newTestWorld(t)

	ce := newMockCE()
	result := w.OnPlayerEnterRoom(player, 99999, ce)
	if result {
		t.Error("expected false for a non-existent room")
	}
}

// TestOnPlayerEnterRoom_AggressiveMob — aggressive mob triggers StartCombat and returns true.
func TestOnPlayerEnterRoom_AggressiveMob(t *testing.T) {
	w, player := newAggroWorld(t)

	_, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	ce := newMockCE()
	result := w.OnPlayerEnterRoom(player, 1001, ce)
	if !result {
		t.Error("expected true when aggressive mob is in room")
	}
}

// TestOnPlayerEnterRoom_AlreadyFighting — mob already in combat is skipped.
func TestOnPlayerEnterRoom_AlreadyFighting(t *testing.T) {
	w, player := newAggroWorld(t)

	mob, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	ce := newMockCE()
	ce.fighting[mob.GetName()] = true // mark mob as already fighting

	result := w.OnPlayerEnterRoom(player, 1001, ce)
	// Mob is aggressive but already fighting — should NOT start new combat
	if result {
		t.Error("expected false when aggressive mob is already fighting")
	}
}

// ---------------------------------------------------------------------------
// Stats test
// ---------------------------------------------------------------------------

func TestStats_ReturnsNonEmpty(t *testing.T) {
	w, _ := newTestWorld(t)

	s := w.Stats()
	if s == "" {
		t.Error("Stats() should return a non-empty string")
	}
	if !strings.Contains(s, "rooms") {
		t.Errorf("Stats() should mention rooms, got: %q", s)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func collectInventoryVNums(p *Player) []int {
	vnums := make([]int, 0, len(p.Inventory.Items))
	for _, item := range p.Inventory.Items {
		vnums = append(vnums, item.VNum)
	}
	return vnums
}

func containsVNum(vnums []int, target int) bool {
	for _, v := range vnums {
		if v == target {
			return true
		}
	}
	return false
}

func findItemByVNum(p *Player, vnum int) *ObjectInstance {
	for _, item := range p.Inventory.Items {
		if item.VNum == vnum {
			return item
		}
	}
	return nil
}

func containerVNums(container *ObjectInstance) []int {
	vnums := make([]int, 0, len(container.Contains))
	for _, item := range container.Contains {
		vnums = append(vnums, item.VNum)
	}
	return vnums
}
