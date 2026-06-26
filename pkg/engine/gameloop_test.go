package engine

import (
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

	gl.Start()
	time.Sleep(150 * time.Millisecond)
	gl.Stop()

	if gl.Pulse.Load() == 0 {
		t.Fatal("expected pulse counter to advance")
	}
}

func TestGameLoopRepeatedStopDoesNotPanic(t *testing.T) {
	gl := NewGameLoop(GameLoopCallbacks{})

	gl.Start()
	time.Sleep(150 * time.Millisecond)
	gl.Stop()
	gl.Stop()
	gl.Stop()
}

func TestGameLoopStartIsIdempotent(t *testing.T) {
	gl := NewGameLoop(GameLoopCallbacks{})

	gl.Start()
	gl.Start()
	gl.Start()

	time.Sleep(150 * time.Millisecond)
	gl.Stop()
}
