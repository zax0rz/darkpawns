package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

func TestCombatBodyLiveGroupSpellLeaders(t *testing.T) {
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001}}, Mobs: []parser.Mob{{VNum: 300, Keywords: "leader", ShortDesc: "a leader", Position: 8}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	a, err := w.SpawnMobQuiet(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	b, err := w.SpawnMobQuiet(300, 1001)
	if err != nil {
		t.Fatal(err)
	}
	x, y := NewPlayer(1, "First", 1001), NewPlayer(2, "Second", 1001)
	for _, p := range []*Player{x, y} {
		p.SetInGroup(true)
		p.SetMaxHP(500)
		p.SetHP(100)
		if err := w.AddPlayer(p); err != nil {
			t.Fatal(err)
		}
	}
	x.followingBody = a
	x.Following = a.GetName()
	y.followingBody = b
	y.Following = b.GetName()
	spells.MagGroups(30, x, spells.SpellGroupHeal, 0, w)
	if y.GetHP() != 100 {
		t.Fatalf("separate duplicate leader group healed: %d", y.GetHP())
	}
	y.followingBody = a
	spells.MagGroups(30, x, spells.SpellGroupHeal, 0, w)
	if y.GetHP() <= 100 {
		t.Fatal("actual shared leader group was skipped")
	}
}
