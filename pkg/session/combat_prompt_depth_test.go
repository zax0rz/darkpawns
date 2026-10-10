package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCombatPromptTargetAndTank(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Viewer", 1001, true)
	target := game.NewPlayer(2, "mIXed", 1001)
	tank := game.NewPlayer(3, "Tank", 1001)
	target.SetHP(75)
	target.MaxHealth = 100
	tank.SetHP(15)
	tank.MaxHealth = 100
	s.player.SetFightingBody(target)
	target.SetFightingBody(tank)
	s.player.SetPlrFlag(game.PrfDispTarget, true)
	s.player.SetPlrFlag(game.PrfDispTank, true)
	if got := s.promptText(); got != "Mixed:(small wounds) Tank:(pretty hurt) > " {
		t.Fatalf("combat prompt=%q", got)
	}
	s.player.SetPlrFlag(game.PrfColor1, true)
	s.player.SetPlrFlag(game.PrfColor2, true)
	if got := s.promptText(); got != "\x1b[31mMixed:(small wounds)\x1b[0m \x1b[31mTank:(pretty hurt)\x1b[0m > " {
		t.Fatalf("colored combat prompt=%q", got)
	}
	target.SetInvisLevel(31)
	if got := s.promptText(); got != "\x1b[31mSomeone:(small wounds)\x1b[0m \x1b[31mTank:(pretty hurt)\x1b[0m > " {
		t.Fatalf("invisible target prompt=%q", got)
	}
	target.SetInvisLevel(0)
	s.infobarMode = InfobarOn
	if got := s.promptText(); got != "> " {
		t.Fatalf("infobar combat prompt=%q", got)
	}
	s.infobarMode = InfobarOff
	target.SetFightingBody(nil)
	if got := s.promptText(); got != "\x1b[31mMixed:(small wounds)\x1b[0m > " {
		t.Fatalf("missing tank prompt=%q", got)
	}
	s.player.SetFightingBody(nil)
	if got := s.promptText(); got != "> " {
		t.Fatalf("nonfighting prompt=%q", got)
	}
}

func TestCombatPromptConditionThresholds(t *testing.T) {
	p := game.NewPlayer(2, "Target", 1001)
	p.MaxHealth = 100
	for _, tc := range []struct {
		hp     int
		status string
	}{
		{101, "excellent"}, {100, "excellent"}, {99, "few scratches"}, {90, "few scratches"}, {89, "small wounds"}, {75, "small wounds"}, {74, "quite a few wounds"}, {50, "quite a few wounds"}, {49, "big nasty wounds"}, {30, "big nasty wounds"}, {29, "pretty hurt"}, {15, "pretty hurt"}, {14, "awful"}, {0, "awful"}, {-1, "nearly dead"},
	} {
		p.SetHP(tc.hp)
		want := "Target:(" + tc.status + ") "
		if got := promptCombatSegment("Target", p, false); got != want {
			t.Fatalf("hp %d: got %q want %q", tc.hp, got, want)
		}
	}
	p.MaxHealth = 0
	if got := promptCombatSegment("Target", p, false); got != "Target:(nearly dead) " {
		t.Fatalf("zero max prompt=%q", got)
	}
	p.MaxHealth = 1000
	p.SetHP(-1)
	if got := promptCombatSegment("Target", p, false); got != "Target:(awful) " {
		t.Fatalf("C negative division prompt=%q", got)
	}
}

func TestPromptStatusColorLevels(t *testing.T) {
	for level := 0; level < 4; level++ {
		m := makeTestManager(t)
		s := makeTestSession(t, m, "Viewer", 1001, true)
		s.player.SetPlrFlag(game.PrfColor1, level&1 != 0)
		s.player.SetPlrFlag(game.PrfColor2, level&2 != 0)
		s.player.SetPlrFlag(game.PrfDisphp, true)
		s.player.SetPlrFlag(game.PrfAFK, true)
		want := "AFK > "
		if level == 3 {
			want = "\x1b[31mAFK\x1b[0m > "
		}
		if got := s.promptText(); got != want {
			t.Fatalf("level %d AFK=%q want %q", level, got, want)
		}
		s.player.SetPlrFlag(game.PrfInactive, true)
		want = "INACTIVE > "
		if level == 3 {
			want = "\x1b[31mINACTIVE\x1b[0m > "
		}
		if got := s.promptText(); got != want {
			t.Fatalf("level %d INACTIVE=%q want %q", level, got, want)
		}
		s.infobarMode = InfobarOn
		if got := s.promptText(); got != "> " {
			t.Fatalf("infobar status prompt=%q", got)
		}
	}
}
