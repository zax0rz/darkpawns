package session

import (
	"encoding/json"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// PF-063 cluster 2: C's process_output adds the "\r\n" frame before the
// prompt only when the descriptor is playing normally AND the player is not
// compact (src/comm.c:1631). The ordinary prompt path had the playing gate
// but not the compact gate, printing a blank line before every prompt a
// compact player received. These tests pin the exact prompt bytes for the
// compact/noncompact, pending/no-pending, and editor states; revert the
// compact clause in sendPromptNow to fail them.

func newPromptFrameSession(t *testing.T, compact bool) *Session {
	t.Helper()
	m := &Manager{}
	s := m.NewSession()
	s.player = game.NewPlayer(1, "Tester", 1001)
	if compact {
		s.player.Flags = 1 << uint(game.PrfCompact)
	}
	return s
}

// promptTextAfterFlush runs one prompt pass with pendingOutput lines since
// the previous prompt and returns the prompt message's exact text bytes.
func promptTextAfterFlush(t *testing.T, s *Session, pendingOutput int64) string {
	t.Helper()
	s.outputSincePrompt.Store(pendingOutput)
	s.sendPromptNow()
	if len(s.send) != 1 {
		t.Fatalf("sendPromptNow enqueued %d messages, want exactly one prompt", len(s.send))
	}
	var msg ServerMessage
	if err := json.Unmarshal(<-s.send, &msg); err != nil {
		t.Fatalf("unmarshal prompt message: %v", err)
	}
	if msg.Type != MsgPrompt {
		t.Fatalf("message type = %q, want %q", msg.Type, MsgPrompt)
	}
	data, ok := msg.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("prompt message data = %#v, want a text map", msg.Data)
	}
	text, _ := data["text"].(string)
	return text
}

// TestSendPromptNowCompactSuppressesFrameWithPendingOutput is the cluster-2
// shape: with output pending, a noncompact player's prompt is framed with a
// leading "\r\n" (the blank line C prints) and a compact player's is bare
// (src/comm.c:1631's !PRF_COMPACT gate).
func TestSendPromptNowCompactSuppressesFrameWithPendingOutput(t *testing.T) {
	if got := promptTextAfterFlush(t, newPromptFrameSession(t, false), 1); got != "\r\n> " {
		t.Fatalf("noncompact framed prompt = %q, want %q", got, "\r\n> ")
	}
	if got := promptTextAfterFlush(t, newPromptFrameSession(t, true), 1); got != "> " {
		t.Fatalf("compact framed prompt = %q, want %q (compact players never get the frame)", got, "> ")
	}
}

// TestSendPromptNowBarePromptWithoutPendingOutput pins the other half of C's
// rule: with no output since the previous prompt, the bare game-loop prompt
// pass writes the prompt alone in both modes — no frame, compact or not.
func TestSendPromptNowBarePromptWithoutPendingOutput(t *testing.T) {
	if got := promptTextAfterFlush(t, newPromptFrameSession(t, false), 0); got != "> " {
		t.Fatalf("noncompact bare prompt = %q, want %q", got, "> ")
	}
	if got := promptTextAfterFlush(t, newPromptFrameSession(t, true), 0); got != "> " {
		t.Fatalf("compact bare prompt = %q, want %q", got, "> ")
	}
}

// TestSendPromptNowEditorFrameBehavior pins the editor interactions of the
// same C rule (comm.c:1630-1632): a CON_* editor's flush is never framed,
// do_string's CON_PLAYING editor's flush is framed for noncompact players,
// and compact suppresses the frame on every playing flush — the playing
// editor's included, because C's gate has no editor exception.
func TestSendPromptNowEditorFrameBehavior(t *testing.T) {
	withEditor := func(s *Session, playing bool) *Session {
		s.textEdit = &textEditState{playingEditor: playing}
		return s
	}

	// CON_* editor (tedit and friends): prompt "] ", no frame, pending output.
	if got := promptTextAfterFlush(t, withEditor(newPromptFrameSession(t, false), false), 1); got != "] " {
		t.Fatalf("CON_* editor prompt = %q, want %q (never framed)", got, "] ")
	}
	// do_string's editor stays in CON_PLAYING: framed for noncompact players.
	if got := promptTextAfterFlush(t, withEditor(newPromptFrameSession(t, false), true), 1); got != "\r\n] " {
		t.Fatalf("playing editor prompt = %q, want %q", got, "\r\n] ")
	}
	// Compact suppresses the frame even on the playing editor's flush.
	if got := promptTextAfterFlush(t, withEditor(newPromptFrameSession(t, true), true), 1); got != "] " {
		t.Fatalf("compact playing editor prompt = %q, want %q (compact has no editor exception)", got, "] ")
	}
	// And a CON_* editor stays bare under compact too.
	if got := promptTextAfterFlush(t, withEditor(newPromptFrameSession(t, true), false), 1); got != "] " {
		t.Fatalf("compact CON_* editor prompt = %q, want %q", got, "] ")
	}
}
