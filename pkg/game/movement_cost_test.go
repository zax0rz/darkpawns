package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// These tests guard the movement-cost table and immortal exemption (DP-1029 / F9).
// See docs/briefs/BRIEF-2026-07-11-glm-dp1029-movement-cost.md.

// TestMovementLossTable asserts the single shared movementLoss table matches the
// one-true table from src/constants.c movement_loss[], guarding the C
// comment/value swap at indices 8/9 from regressing.
func TestMovementLossTable(t *testing.T) {
	want := []int{2, 2, 3, 4, 5, 7, 5, 6, 2, 6, 8, 6, 6, 6, 6, 4}
	if len(movementLoss) != 16 {
		t.Fatalf("movementLoss must have 16 entries, got %d", len(movementLoss))
	}
	for i, w := range want {
		if movementLoss[i] != w {
			t.Errorf("movementLoss[%d] = %d, want %d", i, movementLoss[i], w)
		}
	}

	// Explicitly pin the swapped entries: C's inline comments at idx 8/9 are
	// "Flying"/"Underwater" but the enum is SECT_UNDERWATER=8, SECT_FLYING=9.
	// The runtime indexes by enum, so the VALUES win. See the brief's TRAP section.
	if movementLoss[SECT_UNDERWATER] != 2 {
		t.Errorf("movementLoss[SECT_UNDERWATER=%d] = %d, want 2", SECT_UNDERWATER, movementLoss[SECT_UNDERWATER])
	}
	if movementLoss[SECT_FLYING] != 6 {
		t.Errorf("movementLoss[SECT_FLYING=%d] = %d, want 6", SECT_FLYING, movementLoss[SECT_FLYING])
	}
}

// TestSectorMoveCost samples a few sectors (incl. DESERT which previously fell
// through to default) and confirms an out-of-range index returns the INSIDE
// default rather than panicking.
func TestSectorMoveCost(t *testing.T) {
	cases := []struct {
		sector int
		want   int
	}{
		{SECT_INSIDE, 2},
		{SECT_FIELD, 3},
		{SECT_DESERT, 8}, // sectors 10-15 previously fell through to default 1
		{SECT_SWAMP, 4},
		{SECT_UNDERWATER, 2},
		{SECT_FLYING, 6},
	}
	for _, c := range cases {
		if got := sectorMoveCost(c.sector); got != c.want {
			t.Errorf("sectorMoveCost(%d) = %d, want %d", c.sector, got, c.want)
		}
	}

	// Out-of-range returns the INSIDE cost, not a panic.
	if got := sectorMoveCost(-1); got != movementLoss[SECT_INSIDE] {
		t.Errorf("sectorMoveCost(-1) = %d, want INSIDE default %d", got, movementLoss[SECT_INSIDE])
	}
	if got := sectorMoveCost(len(movementLoss)); got != movementLoss[SECT_INSIDE] {
		t.Errorf("sectorMoveCost(OOB) = %d, want INSIDE default %d", got, movementLoss[SECT_INSIDE])
	}
}

// newMoveCostTestWorld builds a world with two rooms of known sectors so the
// movement cost of moving between them is deterministic: room 1001 (SECT_FIELD,
// cost 3) ↔ room 1002 (SECT_DESERT, cost 8). Expected move cost = (3+8)/2 = 5.
func newMoveCostTestWorld(t *testing.T) *World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{
				VNum: 1001, Name: "Field", Zone: 1, Sector: SECT_FIELD,
				Exits: map[string]parser.Exit{
					"north": {Direction: "north", ToRoom: 1002, DoorState: 0},
				},
			},
			{
				VNum: 1002, Name: "Desert", Zone: 1, Sector: SECT_DESERT,
				Exits: map[string]parser.Exit{
					"south": {Direction: "south", ToRoom: 1001, DoorState: 0},
				},
			},
		},
		Mobs: []parser.Mob{},
		Objs: []parser.Obj{},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })
	return w
}

// TestMovePlayer_MortalPaysMoveCost verifies a mortal player's move points drop
// by the (src+dst)/2 cost when moving between two rooms of known sectors.
func TestMovePlayer_MortalPaysMoveCost(t *testing.T) {
	w := newMoveCostTestWorld(t)

	mortal := NewPlayer(1, "Mortal", 1001)
	mortal.Level = 1 // well below LVL_IMMORT
	mortal.SetMove(100)
	if err := w.AddPlayer(mortal); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	before := mortal.GetMove()
	room, err := w.MovePlayer(mortal, "north")
	if err != nil {
		t.Fatalf("MovePlayer failed: %v", err)
	}
	if room == nil || room.VNum != 1002 {
		t.Fatalf("expected move into room 1002, got %v", room)
	}

	wantCost := (sectorMoveCost(SECT_FIELD) + sectorMoveCost(SECT_DESERT)) / 2 // (3+8)/2 = 5
	if got := before - mortal.GetMove(); got != wantCost {
		t.Errorf("mortal move cost = %d, want %d (before=%d after=%d)", got, wantCost, before, mortal.GetMove())
	}
}

// TestMovePlayer_ImmortalExempt verifies an immortal's move points are unchanged.
func TestMovePlayer_ImmortalExempt(t *testing.T) {
	w := newMoveCostTestWorld(t)

	immort := NewPlayer(2, "Immort", 1001)
	immort.Level = LVL_IMMORT // immortals move free (act.movement.c:210)
	immort.SetMove(100)
	if err := w.AddPlayer(immort); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	before := immort.GetMove()
	room, err := w.MovePlayer(immort, "north")
	if err != nil {
		t.Fatalf("MovePlayer failed: %v", err)
	}
	if room == nil || room.VNum != 1002 {
		t.Fatalf("expected move into room 1002, got %v", room)
	}

	if got := immort.GetMove(); got != before {
		t.Errorf("immortal move points changed: before=%d after=%d (should be unchanged)", before, got)
	}
}

// TestMovePlayer_ImmortalExemptWhenExhausted confirms immortals move even with
// zero move points — they never hit the "too exhausted" path.
func TestMovePlayer_ImmortalExemptWhenExhausted(t *testing.T) {
	w := newMoveCostTestWorld(t)

	immort := NewPlayer(3, "TiredImmort", 1001)
	immort.Level = LVL_IMMORT
	immort.SetMove(0) // zero move — a mortal would be blocked
	if err := w.AddPlayer(immort); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	room, err := w.MovePlayer(immort, "north")
	if err != nil {
		t.Fatalf("immortal with 0 move should still move, got err: %v", err)
	}
	if room == nil || room.VNum != 1002 {
		t.Errorf("expected room 1002, got %v", room)
	}
}

// TestMovePlayer_MortalExhaustedBlocked confirms a mortal with too few move
// points is still blocked (regression guard for the immortal short-circuit).
func TestMovePlayer_MortalExhaustedBlocked(t *testing.T) {
	w := newMoveCostTestWorld(t)

	mortal := NewPlayer(4, "TiredMortal", 1001)
	mortal.Level = 1
	mortal.SetMove(1) // less than the 5 cost
	if err := w.AddPlayer(mortal); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	room, err := w.MovePlayer(mortal, "north")
	if err == nil {
		t.Fatal("exhausted mortal should fail to move, got nil error")
	}
	if room != nil {
		t.Errorf("exhausted mortal should not change rooms, got %v", room)
	}
	if mortal.GetRoom() != 1001 {
		t.Errorf("mortal should still be in 1001, got %d", mortal.GetRoom())
	}
}
