package session

import (
	"encoding/json"
	"testing"
)

// PF-063 cluster 3 cross-check: do_compare's LFCR literals
// ("They're the same thing!\n\r", src/new_cmds.c:1991) are handed to
// Session.SendMessage with exact bytes (proven in pkg/command's
// TestSendSkillResultSendsLFCRBytesExactly), but the telnet renderer's text
// event applies NormalizeCRLF, which folds "\n\r" to "\r\n". This test runs
// the actual rendering path — SendMessage to the send channel, then
// RenderTerminalFrame — and asserts the frame a terminal would display
// carries C's spelling. It fails while the fold exists: the wire ships CRLF
// where C writes LFCR (invisible to both comparators, which fold spellings
// too; only this renderer-level assertion can see it).
func TestTextEventRenderingPreservesLFCR(t *testing.T) {
	m := &Manager{}
	s := m.NewSession()
	const cBytes = "They're the same thing!\n\r"

	s.SendMessage(cBytes)
	if len(s.send) != 1 {
		t.Fatalf("send channel holds %d messages, want 1", len(s.send))
	}
	var msg ServerMessage
	raw := <-s.send
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	frame, ok := RenderTerminalFrame(raw)
	if !ok {
		t.Fatal("RenderTerminalFrame dropped the message")
	}
	if frame.Kind != FrameText {
		t.Fatalf("frame kind = %v, want FrameText", frame.Kind)
	}
	if frame.Text != cBytes {
		t.Fatalf("terminal frame text = %q, want %q — the LFCR terminator was folded at rendering (NormalizeCRLF)", frame.Text, cBytes)
	}
}

// The literal/raw event path is the byte-exact escape hatch: control case
// proving it renders the same bytes untouched.
func TestRawEventRenderingPreservesLFCR(t *testing.T) {
	m := &Manager{}
	s := m.NewSession()
	const cBytes = "They're the same thing!\n\r"

	s.sendLiteralText(cBytes)
	if len(s.send) != 1 {
		t.Fatalf("send channel holds %d messages, want 1", len(s.send))
	}
	frame, ok := RenderTerminalFrame(<-s.send)
	if !ok || frame.Kind != FrameText {
		t.Fatalf("frame = %+v, want a text frame", frame)
	}
	if frame.Text != cBytes {
		t.Fatalf("raw frame text = %q, want %q", frame.Text, cBytes)
	}
}
