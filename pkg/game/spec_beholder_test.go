package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// scriptBeholderDraws pins SPECIAL(beholder)'s number() draws to a fixed
// sequence and records each call's bounds, mirroring scriptHuntDraws. Any draw
// beyond the scripted sequence fails the test, so an extra or reordered gate
// cannot pass silently.
func scriptBeholderDraws(t *testing.T, results ...int) *[][2]int {
	t.Helper()
	previous := beholderNumber
	idx := 0
	bounds := &[][2]int{}
	beholderNumber = func(from, to int) int {
		*bounds = append(*bounds, [2]int{from, to})
		if idx >= len(results) {
			t.Fatalf("unexpected extra beholder draw number(%d,%d)", from, to)
		}
		v := results[idx]
		idx++
		return v
	}
	t.Cleanup(func() { beholderNumber = previous })
	return bounds
}

// TestSpecBeholderPeacefulPulseDraws pins C's not-fighting arm
// (src/new_cmds2.c:345-373): count the room's people, return when there is
// nobody else, pick one with number(0, targets), then pick sleep, charm or
// curse with number(0, 2). C returns FALSE, so the rest of the mobile's pulse
// still runs. The port skipped this arm entirely, which is the draw row in
// spec-proc-beholder-command.
func TestSpecBeholderPeacefulPulseDraws(t *testing.T) {
	w, _ := newActorRoomTestWorld(t, parser.Room{VNum: 1001, Name: "Vault"})
	beholder := newSpecProcTestMob(t, w, 1001, 12)
	beholder.SetPosition(combat.PosStanding)

	// cRoomPeople is newest-first, so both players land ahead of the beholder
	// and number(0, 3) == 1 must pick the newest player.
	first := NewPlayer(1, "First", 1001)
	if err := w.AddPlayer(first); err != nil {
		t.Fatalf("AddPlayer first: %v", err)
	}
	second := NewPlayer(2, "Second", 1001)
	if err := w.AddPlayer(second); err != nil {
		t.Fatalf("AddPlayer second: %v", err)
	}

	bounds := scriptBeholderDraws(t, 1, 2)
	if specBeholder(w, nil, beholder, "", "") {
		t.Fatal("peaceful arm must return FALSE so the mobile's pulse continues")
	}
	if got := len(*bounds); got != 2 {
		t.Fatalf("draws = %d, want 2: number(0, targets) then number(0, 2)", got)
	}
	if got, want := (*bounds)[0], [2]int{0, 3}; got != want {
		t.Fatalf("first draw = number(%d,%d), want number(0,3) counting every occupant", got[0], got[1])
	}
	if got, want := (*bounds)[1], [2]int{0, 2}; got != want {
		t.Fatalf("second draw = number(%d,%d), want number(0,2) for sleep/charm/curse", got[0], got[1])
	}
}

// TestSpecBeholderPeacefulLoneRoomDoesNotDraw pins the `targets <= 1` gate: a
// beholder alone in the room draws nothing at all.
func TestSpecBeholderPeacefulLoneRoomDoesNotDraw(t *testing.T) {
	w, _ := newActorRoomTestWorld(t, parser.Room{VNum: 1001, Name: "Vault"})
	beholder := newSpecProcTestMob(t, w, 1001, 12)
	beholder.SetPosition(combat.PosStanding)

	bounds := scriptBeholderDraws(t)
	if specBeholder(w, nil, beholder, "", "") {
		t.Fatal("lone beholder must decline")
	}
	if got := len(*bounds); got != 0 {
		t.Fatalf("draws = %d, want 0: C returns before number(0, targets)", got)
	}
}

// TestSpecBeholderFightingArmDraws pins C's fighting arm
// (src/new_cmds2.c:377-397): one number(0,4) per candidate whose FIGHTING is
// the beholder, then number(0,10) for disrupt, else number(0,5) for
// disintegrate.
func TestSpecBeholderFightingArmDraws(t *testing.T) {
	w, _ := newActorRoomTestWorld(t, parser.Room{VNum: 1001, Name: "Vault"})
	beholder := newSpecProcTestMob(t, w, 1001, 20)
	beholder.SetPosition(combat.PosFighting)
	player := NewPlayer(1, "Victim", 1001)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	player.SetFightingBody(beholder)
	beholder.SetFightingBody(player)

	// 0 -> the scan takes this candidate; 7 -> number(0,10) misses disrupt;
	// 0 -> number(0,5) hits disintegrate.
	bounds := scriptBeholderDraws(t, 0, 7, 0)
	if !specBeholder(w, nil, beholder, "", "") {
		t.Fatal("disintegrate arm returns TRUE in C")
	}
	if got := len(*bounds); got != 3 {
		t.Fatalf("draws = %d, want 3: scan, disrupt, disintegrate", got)
	}
	if got, want := (*bounds)[0], [2]int{0, 4}; got != want {
		t.Fatalf("scan draw = number(%d,%d), want number(0,4)", got[0], got[1])
	}
	if got, want := (*bounds)[1], [2]int{0, 10}; got != want {
		t.Fatalf("disrupt draw = number(%d,%d), want number(0,10)", got[0], got[1])
	}
	if got, want := (*bounds)[2], [2]int{0, 5}; got != want {
		t.Fatalf("disintegrate draw = number(%d,%d), want number(0,5)", got[0], got[1])
	}
}
