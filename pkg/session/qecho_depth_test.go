package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestQechoRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["qecho"]
	if !ok {
		t.Fatal("qecho command has no C gate")
	}
	if entry.MinLevel != LVL_IMMORT || entry.MinPosition != combat.PosDead {
		t.Fatalf("qecho gate = level %d position %d, want level %d position %d", entry.MinLevel, entry.MinPosition, LVL_IMMORT, combat.PosDead)
	}
}

func TestQechoBroadcastSkipsWritingQuestPlayer(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "QechoActor", 1001, true)
	peer := makeTestSession(t, m, "QechoPeer", 1001, true)
	writing := makeTestSession(t, m, "QechoWriter", 1001, true)
	peer.player.SetPlrFlag(game.PrfQuest, true)
	writing.player.SetPlrFlag(game.PrfQuest, true)
	writing.player.SetPlrFlag(game.PlrWriting, true)

	m.mu.Lock()
	m.sessions[actor.player.Name] = actor
	m.sessions[peer.player.Name] = peer
	m.sessions[writing.player.Name] = writing
	m.mu.Unlock()

	broadcastQuest(actor, "self", "other")
	if got := drainSendChannel(t, peer); !strings.Contains(got, `"text":"other"`) {
		t.Fatalf("questing peer output = %q, want other", got)
	}
	if got := drainSendChannel(t, writing); got != "" {
		t.Fatalf("writing questing peer output = %q, want empty", got)
	}
}
