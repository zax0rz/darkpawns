package session

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestReviewSendsCBytesUntouched drives the real review command and renders
// what the terminal receives. do_review pages its buffer verbatim
// (src/new_cmds.c:1369-1391): C_CMP cyan name, yellow text, KNRM, and a
// "\n\r" entry terminator. The oracle normalizes line endings, so only a byte
// test can hold the terminator (R1, R5h).
func TestReviewSendsCBytesUntouched(t *testing.T) {
	m := makeTestManagerWithMobs(t)
	viewer := makeTestSession(t, m, "Viewer", 1001, true)
	speaker := makeTestSession(t, m, "Stink", 1001, true)
	registerInWorld(t, viewer)
	registerInWorld(t, speaker)
	viewer.player.SetPlrFlag(game.PrfColor1, true)
	viewer.player.SetPlrFlag(game.PrfColor2, true)

	speaker.player.Level = 30
	speaker.player.Stats.Int = 13
	speaker.player.Stats.Wis = 13
	speaker.player.CopyBaseAttributes()
	m.world.DoChannel(speaker.player, "peee-YEW", "gossip")
	_ = drainSendChannel(t, viewer)
	_ = drainSendChannel(t, speaker)

	if err := cmdReview(viewer); err != nil {
		t.Fatalf("cmdReview: %v", err)
	}
	select {
	case msg := <-viewer.send:
		frame, ok := RenderTerminalFrame(msg)
		if !ok {
			t.Fatal("review frame was not rendered")
		}
		want := "Last Gossips:\r\n-------------\r\n\x1b[36mStink: \x1b[33mpeee-YEW\x1b[0m\n\r"
		if frame.Text != want {
			t.Fatalf("review bytes = %q, want %q", frame.Text, want)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("timeout waiting for review output")
	}
}
