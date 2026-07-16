package session

import (
	"reflect"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/game"
)

type commandDrawRange struct {
	from int
	to   int
}

func captureCommandDraws(t *testing.T) *[]commandDrawRange {
	t.Helper()
	draws := make([]commandDrawRange, 0, 1)
	previous := commandNumber
	commandNumber = func(from, to int) int {
		draws = append(draws, commandDrawRange{from: from, to: to})
		return dprng.Number(from, to)
	}
	t.Cleanup(func() { commandNumber = previous })
	return &draws
}

func TestExecuteCommandConsumesOneLeadingDraw(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "DrawTester", 1, 1001)

	const seed = 1
	dprng.ResetStream(seed)
	draws := captureCommandDraws(t)
	if err := ExecuteCommand(s, "not-a-command", nil); err != nil {
		t.Fatal(err)
	}

	wantDraws := []commandDrawRange{{from: 0, to: 3}}
	if !reflect.DeepEqual(*draws, wantDraws) {
		t.Fatalf("command draws = %+v, want %+v", *draws, wantDraws)
	}
	wantStream := dprng.New(seed)
	wantStream.Number(0, 3)
	if got, want := dprng.Next(), wantStream.Next(); got != want {
		t.Fatalf("stream after command = %d, want %d (exactly one draw)", got, want)
	}
}

func TestExecuteCommandHideClearRoll(t *testing.T) {
	t.Run("zero clears hide", func(t *testing.T) {
		m := makeTestManager(t)
		s := makeCommandTestSession(t, m, "Revealed", 1, 1001)
		s.player.SetAffect(game.AffHide, true)

		// Seed 5's first number(0,3) is 0.
		dprng.ResetStream(5)
		if err := ExecuteCommand(s, "not-a-command", nil); err != nil {
			t.Fatal(err)
		}
		if s.player.IsAffected(game.AffHide) {
			t.Fatal("AFF_HIDE remained set after a zero command roll")
		}
	})

	t.Run("nonzero preserves hide", func(t *testing.T) {
		m := makeTestManager(t)
		s := makeCommandTestSession(t, m, "StillHidden", 1, 1001)
		s.player.SetAffect(game.AffHide, true)

		// Seed 1's first number(0,3) is nonzero.
		dprng.ResetStream(1)
		if err := ExecuteCommand(s, "not-a-command", nil); err != nil {
			t.Fatal(err)
		}
		if !s.player.IsAffected(game.AffHide) {
			t.Fatal("AFF_HIDE cleared after a nonzero command roll")
		}
	})
}

func TestExecuteCommandBeforePlayingConsumesNoDraw(t *testing.T) {
	m := makeTestManager(t)
	s := m.NewSession()

	const seed = 1
	dprng.ResetStream(seed)
	draws := captureCommandDraws(t)
	if err := ExecuteCommand(s, "not-a-command", nil); err != nil {
		t.Fatal(err)
	}

	if len(*draws) != 0 {
		t.Fatalf("pre-playing command consumed draws: %+v", *draws)
	}
	wantStream := dprng.New(seed)
	if got, want := dprng.Next(), wantStream.Next(); got != want {
		t.Fatalf("pre-playing stream = %d, want untouched %d", got, want)
	}
}
