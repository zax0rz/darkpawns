package session

import (
	"bytes"
	"encoding/json"
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

func TestHeartbeatOutputPromptAfterPointText(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Promptactor", 1001, true)
	registerTestSession(t, m, s, s.playerName)
	m.BeginHeartbeatOutput()
	s.Send("weather\r\n")
	s.SendPrompt()
	s.Send("point update\r\n")
	s.SendPrompt()
	if len(s.send) != 0 {
		t.Fatal("early prompt escaped active turn")
	}
	m.EndHeartbeatOutput()
	got := renderedOutput(s)
	if got != "weather\r\npoint update\r\n\r\n> " {
		t.Fatalf("prompt order/framing: %q", got)
	}
}

func TestHeartbeatOutputRawAndObservationPaths(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Rawactor", 1001, true)
	registerTestSession(t, m, s, s.playerName)
	m.BeginHeartbeatOutput()
	s.ClearPromptShown()
	s.MarkAliasedInput()
	s.sendRawEvent("\x1b[31m")
	if err := cmdLook(s, nil); err != nil {
		t.Fatal(err)
	}
	if len(s.send) != 0 {
		t.Fatal("raw/input/observation path bypassed staging")
	}
	m.EndHeartbeatOutput()
	if got := <-s.send; !bytes.Equal(got, inputMarkFrame) {
		t.Fatalf("input marker=%q", got)
	}
	if got := <-s.send; !bytes.Equal(got, inputMarkAliasedFrame) {
		t.Fatalf("alias marker=%q", got)
	}
	if f, ok := RenderTerminalFrame(<-s.send); !ok || f.Text != "\x1b[31m" {
		t.Fatalf("raw control=%+v", f)
	}
	state := false
	for len(s.send) > 0 {
		var msg ServerMessage
		if err := json.Unmarshal(<-s.send, &msg); err != nil {
			t.Fatal(err)
		}
		state = state || msg.Type == MsgState
	}
	if !state {
		t.Fatal("structured observation lost")
	}
}
