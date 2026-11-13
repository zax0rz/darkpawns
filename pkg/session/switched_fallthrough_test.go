package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/command"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1516 #2440: an actor lane the switched body cannot satisfy must not
// swallow the command. Several lanes were written for descriptor-less NPCs and
// print nothing on a miss, so returning early on CharacterCommandUnsupported
// made those commands silently do nothing for a switched body. The command must
// fall through to the session handler instead.
func TestSwitchedBodyUnsupportedActorLaneFallsThrough(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Swdispatch", 1, 1001)
	mob := game.NewMob(&parser.Mob{VNum: 99050, Keywords: "probe", ShortDesc: "a probe"}, 1001)
	s.isSwitched = true
	s.switchedMob = mob

	const name = "switchedfallthroughtest"
	cmdRegistry.Register(name, wrapArgs(func(sess *Session, _ []string) error {
		sess.Send("session-handler-ran")
		return nil
	}), "switched fall-through probe", 0, combat.PosDead)
	if err := cmdRegistry.SetActorHandler(name, func(command.ExecutionContext, string, []string) game.CharacterCommandResult {
		return game.CharacterCommandUnsupported
	}); err != nil {
		t.Fatal(err)
	}

	if err := ExecuteCommand(s, name, nil); err != nil {
		t.Fatalf("ExecuteCommand: %v", err)
	}
	if got := readMsgText(t, s); got != "session-handler-ran" {
		t.Fatalf("unsupported actor lane swallowed the command: got %q", got)
	}
}
