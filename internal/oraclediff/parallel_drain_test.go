package oraclediff

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// slowConn is a fake connection whose reads stay quiet for a fixed wall
// time before returning, like a real connection waiting out its quiescence
// window on silence.
type slowConn struct {
	mu     sync.Mutex
	sent   []string
	delay  time.Duration
	closed bool
}

func (c *slowConn) Send(line string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, line)
	return nil
}

func (c *slowConn) ReadUntilQuiescent(time.Duration) (string, error) {
	time.Sleep(c.delay)
	return "", nil
}

func (c *slowConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *slowConn) ObservedClose() bool { return false }

// TestRunAudienceProbeDrainsPeersInParallel proves the per-step peer drain
// runs concurrently: one step with three peers, each quiet for 300 ms,
// must finish well under the 900 ms a sequential drain would take.
func TestRunAudienceProbeDrainsPeersInParallel(t *testing.T) {
	if testing.Short() {
		t.Skip("wall-clock timing test")
	}
	actor := &slowConn{delay: 10 * time.Millisecond}
	peers := map[string]Conn{
		"alpha": &slowConn{delay: 300 * time.Millisecond},
		"beta":  &slowConn{delay: 300 * time.Millisecond},
		"gamma": &slowConn{delay: 300 * time.Millisecond},
	}

	start := time.Now()
	blocks, err := RunAudienceProbe(actor, peers, []string{"look"}, 10*time.Millisecond)
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	// A sequential drain stacks the three windows: >= 900ms. A parallel
	// drain overlaps them: ~300ms plus the actor's own window. The bound
	// is generous against scheduler noise but far under the sequential floor.
	if elapsed >= 850*time.Millisecond {
		t.Fatalf("peer drain took %s; expected parallel (~300ms), not sequential (>=900ms)", elapsed)
	}
	if len(blocks) != 4 { // actor + three peers
		t.Fatalf("blocks = %d, want 4 (actor + 3 peers)", len(blocks))
	}
	t.Logf("drain elapsed: %s (sequential floor 900ms)", elapsed)
}

// TestRunAudienceProbePeerBlockOrderUnchanged proves the collected block
// order is the sequential loop's: the actor's block first, then the peers
// in sorted-name order, regardless of which peer's read finishes first.
func TestRunAudienceProbePeerBlockOrderUnchanged(t *testing.T) {
	// gamma returns fastest, alpha slowest; the order must still be
	// actor, alpha, beta, gamma.
	actor := &slowConn{delay: 10 * time.Millisecond}
	peers := map[string]Conn{
		"alpha": &slowConn{delay: 120 * time.Millisecond},
		"beta":  &slowConn{delay: 60 * time.Millisecond},
		"gamma": &slowConn{delay: 5 * time.Millisecond},
	}

	blocks, err := RunAudienceProbe(actor, peers, []string{"look"}, 10*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"actor", "alpha", "beta", "gamma"}
	if len(blocks) != len(want) {
		t.Fatalf("blocks = %d, want %d", len(blocks), len(want))
	}
	for i, name := range want {
		if blocks[i].Audience != name {
			t.Fatalf("block %d audience = %q, want %q (sorted-name order must not follow read completion)", i, blocks[i].Audience, name)
		}
		if blocks[i].Command != "look" {
			t.Fatalf("block %d command = %q, want look", i, blocks[i].Command)
		}
	}
}

// Earlier successful blocks and sorted error precedence match sequential reads.
func TestRunAudienceProbeRetainsBlocksBeforePeerError(t *testing.T) {
	failure := errors.New("read failed")
	peers := map[string]Conn{
		"alpha": &resultConn{slowConn: slowConn{delay: 30 * time.Millisecond}, output: "alpha output"},
		"beta":  &resultConn{output: "beta partial", err: failure},
		"gamma": &resultConn{err: errors.New("later peer failed")},
	}
	blocks, err := RunAudienceProbe(&slowConn{}, peers, []string{"look"}, time.Millisecond)
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), "read beta after") || !strings.Contains(err.Error(), "beta partial") {
		t.Fatalf("error = %v, want beta failure and partial output", err)
	}
	if len(blocks) != 2 || blocks[0].Audience != "actor" || blocks[1].Audience != "alpha" || blocks[1].Output != "alpha output" {
		t.Fatalf("blocks = %+v, want actor and successful alpha", blocks)
	}
}

type resultConn struct {
	slowConn
	output string
	err    error
}

func (c *resultConn) ReadUntilQuiescent(time.Duration) (string, error) {
	time.Sleep(c.delay)
	return c.output, c.err
}
