package session

import (
	"reflect"
	"testing"

	"golang.org/x/time/rate"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// C's command_interpreter draws number(0,3) at the top of every line it is
// handed — before skip_spaces and the empty-argument return
// (src/interpreter.c:889-895) — so a bare RETURN burns exactly one draw and
// clears AFF_HIDE on 0 even though it prints nothing. Go returned before the
// draw, which left the two streams one Number(0,3) apart at every empty line.
// Taught by the seed-1 draw-parity corpus: prompt-playing-depth,
// prompt-infobar-status-depth and socials-command-depth matched byte-for-byte
// while their RNG streams drifted by one draw per empty probe line.
func TestExecuteCommandEmptyLineConsumesLeadingDraw(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"bare return", ""},
		{"whitespace only", "   "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := makeTestManager(t)
			s := makeCommandTestSession(t, m, "EmptyDraw", 1, 1001)

			const seed = 1
			dprng.ResetStream(seed)
			draws := captureCommandDraws(t)
			if err := ExecuteCommand(s, tc.input, nil); err != nil {
				t.Fatal(err)
			}

			wantDraws := []commandDrawRange{{from: 0, to: 3}}
			if !reflect.DeepEqual(*draws, wantDraws) {
				t.Fatalf("empty-line command draws = %+v, want %+v", *draws, wantDraws)
			}
			wantStream := dprng.New(seed)
			wantStream.Number(0, 3)
			if got, want := dprng.Next(), wantStream.Next(); got != want {
				t.Fatalf("stream after empty line = %d, want %d (exactly one draw)", got, want)
			}
		})
	}
}

// The transport regression: TerminalLine's empty-line case must reach the
// command interpreter, not merely refresh the prompt (src/comm.c:628 hands a
// bare RETURN to command_interpreter like any other dequeued line).
func TestTerminalLineBareReturnConsumesLeadingDraw(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "EmptyTerm", 1001, true)
	s.terminalNamed = true
	s.limiter = rate.NewLimiter(rate.Inf, 1000)
	registerTestSession(t, m, s, "EmptyTerm")
	_ = captureWire(s) // discard the entry state

	const seed = 1
	dprng.ResetStream(seed)
	draws := captureCommandDraws(t)
	s.TerminalLine("")

	wantDraws := []commandDrawRange{{from: 0, to: 3}}
	if !reflect.DeepEqual(*draws, wantDraws) {
		t.Fatalf("bare RETURN command draws = %+v, want %+v", *draws, wantDraws)
	}
	wantStream := dprng.New(seed)
	wantStream.Number(0, 3)
	if got, want := dprng.Next(), wantStream.Next(); got != want {
		t.Fatalf("stream after bare RETURN = %d, want %d (exactly one draw)", got, want)
	}
}

// Seed 5's first number(0,3) is 0, so a bare RETURN must clear AFF_HIDE the
// same way any other command does (src/interpreter.c:889-890).
func TestTerminalLineBareReturnClearsHide(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "EmptyHidden", 1001, true)
	s.terminalNamed = true
	s.limiter = rate.NewLimiter(rate.Inf, 1000)
	registerTestSession(t, m, s, "EmptyHidden")
	_ = captureWire(s) // discard the entry state
	s.player.SetAffect(game.AffHide, true)

	dprng.ResetStream(5)
	s.TerminalLine("")

	if s.player.IsAffected(game.AffHide) {
		t.Fatal("AFF_HIDE survived a bare RETURN whose command roll was zero")
	}
}
