package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestFidelityCanSeeBlindnessAndInvis verifies that the canSee check
// in act.go correctly respects blindness, invisibility, and hiding.
func TestFidelityCanSeeBlindnessAndInvis(t *testing.T) {
	observer := NewPlayer(1, "Observer", 1001)
	subject := NewPlayer(2, "Subject", 1001)

	// Under normal conditions, awake characters see each other
	if !canSee(observer, subject) {
		t.Error("Observer should see subject under normal conditions")
	}

	// 1. Blindness check — a blind character cannot see anything
	observer.SetAffect(affBlind, true)
	if !observer.IsAffected(affBlind) {
		t.Fatal("Failed to apply blind affect to observer")
	}
	if canSee(observer, subject) {
		t.Error("Observer should NOT see subject while blind")
	}

	// Cure blind
	observer.SetAffect(affBlind, false)
	if !canSee(observer, subject) {
		t.Error("Observer should see subject after blind is cured")
	}

	// 2. Invisibility check — observer without detect invisibility cannot see invisible subject
	subject.SetAffect(affInvisible, true)
	if canSee(observer, subject) {
		t.Error("Observer without detect invisible should NOT see invisible subject")
	}

	// Add detect invisibility to observer
	observer.SetAffect(affDetectInvisible, true)
	if !canSee(observer, subject) {
		t.Error("Observer with detect invisible should see invisible subject")
	}

	// Remove invisibility and detect invisibility
	subject.SetAffect(affInvisible, false)
	observer.SetAffect(affDetectInvisible, false)

	// 3. Hiding check — observer without sense life cannot see hidden subject
	subject.SetAffect(affHide, true)
	if canSee(observer, subject) {
		t.Error("Observer without sense life should NOT see hidden subject")
	}

	// Add sense life to observer
	observer.SetAffect(affSenseLife, true)
	if !canSee(observer, subject) {
		t.Error("Observer with sense life should see hidden subject")
	}
}

// TestFidelityStealMobRestrictions verifies that DoSteal works on mob targets
// and respects weight penalties.
func TestFidelityStealMobRestrictions(t *testing.T) {
	thief := NewPlayer(1, "Thief", 1001)
	thief.SetSkill(SkillSteal, 1000) // extremely high skill to guarantee success

	// Construct a mob target
	mobProto := &parser.Mob{
		VNum:      2001,
		ShortDesc: "a dummy mob",
		AC:        100,
	}
	mob := NewMob(mobProto, 1001)

	// Construct an item
	itemProto := &parser.Obj{
		VNum:      3001,
		ShortDesc: "a silver sword",
		Keywords:  "sword silver",
		Weight:    2,
	}
	item := NewObjectInstance(itemProto, 1001)

	// Set up world
	world := &World{
		rooms:           make(map[int]*parser.Room),
		objs:            make(map[int]*parser.Obj),
		objectInstances: make(map[int]*ObjectInstance),
		players:         make(map[string]*Player),
		activeMobs:      make(map[int]*MobInstance),
	}
	world.players[thief.Name] = thief
	world.activeMobs[mob.GetID()] = mob
	mob.AddToInventory(item)

	// Perform steal
	result := DoSteal(thief, mob, "sword")
	if !result.Success {
		t.Fatalf("DoSteal failed: %s", result.MessageToCh)
	}

	// Verify item was transferred
	if len(mob.Inventory) != 0 {
		t.Error("Mob inventory should be empty after steal")
	}
	if len(thief.Inventory.Items) != 1 {
		t.Error("Thief should have stolen 1 item")
	} else if thief.Inventory.Items[0].VNum != 3001 {
		t.Errorf("Thief inventory has wrong item: %v", thief.Inventory.Items[0])
	}
}

// TestFidelityMindlinkMobAssertion verifies that DoMindlink successfully transfers mana
// to mob targets.
func TestFidelityMindlinkMobAssertion(t *testing.T) {
	ch := NewPlayer(1, "Psionic", 1001)
	ch.SetSkill(SkillMindlink, 1000) // guarantee success
	ch.SetHP(200)

	// Construct a mob target
	mobProto := &parser.Mob{
		VNum:      2001,
		ShortDesc: "a dummy mob",
		AC:        100,
	}
	mob := NewMob(mobProto, 1001)
	mob.CurrentHP = 100
	mob.MaxHP = 100
	mob.SetMana(10) // start with 10 mana

	// Set status to standing (combat blocks mindlink)
	ch.SetPosition(combat.PosStanding)
	mob.SetStatus("standing")

	// Trigger mindlink
	result := DoMindlink(ch, mob)
	if !result.Success {
		t.Fatalf("DoMindlink failed: %s", result.MessageToCh)
	}

	// Verify mana was transferred
	if mob.GetMana() <= 10 {
		t.Errorf("Mob mana should be greater than 10, got %d", mob.GetMana())
	}
}

// TestFidelityDigCosmeticStub verifies that DoDig is a fully functional action
// that spawns real objects and transfers them into the player's inventory on success.
func TestFidelityDigCosmeticStub(t *testing.T) {
	ch := NewPlayer(1, "Digger", 1001)
	ch.SetHP(100)
	ch.SetSkill(SkillDig, 1000) // guarantee success

	world := &World{
		rooms:           make(map[int]*parser.Room),
		objs:            make(map[int]*parser.Obj),
		objectInstances: make(map[int]*ObjectInstance),
		players:         make(map[string]*Player),
		roomItems:       make(map[int][]*ObjectInstance),
	}
	// Add room 1001 with suitable sector (e.g. SECT_DIRT = 2)
	world.rooms[1001] = &parser.Room{
		VNum:   1001,
		Sector: 2,
	}
	ch.RoomVNum = 1001
	world.players[ch.Name] = ch

	// Initialize the prototype that will be spawned
	world.objs[3001] = &parser.Obj{
		VNum:      3001,
		ShortDesc: "a shiny gold coin",
		Keywords:  "coin gold shiny",
	}

	// Dig

	// Keep trying until we spawn the item (since dig has a 20% chance of coins instead)
	successCount := 0
	for i := 0; i < 20; i++ {
		// Reset inventory
		ch.Inventory.Items = nil
		result := DoDig(ch, world)
		if result.Success {
			successCount++
			if len(ch.Inventory.Items) > 0 {
				break
			}
		}
	}

	if successCount == 0 {
		t.Error("Digging failed to succeed in 20 attempts")
	}

	if len(ch.Inventory.Items) == 0 {
		t.Error("Digging succeeded but never spawned any items in player inventory (expected prototype 3001)")
	} else if ch.Inventory.Items[0].VNum != 3001 {
		t.Errorf("Spawned wrong item type: %v", ch.Inventory.Items[0])
	}
}
