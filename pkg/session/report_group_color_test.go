package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// groupColorFixture builds a grouped leader, a grouped player follower and a
// grouped mobile follower in one room and returns the leader's session.
func groupColorFixture(t *testing.T) *Session {
	t.Helper()
	w, err := game.NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Hall"}},
		Mobs: []parser.Mob{{
			VNum: 99100, Keywords: "guard", ShortDesc: "a guard",
			Level: 5, Position: combat.PosStanding, DefaultPos: combat.PosStanding,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)

	leader := makeCommandTestSession(t, m, "Leader", 10, 1001)
	leader.player.SetClass(3) // Warrior → "Wa"
	leader.player.SetHealth(100)
	leader.player.SetMaxHP(120)
	leader.player.SetMana(80)
	leader.player.SetMove(60)
	leader.player.SetInGroup(true)
	leader.player.SetAffect(game.AffGroup, true)
	if err := w.AddPlayer(leader.player); err != nil {
		t.Fatal(err)
	}

	follower := makeCommandTestSession(t, m, "Follower", 7, 1001)
	follower.player.SetClass(0) // Magic User → "Mu"
	follower.player.SetHealth(11)
	follower.player.SetMaxHP(20)
	follower.player.SetMana(22)
	follower.player.SetMove(33)
	follower.player.SetInGroup(true)
	follower.player.SetAffect(game.AffGroup, true)
	follower.player.SetFollowing("Leader")
	if err := w.AddPlayer(follower.player); err != nil {
		t.Fatal(err)
	}

	guard, err := w.SpawnMob(99100, 1001)
	if err != nil {
		t.Fatal(err)
	}
	guard.SetAffected(game.AffGroup)
	guard.SetFollowing("Leader")
	// world.MessageSink routes player output by name through Manager.GetSession
	// (pkg/session/manager.go:391), so the viewer's session must be registered
	// for print_group's rows to reach its send channel.
	m.sessions[leader.player.Name] = leader
	m.sessions[follower.player.Name] = follower
	return leader
}

// print_group wraps every stat field at C_CMP and measures the gate against the
// viewer (src/act.other.c:638-681). C_CMP is _clrlevel >= 3 while the reports
// that share these codes use C_NRM (>= 2), so `color normal` is the level that
// discriminates: a port that reused the C_NRM gate would colour the listing
// there, and a port that coloured the mobile row would fail the dashes check.
// The expected bytes are written from C's sprintf with the literal codes from
// src/screen.h:34-41, not from the production constants (R1/R5e/R5h).
func TestGroupListColorLevelsMatchC(t *testing.T) {
	for _, level := range statColorLevels {
		t.Run(level.name, func(t *testing.T) {
			s := groupColorFixture(t)
			statColorLevel(t, s, level.color1, level.color2)

			if err := printGroup(s); err != nil {
				t.Fatal(err)
			}
			got := statColorOutput(t, s)

			green, cyan, reset := "", "", ""
			if level.complete {
				green, cyan, reset = "\x1b[32m", "\x1b[36m", "\x1b[0m"
			}
			wants := []string{
				"Your group consists of:\r\n",
				// Head row: CCGRN on each stat, CCCYN on the class (line 659).
				"     [" + green + "100" + reset + "H " + green + " 80" + reset + "M " +
					green + " 60" + reset + "V] [10 " + cyan + "Wa" + reset + "] Leader (Head of group)\r\n",
				// Follower row: CCGRN on each stat and CCGRN on the class (line 676).
				"     [" + green + " 11" + reset + "H " + green + " 22" + reset + "M " +
					green + " 33" + reset + "V] [ 7 " + green + "Mu" + reset + "] Follower\r\n",
				// Mobile row: literal dashes, no colour at any level (line 669).
				"     [---H ---M ---V] [-- --] ",
			}
			for _, want := range wants {
				if !strings.Contains(got, want) {
					t.Errorf("group listing lacks %q\n  got=%q", want, got)
				}
			}
			// C_CMP needs both bits: below it every macro expands to "".
			if !level.complete && strings.Contains(got, "\x1b") {
				t.Errorf("group listing emitted colour below C_CMP: %q", got)
			}
		})
	}
}

// do_report's wrappers are C_NRM, one level below print_group's, so a viewer
// with only PRF_COLOR_2 must still receive the escapes — a port that reused the
// C_CMP gate would leave this line plain (src/act.other.c:806-818).
func TestReportColorUsesCNRMNotCMP(t *testing.T) {
	m := makeTestManager(t)
	s := makeCommandTestSession(t, m, "Reporter", 10, 1001)
	s.player.SetHealth(100)
	s.player.SetMaxHP(120)
	s.player.SetMana(80)
	s.player.SetMaxMana(90)
	s.player.SetMove(60)
	s.player.MaxMove = 70
	if err := m.world.AddPlayer(s.player); err != nil {
		t.Fatal(err)
	}
	// do_report's lines travel through world.MessageSink, which resolves the
	// recipient by name through Manager.GetSession (manager.go:391).
	m.sessions[s.player.Name] = s

	statColorLevel(t, s, false, true) // C_NRM only
	if err := cmdReport(s, nil); err != nil {
		t.Fatal(err)
	}
	got := statColorOutput(t, s)
	want := "    [\x1b[32m100\x1b[0m/\x1b[32m120\x1b[0m]H" +
		" [\x1b[32m80\x1b[0m/\x1b[32m90\x1b[0m]M" +
		" [\x1b[32m60\x1b[0m/\x1b[32m70\x1b[0m]V" +
		" [\x1b[31m0\x1b[0m]Kills [\x1b[31m0\x1b[0m]PKs [\x1b[31m0\x1b[0m]Deaths\r\n"
	if !strings.Contains(got, want) {
		t.Fatalf("C_NRM report line = %q, want substring %q", got, want)
	}

	statColorLevel(t, s, false, false)
	if err := cmdReport(s, nil); err != nil {
		t.Fatal(err)
	}
	got = statColorOutput(t, s)
	if want := "    [100/120]H [80/90]M [60/70]V [0]Kills [0]PKs [0]Deaths\r\n"; !strings.Contains(got, want) {
		t.Fatalf("colour-off report line = %q, want substring %q", got, want)
	}
}
