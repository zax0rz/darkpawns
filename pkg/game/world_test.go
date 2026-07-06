package game

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestStopAITickerStopsBothTickers verifies that StopAITicker can be called
// without panic and that the World remains usable afterwards. The AI and point
// tickers share the World's done channel; closing it stops both loops.
func TestStopAITickerStopsBothTickers(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	// Stop once — should close the shared done channel and stop both tickers.
	w.StopAITicker()

	// Stop again — should be safe/no-op because the channel is already closed.
	w.StopAITicker()

	// World methods should still work after stopping tickers.
	if _, ok := w.GetRoom(1001); !ok {
		t.Error("GetRoom should still work after StopAITicker")
	}
}

// TestStopPeriodicResetsStopsTicker verifies that the World wrapper safely
// stops the periodic zone reset goroutine and is idempotent.
func TestStopPeriodicResetsStopsTicker(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
		Zones: []parser.Zone{{Number: 1, Lifespan: 10}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	// Start and then stop periodic resets.
	w.StartPeriodicResets(100 * time.Millisecond)
	w.StopPeriodicResets()

	// Second stop should be safe.
	w.StopPeriodicResets()
}

// TestStopPeriodicResetsWithoutSpawner verifies the wrapper is safe when no
// periodic reset goroutine has ever been started.
func TestStopPeriodicResetsWithoutSpawner(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	w.StopPeriodicResets()
}

func TestHasSpecInRoom_NoSpecs(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Empty Room", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	w.RebuildSpecRooms()
	if w.HasSpecInRoom(1001) {
		t.Fatal("HasSpecInRoom(1001) = true for empty room, want false")
	}
}

func TestHasSpecInRoom_MobWithSpec(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Mob Room", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 1, ShortDesc: "puff", Level: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	if _, err := w.SpawnMob(1, 1001); err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	w.RebuildSpecRooms()

	if !w.HasSpecInRoom(1001) {
		t.Fatal("HasSpecInRoom(1001) = false, want true for mob with spec")
	}
}

func TestHasSpecInRoom_BoardObject(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Board Room", Zone: 1}},
		Objs:  []parser.Obj{{VNum: 8099, Keywords: "board"}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	obj := NewObjectInstance(&parsed.Objs[0], 1001)
	w.AddItemToRoom(obj, 1001)
	w.RebuildSpecRooms()

	if !w.HasSpecInRoom(1001) {
		t.Fatal("HasSpecInRoom(1001) = false, want true for board object")
	}
}

func TestHasSpecInRoom_RoomSpec(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 8008, Name: "Pray Room", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	w.RebuildSpecRooms()
	if !w.HasSpecInRoom(8008) {
		t.Fatal("HasSpecInRoom(8008) = false, want true for room with spec")
	}
}

func TestHasSpecInRoom_AfterRefresh(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Mob Room", Zone: 1}},
		Mobs:  []parser.Mob{{VNum: 1, ShortDesc: "puff", Level: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	mob, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	w.RebuildSpecRooms()
	if !w.HasSpecInRoom(1001) {
		t.Fatal("HasSpecInRoom(1001) = false before removal, want true")
	}

	w.ExtractMob(mob)
	w.RebuildSpecRooms()
	if w.HasSpecInRoom(1001) {
		t.Fatal("HasSpecInRoom(1001) = true after removal, want false")
	}
}
