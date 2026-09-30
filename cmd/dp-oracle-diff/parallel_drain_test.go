package main

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/internal/oraclediff"
)

// slowConn is a fake connection whose reads stay quiet for a fixed wall
// time before returning.
type slowConn struct {
	delay time.Duration
}

func (c *slowConn) Send(string) error { return nil }
func (c *slowConn) ReadUntilQuiescent(time.Duration) (string, error) {
	time.Sleep(c.delay)
	return "", nil
}
func (c *slowConn) Close() error        { return nil }
func (c *slowConn) ObservedClose() bool { return false }

// TestDrainClientsDrainsInParallel proves the setup drain runs
// concurrently: the primary plus three peers, each quiet 300 ms, must
// finish well under the 900 ms a sequential drain would take.
func TestDrainClientsDrainsInParallel(t *testing.T) {
	if testing.Short() {
		t.Skip("wall-clock timing test")
	}
	primary := &slowConn{delay: 10 * time.Millisecond}
	peers := map[string]oraclediff.Conn{
		"one":   &slowConn{delay: 300 * time.Millisecond},
		"two":   &slowConn{delay: 300 * time.Millisecond},
		"three": &slowConn{delay: 300 * time.Millisecond},
	}

	start := time.Now()
	if err := drainClients(10*time.Millisecond, primary, peers); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	if elapsed >= 850*time.Millisecond {
		t.Fatalf("drainClients took %s; expected parallel (~310ms), not sequential (>=900ms)", elapsed)
	}
	t.Logf("drainClients elapsed: %s", elapsed)
}
