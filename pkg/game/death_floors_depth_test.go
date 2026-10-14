package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// death_floors_depth_test.go — DP-1429. C has no floor on real_abils.con when
// die_with_killer decrements it (src/fight.c:605-607) and no floor on
// GET_HIT when damage() applies a blow (src/fight.c:1483-1484); affect_total
// and update_pos, not the stores, are what bound the effective values.

// deathFloorsWorld is a room with a level-15 PC victim and a PC killer.
func deathFloorsWorld(t *testing.T) (*World, *Player, *Player) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{{VNum: 1001, Name: "Floors"}}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)

	victim := NewPlayer(21, "Victim", 1001)
	victim.Stats = CharStats{Str: 10, Int: 10, Wis: 10, Dex: 10, Con: 2, Cha: 10}
	victim.CopyBaseAttributes()
	victim.SetLevel(21) // level > 20 also arms the second CON draw (src/fight.c:606)
	victim.SetMaxHP(100)
	victim.SetHP(100)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer victim: %v", err)
	}
	killer := NewPlayer(22, "Killer", 1001)
	killer.SetLevel(15)
	if err := w.AddPlayer(killer); err != nil {
		t.Fatalf("AddPlayer killer: %v", err)
	}
	return w, victim, killer
}

// killOnce runs one combat death through the real handler. die_with_killer is
// reached with GET_HIT(ch) < 0 (src/fight.c:599), and the port latches the
// deferred extraction, which would short-circuit the next call; clear it so
// the loop can die again the way a fresh life would.
func killOnce(w *World, victim, killer *Player) {
	victim.SetHP(-1)
	w.HandleDeath(victim, killer, 0)
	victim.SetPlrFlag(plrExtractBit, false)
	w.mu.Lock()
	delete(w.pendingPlayerExtractions, victim)
	w.mu.Unlock()
}

// TestDieWithKillerConLossHasNoFloor is the DP-1429 CON regression: repeated
// combat deaths drive real CON to 0 and then below it. C's affect_total is what
// clamps the effective CON to 0..18 (src/handler.c:356).
func TestDieWithKillerConLossHasNoFloor(t *testing.T) {
	dprng.ResetStream(20261014)
	w, victim, killer := deathFloorsWorld(t)

	// 1-in-4 gate for the first decrement and 1-in-6 for the second; the bound
	// is far above the deterministic count for this seed, and the loop
	// condition is the assertion. CON must pass -1: with either of the port's
	// old floors present it can never get there.
	for i := 0; i < 400 && victim.Stats.Con > -2; i++ {
		killOnce(w, victim, killer)
	}
	if victim.Stats.Con > -2 {
		t.Fatalf("real CON = %d after 400 combat deaths, want <= -2 (C has no floor on either decrement)", victim.Stats.Con)
	}
	if got := victim.GetCon(); got != 0 {
		t.Fatalf("effective CON = %d, want 0 (affect_total clamps 0..18)", got)
	}
}

// TestDieWithKillerBothConLossDrawsAreUnfloored pins both decrements with the
// draw seam: one death that passes both gates must take real CON down by
// exactly 2, even from 1 (so the value lands at -1). Either of the port's old
// floors leaves it at 1. Source: src/fight.c:599-608.
func TestDieWithKillerBothConLossDrawsAreUnfloored(t *testing.T) {
	w, victim, killer := deathFloorsWorld(t)
	victim.Stats.Con = 1
	victim.CopyBaseAttributes()
	victim.SetLevel(21) // level > 20 arms the second draw (src/fight.c:606)

	previous := conLossNumber
	conLossNumber = func(_, _ int) int { return 0 } // both gates pass
	t.Cleanup(func() { conLossNumber = previous })

	killOnce(w, victim, killer)

	if victim.Stats.Con != -1 {
		t.Fatalf("real CON = %d after one double-decrement death, want -1 (C floors neither decrement)", victim.Stats.Con)
	}
	if got := victim.GetCon(); got != 0 {
		t.Fatalf("effective CON = %d, want 0 (affect_total clamps 0..18)", got)
	}
}

// die_with_killer around the unfloored decrement: the counters, the exp loss
// and the deferred extraction are unchanged by it.
func TestPlayerDeathCountersStillRunWithNegativeCon(t *testing.T) {
	dprng.ResetStream(20261014)
	w, victim, killer := deathFloorsWorld(t)
	victim.SetExp(100000)

	for i := 0; i < 20; i++ {
		killOnce(w, victim, killer)
	}
	if victim.Deaths != 20 {
		t.Fatalf("deaths = %d, want 20", victim.Deaths)
	}
	if victim.Stats.Con >= 2 {
		t.Fatalf("real CON = %d after 20 deaths, want a decrement (no floor)", victim.Stats.Con)
	}
	if victim.LastDeath == 0 {
		t.Fatal("LastDeath not recorded")
	}
}
