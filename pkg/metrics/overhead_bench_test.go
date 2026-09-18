package metrics

import (
	"testing"
	"time"
)

// These pin the cost of the calls wired into hot paths in #1492: every command
// dispatch, every damage application, every combat round. The wiring shipped
// unmeasured, which is exactly the habit that put dead infrastructure in this
// repo — measuring it after the fact is the cheap half of the correction.
func BenchmarkCommandProcessed(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CommandProcessed("movement", 900*time.Microsecond)
	}
}

// The labelled counter is the one worth watching: WithLabelValues is a map
// lookup under a lock, not a bare atomic add, and it fires on every hit any
// player or mob takes, including hunger and thirst ticks.
func BenchmarkDamageTaken(b *testing.B) {
	for i := 0; i < b.N; i++ {
		DamageTaken("mob", 7)
	}
}

func BenchmarkCombatRound(b *testing.B) {
	for i := 0; i < b.N; i++ {
		CombatRound()
	}
}

func BenchmarkConnectionOpenClose(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ConnectionOpened()
		ConnectionClosed()
	}
}
