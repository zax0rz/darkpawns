package session

import (
	"bytes"
	"strings"
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

func TestHeartbeatOutputManagerAndTextPaths(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Queueactor", 1001, true)
	registerTestSession(t, m, s, s.playerName)
	m.BeginHeartbeatOutput()
	s.Send("first\r\n")
	s.sendText("second")
	m.world.MessageSink(s.playerName, []byte("third\r\n"))
	m.SendToAll("fourth\r\n")
	if len(s.send) != 0 {
		t.Fatal("text or manager path bypassed staging")
	}
	m.EndHeartbeatOutput()
	got := renderedOutput(s)
	if !strings.Contains(got, "first\r\nsecond\r\nthird\r\nfourth\r\n") {
		t.Fatalf("delivery order: %q", got)
	}
	s.Send("ordinary\r\n")
	if len(s.send) != 1 {
		t.Fatal("ordinary send did not remain immediate")
	}
}
