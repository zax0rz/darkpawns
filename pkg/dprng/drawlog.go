package dprng

import (
	"fmt"
	"io"
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
	drawLogOn              = os.Getenv("DP_DRAW_LOG") == "1"
	drawLogSink  io.Writer = os.Stderr
)

// When DP_DRAW_LOG_FILE is set, draws go to that file (truncated at start)
// instead of stderr. Harnesses that capture a subprocess's stderr into an
// unread buffer (cmd/dp-oracle-diff) swallow the log otherwise, so a file sink
// is the only way to recover the port's draw sequence.
func init() {
	if path := os.Getenv("DP_DRAW_LOG_FILE"); drawLogOn && path != "" {
		if f, err := os.Create(path); err == nil { // #nosec G304 G703 -- dev draw-parity tool; path is an operator-supplied env var, not request-derived
			drawLogSink = f
		}
	}
}

func drawLog(consumed int, method string, a, b, value int) {
	if !drawLogOn {
		return
	}
	drawLogMu.Lock()
	defer drawLogMu.Unlock()
	idx := drawLogIndex
	drawLogIndex += consumed
	_, _ = fmt.Fprintf(drawLogSink, "DRAW %d %d %s(%d,%d) = %d\n", idx, consumed, method, a, b, value)
}

// DrawLogIndex returns the number of draws consumed so far (testing aid).
func DrawLogIndex() int {
	drawLogMu.Lock()
	defer drawLogMu.Unlock()
	return drawLogIndex
}

// DrawLogEnabled reports whether draw logging is active.
func DrawLogEnabled() bool { return drawLogOn }
