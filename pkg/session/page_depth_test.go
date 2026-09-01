package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPageRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["page"]
	if !ok {
		t.Fatal("page command has no C gate")
	}
	if gate.MinLevel != game.LVL_IMMORT || gate.MinPosition != combat.PosDead {
		t.Fatalf("page gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, game.LVL_IMMORT, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("page")
	if !ok {
		t.Fatal("page command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("page registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestPageAllRequiresStrictlyAboveGod(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Pageactor", game.LVL_GOD, 1001)

	if err := ExecuteCommand(actor, "page", []string{"all", "message"}); err != nil {
		t.Fatalf("ExecuteCommand(page all): %v", err)
	}
	if got := readSessionText(t, actor); got != "You will never be godly enough to do that!" {
		t.Fatalf("page all level gate = %q, want C refusal", got)
	}
}

func TestPageResolvesCaseInsensitiveAndSuppressesSleepingTarget(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Pageactor", game.LVL_IMMORT, 1001)
	awake := makeCommandTestSession(t, m, "Pagetarget", 1, 1001)
	sleeping := makeCommandTestSession(t, m, "Pagesleep", 1, 1001)
	m.sessions[actor.player.Name] = actor
	m.sessions[awake.player.Name] = awake
	m.sessions[sleeping.player.Name] = sleeping
	sleeping.player.SetPosition(combat.PosSleeping)

	if err := cmdPageText(actor, "pagetarget Alpha  beta  gamma  "); err != nil {
		t.Fatalf("cmdPageText(case-insensitive): %v", err)
	}
	if got := readSessionText(t, awake); got != "\a\a*Pageactor* Alpha  beta  gamma  \r\n\r\n" {
		t.Fatalf("awake target output = %q, want C page bytes", got)
	}
	if got := readSessionText(t, actor); got != "\a\a*Pageactor* Alpha  beta  gamma  \r\n\r\n" {
		t.Fatalf("actor output = %q, want C page bytes", got)
	}

	if err := cmdPageText(actor, "Pagesleep sleeping message"); err != nil {
		t.Fatalf("cmdPageText(sleeping): %v", err)
	}
	if got := drainSendChannel(t, sleeping); got != "" {
		t.Fatalf("sleeping target output = %q, want suppressed by C SENDOK", got)
	}
	if got := readSessionText(t, actor); got != "\a\a*Pageactor* sleeping message\r\n\r\n" {
		t.Fatalf("sleeping-target actor output = %q, want C page bytes", got)
	}
}

func TestPageSelfAliasesDeliverTwoCActMessages(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Pageactor", game.LVL_IMMORT, 1001)
	m.sessions[actor.player.Name] = actor

	if err := cmdPageText(actor, "me self message"); err != nil {
		t.Fatalf("cmdPageText(self alias): %v", err)
	}
	want := "\a\a*Pageactor* self message\r\n\r\n"
	if got := readSessionText(t, actor); got != want {
		t.Fatalf("first self page output = %q, want one C act message", got)
	}
	if got := readSessionText(t, actor); got != want {
		t.Fatalf("second self page output = %q, want one C act message", got)
	}
}
