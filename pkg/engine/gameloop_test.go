package engine

import (
	"context"
	"os"
	"slices"
	"sync/atomic"
	"testing"
	"time"
)

func TestGameLoopStopBeforeStartReturns(t *testing.T) {
	gl := NewGameLoop(GameLoopCallbacks{})

	done := make(chan struct{})
	go func() {
		gl.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Stop blocked when called before Start")
	}
}

func TestGameLoopStartStop(t *testing.T) {
	gl := NewGameLoop(GameLoopCallbacks{})

	gl.Start(context.Background())
	time.Sleep(150 * time.Millisecond)
	gl.Stop()

	if gl.Pulse.Load() == 0 {
		t.Fatal("expected pulse counter to advance")
	}
}

func TestGameLoopStartFreezesWhenDPClockIsSet(t *testing.T) {
	t.Setenv("DP_CLOCK", "1")
	gl := NewGameLoop(GameLoopCallbacks{})
	gl.tickerInterval = time.Millisecond

	gl.Start(context.Background())
	time.Sleep(20 * time.Millisecond)
	gl.Stop()

	if got := gl.Pulse.Load(); got != 0 {
		t.Fatalf("pulse advanced under DP_CLOCK: %d", got)
	}
}

func TestGameLoopPumpPulsesDispatchesCOrder(t *testing.T) {
	t.Setenv("DP_CLOCK", "1")
	var calls []string
	gl := NewGameLoop(GameLoopCallbacks{
		OnMobileActivity:  func() { calls = append(calls, "mobile") },
		OnRoomActivity:    func() { calls = append(calls, "room") },
		OnObjectActivity:  func() { calls = append(calls, "object") },
		OnPerformViolence: func() { calls = append(calls, "violence") },
	})

	if err := gl.PumpPulses(PULSE_MOBILE); err != nil {
		t.Fatal(err)
	}
	if got := gl.Pulse.Load(); got != PULSE_MOBILE {
		t.Fatalf("pulse = %d, want %d", got, PULSE_MOBILE)
	}
	want := []string{"violence", "mobile", "room", "object", "violence"}
	if !slices.Equal(calls, want) {
		t.Fatalf("callback order = %v, want %v", calls, want)
	}
}

func TestGameLoopPumpPulsesRequiresDPClock(t *testing.T) {
	if err := os.Unsetenv("DP_CLOCK"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Unsetenv("DP_CLOCK") })
	gl := NewGameLoop(GameLoopCallbacks{})
	if err := gl.PumpPulses(1); err == nil {
		t.Fatal("PumpPulses succeeded without DP_CLOCK")
	}
	if got := gl.Pulse.Load(); got != 0 {
		t.Fatalf("pulse advanced without DP_CLOCK: %d", got)
	}
}

func TestGameLoopRepeatedStopDoesNotPanic(t *testing.T) {
	gl := NewGameLoop(GameLoopCallbacks{})

	gl.Start(context.Background())
	time.Sleep(150 * time.Millisecond)
	gl.Stop()
	gl.Stop()
	gl.Stop()
}

func TestGameLoopStartIsIdempotent(t *testing.T) {
	gl := NewGameLoop(GameLoopCallbacks{})

	gl.Start(context.Background())
	gl.Start(context.Background())
	gl.Start(context.Background())

	time.Sleep(150 * time.Millisecond)
	gl.Stop()
}

// TestHeartbeatRecoversFromPanickingCallback is the DP-1019 regression: a panic
// in one heartbeat callback must not escape (which would kill the loop
// goroutine and the whole server), and must not prevent later callbacks in the
// same tick from running.
func TestHeartbeatRecoversFromPanickingCallback(t *testing.T) {
	ranAfterPanic := false
	gl := NewGameLoop(GameLoopCallbacks{
		OnEventProcess:   func() { panic("boom from a heartbeat callback") },
		OnExtractPending: func() { ranAfterPanic = true },
	})

	// Direct call — heartbeat must not propagate the panic.
	gl.heartbeat(1)

	if !ranAfterPanic {
		t.Error("callback after a panicking one did not run — per-callback recovery failed")
	}
}

// TestGameLoopSurvivesPanickingCallback drives the real ticker goroutine: with a
// callback that panics every tick, the loop must keep advancing pulses instead
// of dying on the first panic (DP-1019).
func TestGameLoopSurvivesPanickingCallback(t *testing.T) {
	var ticks atomic.Int64
	gl := NewGameLoop(GameLoopCallbacks{
		OnEventProcess: func() {
			ticks.Add(1)
			panic("boom every tick")
		},
	})

	gl.Start(context.Background())
	time.Sleep(350 * time.Millisecond)
	gl.Stop()

	if got := ticks.Load(); got < 2 {
		t.Fatalf("loop did not survive panics: callback ran %d times, want >= 2", got)
	}
	if gl.Pulse.Load() < 2 {
		t.Fatalf("pulse did not advance past the first panic: %d", gl.Pulse.Load())
	}
}

// TestGameLoopStopsOnContextCancel is the DP-892 behavior: canceling the context
// passed to Start drains the loop the same way Stop does — the goroutine exits
// (doneCh closes, so a subsequent Stop returns promptly) and the pulse stops
// advancing.
func TestGameLoopStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	gl := NewGameLoop(GameLoopCallbacks{})

	gl.Start(ctx)
	time.Sleep(150 * time.Millisecond)
	if gl.Pulse.Load() == 0 {
		t.Fatal("expected pulse counter to advance before cancel")
	}

	cancel()

	// The loop should exit on its own; Stop must then return without blocking.
	done := make(chan struct{})
	go func() {
		gl.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("loop did not stop after context cancellation")
	}

	// Pulse must be frozen after cancellation.
	after := gl.Pulse.Load()
	time.Sleep(150 * time.Millisecond)
	if gl.Pulse.Load() != after {
		t.Fatalf("pulse advanced after context cancel: %d -> %d", after, gl.Pulse.Load())
	}
}
