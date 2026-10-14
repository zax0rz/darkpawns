package oraclediff

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// DrawCall is one line of a draw log: the C oracle's seam (DP_C_DRAW_LOG_FILE,
// dp-oracle-seam src/utils.c) and the Go port's dprng draw log
// (DP_DRAW_LOG_FILE, pkg/dprng/drawlog.go) both write
//
//	DRAW <idx> <consumed> <Method>(<a>,<b>) = <value>
//
// where <consumed> is the number of raw stream values the call used.
type DrawCall struct {
	Method   string
	A, B     int
	Value    int
	Consumed int
}

func (c DrawCall) String() string {
	return fmt.Sprintf("%s(%d,%d) = %d", c.Method, c.A, c.B, c.Value)
}

var drawLine = regexp.MustCompile(`^DRAW (-?\d+) (\d+) (\w+)\((-?\d+),(-?\d+)\) = (-?\d+)$`)

// ReadDrawLog parses a draw log. A missing file is an empty log: a run that
// never draws never creates one.
func ReadDrawLog(path string) ([]DrawCall, error) {
	f, err := os.Open(path) // #nosec G304 -- dev oracle-diff harness; path is the harness's own per-run file
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var calls []DrawCall
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		m := drawLine.FindStringSubmatch(strings.TrimSpace(scanner.Text()))
		if m == nil {
			continue
		}
		consumed, _ := strconv.Atoi(m[2])
		a, _ := strconv.Atoi(m[4])
		b, _ := strconv.Atoi(m[5])
		v, _ := strconv.Atoi(m[6])
		calls = append(calls, DrawCall{Method: m[3], A: a, B: b, Value: v, Consumed: consumed})
	}
	return calls, scanner.Err()
}

// CountDrawLines returns the number of draw entries currently in a log, for
// recording phase and step boundaries while a run is in progress.
func CountDrawLines(path string) int {
	calls, err := ReadDrawLog(path)
	if err != nil {
		return 0
	}
	return len(calls)
}

// normalizeDraw folds the two spellings of the same single draw together: C rolls
// dice(1,n) where Go may call Number(1,n), and both consume one value.
func normalizeDraw(c DrawCall) DrawCall {
	if c.Method == "Dice" && c.A == 1 {
		c.Method = "Number"
		c.A = 1
	}
	return c
}

// DrawComparison is the result of comparing two draw logs.
type DrawComparison struct {
	Match bool
	// CEntry and GoEntry index the first mismatching entries in each log
	// (entry positions, not stream positions). They equal the log lengths when
	// one log simply ends early.
	CEntry, GoEntry int
	// Stream is the number of raw stream values both sides agreed on before
	// the mismatch.
	Stream          int
	CCount, GoCount int
}

// CompareDraws walks both logs in order. Beyond exact equality it accepts
// C's dice(k,n) against k consecutive Go Number(1,n) calls (or the reverse)
// when the sums agree, because both consume the same k stream values.
func CompareDraws(c, g []DrawCall) DrawComparison {
	r := DrawComparison{CCount: len(c), GoCount: len(g)}
	ci, gi := 0, 0
	for ci < len(c) && gi < len(g) {
		cc, gc := normalizeDraw(c[ci]), normalizeDraw(g[gi])
		if cc == gc {
			r.Stream += max(cc.Consumed, 1)
			ci++
			gi++
			continue
		}
		if k, ok := expandsTo(cc, g[gi:]); ok {
			r.Stream += cc.Consumed
			ci++
			gi += k
			continue
		}
		if k, ok := expandsTo(gc, c[ci:]); ok {
			r.Stream += gc.Consumed
			gi++
			ci += k
			continue
		}
		r.CEntry, r.GoEntry = ci, gi
		return r
	}
	r.CEntry, r.GoEntry = ci, gi
	r.Match = ci == len(c) && gi == len(g)
	return r
}

// expandsTo reports whether dice(k,n) equals the next k Number(1,n) calls in
// rest, returning k.
func expandsTo(dice DrawCall, rest []DrawCall) (int, bool) {
	if dice.Method != "Dice" || dice.A < 2 || len(rest) < dice.A {
		return 0, false
	}
	sum := 0
	for _, n := range rest[:dice.A] {
		n = normalizeDraw(n)
		if n.Method != "Number" || n.A != 1 || n.B != dice.B {
			return 0, false
		}
		sum += n.Value
	}
	return dice.A, sum == dice.Value
}

// DrawBoundary records how many draw entries a side's log held when a phase or
// probe step finished.
type DrawBoundary struct {
	Label string
	Count int
}

// StepFor names the phase or step during which entry index idx was drawn: the
// first boundary whose count exceeds idx. Entries after the last boundary
// belong to the teardown.
func StepFor(boundaries []DrawBoundary, idx int) string {
	for _, b := range boundaries {
		if idx < b.Count {
			return b.Label
		}
	}
	return "after the last recorded step"
}
