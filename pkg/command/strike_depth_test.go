package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestCmdStrike_BlocksMountedBeforeRoll — C's mount refusal sits after target
// resolution and before the roll and WAIT_STATE (new_cmds.c:1466-1470), so a
// mounted striker who names a resolvable target sees only that line.
func TestCmdStrike_BlocksMountedBeforeRoll(t *testing.T) {
	ktw := newKillTestWorld(t, 500, 0, 0, 1, "rat")
	ktw.world.StopAITicker()
	p := ktw.addPlayer(t, 1, "Strikerider", 20, game.ClassWarrior, false)
	p.SetPosition(combat.PosStanding)
	p.SetSkill(game.SkillStrike, 100)
	p.MountName = "pony"
	sess := &killPayoutSession{player: p, world: ktw.world}

	if err := CmdStrike(sess, []string{"rat"}); err != nil {
		t.Fatalf("CmdStrike: %v", err)
	}
	if got := strings.Join(sess.getMessages(), ""); got != "Dismount first!\r\n" {
		t.Errorf("mounted strike bytes = %q, want the C mount refusal", got)
	}
	if p.GetWaitState() != 0 {
		t.Errorf("mounted strike applied wait state %d, want 0 (C returns before WAIT_STATE)", p.GetWaitState())
	}
}
