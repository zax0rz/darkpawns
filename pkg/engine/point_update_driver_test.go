package engine

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"
)

// src/comm.c:825-830: exactly one point_update every 630 pulses,
// after weather and affects and before hunt_items/player-file flush.
func TestHourlyPointUpdateSingleDriverModes(t *testing.T) {
	for _, mode := range []string{"live", "pumped"} {
		t.Run(mode, func(t *testing.T) {
			if mode == "pumped" {
				t.Setenv("DP_CLOCK", "1")
			} else {
				t.Setenv("DP_CLOCK", "")
				if err := os.Unsetenv("DP_CLOCK"); err != nil {
					t.Fatal(err)
				}
			}
			var calls []string
			cb := GameLoopCallbacks{
				OnWeatherAndTime:  func() { calls = append(calls, "weather") },
				OnAffectUpdate:    func() { calls = append(calls, "affect") },
				OnPointUpdate:     func() { calls = append(calls, "point") },
				OnHuntItems:       func() { calls = append(calls, "hunt") },
				OnFlushPlayerFile: func() { calls = append(calls, "file") },
			}
			hour := SECS_PER_MUD_HOUR * PASSES_PER_SEC
			if hour != 630 {
				t.Fatalf("hour=%d pulses, want C 630", hour)
			}
			if mode == "pumped" {
				gl := NewGameLoop(cb)
				if err := gl.PumpPulses(hour - 1); err != nil {
					t.Fatal(err)
				}
				if len(calls) != 0 {
					t.Fatalf("hourly calls before boundary: %v", calls)
				}
				if err := gl.PumpPulses(1); err != nil {
					t.Fatal(err)
				}
				if err := gl.PumpPulses(hour - 1); err != nil {
					t.Fatal(err)
				}
				if got := len(calls); got != 5 {
					t.Fatalf("calls before second hour=%d, want 5: %v", got, calls)
				}
				if err := gl.PumpPulses(1); err != nil {
					t.Fatal(err)
				}
			} else {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				finished := make(chan struct{})
				var gl *GameLoop
				cb.OnFlushOutput = func() {
					if gl.Pulse.Load() == int64(2*hour+1) {
						cancel()
						close(finished)
					}
				}
				gl = NewGameLoop(cb)
				gl.Pulse.Store(int64(hour - 1))
				gl.tickerInterval = time.Millisecond
				gl.Start(ctx)
				select {
				case <-finished:
				case <-time.After(5 * time.Second):
					gl.Stop()
					t.Fatal("live heartbeat did not reach observation boundary")
				}
				gl.Stop()
			}
			want := []string{"weather", "affect", "point", "hunt", "file", "weather", "affect", "point", "hunt", "file"}
			if !slices.Equal(calls, want) {
				t.Fatalf("hourly driver count/order=%v, want %v", calls, want)
			}
		})
	}
}
