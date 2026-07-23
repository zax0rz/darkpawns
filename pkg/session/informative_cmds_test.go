package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCmdTitleEmptyArgsClearsTitle(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)

	if err := cmdTitle(s, nil); err != nil {
		t.Fatalf("cmdTitle: %v", err)
	}
	assertTitleText(t, s, "Okay, you're now Hero .\r\n")
	if s.player.Title != "" {
		t.Errorf("title = %q, want empty", s.player.Title)
	}
}

func TestCmdTitlePreservesFullArgument(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)

	if err := cmdTitle(s, []string{"the", "Brave", "Knight", "of", "Solamnia"}); err != nil {
		t.Fatalf("cmdTitle: %v", err)
	}
	assertTitleText(t, s, "Okay, you're now Hero the Brave Knight of Solamnia.\r\n")
	if s.player.Title != "the Brave Knight of Solamnia" {
		t.Errorf("title = %q, want %q", s.player.Title, "the Brave Knight of Solamnia")
	}
}

func TestCmdTitleStripsDoubledDollarAndANSIMarkers(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)

	if err := cmdTitle(s, []string{"&rthe", "Re$$d", "Knight"}); err != nil {
		t.Fatalf("cmdTitle: %v", err)
	}
	want := "rthe Re$d Knight"
	if s.player.Title != want {
		t.Errorf("title = %q, want %q", s.player.Title, want)
	}
}

func TestCmdTitleRejectsParentheses(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)

	if err := cmdTitle(s, []string{"the", "Brave", "(Knight)"}); err != nil {
		t.Fatalf("cmdTitle: %v", err)
	}
	assertTitleText(t, s, "Titles can't contain the ( or ) characters.\r\n")
	if s.player.Title != "" {
		t.Errorf("title should not have been set, got %q", s.player.Title)
	}
}

func TestCmdTitleRejectsTooLongTitle(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)

	longTitle := strings.Repeat("x", game.MAX_TITLE_LENGTH+1)
	if err := cmdTitle(s, []string{longTitle}); err != nil {
		t.Fatalf("cmdTitle: %v", err)
	}
	assertTitleText(t, s, "Sorry, titles can't be longer than 80 characters.\r\n")
	if s.player.Title != "" {
		t.Errorf("title should not have been set, got %q", s.player.Title)
	}
}

func TestCmdTitleRejectsNotitleFlag(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)
	s.player.SetPlrFlag(game.PlrNotitle, true)

	if err := cmdTitle(s, []string{"the", "Brave"}); err != nil {
		t.Fatalf("cmdTitle: %v", err)
	}
	assertTitleText(t, s, "You can't title yourself -- you shouldn't have abused it!\r\n")
	if s.player.Title != "" {
		t.Errorf("title should not have been set, got %q", s.player.Title)
	}
}

func TestCmdTitleSuccessUsesSetTitle(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)
	s.player.Class = game.ClassWarrior

	if err := cmdTitle(s, []string{"custom"}); err != nil {
		t.Fatalf("cmdTitle: %v", err)
	}
	assertTitleText(t, s, "Okay, you're now Hero custom.\r\n")
	if s.player.Title != "custom" {
		t.Errorf("title = %q, want %q", s.player.Title, "custom")
	}
}

func TestCmdTitleSetThenClear(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)

	if err := cmdTitle(s, []string{"Foo", "the", "Tester"}); err != nil {
		t.Fatalf("cmdTitle set: %v", err)
	}
	assertTitleText(t, s, "Okay, you're now Hero Foo the Tester.\r\n")
	if s.player.Title != "Foo the Tester" {
		t.Errorf("title = %q, want %q", s.player.Title, "Foo the Tester")
	}

	if err := cmdTitle(s, nil); err != nil {
		t.Fatalf("cmdTitle clear: %v", err)
	}
	assertTitleText(t, s, "Okay, you're now Hero .\r\n")
	if s.player.Title != "" {
		t.Errorf("title = %q, want empty after clear", s.player.Title)
	}
}

func TestCmdTitleNotitleFlagWithEmptyArgs(t *testing.T) {
	m := makeTestManager(t)
	s := makeTestSession(t, m, "Hero", 1001, true)
	s.player.SetPlrFlag(game.PlrNotitle, true)

	if err := cmdTitle(s, nil); err != nil {
		t.Fatalf("cmdTitle: %v", err)
	}
	assertTitleText(t, s, "You can't title yourself -- you shouldn't have abused it!\r\n")
	if s.player.Title != "" {
		t.Errorf("title should not have been set, got %q", s.player.Title)
	}
}

func assertTitleText(t *testing.T, s *Session, want string) {
	t.Helper()
	got := readSessionText(t, s)
	if got != want {
		t.Errorf("title text = %q, want %q", got, want)
	}
}
