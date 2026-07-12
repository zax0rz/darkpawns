package game

// Regression tests for QA Batch 1 (BRIEF-2026-07-12-qa-batch1-oneliners.md):
//   - DP-1055: mortal XP must never auto-advance a player past level 30
//     (LVL_IMMORT-1). Previously the gate used LVL_IMPL-1 (=39), letting a
//     level-30 mortal cross into level 31 (LVL_IMMORT) and gain immortal
//     privileges (DT immunity, instakill routing, aggro immunity).

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestGainExp_MortalCannotReachImmortal verifies the DP-1055 fix: a level-30
// mortal whose XP crosses the advancement threshold stays at 30, while a
// level-29 mortal under the same conditions still advances to 30 (the gate
// must let sub-cap advancements through).
func TestGainExp_MortalCannotReachImmortal(t *testing.T) {
	newWorld := func(t *testing.T) *World {
		t.Helper()
		parsed := &parser.World{
			Rooms: []parser.Room{{VNum: 1001, Name: "XP Lab", Zone: 1}},
		}
		w, err := NewWorld(parsed)
		if err != nil {
			t.Fatalf("NewWorld failed: %v", err)
		}
		t.Cleanup(func() { w.StopAITicker() })
		return w
	}

	t.Run("level30_capped", func(t *testing.T) {
		w := newWorld(t)
		p := NewPlayer(1, "Capped", 1001)
		p.Class = ClassWarrior
		p.SetLevel(30)
		// Park XP at exactly the level-30 advancement threshold so any positive
		// gain satisfies the >= ExpNeededForLevel(p) predicate.
		p.Exp = ExpNeededForLevel(p)
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer failed: %v", err)
		}

		w.GainExp(p, 5_000_000)

		if p.Level != 30 {
			t.Errorf("level-30 mortal advanced to %d; XP must never take a mortal past 30 (LVL_IMMORT-1)", p.Level)
		}
	})

	t.Run("level29_advances_to_30", func(t *testing.T) {
		w := newWorld(t)
		p := NewPlayer(2, "Almost", 1001)
		p.Class = ClassWarrior
		p.SetLevel(29)
		// Start just below the level-29 threshold; the large award crosses it.
		p.Exp = ExpNeededForLevel(p) - 1
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer failed: %v", err)
		}

		w.GainExp(p, 5_000_000)

		if p.Level != 30 {
			t.Errorf("level-29 mortal advanced to %d, want 30 (sub-cap advancement must still work)", p.Level)
		}
	})
}
