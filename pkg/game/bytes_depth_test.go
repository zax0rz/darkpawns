package game

// bytes_depth_test.go — byte-level fidelity for PR 3 (bytes): the oracle
// normalizer folds \r\n against \n\r, so neither difference below is
// visible to a census (R1b); direct raw-byte asserts are the proof.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestBytesDraculaMesmerizedTerminator pins buf3's LF CR ending
// (src/spec_procs.c:1815) against the port's former CRLF sendToChar.
func TestBytesDraculaMesmerizedTerminator(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "X", Zone: 1},
			{VNum: 18101, Name: "Y", Zone: 1},
		},
		Mobs: []parser.Mob{{
			VNum: 7903, Keywords: "dracula vlad", ShortDesc: "Vlad the dracula",
			Level: 30, Position: 8, DefaultPos: 8, HP: parser.DiceRoll{Plus: 1000},
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }

	ch := NewPlayer(1, "Tester", 1001)
	if err := w.AddPlayer(ch); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	dracula, err := w.SpawnMob(7903, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	if !specDracula(w, ch, dracula, "look", "dracula") {
		t.Fatal("dracula look was not intercepted")
	}
	const want = "You feel mesmerized... your will weakens.\n\r"
	got := out["Tester"]
	if !strings.Contains(got, want) {
		t.Fatalf("sink = %q, want the bare LF CR send %q", got, want)
	}
	if bad := strings.Contains(got, "weakens.\r\n"); bad {
		t.Fatalf("sink = %q, must not also carry the CRLF terminator", got)
	}
}

// TestBytesSuckInDizzyBreaks pins spec_procs.c:2291's two leading and two
// trailing CRLFs against the doubled port literal.
