package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func switchGateFixture(t *testing.T) (*Manager, *Session, *Session) {
	t.Helper()
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Room A"}, {VNum: 1002, Name: "Room B"}},
		Objs:  []parser.Obj{{VNum: 8023, Keywords: "club", ShortDesc: "a club", WearFlags: [4]int{1}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	wizard := makeTestSession(t, m, "Wizard", 1001, true)
	wizard.player.SetLevel(LVL_IMPL)
	wizard.transportDone = make(chan struct{})
	registerTestSession(t, m, wizard, "Wizard")
	holder := makeTestSession(t, m, "Borrowed", 1002, true)
	holder.player = game.NewPlayer(2, "Borrowed", 1002)
	holder.transportDone = make(chan struct{})
	holder.DetachTransport()
	holder.player.SetLinkless(true)
	registerTestSession(t, m, holder, "Borrowed")
	return m, wizard, holder
}

func TestSwitchCanonicalVisibleWorldGate(t *testing.T) {
	_, s, h := switchGateFixture(t)
	if err := cmdSwitch(s, []string{"the", "Borrowed", "ignored"}); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(s); got != "Okay.\r\n" {
		t.Fatalf("remote fill-word target = %q", got)
	}
	if s.switchedPlayer != h.player {
		t.Fatal("wrong resolved body")
	}
}

func TestSwitchLinkdeadLevelGate(t *testing.T) {
	_, s, _ := switchGateFixture(t)
	s.player.SetLevel(38)
	if err := cmdSwitch(s, []string{"Borrowed"}); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(s); got != "You aren't holy enough to use a mortal's body.\r\n" {
		t.Fatalf("linkdead level refusal = %q", got)
	}
	if s.isSwitched {
		t.Fatal("refusal changed ownership")
	}
}

func TestSwitchLiveInvisibleAndSelfGates(t *testing.T) {
	_, s, h := switchGateFixture(t)
	h.transportDone = make(chan struct{})
	if err := cmdSwitch(s, []string{"Borrowed"}); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(s); !strings.Contains(got, "already in use") {
		t.Fatalf("live descriptor = %q", got)
	}
	h.player.SetInvisLevel(41)
	if err := cmdSwitch(s, []string{"Borrowed"}); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(s); got != "No such character.\r\n" {
		t.Fatalf("invisible target = %q", got)
	}
	if err := cmdSwitch(s, []string{"Wizard"}); err != nil {
		t.Fatal(err)
	}
	if got := renderedOutput(s); got != "Hee hee... we are jolly funny today, eh?\r\n" {
		t.Fatalf("self target = %q", got)
	}
}
