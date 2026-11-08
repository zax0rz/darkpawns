package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// DP-1502: C's do_subdue prints "Subdue who?" both for a missing argument and
// for a get_char_room_vis failure (src/act.offensive.c:1101-1105). The port
// used to answer "They aren't here." for the latter. This test fails on that
// shape (R5h).
func TestCmdSubdueUnresolvedTargetSaysSubdueWho(t *testing.T) {
	ktw := newKillTestWorld(t, 500, 0, 0, 10, "target")
	actor := ktw.addPlayer(t, 1, "Subduer", 12, game.ClassWarrior, false)
	actor.SetPosition(combat.PosStanding)
	actor.SetSkill(game.SkillSubdue, 100)
	session := &killPayoutSession{player: actor, world: ktw.world}

	if err := CmdSubdue(session, []string{"nobody"}); err != nil {
		t.Fatal(err)
	}
	msgs := session.getMessages()
	if len(msgs) != 1 || msgs[0] != "Subdue who?\r\n" {
		t.Fatalf("messages = %q, want [\"Subdue who?\\r\\n\"]", msgs)
	}
}
