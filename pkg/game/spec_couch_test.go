package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func couchFidelityWorld(t *testing.T) (*World, map[string]string, *Player, *ObjectInstance, *MobInstance) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Pillow Room", Zone: 1},
			{VNum: 5798, Name: "Mimic Home", Zone: 1},
			{VNum: 9999, Name: "Elsewhere", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 99020, Keywords: "mimic", ShortDesc: "the mimic", Level: 10, HP: parser.DiceRoll{Num: 1, Sides: 8, Plus: 50}, Alignment: 0, Race: 1, Int: 11, Wis: 11, Sex: 1},
		},
		Objs: []parser.Obj{{VNum: 3002, Keywords: "couch", ShortDesc: "a worn couch"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }
	actor := NewPlayer(1, "Actor", 1001)
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	watcher := NewPlayer(2, "Watcher", 1001)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer watcher: %v", err)
	}
	couch, err := w.SpawnObject(3002, 1001)
	if err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	w.AddItemToRoom(couch, 1001)
	mimic, err := w.SpawnMobQuiet(99020, 5798)
	if err != nil {
		t.Fatalf("SpawnMobQuiet: %v", err)
	}
	out["Actor"] = ""
	out["Watcher"] = ""
	return w, out, actor, couch, mimic
}

// TestSpecCouch_ExactGate pins C's strcmp gate (src/spec_procs2.c:290-291):
// only an exact "couch" argument on look/examine triggers; "at couch" or
// "couch pillow" do not (Go previously used Contains).
func TestSpecCouch_ExactGate(t *testing.T) {
	for _, arg := range []string{"at couch", "couch pillow", "couchxyz"} {
		w, _, actor, _, _ := couchFidelityWorld(t)
		if specCouch(w, actor, nil, "look", arg) {
			t.Fatalf("arg %q should not trigger", arg)
		}
	}
	w, _, actor, _, _ := couchFidelityWorld(t)
	if !specCouch(w, actor, nil, "look", "couch") {
		t.Fatal("exact couch should trigger")
	}
}

// TestSpecCouch_LookerRoom pins ch->in_room (src/spec_procs2.c:293): the
// couch scan uses the looker's room, not a mob's room (the object spec's
// me is nil, as GetObjSpecForObject passes nil).
func TestSpecCouch_LookerRoom(t *testing.T) {
	w, _, actor, _, mimic := couchFidelityWorld(t)
	// Sentinel mob in a different room; the spec must still find the
	// couch in the looker's room 1001.
	sentinel, err := w.SpawnMobQuiet(99020, 9999)
	if err != nil {
		t.Fatalf("SpawnMobQuiet sentinel: %v", err)
	}
	if !specCouch(w, actor, sentinel, "look", "couch") {
		t.Fatal("spec should use the looker's room")
	}
	if got := mimic.GetRoomVNum(); got != actor.GetRoomVNum() {
		t.Fatalf("mimic room = %d, want looker room %d", got, actor.GetRoomVNum())
	}
}

// TestSpecCouch_ExtractsCouch pins obj_from_room + extract_obj
// (src/spec_procs2.c:296-297): the couch leaves the room and its
// ObjectLocation becomes Nowhere (not merely unlisted).
func TestSpecCouch_ExtractsCouch(t *testing.T) {
	w, _, actor, couch, _ := couchFidelityWorld(t)
	if !specCouch(w, actor, nil, "look", "couch") {
		t.Fatal("exact couch should trigger")
	}
	if couch.Location.IsInRoom() {
		t.Fatal("couch should no longer be in a room after extract")
	}
	if len(w.GetItemsInRoom(1001)) != 0 {
		t.Fatalf("room items = %d, want 0", len(w.GetItemsInRoom(1001)))
	}
}
