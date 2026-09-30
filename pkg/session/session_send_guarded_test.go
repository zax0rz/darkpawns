package session

import (
	"testing"
	"time"
)

// A bare blocking send parked on a full send channel is woken by CloseSend
// (the link-dead takeover) into an unrecovered "send on closed channel"
// panic — on the recover-less telnet input goroutine that kills the whole
// process (BUG-R2-C1-A3-H1). Every direct producer must use the guarded
// path. One subtest per producer: fill the channel, start only that
// producer, land the takeover close, and assert no panic and no park —
// then prove the closed-channel branch no-ops by producing once more.
func TestGuardedSendsSurviveTakeoverCloseOnFullChannel(t *testing.T) {
	cases := []struct {
		name    string
		prepare func(t *testing.T, s *Session) func()
	}{
		{"sendObservation", func(t *testing.T, s *Session) func() {
			// The look path the attack parks on: verify the render actually
			// carries a room before the subtest relies on it reaching the send.
			res := s.manager.world.DoLookRoom(s.player, false)
			if res.Room == nil {
				t.Fatal("DoLookRoom returned no room; observation send would be skipped")
			}
			return func() { _ = s.sendObservation(res, "") }
		}},
		{"sendWelcome", func(t *testing.T, s *Session) func() {
			return func() { s.sendWelcome("") }
		}},
		{"sendError", func(t *testing.T, s *Session) func() {
			return func() { s.sendError("boom") }
		}},
		{"sendCharCreatePromptWithSecret", func(t *testing.T, s *Session) func() {
			return func() { s.sendCharCreatePromptWithSecret("login_password", "Password: ", nil, true) }
		}},
		{"sendStatsRollPrompt", func(t *testing.T, s *Session) func() {
			return func() { s.sendStatsRollPrompt() }
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := makeTestManager(t)
			s := makeTestSession(t, m, "Victim", 1001, true)
			produce := tc.prepare(t, s)

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
				produce()
				done <- true
			}()

			// Let the goroutine reach its send (it parks pre-fix), then take over.
			time.Sleep(50 * time.Millisecond)
			s.CloseSend()

			select {
			case ok := <-done:
				if !ok {
					t.Errorf("producer panicked on the closed send channel")
				}
			case <-time.After(2 * time.Second):
				t.Errorf("producer is parked on the full send channel")
			}

			// After the close the producer must no-op, not panic; this call
			// runs unrecovered on the test goroutine on purpose.
			produce()
		})
	}
}
