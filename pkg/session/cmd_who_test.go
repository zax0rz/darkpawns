package session

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func addWhoSession(t *testing.T, manager *Manager, name string, level, class, room int, connected time.Time) *Session {
	t.Helper()
	session := makeTestSession(t, manager, name, room, true)
	session.connectedAt = connected
	session.player.SetLevel(level)
	session.player.Class = class
	if class >= 0 && class < len(game.Titles) {
		game.SetTitle(session.player, game.Titles[class])
	}
	manager.mu.Lock()
	manager.sessions[strings.ToLower(name)] = session
	manager.mu.Unlock()
	return session
}

func runWho(t *testing.T, viewer *Session, args ...string) string {
	t.Helper()
	if err := cmdWho(viewer, args); err != nil {
		t.Fatalf("cmdWho(%q): %v", args, err)
	}
	return readSessionText(t, viewer)
}

func TestCmdWhoMortalFormatAndDescriptorOrder(t *testing.T) {
	manager := makeTestManager(t)
	now := time.Now()
	actor := addWhoSession(t, manager, "Cviewactor", 1, game.ClassWarrior, 1001, now)
	addWhoSession(t, manager, "Cviewobs", 1, game.ClassWarrior, 1001, now.Add(time.Second))

	want := "Players\r\n-------\r\n" +
		"[  1  Wa ] Cviewobs the Warrior\r\n" +
		"[  1  Wa ] Cviewactor the Warrior\r\n" +
		"\r\n2 characters displayed.\r\n"
	if got := runWho(t, actor); got != want {
		t.Fatalf("who output:\n%q\nwant:\n%q", got, want)
	}
}

func TestCmdWhoFilters(t *testing.T) {
	manager := makeTestManager(t)
	room, ok := manager.world.GetRoom(1002)
	if !ok {
		t.Fatal("missing room 1002")
	}
	room.Zone = 2
	now := time.Now()
	viewer := addWhoSession(t, manager, "Viewer", 10, game.ClassWarrior, 1001, now)
	questor := addWhoSession(t, manager, "Questor", 5, game.ClassMageUser, 1001, now.Add(time.Second))
	questor.player.Title = "the Quiet Hero"
	questor.player.SetPlrFlag(game.PrfQuest, true)
	outlaw := addWhoSession(t, manager, "Outlaw", 8, game.ClassThief, 1002, now.Add(2*time.Second))
	outlaw.player.SetPlrFlag(game.PlrOutlaw, true)
	addWhoSession(t, manager, "Remote", 20, game.ClassCleric, 1002, now.Add(3*time.Second))
	hidden := addWhoSession(t, manager, "Hidden", 6, game.ClassWarrior, 1001, now.Add(4*time.Second))
	hidden.player.SetAffectBit(1, true) // AFF_INVISIBLE

	tests := []struct {
		name    string
		args    []string
		present []string
		absent  []string
	}{
		{name: "dash level range", args: []string{"-l", "5-5"}, present: []string{"Questor"}, absent: []string{"Viewer", "Outlaw", "Remote", "Hidden"}},
		{name: "bare level range", args: []string{"5", "5"}, present: []string{"Questor"}, absent: []string{"Viewer", "Outlaw", "Remote", "Hidden"}},
		{name: "name in title", args: []string{"-n", "Quiet"}, present: []string{"Questor"}, absent: []string{"Viewer", "Outlaw", "Remote", "Hidden"}},
		{name: "class", args: []string{"-c", "m"}, present: []string{"Questor"}, absent: []string{"Viewer", "Outlaw", "Remote", "Hidden"}},
		{name: "quest", args: []string{"-q"}, present: []string{"Questor", "(quest)"}, absent: []string{"Viewer", "Outlaw", "Remote", "Hidden"}},
		{name: "outlaw", args: []string{"-o"}, present: []string{"Outlaw", "(OUTLAW)"}, absent: []string{"Viewer", "Questor", "Remote", "Hidden"}},
		{name: "same zone", args: []string{"-r"}, present: []string{"Viewer", "Questor"}, absent: []string{"Outlaw", "Remote", "Hidden"}},
		{name: "same room", args: []string{"-z"}, present: []string{"Viewer", "Questor"}, absent: []string{"Outlaw", "Remote", "Hidden"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := runWho(t, viewer, test.args...)
			for _, want := range test.present {
				if !strings.Contains(got, want) {
					t.Errorf("who %q missing %q in %q", test.args, want, got)
				}
			}
			for _, unwanted := range test.absent {
				if strings.Contains(got, unwanted) {
					t.Errorf("who %q unexpectedly contains %q in %q", test.args, unwanted, got)
				}
			}
		})
	}
}

func TestCmdWhoVisibilityAndNoWhoRoom(t *testing.T) {
	t.Run("invisible and hidden", func(t *testing.T) {
		manager := makeTestManager(t)
		now := time.Now()
		viewer := addWhoSession(t, manager, "Viewer", 10, game.ClassWarrior, 1001, now)
		invisible := addWhoSession(t, manager, "Invisible", 10, game.ClassWarrior, 1001, now.Add(time.Second))
		invisible.player.SetAffectBit(1, true) // AFF_INVISIBLE
		hidden := addWhoSession(t, manager, "Hidden", 10, game.ClassWarrior, 1001, now.Add(2*time.Second))
		hidden.player.SetAffectBit(19, true) // AFF_HIDE

		got := runWho(t, viewer)
		if strings.Contains(got, "Invisible") || strings.Contains(got, "Hidden") {
			t.Fatalf("mortal saw invisible/hidden players: %q", got)
		}
	})

	t.Run("no who room", func(t *testing.T) {
		manager := makeTestManager(t)
		room, ok := manager.world.GetRoom(1002)
		if !ok {
			t.Fatal("missing room 1002")
		}
		room.Flags = []string{"0", "8", "0", "0"} // bit 19, ROOM_NO_WHO_ROOM
		now := time.Now()
		viewer := addWhoSession(t, manager, "Viewer", 10, game.ClassWarrior, 1001, now)
		addWhoSession(t, manager, "Secret", 10, game.ClassWarrior, 1002, now.Add(time.Second))

		if got := runWho(t, viewer); strings.Contains(got, "Secret") {
			t.Fatalf("mortal saw player in NO_WHO_ROOM: %q", got)
		}
		viewer.player.SetLevel(LVL_IMPL)
		if got := runWho(t, viewer); !strings.Contains(got, "Secret") {
			t.Fatalf("implementor did not see player in NO_WHO_ROOM: %q", got)
		}
	})
}

func TestWhoClassAbbreviations(t *testing.T) {
	want := []string{"Mu", "Cl", "Th", "Wa", "Ma", "Av", "As", "Pa", "Ni", "Ps", "Ra", "My"}
	for class, abbrev := range want {
		if got := whoClassAbbrev(class); got != abbrev {
			t.Errorf("whoClassAbbrev(%d) = %q, want %q", class, got, abbrev)
		}
	}
}

func TestWhoImmortalRanksAndShortColor(t *testing.T) {
	ranks := []struct {
		level int
		want  string
	}{
		{LVL_IMMORT, "[ IMMORT ]"},
		{LVL_IMMORT + 1, "[ TITAN  ]"},
		{LVL_GOD, "[  GOD   ]"},
		{LVL_LEGEND, "[ LEGEND ]"},
		{LVL_HIGOD, "[ HIGOD  ]"},
		{LVL_GRGOD, "[ GRGOD  ]"},
		{LVL_IMPL, "[ *IMP*  ]"},
	}
	for _, rank := range ranks {
		if got := whoShortRank(rank.level); got != rank.want {
			t.Errorf("whoShortRank(%d) = %q, want %q", rank.level, got, rank.want)
		}
	}

	viewer := game.NewPlayer(1, "Viewer", 1001)
	viewer.SetPlrFlag(game.PrfColor1, true)
	immortal := game.NewPlayer(2, "Immortal", 1001)
	immortal.SetLevel(LVL_IMMORT)
	if got, want := renderWhoShort(viewer, immortal), "\x1b[33m[ IMMORT ] Immortal    \x1b[0m"; got != want {
		t.Errorf("colored short who = %q, want %q", got, want)
	}
	if got := renderWhoLong(nil, immortal); !strings.HasPrefix(got, "[ Wizard ] Immortal ") {
		t.Errorf("non-short immortal should use executable C's Wizard label, got %q", got)
	}
}

func TestCmdWhoShortAndBadFormat(t *testing.T) {
	manager := makeTestManager(t)
	now := time.Now()
	viewer := addWhoSession(t, manager, "Viewer", 1, game.ClassWarrior, 1001, now)
	addWhoSession(t, manager, "Observer", 1, game.ClassWarrior, 1001, now.Add(time.Second))

	wantShort := "Players\r\n-------\r\n" +
		"[  1  Wa ] Observer    [  1  Wa ] Viewer      [  1  Wa ] Viewer      \r\n" +
		"2 characters displayed.\r\n\r\n"
	if got := runWho(t, viewer, "-s"); got != wantShort {
		t.Errorf("short who = %q", got)
	}
	if got := runWho(t, viewer, "-x"); got != whoFormat {
		t.Errorf("bad who option = %q, want %q", got, whoFormat)
	}

	// Exercise the registry boundary: arguments must reach cmdWho rather than
	// being dropped by wrapNoArgs.
	if err := ExecuteCommand(viewer, "who", []string{"-x"}); err != nil {
		t.Fatalf("ExecuteCommand who: %v", err)
	}
	if got := readSessionText(t, viewer); got != whoFormat {
		t.Errorf("registered who discarded args: %q", got)
	}
}

func TestParseWhoClassLettersCaseInsensitive(t *testing.T) {
	opts, ok := parseWhoArgs([]string{"-c", "Mw"})
	if !ok {
		t.Fatal("parseWhoArgs rejected valid class list")
	}
	want := game.FindClassBitvector('m') | game.FindClassBitvector('w')
	if opts.classMask != want {
		t.Errorf("class mask = %d, want %d", opts.classMask, want)
	}
	if _, ok := parseWhoArgs([]string{fmt.Sprintf("%d-bad", LVL_IMMORT)}); ok {
		t.Error("parseWhoArgs accepted malformed level range")
	}
}
