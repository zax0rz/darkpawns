package session

import (
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestPlayingDescriptorBodiesUsesNewestFirstAndSwitchedBody(t *testing.T) {
	base := time.Date(2026, time.October, 18, 12, 0, 0, 0, time.UTC)
	olderPlayer := game.NewPlayer(1, "Older", 1001)
	newerOwner := game.NewPlayer(2, "Owner", 1001)
	switchedMob := game.NewMob(&parser.Mob{VNum: 4207, Keywords: "bhyroga", ShortDesc: "a Bhyroga"}, 1001)
	newer := &Session{
		player: newerOwner, authenticated: true, connectedAt: base.Add(time.Second),
		isSwitched: true, switchedMob: switchedMob,
	}
	older := &Session{player: olderPlayer, authenticated: true, connectedAt: base}
	menu := &Session{player: game.NewPlayer(3, "Menu", 1001), authenticated: true, connectedAt: base.Add(2 * time.Second), menuActive: true}
	manager := &Manager{sessions: map[string]*Session{
		"older": older,
		"newer": newer,
		"menu":  menu,
	}}

	got := manager.playingDescriptorBodies()
	want := []game.Actor{switchedMob, olderPlayer}
	if len(got) != len(want) {
		t.Fatalf("playing bodies=%d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("body[%d]=%T %q, want %T %q", i, got[i], got[i].GetName(), want[i], want[i].GetName())
		}
	}
}
