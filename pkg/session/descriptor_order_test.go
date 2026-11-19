package session

import (
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestSessionsInDescriptorOrderNewestFirst(t *testing.T) {
	ts := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)

	s1 := &Session{playerName: "Alpha", connectionNumber: 1, connectedAt: ts.Add(2 * time.Second), authenticated: true, player: game.NewPlayer(1, "Alpha", 1001)}
	s2 := &Session{playerName: "Beta", connectionNumber: 2, connectedAt: ts, authenticated: true, player: game.NewPlayer(2, "Beta", 1001)}
	s3 := &Session{playerName: "Gamma", connectionNumber: 3, connectedAt: ts, authenticated: true, player: game.NewPlayer(3, "Gamma", 1001)}
	sMenu := &Session{playerName: "Menu", connectionNumber: 4, connectedAt: ts, authenticated: true, menuActive: true, player: game.NewPlayer(4, "Menu", 1001)}

	m := &Manager{sessions: map[string]*Session{
		"1": s1,
		"2": s2,
		"3": s3,
		"4": sMenu,
	}}

	all := m.sessionsInDescriptorOrder()
	wantAll := []*Session{sMenu, s3, s2, s1}
	if len(all) != len(wantAll) {
		t.Fatalf("sessionsInDescriptorOrder len=%d, want %d", len(all), len(wantAll))
	}
	for i := range wantAll {
		if all[i] != wantAll[i] {
			t.Fatalf("all[%d]=%q (conn %d), want %q (conn %d)",
				i, all[i].playerName, all[i].connectionNumber, wantAll[i].playerName, wantAll[i].connectionNumber)
		}
	}

	playing := m.playingSessionsInDescriptorOrder()
	wantPlaying := []*Session{s3, s2, s1}
	if len(playing) != len(wantPlaying) {
		t.Fatalf("playingSessionsInDescriptorOrder len=%d, want %d", len(playing), len(wantPlaying))
	}
	for i := range wantPlaying {
		if playing[i] != wantPlaying[i] {
			t.Fatalf("playing[%d]=%q (conn %d), want %q (conn %d)",
				i, playing[i].playerName, playing[i].connectionNumber, wantPlaying[i].playerName, wantPlaying[i].connectionNumber)
		}
	}
}

func TestWhoWhereWiznetUseDescriptorOrderWhenTimestampsTie(t *testing.T) {
	ts := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)

	manager := makeTestManager(t)
	s1 := addWhoSession(t, manager, "Firstgod", LVL_IMPL, game.ClassWarrior, 1001, ts)
	s1.connectionNumber = 1
	s2 := addWhoSession(t, manager, "Secondgod", LVL_GOD, game.ClassWarrior, 1001, ts)
	s2.connectionNumber = 2
	s3 := addWhoSession(t, manager, "Thirdgod", LVL_GOD, game.ClassWarrior, 1001, ts)
	s3.connectionNumber = 3

	// Run 10 trials so random map iteration cannot pass by accident.
	for trial := range 10 {
		// 1. wiznetSessions must return connectionNumber descending: 3, 2, 1.
		gotWiz := wiznetSessions(manager)
		if len(gotWiz) != 3 || gotWiz[0] != s3 || gotWiz[1] != s2 || gotWiz[2] != s1 {
			t.Fatalf("trial %d: wiznetSessions order = [%s, %s, %s], want [Thirdgod, Secondgod, Firstgod]",
				trial, gotWiz[0].playerName, gotWiz[1].playerName, gotWiz[2].playerName)
		}

		// 2. cmdWhoForBody must list Thirdgod before Secondgod before Firstgod.
		whoOut := runWho(t, s1)
		i3, i2, i1 := strings.Index(whoOut, "Thirdgod"), strings.Index(whoOut, "Secondgod"), strings.Index(whoOut, "Firstgod")
		if i3 < 0 || i2 < 0 || i1 < 0 || i3 >= i2 || i2 >= i1 {
			t.Fatalf("trial %d: cmdWhoForBody order wrong (indices 3=%d 2=%d 1=%d):\n%s", trial, i3, i2, i1, whoOut)
		}

		// 3. cmdWhere must list Thirdgod before Secondgod before Firstgod.
		if err := cmdWhere(s1, nil); err != nil {
			t.Fatalf("trial %d: cmdWhere: %v", trial, err)
		}
		whereOut := readSessionText(t, s1)
		i3, i2, i1 = strings.Index(whereOut, "Thirdgod"), strings.Index(whereOut, "Secondgod"), strings.Index(whereOut, "Firstgod")
		if i3 < 0 || i2 < 0 || i1 < 0 || i3 >= i2 || i2 >= i1 {
			t.Fatalf("trial %d: cmdWhere order wrong (indices 3=%d 2=%d 1=%d):\n%s", trial, i3, i2, i1, whereOut)
		}
	}
}

func TestDrainInputPassUsesDescriptorOrder(t *testing.T) {
	ts := time.Date(2026, time.November, 19, 12, 0, 0, 0, time.UTC)

	manager := makeTestManager(t)
	sObs := addWhoSession(t, manager, "Observer", 1, game.ClassWarrior, 1001, ts)
	sObs.connectionNumber = 0
	sObs.player.ID = 10
	manager.world.AddPlayer(sObs.player)
	s1 := addWhoSession(t, manager, "First", 1, game.ClassWarrior, 1001, ts)
	s1.connectionNumber = 1
	s1.player.ID = 1
	s1.player.Stats.Int = 15
	s1.player.Stats.Wis = 15
	s1.player.CopyBaseAttributes()
	manager.world.AddPlayer(s1.player)
	s2 := addWhoSession(t, manager, "Second", 1, game.ClassWarrior, 1001, ts)
	s2.connectionNumber = 2
	s2.player.ID = 2
	s2.player.Stats.Int = 15
	s2.player.Stats.Wis = 15
	s2.player.CopyBaseAttributes()
	manager.world.AddPlayer(s2.player)
	s3 := addWhoSession(t, manager, "Third", 1, game.ClassWarrior, 1001, ts)
	s3.connectionNumber = 3
	s3.player.ID = 3
	s3.player.Stats.Int = 15
	s3.player.Stats.Wis = 15
	s3.player.CopyBaseAttributes()
	manager.world.AddPlayer(s3.player)

	for trial := range 10 {
		s1.inputQueue = append(s1.inputQueue, queuedInput{cmd: "say", args: []string{"one"}, rawArgs: "one"})
		s2.inputQueue = append(s2.inputQueue, queuedInput{cmd: "say", args: []string{"two"}, rawArgs: "two"})
		s3.inputQueue = append(s3.inputQueue, queuedInput{cmd: "say", args: []string{"three"}, rawArgs: "three"})

		manager.drainInputPass(false)

		got := drainSendChannel(t, sObs)
		got1 := drainSendChannel(t, s1)
		_ = drainSendChannel(t, s2)
		_ = drainSendChannel(t, s3)

		i3 := strings.Index(got, "Third says, 'three'")
		i2 := strings.Index(got, "Second says, 'two'")
		i1 := strings.Index(got, "First says, 'one'")
		if i3 < 0 || i2 < 0 || i1 < 0 || i3 >= i2 || i2 >= i1 {
			t.Fatalf("trial %d: drainInputPass order wrong (indices 3=%d 2=%d 1=%d):\nobs=%q\ns1=%q", trial, i3, i2, i1, got, got1)
		}
	}
}
