package command

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// TestBiteNightCreatureRefusalHasNoTerminator pins R1: C's do_bite sends
// "Your victim is already a creature of the night!" with NO trailing CRLF
// (src/new_cmds.c:1260) — the only bite refusal without one. sendSkillResult
// would otherwise append two bytes C never sends, and the terminator-
// normalizing oracle cannot see the difference. This asserts the raw actor
// bytes the command surfaces.
func TestBiteNightCreatureRefusalHasNoTerminator(t *testing.T) {
	s := newRescueCommandSession(t, nil)
	t.Cleanup(s.world.StopAITicker)

	actor := s.player
	actor.SetLevel(20)
	actor.SetPlrFlag(game.PlrVampire, true)
	actor.SetAffect(game.AffVampire, true)

	victim := game.NewPlayer(2, "NightFoe", 1001)
	victim.SetLevel(8)
	victim.SetPlrFlag(game.PlrVampire, true)
	if err := s.world.AddPlayer(victim); err != nil {
		t.Fatal(err)
	}

	result := game.DoBite(actor, victim)
	if result.MessageToCh != "Your victim is already a creature of the night!" {
		t.Fatalf("DoBite message = %q", result.MessageToCh)
	}
	if !result.NoMessageToChTerminator {
		t.Fatal("DoBite did not mark the terminator-less night-creature refusal")
	}

	if err := sendSkillResult(s, actor, victim, result); err != nil {
		t.Fatal(err)
	}

	const want = "Your victim is already a creature of the night!"
	if len(s.messages) != 1 || s.messages[0] != want {
		t.Fatalf("actor messages = %q, want [%q] (C sends this line with no CRLF)", s.messages, want)
	}
}
