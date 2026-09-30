package session

import (
	"testing"
	"time"
)

// A bare blocking send parked on a full send channel is woken by CloseSend
// (the link-dead takeover) into an unrecovered "send on closed channel"
// panic — on the recover-less telnet input goroutine that kills the whole
// process (BUG-R2-C1-A3-H1). Every direct producer must use the guarded
// path: this drives the previously-bare producers against a full channel
// while a takeover close lands, in both interleavings (parked-then-closed
// and closed-then-send).
func TestGuardedSendsSurviveTakeoverCloseOnFullChannel(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Victim", 1001, true)

	// Fill every slot so an unguarded send parks and a guarded send drops.
	for i := 0; i < cap(s.send); i++ {
		s.send <- []byte("{}")
	}

	done := make(chan bool, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- false
			}
		}()
		// char_creation.go: both prompt producers that used to send bare.
		s.sendCharCreatePromptWithSecret("login_password", "Password: ", nil, true)
		s.sendStatsRollPrompt()
		// session_send.go: the error producer that leaned on recover().
		s.sendError("boom")
		done <- true
	}()

	// Let the goroutine reach its send (it parks pre-fix), then take over.
	time.Sleep(50 * time.Millisecond)
	s.CloseSend()

	select {
	case ok := <-done:
		if !ok {
			t.Fatal("a producer panicked on the closed send channel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a producer is parked on the full send channel")
	}

	// After the close every producer must no-op, not panic.
	s.sendCharCreatePromptWithSecret("login_password", "Password: ", nil, true)
	s.sendError("after close")
}
