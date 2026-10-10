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

func TestRoomsPreservesParsedRNumOrder(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 300},
		{VNum: 100},
		{VNum: 200},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	rooms := w.Rooms()
	if len(rooms) != 3 {
		t.Fatalf("Rooms returned %d rooms, want 3", len(rooms))
	}
	for i, want := range []int{300, 100, 200} {
		if got := rooms[i].VNum; got != want {
			t.Fatalf("Rooms()[%d].VNum = %d, want %d", i, got, want)
		}
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

func TestRealtimeWorldTickersFreezeWhenDPClockIsSet(t *testing.T) {
	t.Setenv("DP_CLOCK", "1")
	w, err := NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	player := NewCharacter(1, "Clocktest", ClassWarrior, RaceHuman)
	if err := w.AddPlayer(player); err != nil {
		t.Fatal(err)
	}
	full := player.GetCondition(CondFull)

	time.Sleep(20 * time.Millisecond)

	if got := player.GetCondition(CondFull); got != full {
		t.Fatalf("point update changed fullness under DP_CLOCK: %d -> %d", full, got)
	}
}

// TestShutdown_NoMutationAfterWorldStops verifies that the shutdown sequence
// (StopAITicker + stopping the heartbeat) stops world-mutating goroutines before
// sessions drain (COV-3 / DP-964). C's init_game() saves no world state on the
// way out — only the clan table, the in-game date and the player file — so the
// guarantee here is that nothing keeps mutating while sessions close.
//
// Pre-shutdown: manual AITick proves mobs CAN wander (mutation mechanism works).
// Shutdown: StopAITicker closes the shared done channel. Zone updates are
// synchronous heartbeat callbacks, with no independent goroutine.
// Post-shutdown: World is still readable (GetRoom etc. still work) and AITick
// is safe to call (no corrupted state). Both methods are idempotent.
