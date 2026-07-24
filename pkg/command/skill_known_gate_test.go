package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestCmdBash_UnknownSkill_WireOutputIsExact — end-to-end: the whole design
// hinges on the handler's `SendMessage(msg + "\r\n")` append producing exactly
// one terminator. A warrior with bash==0 runs `bash` and the captured wire
// output must be exactly "You'd better leave all the martial arts to fighters.\r\n"
// (single \r\n, not double). This locks the load-bearing append against a future
// refactor. DP-1206. (Claude review note #1 — belt-and-suspenders with the
// CanUseSkill unit tests + the oracle gate.)
func TestCmdBash_UnknownSkill_WireOutputIsExact(t *testing.T) {
	// A bare world is enough: CmdBash fails at the CanUseSkill gate before any
	// target resolution or world access.
	w := &game.World{}
	p := game.NewPlayer(1, "Hero", 1001)
	p.Class = game.ClassWarrior
	// bash stays 0 (default) → CanUseSkill returns the audited message.
	sess := &killPayoutSession{player: p, world: w}

	if err := CmdBash(sess, []string{"target"}); err != nil {
		t.Fatalf("CmdBash returned error: %v", err)
	}
	if len(sess.messages) != 1 {
		t.Fatalf("expected exactly 1 message, got %d: %q", len(sess.messages), sess.messages)
	}
	const want = "You'd better leave all the martial arts to fighters.\r\n"
	if got := sess.messages[0]; got != want {
		t.Errorf("wire output = %q, want %q (single \\r\\n, not double)", got, want)
	}
	// Belt-and-suspenders: assert no double terminator snuck in.
	if strings.Contains(sess.messages[0], "\r\n\r\n") {
		t.Errorf("wire output has a double \\r\\n: %q", sess.messages[0])
	}
}
