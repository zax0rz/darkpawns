package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestWnewbieRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["wnewbie"]
	if !ok {
		t.Fatal("wnewbie command has no C gate")
	}
	if entry.MinLevel != LVL_IMMORT || entry.MinPosition != combat.PosDead {
		t.Fatalf("wnewbie gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_IMMORT, combat.PosDead)
	}
}

func TestCmdNewbieCreatesCOrderAndAudiences(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Newbie room", Zone: 1}},
		Objs: []parser.Obj{
			{VNum: 8019, Keywords: "tunic", ShortDesc: "a tunic"},
			{VNum: 8062, Keywords: "bread", ShortDesc: "a loaf of bread"},
			{VNum: 8063, Keywords: "skin", ShortDesc: "a waterskin"},
			{VNum: 8023, Keywords: "club", ShortDesc: "a club"},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	m := newTestManager(t, w, nil)
	actor := makeCommandTestSession(t, m, "Newbiewizard", LVL_IMPL, 1001)
	target := makeCommandTestSession(t, m, "Newbiepeer", 1, 1001)
	registerInWorld(t, actor)
	registerInWorld(t, target)

	if err := cmdNewbie(actor, []string{"newbiepeer", "trailing", "words"}); err != nil {
		t.Fatalf("cmdNewbie: %v", err)
	}

	actorText := drainSendChannel(t, actor)
	if !strings.Contains(actorText, "Newbied.\\r\\n") || strings.Contains(actorText, "magickal gesture") {
		t.Fatalf("actor output = %q, want Newbied acknowledgement only", actorText)
	}
	targetText := drainSendChannel(t, target)
	wantTarget := "Newbiewizard makes a magickal gesture, creating a bunch of equipment, and hands it to you!"
	if !strings.Contains(targetText, wantTarget) {
		t.Fatalf("target output = %q, want %q", targetText, wantTarget)
	}

	items := target.player.GetInventory()
	if len(items) != 4 {
		t.Fatalf("target inventory length = %d, want 4", len(items))
	}
	wantVNums := []int{8023, 8063, 8062, 8019}
	for i, want := range wantVNums {
		if items[i].VNum != want {
			t.Errorf("target inventory item %d = %d, want %d", i, items[i].VNum, want)
		}
	}
}

func TestCmdNewbieCreatesItemsForMobTarget(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Newbie room", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 2001, Keywords: "guard", ShortDesc: "a guard"}},
		Objs: []parser.Obj{
			{VNum: 8019, Keywords: "tunic", ShortDesc: "a tunic"},
			{VNum: 8062, Keywords: "bread", ShortDesc: "a loaf of bread"},
			{VNum: 8063, Keywords: "skin", ShortDesc: "a waterskin"},
			{VNum: 8023, Keywords: "club", ShortDesc: "a club"},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	m := newTestManager(t, w, nil)
	actor := makeCommandTestSession(t, m, "Newbiewizard", LVL_IMPL, 1001)
	registerInWorld(t, actor)
	mob, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}

	if err := cmdNewbie(actor, []string{"guard"}); err != nil {
		t.Fatalf("cmdNewbie mob: %v", err)
	}
	if got := drainSendChannel(t, actor); !strings.Contains(got, "Newbied.\\r\\n") {
		t.Fatalf("actor output = %q, want Newbied acknowledgement", got)
	}
	wantVNums := []int{8023, 8063, 8062, 8019}
	if len(mob.Inventory) != len(wantVNums) {
		t.Fatalf("mob inventory length = %d, want %d", len(mob.Inventory), len(wantVNums))
	}
	for i, want := range wantVNums {
		if mob.Inventory[i].VNum != want {
			t.Errorf("mob inventory item %d = %d, want %d", i, mob.Inventory[i].VNum, want)
		}
	}
}

func TestCmdNewbieUsesCNoPersonAndEmptyArgumentBytes(t *testing.T) {
	parsed := &parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Newbie room", Zone: 1}}}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	m := newTestManager(t, w, nil)
	actor := makeCommandTestSession(t, m, "Newbiewizard", LVL_IMPL, 1001)
	registerInWorld(t, actor)

	if err := cmdNewbie(actor, nil); err != nil {
		t.Fatalf("cmdNewbie empty: %v", err)
	}
	if got := drainSendChannel(t, actor); !strings.Contains(got, "Whom do you wish to newbie?\\r\\n") {
		t.Fatalf("empty-argument output = %q", got)
	}

	if err := cmdNewbie(actor, []string{"Nobody"}); err != nil {
		t.Fatalf("cmdNewbie missing: %v", err)
	}
	if got := drainSendChannel(t, actor); !strings.Contains(got, "No-one by that name here.\\r\\n") {
		t.Fatalf("missing-target output = %q", got)
	}
}
