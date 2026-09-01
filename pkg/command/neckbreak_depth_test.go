package command

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCmdNeckbreakUsesCTargetBoundary(t *testing.T) {
	session := newSkillCommandSession(t)
	t.Cleanup(session.world.StopAITicker)
	session.player.SetSkill(game.SkillNeckbreak, 100)

	if err := CmdNeckbreak(session, nil); err != nil {
		t.Fatalf("bare CmdNeckbreak: %v", err)
	}
	if got := joinMessages(session.messages); got != "I don't see them here.\r\n" {
		t.Fatalf("bare output = %q, want C target-lookup response", got)
	}

	session.messages = nil
	if err := CmdNeckbreak(session, []string{"nobody", "trailing"}); err != nil {
		t.Fatalf("missing CmdNeckbreak: %v", err)
	}
	if got := joinMessages(session.messages); got != "I don't see them here.\r\n" {
		t.Fatalf("missing output = %q, want C target-lookup response", got)
	}
}

func TestCmdNeckbreakSkipsFillWordsAndTrailingArguments(t *testing.T) {
	session := newSkillCommandSession(t)
	t.Cleanup(session.world.StopAITicker)
	session.player.SetSkill(game.SkillNeckbreak, 100)
	victim := game.NewPlayer(2, "Victim", session.player.GetRoom())
	if err := session.world.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer(victim): %v", err)
	}
	session.world.GetRoomInWorld(session.player.GetRoom()).Flags = []string{"peaceful"}

	if err := CmdNeckbreak(session, []string{"at", "Victim", "trailing", "words"}); err != nil {
		t.Fatalf("CmdNeckbreak: %v", err)
	}
	if got := joinMessages(session.messages); !strings.Contains(got, "You can't contemplate violence in such a place!\r\n") {
		t.Fatalf("output = %q, want peaceful result for first non-fill token", got)
	}
}
