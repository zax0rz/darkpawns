package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestMindlinkDrainedLineAfterRoomHasNoTerminator pins the second actor-message
// emitter: a result that carries MessageToChAfterRoom and the bare-line flag
// must not gain a CRLF. C's do_mindlink reaches this emitter with
// "You feel a little drained..." (src/new_cmds2.c:314), which has no terminator.
func TestMindlinkDrainedLineAfterRoomHasNoTerminator(t *testing.T) {
	s := newRescueCommandSession(t, nil)
	t.Cleanup(s.world.StopAITicker)

	result := game.SkillResult{
		MessageToCh:             "You feel a little drained...",
		NoMessageToChTerminator: true,
		MessageToChAfterRoom:    true,
	}
	if err := sendSkillResult(s, s.player, nil, result); err != nil {
		t.Fatal(err)
	}
	const want = "You feel a little drained..."
	if len(s.messages) != 1 || s.messages[0] != want {
		t.Fatalf("actor messages = %q, want [%q] (C sends this line with no CRLF)", s.messages, want)
	}
}
