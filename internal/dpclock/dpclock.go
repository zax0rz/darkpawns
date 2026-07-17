// Package dpclock exposes the deterministic-clock feature gate shared by the
// server's real-time game-state drivers.
package dpclock

import "os"

// Frozen reports whether wall-clock-driven game pulses must remain stopped.
// Presence, rather than value, mirrors C's getenv("DP_CLOCK") seam.
func Frozen() bool {
	_, ok := os.LookupEnv("DP_CLOCK")
	return ok
}
