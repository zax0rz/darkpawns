package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestMobSpawnPreservesDeadAndFloating pins DP-1553: a prototype built with
// medit (src/medit.c:1061, src/olc.h:46 NUM_POSITIONS=15) and copied by
// read_mobile (src/db.c:1092,1742-1746) keeps POS_DEAD (0) and floating
// (9..14) as spawn positions. C reports position 0 as dead and otherwise
// falls through to floating for unlisted positions (src/act.informative.c:1322-1356).
func TestMobSpawnPreservesDeadAndFloating(t *testing.T) {
	for pos := 0; pos < 15; pos++ {
		mob := NewMob(&parser.Mob{
			VNum: 9000, Keywords: "trainee", ShortDesc: "a trainee",
			Position: pos, DefaultPos: 8,
		}, 1001)
		if got := mob.GetPosition(); got != pos {
			t.Errorf("spawn position %d: GetPosition() = %d, want %d", pos, got, pos)
		}
		if got := mob.GetStatus(); pos == 0 && got != "dead" {
			t.Errorf("spawn position 0: GetStatus() = %q, want dead", got)
		}
		if got := mob.GetStatus(); pos >= 9 && got != "floating" {
			t.Errorf("spawn position %d: GetStatus() = %q, want floating", pos, got)
		}
	}
}

// TestMobSetPositionPreservesDeadAndFloating pins the SetPosition round-trip
// for every C position, including floating values above standing.
func TestMobSetPositionPreservesDeadAndFloating(t *testing.T) {
	mob := NewMob(&parser.Mob{
		VNum: 9001, Keywords: "trainee", ShortDesc: "a trainee",
		Position: combat.PosStanding, DefaultPos: combat.PosStanding,
	}, 1001)
	for pos := 0; pos < 15; pos++ {
		mob.SetPosition(pos)
		if got := mob.GetPosition(); got != pos {
			t.Errorf("SetPosition(%d): GetPosition() = %d, want %d", pos, got, pos)
		}
	}
	// Standing clears any floating value so a later read cannot report it.
	mob.SetPosition(9)
	mob.SetPosition(combat.PosStanding)
	if got := mob.GetPosition(); got != combat.PosStanding {
		t.Errorf("after clearing floating: GetPosition() = %d, want %d", got, combat.PosStanding)
	}
	// An unset proto (0,0) spawns standing, not dead: the parser defaults a
	// missing position line to 8/8, so (0,0) never reaches the oracle.
	unset := NewMob(&parser.Mob{VNum: 9002, Keywords: "unset", ShortDesc: "an unset"}, 1001)
	if got := unset.GetPosition(); got != combat.PosStanding {
		t.Errorf("unset proto: GetPosition() = %d, want %d", got, combat.PosStanding)
	}
}
