package session

import (
	"bytes"
	"testing"
)

func TestHeartbeatOutputHelperFIFOAndBoundary(t *testing.T) {
	m := &Manager{}
	s := m.NewSession()
	if s.stageHeartbeat([]byte("ordinary"), "", false) {
		t.Fatal("ordinary delivery intercepted")
	}
	m.BeginHeartbeatOutput()
	frame := []byte("first")
	if !s.stageHeartbeat(frame, "", false) {
		t.Fatal("active delivery bypassed")
	}
	frame[0] = 'X'
	s.stageHeartbeat([]byte("second"), "", false)
	if len(s.send) != 0 {
		t.Fatal("output escaped before commit")
	}
	m.EndHeartbeatOutput()
	for _, want := range []string{"first", "second"} {
		if got := <-s.send; !bytes.Equal(got, []byte(want)) {
			t.Fatalf("got %q want %q", got, want)
		}
	}
	if s.stageHeartbeat([]byte("ordinary again"), "", false) {
		t.Fatal("ordinary delivery intercepted after commit")
	}
}

func TestHeartbeatOutputHelperDiscardAndCapacity(t *testing.T) {
	m := &Manager{}
	s := m.NewSession()
	s.send = make(chan []byte, 1)
	m.BeginHeartbeatOutput()
	s.stageHeartbeat([]byte("accepted"), "", true)
	s.stageHeartbeat([]byte("overflow"), "", true)
	m.EndHeartbeatOutput()
	if len(s.send) != 1 || string(<-s.send) != "accepted" {
		t.Fatal("bounded queue policy changed")
	}
	m.BeginHeartbeatOutput()
	s.stageHeartbeat([]byte("discard"), "", true)
	s.discardHeartbeatOutput()
	m.EndHeartbeatOutput()
	if got, ok := <-s.send; ok {
		t.Fatalf("discard delivered %q", got)
	}
	if s.outputSincePrompt.Load() != 0 {
		t.Fatal("discard left prompt bookkeeping")
	}
}

func TestHeartbeatOutputHelperOrderlyClose(t *testing.T) {
	m := &Manager{}
	s := m.NewSession()
	m.BeginHeartbeatOutput()
	s.stageHeartbeat([]byte("goodbye"), "", false)
	if !s.deferHeartbeatPrompt() || !s.queueHeartbeatClose() {
		t.Fatal("active barriers not accepted")
	}
	s.stageHeartbeat([]byte("late"), "", false)
	m.EndHeartbeatOutput()
	if got := string(<-s.send); got != "goodbye" {
		t.Fatalf("lost accepted goodbye: %q", got)
	}
	if got, ok := <-s.send; ok {
		t.Fatalf("post-barrier frame delivered: %q", got)
	}
}
