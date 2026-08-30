package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCmdDisembowel_NoTargetAtFightingPosition(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetSkill(game.SkillDisembowel, 1)
	session.player.SetPosition(combat.PosFighting)

	if err := CmdDisembowel(session, nil); err != nil {
		t.Fatalf("CmdDisembowel: %v", err)
	}
	if got := joinMessages(session.messages); !strings.Contains(got, "Disembowel who?") {
		t.Fatalf("messages = %q, want Disembowel who?", got)
	}
}

func TestCmdDisembowel_FallsBackToFightingTarget(t *testing.T) {
	session := newSkillCommandSession(t)
	session.player.SetSkill(game.SkillDisembowel, 1)
	session.player.SetPosition(combat.PosFighting)
	target := game.NewPlayer(2, "Target", session.player.GetRoom())
	if err := session.world.AddPlayer(target); err != nil {
		t.Fatalf("AddPlayer(target): %v", err)
	}
	session.player.SetFighting(target.GetName())

	if err := CmdDisembowel(session, []string{"nobody"}); err != nil {
		t.Fatalf("CmdDisembowel: %v", err)
	}
	got := joinMessages(session.messages)
	if strings.Contains(got, "Disembowel who?") {
		t.Fatalf("messages = %q, explicit miss did not fall back to fighting target", got)
	}
	if !strings.Contains(got, "You need to wield a weapon to make it a success.") {
		t.Fatalf("messages = %q, want the resolved target's weapon gate", got)
	}
}
