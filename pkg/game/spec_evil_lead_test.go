package game

import (
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

// TestSpecEvilLead_NeverHandles pins C's always-FALSE evillead
// (src/spec_procs2.c:1345,1354-1355): obj is never fetched, so the
// junk, goody-goody and cheese gives all fall through to the ordinary
// give with no output, no object moved and no teleport.
func TestSpecEvilLead_NeverHandles(t *testing.T) {
	cases := []struct {
		name      string
		vnum      int
		alignment int
		arg       string
	}{
		{"junk", 13111, -100, "watch swiss"},
		{"goody", 13100, 100, "cheese swiss"},
		{"cheese", 13100, -100, "cheese swiss"},
		{"wrong target", 13100, -100, "cheese wrongtarget"},
		{"empty", 13100, -100, "  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, out, giver, _, lead := evilLeadTestWorld(t)
			giver.SetAlignment(tc.alignment)
			giveObjectToPlayer(t, w, giver, tc.vnum)
			if specEvilLead(w, giver, lead, "give", tc.arg) {
				t.Fatal("C's evillead returns FALSE on every give")
			}
			if out["Giver"] != "" || out["Watcher"] != "" {
				t.Fatalf("output giver=%q watcher=%q, want none", out["Giver"], out["Watcher"])
			}
			if got := giver.GetRoomVNum(); got != 1001 {
				t.Fatalf("giver room = %d, want origin 1001", got)
			}
			if len(lead.Inventory) != 0 {
				t.Fatalf("lead inventory = %d items, want 0", len(lead.Inventory))
			}
		})
	}
}
