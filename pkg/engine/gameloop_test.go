package engine

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestGameLoopStopPreventsFurtherCallbacks(t *testing.T) {
	var callbacks atomic.Int64
	gl := NewGameLoop(GameLoopCallbacks{
		OnEventProcess: func() {
			callbacks.Add(1)
		},
	})
	gl.tickerInterval = time.Millisecond

	gl.Start()
	waitUntil(t, func() bool {
		return callbacks.Load() > 0
	})

	gl.Stop()
	stoppedAt := callbacks.Load()
	time.Sleep(5 * time.Millisecond)

	if got := callbacks.Load(); got != stoppedAt {
		t.Fatalf("callback count changed after Stop: before=%d after=%d", stoppedAt, got)
	}
}

func TestGameLoopStopIsIdempotent(t *testing.T) {
	gl := NewGameLoop(GameLoopCallbacks{})
	gl.tickerInterval = time.Millisecond
	gl.Start()

	gl.Stop()
	gl.Stop()
}

func waitUntil(t *testing.T, condition func() bool) {
	t.Helper()

	deadline := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}
