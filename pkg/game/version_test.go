package game

import (
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestFormatCompileTimeMatchesC pins the intended output of
// load_compile_time() (src/version.c:52-60): long month, day without zero
// padding, comma, four-digit year, three spaces, zero-padded 24-hour HH:MM.
// C's sprintf calls read and write the same buffer, which is undefined
// behaviour, so the layout — not the UB — is the contract (R1a).
func TestFormatCompileTimeMatchesC(t *testing.T) {
	tests := []struct {
		name string
		when time.Time
		want string
	}{
		{"two-digit hour and day", time.Date(2026, 2, 11, 19, 34, 0, 0, time.UTC), "February 11, 2026   19:34"},
		{"single-digit day", time.Date(2026, 10, 5, 9, 7, 0, 0, time.UTC), "October 5, 2026   09:07"},
		{"midnight keeps two digits", time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC), "December 31, 2026   00:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatCompileTime(tt.when); got != tt.want {
				t.Fatalf("formatCompileTime = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestLoadCompileTimeUsesTheBootClock pins load_compile_time()'s input: C sets
// compile_time = time(0) at boot, so the string must be the real boot clock
// formatted in local time, not a build stamp.
func TestLoadCompileTimeUsesTheBootClock(t *testing.T) {
	restoreCompileTimeClock(t)

	// A fixed local instant in 2000; no run of this test happens in 2000.
	when := time.Date(2000, 1, 2, 3, 4, 0, 0, time.Local)
	compileTimeNow = func() time.Time { return when }
	loadCompileTime()

	if got, want := CompileTimeString(), "January 2, 2000   03:04"; got != want {
		t.Fatalf("CompileTimeString = %q, want %q", got, want)
	}
}

// TestLoadCompileTimeIgnoresThePinnedGameClock pins the R1 decision: C's
// compile_time is time(0), so DP_FIXED_TIME (nowFunc) must not leak into the
// player-visible stamp. If it did, the port would print a date the C oracle
// never prints.
func TestLoadCompileTimeIgnoresThePinnedGameClock(t *testing.T) {
	restoreCompileTimeClock(t)
	previousNow := nowFunc
	t.Cleanup(func() { nowFunc = previousNow })

	nowFunc = func() int64 { return 946782245 } // 2000-01-02, pinned game clock
	when := time.Date(2026, 10, 21, 9, 27, 0, 0, time.Local)
	compileTimeNow = func() time.Time { return when }
	loadCompileTime()

	if got, want := CompileTimeString(), "October 21, 2026   09:27"; got != want {
		t.Fatalf("CompileTimeString = %q, want %q from the boot clock, not DP_FIXED_TIME", got, want)
	}
}

// restoreCompileTimeClock restores the package clock seams a test overrides.
func restoreCompileTimeClock(t *testing.T) {
	t.Helper()
	previousNow := compileTimeNow
	previousStr := compileTimeStr
	t.Cleanup(func() {
		compileTimeNow = previousNow
		compileTimeStr = previousStr
	})
}

// TestNewWorldStampsTheCompileTime pins the boot hook: C's boot_db calls
// load_compile_time() before boot_world (src/db.c:313), so constructing a
// world must leave the string set.
func TestNewWorldStampsTheCompileTime(t *testing.T) {
	restoreCompileTimeClock(t)

	compileTimeNow = func() time.Time { return time.Date(2000, 1, 2, 3, 4, 0, 0, time.Local) }
	compileTimeStr = ""

	w, err := NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	if got := CompileTimeString(); got == "" || !strings.Contains(got, "2000") {
		t.Fatalf("CompileTimeString after NewWorld = %q, want the boot stamp", got)
	}
}
