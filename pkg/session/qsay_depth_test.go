package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestQsayRegistrationUsesCEntryGate(t *testing.T) {
	entry, ok := commandGates["qsay"]
	if !ok {
		t.Fatal("qsay command has no C gate")
	}
	if entry.MinLevel != 0 || entry.MinPosition != combat.PosSleeping {
		t.Fatalf("qsay gate = level %d position %d, want level 0 position %d", entry.MinLevel, entry.MinPosition, combat.PosSleeping)
	}
}

func TestQsayBroadcastSkipsWritingQuestPlayer(t *testing.T) {
	m := makeTestManager(t)
	actor := makeTestSession(t, m, "QsayActor", 1001, true)
	peer := makeTestSession(t, m, "QsayPeer", 1001, true)
	writing := makeTestSession(t, m, "QsayWriter", 1001, true)
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
