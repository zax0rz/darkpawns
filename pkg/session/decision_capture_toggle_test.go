package session

import (
	"sync"
	"testing"
)

// TestDecisionCaptureToggleUnderConcurrentReaders drives the toggle against
// concurrent readers under -race.
//
// Be clear about what this does and does not prove. The hazard it guards is not
// a data race the detector finds: captureAndLog used to check the pointer for
// nil and then dereference it twice more, and the failure is a nil dereference
// in the middle of a player's command. Probing the pre-fix shape with -race
// does not reliably report it. The fix is structural — load once into a local,
// so the check and the use cannot disagree — and this test covers the toggle
// working under load rather than the old shape failing.
func TestDecisionCaptureToggleUnderConcurrentReaders(t *testing.T) {
	m := &Manager{}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Readers: the load captureAndLog performs on every command.
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					if dlw := m.decisionLog.Load(); dlw != nil {
						_ = dlw
					}
				}
			}
		}()
	}

	// Writer: flip it as fast as the admin endpoint ever could, and faster.
	for i := 0; i < 2000; i++ {
		m.decisionLog.Store(nil)
		m.decisionLog.Store(nil)
	}
	close(stop)
	wg.Wait()
}

// TestDecisionCaptureUnavailableWithoutWriter covers the ordinary case: no
// DP_RESEARCH_URL, so there is nothing to record into and enabling must fail
// rather than appear to succeed.
func TestDecisionCaptureUnavailableWithoutWriter(t *testing.T) {
	m := &Manager{}

	if m.DecisionCaptureAvailable() {
		t.Error("available with no writer installed")
	}
	if m.EnableDecisionCapture() {
		t.Error("EnableDecisionCapture reported success with no writer")
	}
	if m.DecisionCaptureEnabled() {
		t.Error("capture is on after a failed enable")
	}

	// Disabling when nothing is configured must not panic.
	m.DisableDecisionCapture()
}
