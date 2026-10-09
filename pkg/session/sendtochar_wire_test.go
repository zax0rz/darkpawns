package session

import (
	"strings"
	"testing"

	"golang.org/x/time/rate"
)

// captureWire drains a session's send channel exactly as the telnet writeLoop
// does (pkg/telnet/listener.go:476-514): render each queued message, apply the
// prompt state, and concatenate the bytes a terminal receives. Telnet writes
// FrameText via tc.write(escapeIAC(f.Text)) and FramePrompt via
// tc.writePrompt(f.Text); markPrompt only appends the IAC EOR marker and
// escapeIAC only doubles 0xFF, neither of which occurs in these transcripts.
func captureWire(s *Session) string {
	var out strings.Builder
	for {
		select {
		case msg, open := <-s.send:
			if !open {
				return out.String()
			}
			f, ok := s.RenderTerminalFrame(msg)
			if !ok {
				continue
			}
			f = s.TrackPrompt(f)
			switch f.Kind {
			case FrameText, FramePrompt:
				out.WriteString(f.Text)
			}
		default:
			return out.String()
		}
	}
}

// doorWireFixture registers a playing mortal in room 1001 and returns a closure
// that runs one command line and captures the bytes a telnet client would see.
func doorWireFixture(t *testing.T) func(string) string {
	t.Helper()
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Opener", 1001, true)
	s.terminalNamed = true
	s.limiter = rate.NewLimiter(rate.Inf, 1000)
	registerTestSession(t, m, s, "Opener")
	_ = captureWire(s) // discard the entry state
	return func(line string) string {
		s.TerminalLine(line)
		return captureWire(s)
	}
}

// TestDoorOpenWireMatchesC is the R1 regression for the sendToChar double-CRLF
// class. C's do_gen_door sends "Open what?\r\n" via send_to_char, and
// process_output appends one "\r\n" plus the prompt (src/act.movement.c:606-609,
// src/comm.c:1620-1644). The wire is therefore exactly two line breaks before
// the prompt. Go's sendToChar appended a second "\r\n" of its own, producing a
// third break -- an extra blank line the C oracle never wrote. The divergence is
// invisible to the census: Normalize deletes the "<PROMPT>" line as framing,
// which leaves the surplus blank line trailing, and the final
// strings.Trim(...,"\n") removes it (internal/oraclediff/normalize.go:115-124).
func TestDoorOpenWireMatchesC(t *testing.T) {
	run := doorWireFixture(t)

	cases := []struct {
		line string
		want string
	}{
		{"open", "Open what?\r\n\r\n> "},
		{"open xyzzy", "There doesn't seem to be a xyzzy here.\r\n\r\n> "},
		{"open north", "There doesn't seem to be a north here.\r\n\r\n> "},
	}
	for _, c := range cases {
		if got := run(c.line); got != c.want {
			t.Errorf("%q wire = %q, want %q (C emits exactly one blank line before the prompt)", c.line, got, c.want)
		}
	}
}
