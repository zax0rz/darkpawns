package session

import (
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestLastRegistrationUsesCEntryGate(t *testing.T) {
	gate, ok := commandGates["last"]
	if !ok {
		t.Fatal("last command has no C gate")
	}
	if gate.MinLevel != LVL_GOD-1 || gate.MinPosition != combat.PosDead {
		t.Fatalf("last gate = level %d position %d, want level %d position %d", gate.MinLevel, gate.MinPosition, LVL_GOD-1, combat.PosDead)
	}

	entry, ok := cmdRegistry.Lookup("last")
	if !ok {
		t.Fatal("last command is not registered")
	}
	if entry.MinLevel != gate.MinLevel || entry.MinPosition != gate.MinPosition {
		t.Fatalf("last registry gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, gate.MinLevel, gate.MinPosition)
	}
}

func TestLastUsesCOneArgumentAndMissingPlayerText(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Lastgod", LVL_IMPL, 1001)

	if err := cmdLast(s, []string{"the", "Nobody", "trailing"}); err != nil {
		t.Fatalf("cmdLast: %v", err)
	}
	if got, want := readSessionText(t, s), "There is no such player.\r\n"; got != want {
		t.Fatalf("cmdLast missing target = %q, want %q", got, want)
	}
}

func TestLastNoDatabaseFindsOnlinePlayer(t *testing.T) {
	m := makeTestManager(t)
	target := makeTestSession(t, m, "Lastpeer", 1001, true)
	target.remoteIP = "127.0.0.1"
	target.connectedAt = time.Date(2026, time.September, 6, 11, 44, 17, 0, time.UTC)
	target.player = game.NewCharacter(2, "Lastpeer", game.ClassWarrior, game.RaceHuman)
	if err := m.world.AddPlayer(target.player); err != nil {
		t.Fatalf("add player: %v", err)
	}
	m.mu.Lock()
	m.sessions[target.playerName] = target
	m.mu.Unlock()

	actor := makeTestSession(t, m, "Lastactor", 1001, true)
	actor.player = game.NewCharacter(1, "Lastactor", game.ClassWarrior, game.RaceHuman)
	actor.player.Level = game.LVL_IMPL
	if err := cmdLast(actor, []string{"the", "LASTPEER", "ignored"}); err != nil {
		t.Fatalf("cmdLast: %v", err)
	}
	got := readSessionText(t, actor)
	if !strings.Contains(got, "[    2] [ 1 Wa] Lastpeer") {
		t.Fatalf("online last lookup = %q, want C-shaped player row", got)
	}
}
