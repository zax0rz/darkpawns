package game

import (
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
	if got, want := messagesForPlayer(received, peer.Name), []string{"You blink and suddenly realize that Player1 is gone.\r\n"}; !equalStrings(got, want) {
		t.Fatalf("threshold audience messages = %q, want exactly %q", got, want)
	}

	received = nil
	w.DoInvis(actor, "")
	if got := actor.GetInvisLevel(); got != 0 {
		t.Fatalf("invis level after toggle off = %d, want 0", got)
	}
	if got, want := messagesForPlayer(received, peer.Name), []string{"You feel a strange presence as Player1 appears, seemingly from nowhere.\r\n"}; !equalStrings(got, want) {
		t.Fatalf("appear audience messages = %q, want exactly %q", got, want)
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

func messagesForPlayer(messages []struct {
	player string
	msg    string
}, player string,
) []string {
	var result []string
	for _, message := range messages {
		if message.player == player {
			result = append(result, message.msg)
		}
	}
	return result
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
