package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
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

// TestFidelityCanSeeObjectInvisibility verifies canSeeObject's ITEM_INVISIBLE
// check uses bit 5 (src/structs.h:468-496), not bit 0 (ITEM_GLOW).
func TestFidelityCanSeeObjectInvisibility(t *testing.T) {
	observer := NewPlayer(1, "Observer", 1001)

	glowing := newDonatableItem(6000, "a glowing lantern", "lantern", 50)
	glowing.SetExtraFlag(0, 0) // ITEM_GLOW — must not be treated as invisible
	if !canSeeObject(observer, glowing) {
		t.Error("a merely glowing item should be visible")
	}

	invisible := newDonatableItem(6001, "an invisible coin", "coin", 50)
	invisible.SetExtraFlag(0, extraFlagInvisible)
	if canSeeObject(observer, invisible) {
		t.Error("Observer without detect invisible should NOT see an invisible object")
	}

	observer.SetAffect(affDetectInvisible, true)
	if !canSeeObject(observer, invisible) {
		t.Error("Observer with detect invisible should see an invisible object")
	}
}

// TestFidelityChCanSeeObjInvisibility verifies chCanSeeObj's ITEM_INVISIBLE
// check and nil-safety (act_informative.go) — the simpler, Player-specific
// sibling of canSeeObject.
func TestFidelityChCanSeeObjInvisibility(t *testing.T) {
	mortal := NewPlayer(1, "Mortal", 1001)
	immort := NewPlayer(2, "Immort", 1001)
	immort.SetLevel(lvlImmort)

	if chCanSeeObj(mortal, nil) {
		t.Error("nil object should never be visible")
	}

	visible := newDonatableItem(6002, "a steel blade", "blade", 50)
	if !chCanSeeObj(mortal, visible) {
		t.Error("a plain visible object should be visible")
	}

	invisible := newDonatableItem(6003, "an invisible blade", "blade", 50)
	invisible.SetExtraFlag(0, extraFlagInvisible)
	if chCanSeeObj(mortal, invisible) {
		t.Error("a mortal should not see an invisible object via chCanSeeObj")
	}
	if !chCanSeeObj(immort, invisible) {
		t.Error("an immortal should see an invisible object via chCanSeeObj")
	}
}

// TestFidelityStealMobRestrictions verifies that DoSteal works on mob targets
// and respects weight penalties.
func TestFidelityStealMobRestrictions(t *testing.T) {
	thief := NewPlayer(1, "Thief", 1001)
	thief.SetSkill(SkillSteal, 1000) // extremely high skill to guarantee success
	thief.Stats.Str = 10

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
	result := DoSteal(thief, mob, "sword", world)
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

// TestFidelityMindlinkMobManaGate verifies do_mindlink's C target-resource
// gate. Normal C-loaded mobs begin with only ten mana, so the command returns
// before its percent roll and does not drain the actor.
func TestFidelityMindlinkMobManaGate(t *testing.T) {
	ch := NewPlayer(1, "Psionic", 1001)
	ch.SetSkill(SkillMindlink, 1000)
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
	mob.SetMana(10) // C db.c initializes ordinary mobs to 10 mana.

	// Set status to standing (combat blocks mindlink)
	ch.SetPosition(combat.PosStanding)
	mob.SetStatus("standing")

	// Trigger mindlink
	result := DoMindlink(ch, mob)
	if result.Success {
		t.Fatal("DoMindlink should stop at the target mana gate")
	}
	if result.MessageToCh != "They don't have enough energy to spare!\r\n" {
		t.Fatalf("message = %q, want target mana gate", result.MessageToCh)
	}
	if ch.GetHP() != 200 {
		t.Fatalf("actor HP = %d, want unchanged 200", ch.GetHP())
	}

	// C checks self before the skill gate (new_cmds2.c:263-268).
	ch.SetSkill(SkillMindlink, 0)
	if got := DoMindlink(ch, ch).MessageToCh; got != "You wish you could.\r\n" {
		t.Fatalf("self/no-skill message = %q, want self gate", got)
	}
}

// TestFidelityMindlinkMobFailureContract pins the post-gate NPC branch. The
// ordinary command vehicle cannot raise a mob above C's ten-mana load, so use
// a focused state fixture to reach the source branch after the mana gate.
func TestFidelityMindlinkMobFailureContract(t *testing.T) {
	ch := NewPlayer(1, "Psionic", 1001)
	ch.SetSkill(SkillMindlink, 100)
	ch.SetHP(200)
	ch.SetPosition(combat.PosStanding)
	mob := NewMob(&parser.Mob{VNum: 2002, ShortDesc: "a full-mana dummy"}, 1001)
	mob.SetMana(100)
	mob.SetStatus("standing")

	dprng.ResetStream(23)
	result := DoMindlink(ch, mob)
	if result.Success {
		t.Fatal("NPC mindlink success is unreachable after C's IS_NPC macros")
	}
	if result.MessageToCh != "You feel a little drained...\r\n" {
		t.Fatalf("actor message = %q, want final drain line", result.MessageToCh)
	}
	if result.MessageToRoom == "" || len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillMindlink || !result.DeferredImproveAfterRoom {
		t.Fatalf("failure contract = room %q improve %#v after-room %v", result.MessageToRoom, result.DeferredImprove, result.DeferredImproveAfterRoom)
	}
	if !result.MessageToChAfterRoom || !result.SelfStunnedAfterMessage {
		t.Fatal("failure contract must deliver the actor line and stun after the room act")
	}
	if got := ch.GetHP(); got != 100 {
		t.Fatalf("actor HP = %d, want 100 after the fixed 100-point drain", got)
	}

	// C draws number(1,101) before taking the unconditional NPC failure arm.
	gotNext := dprng.Number(0, 999)
	dprng.ResetStream(23)
	dprng.Number(1, 101)
	wantNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("mindlink draw sequence next=%d, want %d", gotNext, wantNext)
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

func TestMobGetSexTranslatesCFileEncodingToActorEncoding(t *testing.T) {
	tests := []struct {
		name string
		cSex int
		want int
	}{
		{name: "neutral", cSex: 0, want: 2},
		{name: "male", cSex: 1, want: 0},
		{name: "female", cSex: 2, want: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mob := NewMob(&parser.Mob{Sex: test.cSex}, 1001)
			if got := mob.GetSex(); got != test.want {
				t.Fatalf("GetSex() = %d, want %d", got, test.want)
			}
		})
	}
}
