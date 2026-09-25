package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newMpGiveTestWorld is a one-room world with a message sink and a player.
func newMpGiveTestWorld(t *testing.T) (*World, *Player, func() string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Give Room", Zone: 1}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	var out strings.Builder
	w.MessageSink = func(_ string, msg []byte) { out.Write(msg) }

	ch := NewPlayer(1, "Tester", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	return w, ch, func() string { s := out.String(); out.Reset(); return s }
}

// spawnMpGiveTestMob registers and spawns a mob with an exact vnum (mp_give
// branches key on 8061/8063/8065/14401).
func spawnMpGiveTestMob(t *testing.T, w *World, roomVNum, vnum int) *MobInstance {
	t.Helper()
	proto := &parser.Mob{
		VNum:      vnum,
		Keywords:  "testmob",
		ShortDesc: "a test mob",
		LongDesc:  "A test mob is here.",
		Level:     10,
		HP:        parser.DiceRoll{Num: 1, Sides: 8, Plus: 20},
		Race:      1,
		Sex:       1, // male, like the real warg/gatekeeper/janitor
	}
	w.mu.Lock()
	w.mobs[vnum] = proto
	w.mu.Unlock()

	mob, err := w.SpawnMob(vnum, roomVNum)
	if err != nil {
		t.Fatalf("SpawnMob %d: %v", vnum, err)
	}
	mob.SetPosition(combat.PosStanding)
	return mob
}

func registerMpGiveObj(t *testing.T, w *World, proto *parser.Obj) *ObjectInstance {
	t.Helper()
	w.mu.Lock()
	w.objs[proto.VNum] = proto
	w.mu.Unlock()
	obj, err := w.SpawnObject(proto.VNum, -1)
	if err != nil {
		t.Fatalf("SpawnObject %d: %v", proto.VNum, err)
	}
	return obj
}

func mpGiveSwordProto(vnum int, cost int) *parser.Obj {
	return &parser.Obj{
		VNum:      vnum,
		Keywords:  "sword",
		ShortDesc: "a short sword",
		WearFlags: [4]int{1},
		Cost:      cost,
		Weight:    0, // test mobs carry STR 0
	}
}

func TestMpGiveObjectDogFoodExtracts(t *testing.T) {
	w, _, lastMsg := newMpGiveTestWorld(t)
	dog := spawnMpGiveTestMob(t, w, 1001, 8063)
	bread := registerMpGiveObj(t, w, &parser.Obj{
		VNum: 8010, Keywords: "bread", ShortDesc: "a loaf of bread",
		TypeFlag: ITEM_FOOD, WearFlags: [4]int{1}, Cost: 3,
	})

	w.MpGiveObject(nil, dog, bread)

	if got := lastMsg(); !strings.Contains(got, "devours a loaf of bread and wags his tail happily") {
		t.Fatalf("devour act: got %q", got)
	}
	w.mu.RLock()
	_, stillThere := w.objectInstances[bread.ID]
	w.mu.RUnlock()
	if stillThere {
		t.Fatal("dog food should be extracted from the world")
	}
}

func TestMpGiveObjectDogJunkDropsInRoom(t *testing.T) {
	w, _, lastMsg := newMpGiveTestWorld(t)
	dog := spawnMpGiveTestMob(t, w, 1001, 8063)
	sword := registerMpGiveObj(t, w, mpGiveSwordProto(8037, 30))

	w.MpGiveObject(nil, dog, sword)

	got := lastMsg()
	if !strings.Contains(got, "sniffs around and plays with a short sword for a while") ||
		!strings.Contains(got, "quickly loses interest") {
		t.Fatalf("dog junk acts: got %q", got)
	}
	found := false
	for _, item := range w.GetItemsInRoom(1001) {
		if item == sword {
			found = true
		}
	}
	if !found {
		t.Fatal("dog junk should be dropped in the room, not destroyed")
	}
}

func TestMpGiveObjectDemonNonSoulReturnsToMobGiver(t *testing.T) {
	w, _, lastMsg := newMpGiveTestWorld(t)
	demon := spawnMpGiveTestMob(t, w, 1001, 14401)
	giver := spawnMpGiveTestMob(t, w, 1001, 99001)
	bread := registerMpGiveObj(t, w, &parser.Obj{
		VNum: 8010, Keywords: "bread", ShortDesc: "a loaf of bread",
		TypeFlag: ITEM_FOOD, WearFlags: [4]int{1}, Cost: 3,
	})
	if err := w.MoveObjectToMobInventory(bread, giver); err != nil {
		t.Fatalf("arm giver: %v", err)
	}

	w.MpGiveObject(giver, demon, bread)

	got := lastMsg()
	if !strings.Contains(got, "peers at a loaf of bread closely, then hands it back") ||
		!strings.Contains(got, "growls, 'Are you mocking me?'") {
		t.Fatalf("demon mock acts: got %q", got)
	}
	if bread.Location.Kind != ObjInInventory || bread.Location.MobID != giver.GetID() {
		t.Fatalf("non-soul should return to the mob giver, got location %+v", bread.Location)
	}
}

func TestMpGiveObjectDemonSoulSpawnsPortal(t *testing.T) {
	w, ch, lastMsg := newMpGiveTestWorld(t)
	demon := spawnMpGiveTestMob(t, w, 1001, 14401)
	soul := registerMpGiveObj(t, w, &parser.Obj{
		VNum: 9900, Keywords: "stone soul", ShortDesc: "a dull black stone",
		WearFlags: [4]int{1}, Cost: 5,
	})
	registerMpGiveObj(t, w, &parser.Obj{
		VNum: 19611, Keywords: "portal", ShortDesc: "a shimmering black portal",
		Values: [4]int{0, 0, -1, 0},
	})

	w.MpGiveObject(ch, demon, soul)

	got := lastMsg()
	for _, want := range []string{
		"peers at the soul, then licks his lips",
		"says, 'This will do nicely.. you may enter!'",
		"hideous\r\n screaming rings in your ears",
		"a shimmering black portal materializes before you",
		"exclaims, 'Enter the portal quickly! It will not last long!'",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("soul quest act missing %q: got %q", want, got)
		}
	}
	w.mu.RLock()
	_, soulThere := w.objectInstances[soul.ID]
	w.mu.RUnlock()
	if soulThere {
		t.Fatal("soul should be extracted after the swallow act")
	}
	var portal *ObjectInstance
	for _, item := range w.GetItemsInRoom(1001) {
		if item.VNum == 19611 {
			portal = item
		}
	}
	if portal == nil {
		t.Fatal("portal 19611 should load in the giver's room")
	}
	if v := portal.GetValue(2); v != 2 {
		t.Fatalf("portal value[2] = %d, want 2 (GET_OBJ_VAL(portal, 2) = 2)", v)
	}
}

func TestMpGiveObjectJanitorJunkPredicate(t *testing.T) {
	w, _, lastMsg := newMpGiveTestWorld(t)
	janitor := spawnMpGiveTestMob(t, w, 1001, 8061)

	// Junk: takeable and cost <= 10 (bread, cost 3). mp_give's contract is
	// that perform_give has already moved the object to the mob.
	bread := registerMpGiveObj(t, w, &parser.Obj{
		VNum: 8010, Keywords: "bread", ShortDesc: "a loaf of bread",
		WearFlags: [4]int{1}, Cost: 3,
	})
	if err := w.MoveObjectToMobInventory(bread, janitor); err != nil {
		t.Fatalf("hand bread to janitor: %v", err)
	}
	w.MpGiveObject(nil, janitor, bread)
	if got := lastMsg(); !strings.Contains(got, "Thanks for helping clean this place up") {
		t.Fatalf("janitor junk thanks: got %q", got)
	}

	// Non-junk: takeable but cost > 10 (short sword, cost 30).
	sword := registerMpGiveObj(t, w, mpGiveSwordProto(8037, 30))
	if err := w.MoveObjectToMobInventory(sword, janitor); err != nil {
		t.Fatalf("hand sword to janitor: %v", err)
	}
	w.MpGiveObject(nil, janitor, sword)
	if got := lastMsg(); !strings.Contains(got, "Wow, this is pretty neat, thanks.") {
		t.Fatalf("janitor non-junk thanks: got %q", got)
	}

	// The janitor says its line but keeps nothing and moves nothing: the
	// objects remain wherever perform_give put them (already the janitor's).
	if bread.Location.Kind != ObjInInventory || bread.Location.MobID != janitor.GetID() {
		t.Fatalf("janitor should keep the bread, got location %+v", bread.Location)
	}
}

// TestNpcPerformGiveRunsMpGive proves the R5c class wiring: the mobile-giver
// perform_give port (scripted action(me, "give ...")) runs mp_give for its
// NPC victim just as the player-giver path does.
func TestNpcPerformGiveRunsMpGive(t *testing.T) {
	w, _, lastMsg := newMpGiveTestWorld(t)
	warg := spawnMpGiveTestMob(t, w, 1001, 8063)
	// C perform_give's OKGIVE gate must pass for the scripted mob-to-mob give
	// to reach mp_give at all (MOB_OKGIVE is bit 24).
	janitor := spawnMpGiveTestMob(t, w, 1001, 8061)
	janitor.SetMobFlag(24)
	sword := registerMpGiveObj(t, w, mpGiveSwordProto(8037, 30))
	if err := w.MoveObjectToMobInventory(sword, warg); err != nil {
		t.Fatalf("arm warg: %v", err)
	}

	w.npcPerformGive(warg, janitor, sword)

	got := lastMsg()
	if !strings.Contains(got, "gives a short sword to a test mob") ||
		!strings.Contains(got, "Wow, this is pretty neat, thanks.") {
		t.Fatalf("mob-to-mob give should run mp_give: got %q", got)
	}
	if sword.Location.Kind != ObjInInventory || sword.Location.MobID != janitor.GetID() {
		t.Fatalf("object should end with the janitor, got location %+v", sword.Location)
	}
}
