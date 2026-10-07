package session

import (
	"encoding/json"
	"strings"
	"testing"

	"golang.org/x/time/rate"
)

// C process_input copies only isascii() && isprint() bytes and applies
// backspaces (src/comm.c:1965-1982).
func TestCInputLine(t *testing.T) {
	cases := []struct{ in, want string }{
		{"say hello", "say hello"},
		{"say a\x1b[2Jb", "say a[2Jb"},             // ESC dropped
		{"say \x1b]0;title\x07x", "say ]0;titlex"}, // OSC and BEL dropped
		{"say a\xffb", "say ab"},                   // decoded telnet IAC dropped
		{"say a\tb\x7fc", "say abc"},               // tab and DEL are not isprint
		{"say caf\xc3\xa9", "say caf"},             // non-ASCII bytes are not isascii
		{"sax\by hi", "say hi"},                    // backspace removes the kept byte
		{"\b\bsay", "say"},                         // backspace on empty is a no-op
		{"say 100$", "say 100$"},                   // $ survives; doubling is elsewhere
		{"", ""},
	}
	for _, tc := range cases {
		if got := cInputLine(tc.in); got != tc.want {
			t.Errorf("cInputLine(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func filterTestSpeaker(t *testing.T) *Session {
	t.Helper()
	m := makeTestManagerWithVoidRooms(t)
	s := makeTestSession(t, m, "Speaker", 1001, true)
	s.terminalNamed = true
	s.limiter = rate.NewLimiter(rate.Inf, 1000)
	s.player.Stats.Int, s.player.Stats.Wis = 13, 13 // C refuses thought at 0
	s.player.CopyBaseAttributes()
	registerTestSession(t, m, s, "Speaker")
	return s
}

// A terminal line (telnet or the browser terminal) reaches commands only
// after C's character pass. think echoes its argument verbatim, so control
// bytes in the echo would reach the terminal (VULN-014, REF-1 inbound).
func TestTerminalLineAppliesCInputFilter(t *testing.T) {
	s := filterTestSpeaker(t)
	s.TerminalLine("think a\x1b[2Jb\xffc\bd")
	out := drainFrames(s)
	if !strings.Contains(out, "( a[2Jbd )") {
		t.Fatalf("think echo = %q, want the filtered argument", out)
	}
	if strings.ContainsRune(out, '\x1b') || strings.ContainsRune(out, '\b') || strings.Contains(out, "\xff") {
		t.Fatalf("control bytes reached the echo: %q", out)
	}
}

// Structured WebSocket clients get the same pass on every string they send.
func TestStructuredCommandAppliesCInputFilter(t *testing.T) {
	s := filterTestSpeaker(t)
	msg, err := json.Marshal(map[string]any{
		"type": MsgCommand,
		"data": map[string]any{"command": "think", "args": []string{"x\x1b[31my"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.handleMessage(msg); err != nil {
		t.Fatalf("handleMessage: %v", err)
	}
	out := strings.Join(drainSessionText(t, s), "")
	if !strings.Contains(out, "x[31my") || strings.Contains(out, "\x1b[31m") {
		t.Fatalf("think echo = %q, want ESC removed from the structured argument", out)
	}
}
