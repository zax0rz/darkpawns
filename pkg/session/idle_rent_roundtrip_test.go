package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// C saves Crash_rentsave before extraction (src/limits.c:438-451), destroys
// unrentable objects (src/objsave.c:730-760), and restores at menu entry
// (src/interpreter.c:2184-2194). Exercise that boundary with real SQLite.
func idleRentRoundtrip(t *testing.T, norent bool) *Session {
	t.Helper()
	t.Setenv("JWT_SECRET", "idle-rent-test-secret-at-least-32-bytes")
	database := entryDatabase(t)
	record := entrySeed(t, database, "Renter")
	parsed := &parser.World{Rooms: []parser.Room{{VNum: 1}, {VNum: 3}, {VNum: 1001}, {VNum: game.MortalStartRoom}}, Objs: []parser.Obj{
		{VNum: 8032, Keywords: "bag", ShortDesc: "a bag", TypeFlag: game.ITEM_CONTAINER, WearFlags: [4]int{1}, Values: [4]int{100}},
		{VNum: 8010, Keywords: "bread", ShortDesc: "bread", WearFlags: [4]int{1}},
		{VNum: 8019, Keywords: "tunic", ShortDesc: "a tunic", TypeFlag: 11, WearFlags: [4]int{9}, ExtraFlags: [4]int{1 << 17}},
		{VNum: 4291, Keywords: "cloak", ShortDesc: "a cloak", TypeFlag: 11, WearFlags: [4]int{1025}, ExtraFlags: [4]int{game.FlagNoRent}},
		{VNum: 4320, Keywords: "fruit", ShortDesc: "fruit", WearFlags: [4]int{1}, ExtraFlags: [4]int{game.FlagNoRent}},
	}}
	world, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	world.StopAITicker()
	t.Cleanup(world.StopAITicker)
	m := newTestManager(t, world, database)
	m.WirePlayerSaver(world)
	s := makeTestSession(t, m, "Renter", 1001, true)
	s.player, err = db.RecordToPlayer(record, world)
	if err != nil {
		t.Fatal(err)
	}
	s.player.SetRoom(1001)
	s.player.SetLevel(1)
	registerTestSession(t, m, s, "Renter")
	spawn := func(vnum int) *game.ObjectInstance {
		obj, e := world.SpawnObject(vnum, -1)
		if e != nil {
			t.Fatal(e)
		}
		if e = world.MoveObjectToPlayerInventory(obj, s.player); e != nil {
			t.Fatal(e)
		}
		return obj
	}
	bag := spawn(8032)
	bread := spawn(8010)
	if err = world.MoveObjectToContainer(bread, bag); err != nil {
		t.Fatal(err)
	}
	tunic := spawn(8019)
	if err = world.MoveObject(tunic, game.LocEquippedPlayer(s.player.Name, game.SlotBody)); err != nil {
		t.Fatal(err)
	}
	if norent {
		cloak := spawn(4291)
		if err = world.MoveObject(cloak, game.LocEquippedPlayer(s.player.Name, game.SlotAbout)); err != nil {
			t.Fatal(err)
		}
		spawn(4320)
	}
	for tick := 0; tick < 31; tick++ {
		world.CheckIdling(s.player)
	}
	m.ExtractPendingChars()
	if _, ok := world.GetPlayer("Renter"); ok {
		t.Fatal("force-rented body remained in world")
	}
	fresh := makeCharSession(t, m)
	fresh.player = game.NewPlayer(int(record.ID), "Renter", 1001)
	fresh.player.RentedOut = true
	fresh.playerName = "Renter"
	fresh.authenticated = true
	fresh.menuActive = true
	fresh.menuStage = "menu"
	if err = fresh.enterReturningPlayer(); err != nil {
		t.Fatal(err)
	}
	if len(world.GetAllObjects()) != 3 {
		t.Fatalf("restored world objects=%d, want only bag, bread, tunic", len(world.GetAllObjects()))
	}
	return fresh
}

func TestIdleRentRoundtripInventory(t *testing.T) {
	s := idleRentRoundtrip(t, false)
	items := s.player.Inventory.Items
	if len(items) != 1 || items[0].GetVNum() != 8032 {
		t.Fatalf("restored inventory=%v, want bag", items)
	}
	if len(items[0].Contains) != 1 || items[0].Contains[0].GetVNum() != 8010 {
		t.Fatal("nested rentable bread was not restored")
	}
	tunic, ok := s.player.Equipment.GetItemInSlot(game.SlotBody)
	if !ok || tunic.GetVNum() != 8019 {
		t.Fatal("rentable tunic was not restored to its slot")
	}
	if tunic.GetShortDesc() != "Renter's tunic" {
		t.Fatalf("restored TAKE_NAME description=%q", tunic.GetShortDesc())
	}
}
