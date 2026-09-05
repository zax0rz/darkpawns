package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestCmdSlug_FallsBackToFightingTarget pins do_slug's direct FIGHTING(ch)
// fallback even when a named lookup misses (new_cmds.c:833-840).
func TestCmdSlug_FallsBackToFightingTarget(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosFighting)
	session.player.SetSkill(game.SkillSlug, 75)
	session.player.MountName = "a pony"

	target := game.NewPlayer(2, "Target", 1001)
	target.SetPosition(combat.PosFighting)
	if err := session.world.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer target: %v", err)
	}
	session.player.SetFighting(target.GetName())

	if err := CmdSlug(session, []string{"nobody"}); err != nil {
		t.Fatalf("CmdSlug: %v", err)
	}
	got := strings.Join(session.messages, "")
	if got != "Dismount first!\r\n" {
		t.Fatalf("explicit-miss fallback output = %q, want mounted gate", got)
	}

	session.messages = nil
	if err := CmdSlug(session, nil); err != nil {
		t.Fatalf("CmdSlug without argument: %v", err)
	}
	if got := strings.Join(session.messages, ""); got != "Dismount first!\r\n" {
		t.Fatalf("empty-argument fallback output = %q, want mounted gate", got)
	}
}

func TestCmdSlug_SelfTargetUsesCMessage(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetPosition(combat.PosFighting)
	session.player.SetSkill(game.SkillSlug, 75)

	if err := CmdSlug(session, []string{"Tester"}); err != nil {
		t.Fatalf("CmdSlug: %v", err)
	}
	if got := strings.Join(session.messages, ""); got != "You curl up your fist and slug yourself in the nose! Ouch!\r\n" {
		t.Fatalf("self-target output = %q", got)
	}
}
