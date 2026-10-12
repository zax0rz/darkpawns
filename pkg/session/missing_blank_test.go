package session

// missing_blank_test.go — PF-063 cluster 5 proofs for the session side:
// sendText-only output owes the prompt frame (C's process_output frames any
// flushed output, comm.c:1631), and do_dark's room act carries its own
// trailing break (act.wizard.c:3233-3235). Exact rendered bytes; reverting
// either fix fails (mutation checks).

import (
	"encoding/json"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestSendTextOutputOwesThePromptFrame pins C's frame rule for sendText
// responses: the message renders line-ended, and the following prompt pass
// is framed — the blank line do_ungroup's reply shows (act.other.c:754).
// Fails without notePlayerOutput in sendText.
func TestSendTextOutputOwesThePromptFrame(t *testing.T) {
	m := &Manager{}
	s := m.NewSession()
	s.player = game.NewPlayer(1, "Tester", 1001)

	s.sendText("But you lead no group!")
	s.SendPrompt()
	if got := renderAllOutput(s); got != "But you lead no group!\r\n\r\n> " {
		t.Fatalf("rendered wire = %q, want %q (message line-ended, framed prompt)", got, "But you lead no group!\r\n\r\n> ")
	}
}

// TestDarkRoomActCarriesItsOwnBreak — C's act format ends with \r\n and
// act() appends the second (act.wizard.c:3233-3235): the room audience sees
// the stop line followed by a blank line before the prompt.
func TestDarkRoomActCarriesItsOwnBreak(t *testing.T) {
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Study"}},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	god := makeCommandTestSession(t, m, "Darkgod", LVL_IMMORT+1, 1001)
	if err := w.AddPlayer(god.player); err != nil {
		t.Fatalf("AddPlayer god: %v", err)
	}
	watcher := makeTestSession(t, m, "Watcher", 1001, true)
	m.mu.Lock()
	m.sessions["watcher"] = watcher
	m.mu.Unlock()
	if err := w.AddPlayer(watcher.player); err != nil {
		t.Fatalf("AddPlayer watcher: %v", err)
	}

	if err := cmdDark(god, nil); err != nil {
		t.Fatalf("cmdDark: %v", err)
	}
	want := "Darkgod stops the senseless violence in the room with a wave of his hand.\r\n\r\n"
	if got := renderAllOutput(watcher); got != want {
		t.Fatalf("watcher bytes = %q, want %q (act's own break plus act()'s append)", got, want)
	}
}

// TestUnknownCommandRendersCTerminatedReplyWithFrame — the generic
// not-found reply is C's terminated "Huh?!?\r\n" (interpreter.c:916); on the
// wire that is message, blank (the frame), prompt. Pins the Send seam: the
// old sendText doubled the terminator, which the missing frame cancelled.
func TestUnknownCommandRendersCTerminatedReplyWithFrame(t *testing.T) {
	m := &Manager{}
	s := m.NewSession()
	s.player = game.NewPlayer(1, "Tester", 1001)

	data, _ := json.Marshal(CommandData{Command: "xyzzy"})
	if err := s.handleCommand(data); err != nil {
		t.Fatalf("handleCommand: %v", err)
	}
	s.SendPrompt()
	if got := renderAllOutput(s); got != "Huh?!?\r\n\r\n> " {
		t.Fatalf("rendered wire = %q, want %q", got, "Huh?!?\r\n\r\n> ")
	}
}
