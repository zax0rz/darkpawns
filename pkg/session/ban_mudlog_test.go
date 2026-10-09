package session

import (
	"os"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// mudlogTestWorld builds a world whose ban file lands in a temporary working
// directory: C's write_ban_list() path is relative, and DoBan writes it.
func mudlogTestWorld(t *testing.T) (*Manager, *game.World) {
	t.Helper()
	t.Chdir(t.TempDir())
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001}, {VNum: 1002}},
		Mobs:  []parser.Mob{{VNum: 90, Keywords: "guard", ShortDesc: "a retained guard", Level: 40}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	return newTestManager(t, w, nil), w
}

// mudlogObserver registers an immortal whose syslog and level decide whether a
// producer reaches it.
func mudlogObserver(t *testing.T, m *Manager, name string, level int, flags ...int) *Session {
	t.Helper()
	s := makeTestSession(t, m, name, 1001, true)
	s.player.SetLevel(level)
	for _, flag := range flags {
		s.player.SetPlrFlag(flag, true)
	}
	registerTestSession(t, m, s, name)
	drainSessionText(t, s)
	return s
}

// ban.c:205-209 (producer, acknowledgement, file) and ban.c:237-243
// (acknowledgement, producer, file), with the NRM / MAX(LVL_GOD, invis)
// audience. The actor's own stream proves the order relative to the
// acknowledgement; the observers prove the level and type filters.
func TestBanMudlogProducerBytesAndOrder(t *testing.T) {
	m, _ := mudlogTestWorld(t)

	actor := mudlogObserver(t, m, "Godactor", 40, game.PrfLog1, game.PrfLog2)
	watcher := mudlogObserver(t, m, "Banwatch", game.LVL_GOD, game.PrfLog2)
	below := mudlogObserver(t, m, "Banbelow", game.LVL_GOD-1, game.PrfLog1, game.PrfLog2)
	brief := mudlogObserver(t, m, "Banbrief", 40, game.PrfLog1)
	file := captureMudlogFile(t)

	if err := ExecuteCommand(actor, "ban", []string{"all", "127.000.000.*"}); err != nil {
		t.Fatal(err)
	}
	banLine := "[ Godactor has banned 127.000.000.* for all players. ]\r\n"
	if got := strings.Join(drainSessionText(t, actor), ""); got != banLine+"Site banned.\r\n" {
		t.Fatalf("ban actor stream = %q, want the producer before the acknowledgement", got)
	}
	if got := strings.Join(drainSessionText(t, watcher), ""); got != banLine {
		t.Fatalf("ban observer stream = %q, want %q", got, banLine)
	}
	if got := strings.Join(drainSessionText(t, below), ""); got != "" {
		t.Fatalf("below-threshold observer saw %q", got)
	}
	if got := strings.Join(drainSessionText(t, brief), ""); got != "" {
		t.Fatalf("brief-syslog observer saw %q", got)
	}
	if !strings.Contains(file.String(), "Godactor has banned 127.000.000.* for all players.") {
		t.Fatalf("ban file line = %q", file.String())
	}
	if got := AcceptBanLevel(m.GetBanManager(), "127.000.000.001", false); got != game.BanAll {
		// The ban is written in C's wildhost spelling, so the accept-time check
		// (host + both wildcard strings) is what matches it.
		t.Fatalf("ban not enforced: accept-time level %d", got)
	}
	if _, err := readBanFileLine(t, "./data/badsites"); err != nil {
		t.Fatalf("ban file: %v", err)
	}

	file.Reset()
	if err := ExecuteCommand(actor, "unban", []string{"127.000.000.*"}); err != nil {
		t.Fatal(err)
	}
	unbanLine := "[ Godactor removed the all-player ban on 127.000.000.*. ]\r\n"
	if got := strings.Join(drainSessionText(t, actor), ""); got != "Site unbanned.\r\n"+unbanLine {
		t.Fatalf("unban actor stream = %q, want the acknowledgement before the producer", got)
	}
	if got := strings.Join(drainSessionText(t, watcher), ""); got != unbanLine {
		t.Fatalf("unban observer stream = %q, want %q", got, unbanLine)
	}
	if !strings.Contains(file.String(), "Godactor removed the all-player ban on 127.000.000.*.") {
		t.Fatalf("unban file line = %q", file.String())
	}
	if got := AcceptBanLevel(m.GetBanManager(), "127.000.000.001", false); got != game.BanNot {
		t.Fatalf("ban still enforced after unban: accept-time level %d", got)
	}
}

// MAX(LVL_GOD, GET_INVIS_LEV(ch)): an invisible actor raises the threshold above
// LVL_GOD, so an ordinary god no longer sees the line.
func TestBanMudlogInvisThreshold(t *testing.T) {
	m, _ := mudlogTestWorld(t)

	actor := mudlogObserver(t, m, "Invisactor", 40, game.PrfLog2)
	actor.player.SetInvisLevel(40)
	god := mudlogObserver(t, m, "Godwatch", game.LVL_GOD, game.PrfLog2)
	hidden := mudlogObserver(t, m, "Implwatch", 40, game.PrfLog2)

	if err := ExecuteCommand(actor, "ban", []string{"new", "evil.example"}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(drainSessionText(t, god), ""); got != "" {
		t.Fatalf("a LVL_GOD observer saw an invis-40 producer: %q", got)
	}
	line := "[ Invisactor has banned evil.example for new players. ]\r\n"
	if got := strings.Join(drainSessionText(t, hidden), ""); got != line {
		t.Fatalf("level-40 observer stream = %q, want %q", got, line)
	}
}

// R4: the producer names the acting body, not the immortal behind it. C's ch is
// the switched-into body, and for a mob its player_specials are zeroed, so
// GET_INVIS_LEV is 0 and the threshold stays LVL_GOD.
func TestMudlogActorNamesActingBody(t *testing.T) {
	t.Run("ordinary immortal", func(t *testing.T) {
		m, _ := mudlogTestWorld(t)
		actor := mudlogObserver(t, m, "Godactor", 40, game.PrfLog2)
		actor.player.SetInvisLevel(37)
		name, invis := actor.mudlogActor()
		if name != "Godactor" || invis != 37 {
			t.Fatalf("mudlogActor() = %q/%d, want Godactor/37", name, invis)
		}
	})

	t.Run("switched into a player", func(t *testing.T) {
		m, _ := mudlogTestWorld(t)
		actor := mudlogObserver(t, m, "Godactor", 40, game.PrfLog2)
		borrowed := makeTestSession(t, m, "Borrowed", 1002, true)
		borrowed.player = game.NewPlayer(2, "Borrowed", 1002)
		borrowed.player.SetInvisLevel(5)
		borrowed.transportDone = make(chan struct{})
		borrowed.DetachTransport()
		borrowed.player.SetLinkless(true)
		registerTestSession(t, m, borrowed, "Borrowed")
		if err := cmdSwitch(actor, []string{"Borrowed"}); err != nil {
			t.Fatal(err)
		}
		name, invis := actor.mudlogActor()
		if name != "Borrowed" || invis != 5 {
			t.Fatalf("mudlogActor() = %q/%d, want the borrowed body Borrowed/5", name, invis)
		}
	})

	t.Run("switched into a mob", func(t *testing.T) {
		m, w := mudlogTestWorld(t)
		actor := mudlogObserver(t, m, "Godactor", 40, game.PrfLog2)
		actor.player.SetInvisLevel(37)
		mob, err := w.SpawnMobQuiet(90, 1001)
		if err != nil {
			t.Fatal(err)
		}
		// Set directly: Go's immortal-command gate refuses a session acting as a
		// mobile, so no command can reach this state. The helper still resolves
		// it the way the help and file-editor producers do.
		actor.isSwitched, actor.switchedMob = true, mob
		name, invis := actor.mudlogActor()
		if name != mob.GetName() || invis != 0 {
			t.Fatalf("mudlogActor() = %q/%d, want the mob's name and invis 0", name, invis)
		}
	})
}

// readBanFileLine reads the ban file the command wrote, proving the file write
// follows the producer instead of replacing it.
func readBanFileLine(t *testing.T, path string) (string, error) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
