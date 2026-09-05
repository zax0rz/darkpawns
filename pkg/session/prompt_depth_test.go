package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPromptRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["prompt"]
	if !ok {
		t.Fatal("prompt command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosDead {
		t.Fatalf("prompt gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosDead)
	}
	registered, ok := cmdRegistry.Lookup("prompt")
	if !ok {
		t.Fatal("prompt command is not registered")
	}
	if registered.MinLevel != entry.MinLevel || registered.MinPosition != entry.MinPosition {
		t.Fatalf("prompt registry gate = level %d position %d, want level %d position %d", registered.MinLevel, registered.MinPosition, entry.MinLevel, entry.MinPosition)
	}
}

func TestCmdPromptMatchesCDoDisplay(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Prompttest", 1, 1001)
	registerInWorld(t, actor)

	if err := cmdPrompt(actor, nil); err != nil {
		t.Fatalf("cmdPrompt no argument: %v", err)
	}
	if got, want := readMsgText(t, actor), "Usage: prompt { H | M | V | T | F | all | none }\r\n"; got != want {
		t.Fatalf("no-argument response = %q, want %q", got, want)
	}

	if err := cmdPrompt(actor, []string{"all"}); err != nil {
		t.Fatalf("cmdPrompt all: %v", err)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("all response = %q, want %q", got, want)
	}
	for _, bit := range []int{game.PrfDisphp, game.PrfDispmmana, game.PrfDispmove, game.PrfDispTank, game.PrfDispTarget} {
		if actor.player.GetFlags()&(1<<uint(bit)) == 0 {
			t.Errorf("all: display bit %d is not set", bit)
		}
	}

	if err := cmdPrompt(actor, []string{"off"}); err != nil {
		t.Fatalf("cmdPrompt off: %v", err)
	}
	assertNoSessionMessage(t, actor)
	for _, bit := range []int{game.PrfDisphp, game.PrfDispmmana, game.PrfDispmove, game.PrfDispTank, game.PrfDispTarget} {
		if actor.player.GetFlags()&(1<<uint(bit)) != 0 {
			t.Errorf("off: display bit %d is still set", bit)
		}
	}

	if err := cmdPrompt(actor, []string{"hmv"}); err != nil {
		t.Fatalf("cmdPrompt hmv: %v", err)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("letter-mask response = %q, want %q", got, want)
	}
	for _, bit := range []int{game.PrfDisphp, game.PrfDispmmana, game.PrfDispmove} {
		if actor.player.GetFlags()&(1<<uint(bit)) == 0 {
			t.Errorf("hmv: display bit %d is not set", bit)
		}
	}
	for _, bit := range []int{game.PrfDispTank, game.PrfDispTarget} {
		if actor.player.GetFlags()&(1<<uint(bit)) != 0 {
			t.Errorf("hmv: display bit %d is unexpectedly set", bit)
		}
	}

	if err := cmdPrompt(actor, []string{"none"}); err != nil {
		t.Fatalf("cmdPrompt none: %v", err)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("none response = %q, want %q", got, want)
	}
	for _, bit := range []int{game.PrfDisphp, game.PrfDispmmana, game.PrfDispmove, game.PrfDispTank, game.PrfDispTarget} {
		if actor.player.GetFlags()&(1<<uint(bit)) != 0 {
			t.Errorf("none: display bit %d is still set", bit)
		}
	}

	if err := cmdPrompt(actor, []string{"on"}); err != nil {
		t.Fatalf("cmdPrompt on: %v", err)
	}
	if got, want := readMsgText(t, actor), "Okay.\r\n"; got != want {
		t.Fatalf("on response = %q, want %q", got, want)
	}
}
