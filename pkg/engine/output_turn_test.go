package engine

import (
	"reflect"
	"testing"
)

func TestHeartbeatOutputTurnLiveAndPumpBoundaries(t *testing.T) {
	for _, mode := range []string{"live", "pump"} {
		t.Run(mode, func(t *testing.T) {
			var got []string
			gl := NewGameLoop(GameLoopCallbacks{
				OnBeginOutputTurn: func() { got = append(got, "begin") },
				OnEndOutputTurn:   func() { got = append(got, "end") },
				OnDrainInput:      func() { got = append(got, "input") },
				OnEventProcess:    func() { got = append(got, "work"); panic("callback control") },
				OnExtractPending:  func() { got = append(got, "extract") },
			})
			want := []string{"input", "begin", "work", "extract", "end"}
			if mode == "live" {
				gl.heartbeatTurn(1, true)
			} else {
				t.Setenv("DP_CLOCK", "1")
				if err := gl.PumpPulses(2); err != nil {
					t.Fatal(err)
				}
				want = []string{"begin", "input", "work", "extract", "input", "work", "extract", "end"}
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("output boundary: %v want %v", got, want)
			}
		})
	}
}
