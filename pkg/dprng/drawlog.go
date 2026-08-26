package dprng

import (
	"fmt"
	"os"
	"sync"
)

// Draw logging seam (DP_DRAW_LOG=1): appends one line per package-level
// consumer draw to stderr in the form
//
//	DRAW <index> <consumed> <method>(<a>,<b>) = <value>
//
// <index> is the absolute draw index since process start (monotonic across
// Number/Uniform/Dice); <consumed> is how many raw draws the call burned
// (Dice(n,s) consumes n). This is an R3 debugging aid for draw-parity
// investigations: the absolute index lets offline toolry replay the seeded
// stream draw-for-draw and compare against the C oracle's observed values,
// pinpointing hidden extra/missing upstream draws. Off by default; it never
// touches the stream.
var (
	drawLogMu    sync.Mutex
	drawLogIndex int
	drawLogOn    = os.Getenv("DP_DRAW_LOG") == "1"
)

func drawLog(consumed int, method string, a, b, value int) {
	if !drawLogOn {
		return
	}
	drawLogMu.Lock()
	defer drawLogMu.Unlock()
	idx := drawLogIndex
	drawLogIndex += consumed
	fmt.Fprintf(os.Stderr, "DRAW %d %d %s(%d,%d) = %d\n", idx, consumed, method, a, b, value)
}

// DrawLogIndex returns the number of draws consumed so far (testing aid).
func DrawLogIndex() int {
	drawLogMu.Lock()
	defer drawLogMu.Unlock()
	return drawLogIndex
}

// DrawLogEnabled reports whether draw logging is active.
func DrawLogEnabled() bool { return drawLogOn }
