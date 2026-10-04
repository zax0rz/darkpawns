package session

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
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

func TestHeartbeatOutputGMCPAndVarsPaths(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Dataactor", 1001, true)
	s.wantsStructuredData = true
	s.subscribedVars[VarHealth] = true
	m.BeginHeartbeatOutput()
	s.sendGMCPRaw("Char.Vitals", `{"hp":12}`)
	s.markDirty(VarHealth)
	s.flushDirtyVars()
	s.sendFullVarDump()
	if len(s.send) != 0 {
		t.Fatal("GMCP/vars path bypassed staging")
	}
	m.EndHeartbeatOutput()
	for _, want := range []string{MsgGMCP, MsgVars, MsgVars} {
		var msg ServerMessage
		if err := json.Unmarshal(<-s.send, &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Type != want {
			t.Fatalf("envelope/order: %s want %s", msg.Type, want)
		}
	}
}

func TestHeartbeatOutputIdleDiscardBeforeWriter(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Idleactor", 1, true)
	listener := makeTestSession(t, m, "Listener", 4, true)
	registerTestSession(t, m, s, s.playerName)
	registerTestSession(t, m, listener, listener.playerName)
	s.player.SetLevel(1)
	s.player.WasInRoom = 1001
	s.player.SetIdleTimer(30)
	closes := 0
	s.SetCloseFunc(func() { closes++ })
	t.Cleanup(s.closeSendNow)
	got := make(chan string, 1)
	ready := make(chan struct{})
	go func() {
		close(ready)
		var text strings.Builder
		for msg := range s.send {
			if f, ok := RenderTerminalFrame(msg); ok {
				text.WriteString(f.Text)
			}
		}
		got <- text.String()
	}()
	<-ready
	m.BeginHeartbeatOutput()
	// Real outdoor broadcast, not a weather-only special case.
	for game.TimeSnapshot().Hours != 20 {
		game.AnotherHour(false, nil)
	}
	game.AnotherHour(true, m.SendToOutdoor)
	s.Send("point update before close\r\n")
	m.world.CheckIdling(s.player)
	if !s.SendClosed() || closes != 1 {
		t.Fatal("idle close deferred past rent/extraction")
	}
	m.EndHeartbeatOutput()
	if text := <-got; text != "" {
		t.Fatalf("writer received terminal output: %q", text)
	}
	if !m.HandleTransportDisconnect(s) {
		t.Fatal("transport teardown must retain pending rent/extraction owner")
	}
	if closes != 1 {
		t.Fatal("transport repeated idle close")
	}
	heard := renderedOutput(listener)
	if strings.Count(heard, "Idleactor has lost") != 1 || strings.Count(heard, "> ") != 1 {
		t.Fatalf("room act/prompt duplicated or lost: %q", heard)
	}
	m.ExtractPendingChars()
	if closes != 1 {
		t.Fatal("extraction repeated transport close")
	}
	if _, ok := m.GetSession(s.playerName); ok {
		t.Fatal("idle owner not retired at extraction")
	}
	s.Send("late") // discarded descriptor remains safe across the final boundary
}

func TestHeartbeatOutputIdleSwitchAttachment(t *testing.T) {
	for _, which := range []string{"original", "borrowed"} {
		t.Run(which, func(t *testing.T) {
			m, s, h, original := switchedPCFixture(t)
			victim := original
			if which == "borrowed" {
				victim = h.player
			}
			victim.SetLevel(1)
			victim.WasInRoom = 1001
			victim.SetIdleTimer(30)
			m.BeginHeartbeatOutput()
			s.Send("queued borrowed output\r\n")
			m.world.CheckIdling(victim)
			m.EndHeartbeatOutput()
			if which == "original" {
				if s.SendClosed() || !s.isSwitched {
					t.Fatal("descriptorless original closed borrowed descriptor")
				}
				if !strings.Contains(renderedOutput(s), "queued borrowed output") {
					t.Fatal("borrowed descriptor lost output")
				}
			} else {
				if !s.SendClosed() || s.isSwitched || s.player != original || !original.IsLinkless() || !h.player.IsLinkless() {
					t.Fatal("borrowed idle close failed to detach both identities")
				}
				if got := renderedOutput(s); got != "" {
					t.Fatalf("closed switched actor received %q", got)
				}
			}
		})
	}
}

func TestHeartbeatOutputSnoopAndOrderlyBarrier(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	target := makeTestSession(t, m, "Target", 1001, true)
	spy := makeTestSession(t, m, "Spy", 1001, true)
	m.snoopMu.Lock()
	target.snoopBy = spy
	spy.snooping = target
	m.snoopMu.Unlock()
	m.BeginHeartbeatOutput()
	target.Send("first\r\n")
	target.Send("second\r\n")
	target.CloseSend()
	target.Send("late\r\n")
	if len(target.send) != 0 || len(spy.send) != 0 || target.SendClosed() {
		t.Fatal("orderly barrier bypassed staged flush")
	}
	m.EndHeartbeatOutput()
	if got := renderedOutput(target); got != "first\r\nsecond\r\n" {
		t.Fatalf("goodbye queue=%q", got)
	}
	if !target.SendClosed() {
		t.Fatal("FIFO close barrier not committed")
	}
	if got := renderedOutput(spy); got != "% first\r\nsecond\r\n%%\r\n" {
		t.Fatalf("snoop flush delimiters/order=%q", got)
	}
}

func TestHeartbeatOutputIdleSnoopDiscard(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	target := makeTestSession(t, m, "Target", 1, true)
	spy := makeTestSession(t, m, "Spy", 1001, true)
	registerTestSession(t, m, target, target.playerName)
	registerTestSession(t, m, spy, spy.playerName)
	target.player.SetLevel(1)
	target.player.SetIdleTimer(30)
	target.player.WasInRoom = 1001
	m.snoopMu.Lock()
	target.snoopBy = spy
	spy.snooping = target
	m.snoopMu.Unlock()
	m.BeginHeartbeatOutput()
	target.Send("discarded weather\r\n")
	m.world.CheckIdling(target.player)
	m.EndHeartbeatOutput()
	if got := renderedOutput(target); got != "" {
		t.Fatalf("closed target=%q", got)
	}
	got := renderedOutput(spy)
	if strings.Contains(got, "discarded weather") || strings.Count(got, "Your victim is no longer among us.") != 1 {
		t.Fatalf("snoop close=%q", got)
	}
	if target.snoopBy != nil || spy.snooping != nil {
		t.Fatal("idle close left snoop links")
	}
}

func TestHeartbeatOutputConcurrentEnqueueCommitClose(t *testing.T) {
	for _, immediate := range []bool{false, true} {
		m := &Manager{}
		s := m.NewSession()
		m.BeginHeartbeatOutput()
		start := make(chan struct{})
		var wg sync.WaitGroup
		got := make(chan [][]byte, 1)
		go func() {
			var frames [][]byte
			for msg := range s.send {
				frames = append(frames, msg)
			}
			got <- frames
		}()
		wg.Add(3)
		go func() {
			defer wg.Done()
			<-start
			for i := 0; i < 100; i++ {
				s.sendGuarded([]byte(strconv.Itoa(i)))
			}
		}()
		go func() { defer wg.Done(); <-start; m.EndHeartbeatOutput() }()
		go func() {
			defer wg.Done()
			<-start
			if immediate {
				s.discardHeartbeatOutput()
			} else {
				s.CloseSend()
			}
		}()
		close(start)
		wg.Wait()
		previous := -1
		for _, frame := range <-got {
			n, err := strconv.Atoi(string(frame))
			if err != nil {
				t.Fatal(err)
			}
			if n <= previous {
				t.Fatalf("FIFO overtaken: %d after %d", n, previous)
			}
			previous = n
		}
		if !s.SendClosed() {
			t.Fatal("concurrent close lost")
		}
	}
}
