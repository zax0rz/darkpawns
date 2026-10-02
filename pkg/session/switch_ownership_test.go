package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSwitchAttachedBodyLookup(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Wizard", 1001, true)
	s.transportDone = make(chan struct{})
	registerTestSession(t, m, s, "Wizard")
	if m.attachedBody(s.player) != s {
		t.Fatal("ordinary live descriptor not found")
	}
	s.DetachTransport()
	if m.attachedBody(s.player) != nil {
		t.Fatal("linkdead holder is not a descriptor")
	}
	original := s.player
	body := game.NewPlayer(2, "Borrowed", 1002)
	s.transportDone = make(chan struct{})
	s.player = body
	s.isSwitched = true
	s.switchedOriginal = original
	s.switchedPlayer = body
	if m.attachedBody(body) != s || m.attachedBody(original) != nil {
		t.Fatal("attached/original identity confused")
	}
	if got, _ := m.GetSession("Wizard"); got != s {
		t.Fatal("identity lookup changed")
	}
}
