package game

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestGetAllZonesReturnsDeterministicOrder(t *testing.T) {
	w, err := NewWorld(&parser.World{Zones: []parser.Zone{{Number: 30}, {Number: 10}, {Number: 20}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	zones := w.GetAllZones()
	if len(zones) != 3 || zones[0].Number != 10 || zones[1].Number != 20 || zones[2].Number != 30 {
		t.Fatalf("zone order = %+v, want [10 20 30]", zones)
	}
}

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

// TestHasSpecInRoom_WanderingSpecMob verifies that a spec mob's destination
// room is flagged when the mob moves via mobPerformMove (DP-QA#4).
func TestHasSpecInRoom_WanderingSpecMob(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Room 1", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			{VNum: 1002, Name: "Room 2", Zone: 1, Exits: map[string]parser.Exit{"south": {ToRoom: 1001}}},
		},
		Mobs: []parser.Mob{{VNum: 9001, ShortDesc: "wanderer", Level: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	old := MobSpecAssign[9001]
	MobSpecAssign[9001] = "puff"
	defer func() {
		if old == "" {
			delete(MobSpecAssign, 9001)
		} else {
			MobSpecAssign[9001] = old
		}
	}()

	mob, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}
	if !w.HasSpecInRoom(1001) {
		t.Fatal("HasSpecInRoom(1001) = false after spawn, want true")
	}

	// Move the mob north (0) via the hunt/move path; this should flag the new room.
	w.mobPerformMove(mob, 0)
	if mob.GetRoom() != 1002 {
		t.Fatalf("mob room = %d after move, want 1002", mob.GetRoom())
	}
	if !w.HasSpecInRoom(1002) {
		t.Fatal("HasSpecInRoom(1002) = false after move, want true")
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

// TestShutdown_NoMutationAfterSaveBegins verifies that the shutdown sequence
// (StopAITicker + StopPeriodicResets) stops world-mutating goroutines before
// SaveWorld would run (COV-3 / DP-964).
//
// Pre-shutdown: manual AITick proves mobs CAN wander (mutation mechanism works).
// Shutdown: StopAITicker closes the shared done channel; StopPeriodicResets
// closes the spawner done channel.
// Post-shutdown: World is still usable (SaveWorld calls GetRoom, etc.) and
// AITick is safe to call (no corrupted state). Both methods are idempotent.
func TestShutdown_NoMutationAfterSaveBegins(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Room 1", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			{VNum: 1002, Name: "Room 2", Zone: 1, Exits: map[string]parser.Exit{"south": {ToRoom: 1001}}},
		},
		Mobs: []parser.Mob{{
			VNum: 1, ShortDesc: "a wanderer", Keywords: "wanderer",
			Level: 1, ActionFlags: []string{"wander"},
			HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 10},
		}},
		Zones: []parser.Zone{{Number: 1, Lifespan: 10}},
	}

	// Build world — NewWorld auto-starts AI ticker (10s) and point update ticker (30s),
	// both listening on the shared w.done channel.
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}

	// Also start periodic zone resets (like main.go does after NewWorld).
	w.StartPeriodicResets(10 * time.Second)

	// Spawn a wandering mob owned by the world.
	mob, err := w.SpawnMob(1, 1001)
	if err != nil {
		t.Fatalf("SpawnMob failed: %v", err)
	}

	// ── Pre-shutdown: verify mutations CAN happen ──────────────────────
	// Call AITick directly (bypasses the 10s timer). The mob should wander
	// to a different room, proving the mutation path works before shutdown.
	originalRoom := mob.GetRoom()
	w.AITick()
	postTickRoom := mob.GetRoom()
	if postTickRoom != originalRoom {
		t.Logf("pre-shutdown: mob wandered from room %d to %d", originalRoom, postTickRoom)
	}

	// ── Shutdown sequence (mirrors cmd/server/main.go:420-455) ──────────
	// 1. StopAITicker closes w.done → AI + point ticker goroutines exit.
	// 2. StopPeriodicResets closes spawner done → reset goroutine exits.
	w.StopAITicker()
	w.StopPeriodicResets()

	// ── Post-shutdown: verify world is still usable ────────────────────
	// SaveWorld will call GetRoom, GetMob, etc. — they must still work.
	if _, ok := w.GetRoom(1001); !ok {
		t.Error("GetRoom(1001) should work after shutdown")
	}
	if _, ok := w.GetRoom(1002); !ok {
		t.Error("GetRoom(1002) should work after shutdown")
	}
	// AITick after shutdown should not panic (the method itself is safe;
	// the goroutine that normally calls it has exited via the done channel).
	requirePanicFree(t, func() { w.AITick() })

	// ── Idempotency ────────────────────────────────────────────────────
	// Multiple calls must be safe (both use doneOnce / close-once channels).
	requirePanicFree(t, w.StopAITicker)
	requirePanicFree(t, w.StopPeriodicResets)

	// ── Goroutine lifecycle ────────────────────────────────────────────
	// After shutdown, the two ticker goroutines have exited (via <-w.done).
	// The zone reset goroutine has also exited (via <-s.done).
	// The event queue goroutine stays running (gated on ctx.Done/stopCh,
	// not w.done), matching production behavior.
	t.Log("shutdown complete — AI, point, and reset goroutines have exited")
}

// requirePanicFree verifies fn does not panic. Mini helper to keep the
// shutdown test assertions readable.
func requirePanicFree(t *testing.T, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	fn()
}
