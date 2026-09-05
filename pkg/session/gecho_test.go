package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestGechoRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["gecho"]
	if !ok {
		t.Fatal("gecho command has no C gate")
	}
	if entry.MinLevel != LVL_GOD || entry.MinPosition != combat.PosDead {
		t.Fatalf("gecho gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_GOD, combat.PosDead)
	}
}

func TestCommandArgumentTextPreservesCSpacing(t *testing.T) {
	if got, want := CommandArgumentText("  gecho Alpha  beta  "), "Alpha  beta  "; got != want {
		t.Fatalf("CommandArgumentText = %q, want %q", got, want)
	}
	if got := CommandArgumentText("gecho"); got != "" {
		t.Fatalf("CommandArgumentText without argument = %q, want empty", got)
	}
}

func TestGechoNoRepeatUsesCOkayAndExcludesActor(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Wizard", LVL_GOD, 1001)
	recipient := makeCommandTestSession(t, m, "Recipient", 1, 1001)
	actor.player.SetPlrFlag(game.PrfNoRepeat, true)
	m.sessions[actor.player.Name] = actor
	m.sessions[recipient.player.Name] = recipient

	if err := cmdGechoText(actor, "Quiet  signal  "); err != nil {
		t.Fatal(err)
	}
	if got := readMsgText(t, actor); got != "Okay.\r\n" {
		t.Fatalf("actor output = %q, want C OK", got)
	}
	if got := readMsgText(t, recipient); got != "Quiet  signal  " {
		t.Fatalf("recipient output = %q, want exact global message", got)
	}
}

func TestGechoQueuedInputPreservesRawArgs(t *testing.T) {
	m := makeTestManager(t)
	actor := makeCommandTestSession(t, m, "Wizard", LVL_GOD, 1001)
	recipient := makeCommandTestSession(t, m, "Recipient", 1, 1001)
	m.sessions[actor.player.Name] = actor
	m.sessions[recipient.player.Name] = recipient
	actor.player.SetWaitStatePulses(1)

	if !actor.tryExecuteNow("gecho", []string{"Queued", "signal"}, "Queued  signal  ") {
		t.Fatal("gecho should queue while the actor has wait state")
	}
	m.DrainInputQueues()

	if got := readMsgText(t, actor); got != "Queued  signal  " {
		t.Fatalf("actor output = %q, want exact queued global message", got)
	}
	if got := readMsgText(t, recipient); got != "Queued  signal  " {
		t.Fatalf("recipient output = %q, want exact queued global message", got)
	}
}
