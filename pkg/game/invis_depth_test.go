package game

import (
	"strings"
	"testing"
)

func TestDoInvisUsesLevelAndThresholdAudience(t *testing.T) {
	w, players := newMessageTestWorld(t)
	actor, peer := players[0], players[1]
	actor.SetLevel(LVL_IMPL)
	peer.SetLevel(30)

	var received []struct {
		player string
		msg    string
	}
	w.MessageSink = func(playerName string, msg []byte) {
		received = append(received, struct {
			player string
			msg    string
		}{playerName, string(msg)})
	}

	w.DoInvis(actor, "")
	if got := actor.GetInvisLevel(); got != LVL_IMPL {
		t.Fatalf("invis level after toggle on = %d, want %d", got, LVL_IMPL)
	}
	if !hasPlayerMessage(received, peer.Name, "You blink and suddenly realize that Player1 is gone.\r\n") {
		t.Fatalf("threshold audience missing from %v", received)
	}

	received = nil
	w.DoInvis(actor, "")
	if got := actor.GetInvisLevel(); got != 0 {
		t.Fatalf("invis level after toggle off = %d, want 0", got)
	}
	if !hasPlayerMessage(received, peer.Name, "You feel a strange presence as Player1 appears, seemingly from nowhere.\r\n") {
		t.Fatalf("appear audience missing from %v", received)
	}
}

func TestCanSeeHonorsWizinvisLevelBeforeImmortalShortcut(t *testing.T) {
	_, players := newMessageTestWorld(t)
	actor, peer := players[0], players[1]
	actor.SetLevel(LVL_IMPL)
	peer.SetLevel(LVL_IMMORT)
	actor.SetInvisLevel(LVL_IMPL)

	if CanSee(peer, actor) {
		t.Fatal("level-31 observer should not see level-40 wizinvis actor")
	}

	peer.SetLevel(LVL_IMPL)
	if !CanSee(peer, actor) {
		t.Fatal("equal-level observer should see wizinvis actor")
	}
}

func hasPlayerMessage(messages []struct {
	player string
	msg    string
}, player, want string,
) bool {
	for _, message := range messages {
		if message.player == player && strings.Contains(message.msg, want) {
			return true
		}
	}
	return false
}
