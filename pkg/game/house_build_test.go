package game

// house_build_test.go — hcontrol build through the startup-wired player
// lookup (BRIEF-2026-10-24 Investigation 4): the command was registered but
// its lookup dependency was only ever wired in tests, so every build died at
// "Player lookup not available." and no house record could exist.

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// wireHouseLookup registers a lookup that resolves exactly Houselord (id 7),
// with save/restore of whatever was registered before.
func wireHouseLookup(t *testing.T) {
	t.Helper()
	oldNameByID, oldIDByName := getPlayerNameByID, getPlayerIDByName
	RegisterHousePlayerLookup(
		func(id int64) string {
			if id == 7 {
				return "Houselord"
			}
			return ""
		},
		func(name string) int64 {
			if strings.EqualFold(name, "Houselord") {
				return 7
			}
			return -1
		},
	)
	t.Cleanup(func() { RegisterHousePlayerLookup(oldNameByID, oldIDByName) })
}

func redirectHouseControl(t *testing.T) {
	t.Helper()
	oldFilename := houseControlFilename
	houseControlFilename = filepath.Join(t.TempDir(), "house", "house_control.json")
	t.Cleanup(func() { houseControlFilename = oldFilename })
}

// houseBuildWorld is 100 -north-> 200 with a two-way exit: a valid house
// (100) and atrium (200).
func houseBuildWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 100, Name: "Tower Second Floor", Zone: 1, Exits: map[string]parser.Exit{
			"north": {Direction: "north", ToRoom: 200},
		}},
		{VNum: 200, Name: "Tower Ground Floor", Zone: 1, Exits: map[string]parser.Exit{
			"south": {Direction: "south", ToRoom: 100},
		}},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

func TestHcontrolBuildHouseWithWiredLookup(t *testing.T) {
	redirectHouseControl(t)
	wireHouseLookup(t)
	w := houseBuildWorld(t)
	var out strings.Builder
	w.MessageSink = func(name string, msg []byte) { out.Write(msg) }
	ch := NewPlayer(1, "Zach", 100)
	ch.worldRef = w
	ch.SetLevel(LVL_GRGOD)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	w.HcontrolBuildHouse(ch, "100 north Houselord")

	if got, want := out.String(), "House built.  Mazel tov!\r\n"; got != want {
		t.Fatalf("build output = %q, want %q", got, want)
	}
	if len(w.HouseControl) != 1 {
		t.Fatalf("house records = %d, want 1", len(w.HouseControl))
	}
	rec := w.HouseControl[0]
	if rec.VNum != 100 || rec.Atrium != 200 || rec.ExitNum != 0 || rec.Owner != 7 {
		t.Fatalf("record = %+v", rec)
	}
	if !HasRoomFlag(w.GetRoomInWorld(100), "house") || !HasRoomFlag(w.GetRoomInWorld(100), "private") {
		t.Fatal("house room not flagged house+private")
	}
	if !HasRoomFlag(w.GetRoomInWorld(200), "atrium") {
		t.Fatal("atrium room not flagged")
	}
}

func TestHcontrolBuildHouseRejects(t *testing.T) {
	redirectHouseControl(t)
	wireHouseLookup(t)

	t.Run("unknown owner", func(t *testing.T) {
		w := houseBuildWorld(t)
		var out strings.Builder
		w.MessageSink = func(name string, msg []byte) { out.Write(msg) }
		ch := NewPlayer(1, "Zach", 100)
		ch.worldRef = w
		if err := w.AddPlayer(ch); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}

		w.HcontrolBuildHouse(ch, "100 north nobody")
		if got, want := out.String(), "Unknown player 'nobody'.\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
		if len(w.HouseControl) != 0 {
			t.Fatalf("record created for unknown owner: %+v", w.HouseControl)
		}
	})

	t.Run("exit must be two-way", func(t *testing.T) {
		w := houseBuildWorld(t)
		// Drop the reverse exit.
		w.mu.Lock()
		delete(w.rooms[200].Exits, "south")
		w.mu.Unlock()
		var out strings.Builder
		w.MessageSink = func(name string, msg []byte) { out.Write(msg) }
		ch := NewPlayer(1, "Zach", 100)
		ch.worldRef = w
		if err := w.AddPlayer(ch); err != nil {
			t.Fatalf("AddPlayer: %v", err)
		}

		w.HcontrolBuildHouse(ch, "100 north Houselord")
		if got, want := out.String(), "A house's exit must be a two-way door.\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})
}

// TestHouseBootDropsUnresolvableOwner pins the boot behavior the wiring
// activates: with the lookup live, House_boot skips a record whose owner no
// longer resolves (src/house.c:290 "owner no longer exists -- skip") and
// rewrites the control file without it.
func TestHouseBootDropsUnresolvableOwner(t *testing.T) {
	redirectHouseControl(t)
	wireHouseLookup(t)
	w := houseBuildWorld(t)

	w.mu.Lock()
	w.HouseControl = []HouseControl{
		{VNum: 100, Atrium: 200, ExitNum: 0, Owner: 7, Key: -1},   // resolves
		{VNum: 99999, Atrium: 200, ExitNum: 0, Owner: 7, Key: -1}, // room gone
	}
	w.mu.Unlock()
	w.saveHouseControl()

	// A second record whose owner is gone shares the file with a good one.
	w.mu.Lock()
	w.HouseControl = append(w.HouseControl, HouseControl{VNum: 200, Atrium: 100, ExitNum: 1, Owner: 12345, Key: -1})
	w.mu.Unlock()
	w.saveHouseControl()

	w.HouseBoot()

	if len(w.HouseControl) != 1 {
		t.Fatalf("records after boot = %d (%+v), want 1", len(w.HouseControl), w.HouseControl)
	}
	if w.HouseControl[0].VNum != 100 {
		t.Fatalf("surviving record = %+v, want the resolvable one", w.HouseControl[0])
	}
}

// TestHcontrolBuildHouseWithoutLookup documents the pre-wiring dead end: with
// no lookup registered (the production state before this change),
// hcontrol build refused every request and no house could be built.
func TestHcontrolBuildHouseWithoutLookup(t *testing.T) {
	redirectHouseControl(t)
	oldNameByID, oldIDByName := getPlayerNameByID, getPlayerIDByName
	RegisterHousePlayerLookup(nil, nil)
	t.Cleanup(func() { RegisterHousePlayerLookup(oldNameByID, oldIDByName) })

	w := houseBuildWorld(t)
	var out strings.Builder
	w.MessageSink = func(name string, msg []byte) { out.Write(msg) }
	ch := NewPlayer(1, "Zach", 100)
	ch.worldRef = w
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	w.HcontrolBuildHouse(ch, "100 north Houselord")
	if got, want := out.String(), "Player lookup not available.\r\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}
