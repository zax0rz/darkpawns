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
	var drainCount int64
	gl := NewGameLoop(GameLoopCallbacks{
		OnDrainInput:      func() { drainCount++ }, // fires every tick; asserted separately
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
	if got := drainCount; got != int64(PULSE_MOBILE) {
		t.Fatalf("OnDrainInput fired %d times, want %d (once per pulse)", got, PULSE_MOBILE)
	}
}

// TestGameLoopDrainBeforeViolence verifies fact 4 (comm.c:603): the per-pulse
// command drain (OnDrainInput) runs BEFORE perform_violence (OnPerformViolence)
// within a PULSE_VIOLENCE tick. This ordering is what lets a command that
// becomes eligible on a violence pulse act before that pulse's combat round.
//
// OnDrainInput fires every tick; OnPerformViolence fires only on the
// PULSE_VIOLENCE-th. Across PumpPulses(PULSE_VIOLENCE) we see PULSE_VIOLENCE
// drains then one violence. The load-bearing assertion is that a drain
// immediately precedes the violence on the violence pulse — i.e. the element
// right before "violence" in the order is "drain", not the other way around.
func TestGameLoopDrainBeforeViolence(t *testing.T) {
	t.Setenv("DP_CLOCK", "1")
	var order []string
	gl := NewGameLoop(GameLoopCallbacks{
		OnDrainInput:      func() { order = append(order, "drain") },
		OnPerformViolence: func() { order = append(order, "violence") },
	})

	if err := gl.PumpPulses(PULSE_VIOLENCE); err != nil {
		t.Fatal(err)
	}
	// PULSE_VIOLENCE drains (one per tick) + one violence on the final tick.
	wantLen := PULSE_VIOLENCE + 1
	if len(order) != wantLen {
		t.Fatalf("expected %d callbacks, got %d: %v", wantLen, len(order), order)
	}
	// Every tick drains first; violence only appears once, as the LAST entry,
	// preceded immediately by a drain.
	if order[len(order)-1] != "violence" {
		t.Fatalf("violence must be the last callback on a PULSE_VIOLENCE pump, got %v", order)
	}
	if order[len(order)-2] != "drain" {
		t.Fatalf("a drain must immediately precede violence, got %v", order)
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

func TestGameLoopStopContextTimesOutDuringBlockedCallback(t *testing.T) {
	callbackStarted := make(chan struct{})
	releaseCallback := make(chan struct{})
	gl := NewGameLoop(GameLoopCallbacks{
		OnDrainInput: func() {
			select {
			case <-callbackStarted:
			default:
				close(callbackStarted)
			}
			<-releaseCallback
		},
	})
	gl.tickerInterval = time.Millisecond
	gl.Start(context.Background())
	<-callbackStarted

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if err := gl.StopContext(ctx); err == nil {
		t.Fatal("StopContext returned nil while heartbeat callback was blocked")
	}

	close(releaseCallback)
	done := make(chan struct{})
	go func() {
		gl.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("game loop did not finish after blocked callback was released")
	}
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
