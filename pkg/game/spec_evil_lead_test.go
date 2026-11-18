package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// evilLeadTestWorld builds rooms 1001 (origin) and 13195 (trapdoor
// destination, src/spec_procs2.c:1378), with a cheese 13100 and a junk
// 13111 object, an evillead mob, a giver and an observer.
func evilLeadTestWorld(t *testing.T) (*World, map[string]string, *Player, *Player, *MobInstance) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Origin", Zone: 1},
			{VNum: 13195, Name: "Destination", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 99002, Keywords: "swiss", ShortDesc: "Swiss", LongDesc: "Swiss is here.", Level: 20, HP: parser.DiceRoll{Num: 1, Sides: 8, Plus: 100}, Alignment: -1000, Race: 1, Int: 11, Wis: 11},
		},
		Objs: []parser.Obj{
			{VNum: 13100, Keywords: "cheese slice", ShortDesc: "a slice of cheese"},
			{VNum: 13111, Keywords: "watch gold", ShortDesc: "a gold watch"},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	giver := NewPlayer(1, "Giver", 1001)
	giver.SetAlignment(-100)
	if err := w.AddPlayer(giver); err != nil {
		t.Fatalf("AddPlayer giver: %v", err)
	}
	watcher := NewPlayer(2, "Watcher", 1001)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer watcher: %v", err)
	}
	lead, err := w.SpawnMobQuiet(99002, 1001)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	lead.SetLevel(20)
	out["Giver"] = ""
	out["Watcher"] = ""
	return w, out, giver, watcher, lead
}

func giveObjectToPlayer(t *testing.T, w *World, giver *Player, vnum int) {
	t.Helper()
	obj, err := w.SpawnObject(vnum, -1)
	if err != nil {
		t.Fatalf("SpawnObject %d: %v", vnum, err)
	}
	if err := w.MoveObjectToPlayerInventory(obj, giver); err != nil {
		t.Fatalf("MoveObjectToPlayerInventory: %v", err)
	}
}

// TestSpecEvilLead_WrongObject pins the junk-give tell
// (src/spec_procs2.c:1356-1361): a non-cheese give tells "Don't give me
// that crap." and returns TRUE without taking the object or moving.
func TestSpecEvilLead_WrongObject(t *testing.T) {
	w, out, giver, _, lead := evilLeadTestWorld(t)
	giveObjectToPlayer(t, w, giver, 13111)
	if !specEvilLead(w, giver, lead, "give", "watch swiss") {
		t.Fatal("junk give should be handled")
	}
	if got := out["Giver"]; !strings.Contains(got, "Swiss tells you, 'Don't give me that crap.'") {
		t.Fatalf("giver bytes = %q, want crap tell", got)
	}
	if got := giver.GetRoomVNum(); got != 1001 {
		t.Fatalf("giver room = %d, want origin 1001", got)
	}
}

// TestSpecEvilLead_GoodAlignment pins the good-giver tell
// (src/spec_procs2.c:1362-1367): cheese from a good-aligned giver tells
// "Get lost, goody-goody." and returns TRUE without moving.
func TestSpecEvilLead_GoodAlignment(t *testing.T) {
	w, out, giver, _, lead := evilLeadTestWorld(t)
	giver.SetAlignment(100)
	giveObjectToPlayer(t, w, giver, 13100)
	if !specEvilLead(w, giver, lead, "give", "cheese swiss") {
		t.Fatal("good-giver cheese should be handled")
	}
	if got := out["Giver"]; !strings.Contains(got, "Swiss tells you, 'Get lost, goody-goody.'") {
		t.Fatalf("giver bytes = %q, want goody-goody tell", got)
	}
	if got := giver.GetRoomVNum(); got != 1001 {
		t.Fatalf("giver room = %d, want origin 1001", got)
	}
}

// TestSpecEvilLead_Trapdoor pins the cheese trapdoor
// (src/spec_procs2.c:1368-1385): the cheese moves to the lead, the room
// hears the trapdoor act, the giver hears the follow-talk, moves to
// 13195 and looks.
func TestSpecEvilLead_Trapdoor(t *testing.T) {
	w, out, giver, watcher, lead := evilLeadTestWorld(t)
	giveObjectToPlayer(t, w, giver, 13100)
	if !specEvilLead(w, giver, lead, "give", "cheese swiss") {
		t.Fatal("evil-giver cheese should be handled")
	}
	if got := out["Watcher"]; !strings.Contains(got, "Swiss leads Giver through a hidden trapdoor") {
		t.Fatalf("room bytes = %q, want trapdoor act", got)
	}
	if got := out["Giver"]; !strings.Contains(got, "You follow Swiss though a small trapdoor") {
		t.Fatalf("giver bytes = %q, want follow-talk", got)
	}
	if got := giver.GetRoomVNum(); got != 13195 {
		t.Fatalf("giver room = %d, want 13195", got)
	}
	// The cheese left the giver's inventory for the lead.
	found := false
	for _, obj := range lead.Inventory {
		if obj.GetVNum() == 13100 {
			found = true
		}
	}
	if !found {
		t.Fatal("cheese should be in the lead's inventory")
	}
	_ = watcher
}

// TestSpecEvilLead_Gates pins the give gates (src/spec_procs2.c:1347-1355):
// non-give, empty argument, wrong target and missing object all return
// false with no teleport.
func TestSpecEvilLead_Gates(t *testing.T) {
	w, out, giver, _, lead := evilLeadTestWorld(t)
	giveObjectToPlayer(t, w, giver, 13100)
	if specEvilLead(w, giver, lead, "look", "cheese") {
		t.Fatal("non-give must not be handled")
	}
	if specEvilLead(w, giver, lead, "give", "  ") {
		t.Fatal("empty give must not be handled")
	}
	if specEvilLead(w, giver, lead, "give", "cheese wrongtarget") {
		t.Fatal("wrong target must not be handled")
	}
	if specEvilLead(w, giver, lead, "give", "nonexistent swiss") {
		t.Fatal("missing object must not be handled")
	}
	if got := giver.GetRoomVNum(); got != 1001 {
		t.Fatalf("giver room = %d, want origin 1001", got)
	}
	if got := out["Giver"]; got != "" {
		t.Fatalf("gate emitted %q", got)
	}
}
