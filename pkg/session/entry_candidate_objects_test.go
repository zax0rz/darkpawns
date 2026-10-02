package session

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/db"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Go restores crash objects before perform_dupe_check; C frees its unloaded
// candidate at src/interpreter.c:1619, before Crash_load at 2184-2194.
func TestEntryDuplicateCandidateObjects(t *testing.T) {
	for _, mode := range []string{"reconnect", "usurp", "unswitch", "editor"} {
		t.Run(mode, func(t *testing.T) {
			database := entryDatabase(t)
			record := entrySeed(t, database, "Returner")
			parsed := &parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Saved"}}, Objs: []parser.Obj{
				{VNum: 8023, Keywords: "club", ShortDesc: "a club", WearFlags: [4]int{1}},
				{VNum: 8019, Keywords: "tunic", ShortDesc: "a tunic", WearFlags: [4]int{9}, Affects: []parser.ObjAffect{{Location: game.ApplyCon, Modifier: 2}}},
				{VNum: 8020, Keywords: "pack", ShortDesc: "a pack", TypeFlag: game.ITEM_CONTAINER, WearFlags: [4]int{1}},
			}}
			world, err := game.NewWorld(parsed)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(world.StopAITicker)
			inv := []game.SaveItemData{{VNum: 8020, Count: 1}, {VNum: 8023, Count: 1, ContainerIndex: 1, ContainerVNum: 8020}, {VNum: 8023, Count: 1}}
			record.Inventory, err = json.Marshal(inv)
			if err != nil {
				t.Fatal(err)
			}
			record.Equipment, err = json.Marshal([]game.SaveItemData{{VNum: 8019, Count: 1, Locate: 6}})
			if err != nil {
				t.Fatal(err)
			}
			if err := database.SavePlayer(record); err != nil {
				t.Fatal(err)
			}
			m := newTestManager(t, world, database)
			old := makeTestSession(t, m, "Returner", 1001, true)
			old.player, err = db.RecordToPlayer(record, world)
			if err != nil {
				t.Fatal(err)
			}
			registerTestSession(t, m, old, "Returner")
			oldItems := append([]*game.ObjectInstance(nil), world.GetAllObjects()...)
			old.player.SetPlrFlag(game.PlrCrash, false)
			switch mode {
			case "reconnect":
				old.player.SetLinkless(true)
			case "usurp":
				old.transportDone = make(chan struct{})
			case "unswitch":
				old.isSwitched = true
				old.switchedOriginal = old.player
			case "editor":
				old.transportDone = make(chan struct{})
				old.textEdit = &textEditState{}
			}
			candidate, err := db.RecordToPlayer(record, world)
			if err != nil {
				t.Fatal(err)
			}
			candidate.Inventory.Capacity = 0 // even a full/over-capacity candidate must discard worn items
			if len(world.GetAllObjects()) != len(oldItems)*2 {
				t.Fatal("fixture did not restore separate object trees")
			}
			fresh := makeCharSession(t, m)
			fresh.player = candidate
			fresh.authenticated = true
			if !fresh.performDupeCheck() || fresh.player != old.player {
				t.Fatal("did not adopt retained body")
			}
			if got := len(world.GetAllObjects()); got != len(oldItems) {
				t.Fatalf("loaded candidate leaked objects: got=%d want=%d", got, len(oldItems))
			}
			if old.player.NeedsCrashSave() {
				t.Fatal("discard marked retained body PLR_CRASH through shared name")
			}
			if candidate.Inventory.GetItemCount() != 0 || len(candidate.Equipment.GetEquippedItems()) != 0 {
				t.Fatal("candidate still references discarded objects")
			}
			for _, obj := range oldItems {
				found := false
				for _, live := range world.GetAllObjects() {
					if live == obj {
						found = true
					}
				}
				if !found || obj.Location.Kind == game.ObjNowhere {
					t.Fatal("retained object removed", obj.ID)
				}
			}
			rec, err := database.GetPlayer("Returner")
			if err != nil || !bytes.Equal(rec.Inventory, record.Inventory) || !bytes.Equal(rec.Equipment, record.Equipment) {
				t.Fatal("takeover rewrote durable object payload")
			}
			m.UnregisterSession(old)
			if m.world.GetPlayerCount() != 1 || fresh.player.Inventory.GetItemCount() != 2 {
				t.Fatal("stale teardown lost retained body/items")
			}
		})
	}
}
