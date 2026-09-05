package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSendRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := cmdRegistry.Lookup("send")
	if !ok {
		t.Fatal("send is not registered")
	}
	if entry.MinLevel != LVL_GOD || entry.MinPosition != combat.PosSleeping {
		t.Fatalf("send gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_GOD, combat.PosSleeping)
	}
}

func makeSendTestSessions(t *testing.T) (*Manager, *Session, *Session) {
	t.Helper()
	m := makeTestManager(t)
	actor := m.NewSession()
	actor.player = game.NewPlayer(1, "God", 1001)
	actor.player.Level = LVL_GOD
	actor.playerName = "God"
	actor.authenticated = true
	target := m.NewSession()
	target.player = game.NewPlayer(2, "Target", 1001)
	target.playerName = "Target"
	target.authenticated = true
	if err := m.world.AddPlayer(actor.player); err != nil {
		t.Fatalf("add actor: %v", err)
	}
	if err := m.world.AddPlayer(target.player); err != nil {
		t.Fatalf("add target: %v", err)
	}
	m.mu.Lock()
	m.sessions[actor.player.Name] = actor
	m.sessions[target.player.Name] = target
	m.mu.Unlock()
	return m, actor, target
}

func TestSendTextMatchesCHalfChopAndAudiences(t *testing.T) {
	_, actor, target := makeSendTestSessions(t)

	if err := cmdSendText(actor, nil, ""); err != nil {
		t.Fatalf("empty send: %v", err)
	}
	if got := readSessionText(t, actor); got != "Send what to who?\r\n" {
		t.Fatalf("empty send = %q, want exact C prompt", got)
	}

	if err := cmdSendText(actor, nil, "Ta hello  world  "); err != nil {
		t.Fatalf("spaced send: %v", err)
	}
	if got := readSessionText(t, target); got != "hello  world  \r\n" {
		t.Errorf("target message = %q, want raw half_chop remainder", got)
	}
	if got := readSessionText(t, actor); got != "You send 'hello  world  ' to Target.\r\n" {
		t.Errorf("actor confirmation = %q, want normal C confirmation", got)
	}

	if err := cmdSendText(actor, nil, "Target"); err != nil {
		t.Fatalf("empty-message send: %v", err)
	}
	if got := readSessionText(t, target); got != "\r\n" {
		t.Errorf("empty target message = %q, want CRLF", got)
	}
	if got := readSessionText(t, actor); got != "You send '' to Target.\r\n" {
		t.Errorf("empty-message confirmation = %q, want exact C confirmation", got)
	}
}

func TestSendTextMatchesCNotFoundSelfAndNoRepeat(t *testing.T) {
	_, actor, target := makeSendTestSessions(t)

	if err := cmdSendText(actor, nil, "nobody hello"); err != nil {
		t.Fatalf("missing target: %v", err)
	}
	if got := readSessionText(t, actor); got != "No-one by that name here.\r\n" {
		t.Errorf("missing target = %q, want C NOPERSON", got)
	}

	if err := cmdSendText(actor, nil, "me self-message"); err != nil {
		t.Fatalf("self target: %v", err)
	}
	if got := readSessionText(t, actor); got != "self-message\r\n" {
		t.Errorf("self message = %q, want victim copy first", got)
	}
	if got := readSessionText(t, actor); got != "You send 'self-message' to God.\r\n" {
		t.Errorf("self confirmation = %q, want actor copy second", got)
	}

	actor.player.SetPlrFlag(game.PrfNoRepeat, true)
	if err := cmdSendText(actor, nil, "Target quiet  message"); err != nil {
		t.Fatalf("norepeat send: %v", err)
	}
	if got := readSessionText(t, target); got != "quiet  message\r\n" {
		t.Errorf("norepeat target message = %q, want raw message", got)
	}
	if got := readSessionText(t, actor); got != "Sent.\r\n" {
		t.Errorf("norepeat actor confirmation = %q, want Sent.", got)
	}
}
