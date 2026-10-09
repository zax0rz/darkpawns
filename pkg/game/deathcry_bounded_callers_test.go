package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// deathTrapMudlogWorld is the room set the death-trap path needs, plus a
// witness standing in the trap room whose delivered bytes the test can read.
func deathTrapMudlogWorld(t *testing.T) (*World, *Player, *Player, *strings.Builder) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Safe Room", Zone: 1, Exits: map[string]parser.Exit{"north": {ToRoom: 1002}}},
			// ROOM_DEATH is bit 1 → bitmask value 2.
			{VNum: 1002, Name: "Death Trap", Zone: 1, Flags: []string{"2"}},
			{VNum: MortalStartRoom, Name: "Temple", Zone: 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)

	output := &strings.Builder{}
	w.MessageSink = func(name string, message []byte) {
		if name == "Witness" {
			output.Write(message)
		}
	}
	victim := NewPlayer(1, "Victim", 1001)
	victim.SetLevel(10)
	victim.SetMove(100)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatal(err)
	}
	// The witness stands in the trap room, where death_cry broadcasts.
	witness := NewPlayer(2, "Witness", 1002)
	if err := w.AddPlayer(witness); err != nil {
		t.Fatal(err)
	}
	return w, victim, witness, output
}

// TestDeathCryBoundedCallers is the bounded caller proof for fight.c:506-516,
// the "death_cry() in fight.c called with ch->in_room = NOWHERE" diagnostic.
// Go never ported that arm because Go's callers cannot hand death_cry a
// NOWHERE room: the death-trap path is gated on a resolved ROOM_DEATH room
// (act_movement.go:427-430) and re-resolves the mount in the rider's room
// instead of reusing a pre-entry pointer the way C's do_move does
// (src/act.movement.c:195-202, :296-300). This test pins the second half: a
// rider whose mount has already left the world produces no mount cry at all,
// so there is no NOWHERE mount to cry for. Removing the re-resolution fails
// the assertion.
func TestDeathCryBoundedCallers(t *testing.T) {
	t.Run("the rider still cries in the trap room", func(t *testing.T) {
		w, victim, _, output := deathTrapMudlogWorld(t)
		if _, err := w.MovePlayer(victim, "north"); err != nil {
			t.Fatal(err)
		}
		want := "Your blood freezes as you hear Victim's death cry.\r\n"
		if got := output.String(); got != want {
			t.Fatalf("witness = %q, want the rider's cry in the real room", got)
		}
	})

	t.Run("an absent mount never reaches death_cry", func(t *testing.T) {
		w, victim, _, output := deathTrapMudlogWorld(t)
		// C keeps the pre-entry mount pointer (src/act.movement.c:195) and
		// calls death_cry(mount) with whatever room it then holds; Go looks the
		// mount up in the rider's room, so an extracted mount yields no cry.
		victim.MountName = "a warhorse"
		victim.SetAffect(affMounted, true)

		if _, err := w.MovePlayer(victim, "north"); err != nil {
			t.Fatal(err)
		}
		if got := output.String(); got != "Your blood freezes as you hear Victim's death cry.\r\n" {
			t.Fatalf("witness = %q, want only the rider's cry", got)
		}
		if victim.GetMountName() != "" || victim.IsAffected(affMounted) {
			t.Fatal("the rider kept riding a mount that is not in the room")
		}
	})
}
