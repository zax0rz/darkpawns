package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Recall fidelity tests — do_recall (act.other.c:1727-1748) with spell_recall
// (spells.c:124-163) inlined. Room vnums: 8162 Temple Infirmary (fresh
// Kir Drax'in char), 8004 mortal_start_room, 18201 kiroshi_start_room,
// 21258 alaozar_start_room (config.c).
func newRecallTestWorld(t *testing.T) (*World, map[string]*strings.Builder) {
	t.Helper()

	parsed := &parser.World{Rooms: []parser.Room{
		{VNum: 8162, Name: "Temple Infirmary", Zone: 80, Flags: []string{"28"}},
		{VNum: 8004, Name: "At the Temple Altar", Zone: 80},
		{VNum: 18201, Name: "Kir-Oshi Start", Zone: 182},
		{VNum: 21258, Name: "Alaozar Start", Zone: 212},
		{VNum: 9000, Name: "Magickal Place", Zone: 90, Flags: []string{"131072"}}, // ROOM_BFR (bit 17)
	}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	outputs := make(map[string]*strings.Builder)
	w.MessageSink = func(name string, msg []byte) {
		if b, ok := outputs[name]; ok {
			b.Write(msg)
		}
	}
	return w, outputs
}

func addRecallPlayer(t *testing.T, w *World, outputs map[string]*strings.Builder, id int, name string, room int) *Player {
	t.Helper()
	p := NewPlayer(id, name, room)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer(%s): %v", name, err)
	}
	outputs[name] = &strings.Builder{}
	return p
}

// Fresh L1 default-home char in 8162: recall relocates to mortal_start_room
// and the recaller sees ONLY the new room's look output — no gate text, no
// fabricated "You close your eyes and pray..."/"You are recalled!" lines.
func TestRecallFreshCharMovesAndSeesOnlyLook(t *testing.T) {
	w, outputs := newRecallTestWorld(t)
	recaller := addRecallPlayer(t, w, outputs, 1, "Recaller", 8162)
	addRecallPlayer(t, w, outputs, 2, "OldRoommate", 8162)
	addRecallPlayer(t, w, outputs, 3, "NewRoommate", 8004)

	w.ExecRecall(recaller, "")

	if got := recaller.GetRoomVNum(); got != MortalStartRoom {
		t.Fatalf("room after recall = %d, want %d", got, MortalStartRoom)
	}

	self := outputs["Recaller"].String()
	for _, fabricated := range []string{
		"You close your eyes and pray",
		"You are recalled!",
		"appears in the middle of the room", // TO_ROOM excludes the actor
		"This command is not available",
		"magickal place",
		"concentration is broken",
	} {
		if strings.Contains(self, fabricated) {
			t.Errorf("recaller output contains %q, full output %q", fabricated, self)
		}
	}
	if !strings.Contains(self, "At the Temple Altar") || !strings.Contains(self, "[ Exits:") {
		t.Errorf("recaller output should be only the new room's look, got %q", self)
	}

	if got := outputs["OldRoommate"].String(); got != "Recaller disappears.\r\n" {
		t.Errorf("old room broadcast = %q, want %q", got, "Recaller disappears.\r\n")
	}
	if got := outputs["NewRoommate"].String(); got != "Recaller appears in the middle of the room.\r\n" {
		t.Errorf("new room broadcast = %q, want %q", got, "Recaller appears in the middle of the room.\r\n")
	}
}

// The recall target is hometown-dependent (GET_HOME split, same as quit's
// isokquit): 2 → kiroshi_start_room, 3 → alaozar_start_room, else mortal.
func TestRecallHometownTargets(t *testing.T) {
	tests := []struct {
		name     string
		hometown int
		want     int
	}{
		{name: "Kir Drax'in", hometown: 1, want: 8004},
		{name: "Kir-Oshi", hometown: 2, want: 18201},
		{name: "Alaozar", hometown: 3, want: 21258},
		{name: "unset", hometown: 0, want: 8004},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, outputs := newRecallTestWorld(t)
			p := addRecallPlayer(t, w, outputs, 1, "Recaller", 8162)
			p.Hometown = tt.hometown

			w.ExecRecall(p, "")

			if got := p.GetRoomVNum(); got != tt.want {
				t.Errorf("hometown %d: room after recall = %d, want %d", tt.hometown, got, tt.want)
			}
		})
	}
}

// Gates in C's order: level > 5, ROOM_BFR, fighting — with C's exact bytes
// (the fighting message has no trailing CRLF in do_recall).
func TestRecallGates(t *testing.T) {
	t.Run("level too high", func(t *testing.T) {
		w, outputs := newRecallTestWorld(t)
		p := addRecallPlayer(t, w, outputs, 1, "Recaller", 8162)
		p.SetLevel(6)

		w.ExecRecall(p, "")

		if got := outputs["Recaller"].String(); got != "This command is not available for someone of your experience!\r\n" {
			t.Errorf("gate message = %q", got)
		}
		if got := p.GetRoomVNum(); got != 8162 {
			t.Errorf("gated recall moved player to %d", got)
		}
	})

	t.Run("bfr room", func(t *testing.T) {
		w, outputs := newRecallTestWorld(t)
		p := addRecallPlayer(t, w, outputs, 1, "Recaller", 9000)

		w.ExecRecall(p, "")

		if got := outputs["Recaller"].String(); got != "You can't recall from this magickal place.\r\n" {
			t.Errorf("gate message = %q", got)
		}
		if got := p.GetRoomVNum(); got != 9000 {
			t.Errorf("gated recall moved player to %d", got)
		}
	})

	t.Run("fighting", func(t *testing.T) {
		w, outputs := newRecallTestWorld(t)
		p := addRecallPlayer(t, w, outputs, 1, "Recaller", 8162)
		p.SetFighting("Target")

		w.ExecRecall(p, "")

		if got := outputs["Recaller"].String(); got != "Your concentration is broken by your fighting!" {
			t.Errorf("gate message = %q (no trailing CRLF in C)", got)
		}
		if got := p.GetRoomVNum(); got != 8162 {
			t.Errorf("gated recall moved player to %d", got)
		}
	})
}

// A sleeping recaller moves but dreams instead of seeing the room (spells.c).
func TestRecallSleepingDreams(t *testing.T) {
	w, outputs := newRecallTestWorld(t)
	p := addRecallPlayer(t, w, outputs, 1, "Recaller", 8162)
	p.SetPosition(PosSleeping)

	w.ExecRecall(p, "")

	if got := p.GetRoomVNum(); got != MortalStartRoom {
		t.Fatalf("room after recall = %d, want %d", got, MortalStartRoom)
	}
	if got := outputs["Recaller"].String(); got != "You have a strange dream about falling..\r\n" {
		t.Errorf("sleeping recaller output = %q", got)
	}
}
