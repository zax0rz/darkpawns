package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPracticeCommandPagesLongImmortalCatalog(t *testing.T) {
	w := makeZeditTestWorld(t)
	m := newTestManager(t, w, nil)
	s := makeCommandTestSession(t, m, "Pracpage", game.LVL_IMMORT, 3000)
	s.player.Class = game.ClassPsionic
	s.player.SetLevel(game.LVL_IMMORT)
	s.player.SetPlrFlag(game.PrfColor1, true)
	s.player.SetPlrFlag(game.PrfColor2, true)
	registerTestSession(t, m, s, s.player.Name)

	if err := ExecuteCommand(s, "practice", nil); err != nil {
		t.Fatal(err)
	}
	if !s.IsPaging() || s.pagerCount < 2 {
		t.Fatalf("practice catalog pager state = %d pages, want multiple pages", s.pagerCount)
	}
	if !strings.Contains(string(s.pagerPages[0]), "You know of the following arts:") {
		t.Fatalf("first page omitted catalog header: %q", s.pagerPages[0])
	}
}
