package session

import (
	"strings"
	"testing"

	"golang.org/x/time/rate"
)

// drainFrames returns the rendered terminal bytes queued on a session.
func drainFrames(s *Session) string {
	var out strings.Builder
	for {
		select {
		case msg, open := <-s.send:
			if !open {
				return out.String()
			}
			if f, ok := RenderTerminalFrame(msg); ok && f.Kind != FrameGMCP {
				out.WriteString(f.Text)
			}
		default:
			return out.String()
		}
	}
}

// TestInputLinePromptsOthersItReached is the DP-1307 regression. C reads one
// line, then flushes every descriptor's pending output with its prompt in the
// same pass (comm.c:626-642). A player who hears another player's say gets a
// prompt after it; a player who heard nothing gets none.
func TestInputLinePromptsOthersItReached(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	speaker := makeTestSession(t, m, "Speaker", 1001, true)
	speaker.terminalNamed = true
	speaker.limiter = rate.NewLimiter(rate.Inf, 1000)
	speaker.player.Stats.Int, speaker.player.Stats.Wis = 13, 13 // C refuses speech at 0
	registerTestSession(t, m, speaker, "Speaker")
	listener := makeTestSession(t, m, "Listener", 1001, true)
	listener.player.ID = 2
	registerTestSession(t, m, listener, "Listener")
	elsewhere := makeTestSession(t, m, "Elsewhere", 3, true)
	elsewhere.player.ID = 3
	registerTestSession(t, m, elsewhere, "Elsewhere")

	speaker.TerminalLine("say hello")

	heard := drainFrames(listener)
	if !strings.Contains(heard, "Speaker says, 'hello'") {
		t.Fatalf("listener output = %q, want the say", heard)
	}
	if !strings.HasSuffix(heard, "> ") && !strings.Contains(heard, "H ") {
		t.Fatalf("listener output = %q, want a prompt after the say", heard)
	}
	if got := drainFrames(elsewhere); got != "" {
		t.Fatalf("a player who heard nothing got %q", got)
	}
}

// A session busy with its own line is never prompted by someone else's
// sweep: its prompt would land in the middle of its own output.
func TestAsyncPromptSweepSkipsBusySession(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	busy := makeTestSession(t, m, "Busy", 1001, true)
	registerTestSession(t, m, busy, "Busy")
	busy.notePlayerOutput()
	busy.inputBusy.Store(true)

	m.FlushAsyncPrompts()
	if got := drainFrames(busy); got != "" {
		t.Fatalf("busy session was prompted: %q", got)
	}

	busy.inputBusy.Store(false)
	m.FlushAsyncPrompts()
	if got := drainFrames(busy); got == "" {
		t.Fatal("idle session with pending output was not prompted")
	}
}

// TestAliasedLineKeepsPromptState: C clears has_prompt when it takes a line
// (comm.c:613) but sets it again for a line from an alias expansion
// (comm.c:621-623), so that line's output starts on a new line.
func TestAliasedLineKeepsPromptState(t *testing.T) {
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Aliaser", 1001, true)

	render := func() []TerminalFrame {
		var frames []TerminalFrame
		for {
			select {
			case msg := <-s.send:
				if f, ok := RenderTerminalFrame(msg); ok {
					frames = append(frames, s.TrackPrompt(f))
				}
			default:
				return frames
			}
		}
	}
	textAfter := func(mark func()) string {
		s.promptShown.Store(false)
		s.SendPrompt()
		mark()
		s.Send("output\r\n")
		for _, f := range render() {
			if f.Kind == FrameText {
				return f.Text
			}
		}
		return ""
	}

	if got := textAfter(s.ClearPromptShown); got != "output\r\n" {
		t.Fatalf("ordinary line output = %q, want no interruption CR LF", got)
	}
	if got := textAfter(s.MarkAliasedInput); got != "\r\noutput\r\n" {
		t.Fatalf("aliased line output = %q, want the interruption CR LF", got)
	}
}
