package session

import (
	"testing"
)

func TestBroadcastCombatMsg_NilPlayer(t *testing.T) {
	s := &Session{
		player: nil,
	}
	// This should return immediately and not panic because s.player is nil
	broadcastCombatMsg(s, 100, "hit", "some text")
}
