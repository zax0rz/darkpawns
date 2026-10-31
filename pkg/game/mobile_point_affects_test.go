package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// TestMobilePointAffectsRecomputeFromTimedRecords pins C's affect_total for
// mobiles (src/handler.c:337-372): every timed affect record with APPLY_AC,
// APPLY_HITROLL or APPLY_DAMROLL moves the mobile's combat points, exactly as
// affect_to_char's berserk records do (src/new_cmds.c:2146-2160 — +25 AC, +2
// hitroll, +2 damroll; the failed attempt installs zeroes). The port's
// getters used to read only the wizard override/prototype, so a berserked
// mobile fought with its base points and split the shared draw stream
// against C (the fighter-round divergence, mob.fighter-round-draw-parity).
func TestMobilePointAffectsRecomputeFromTimedRecords(t *testing.T) {
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 100}},
		Mobs:  []parser.Mob{{VNum: 300, Keywords: "guard", ShortDesc: "a guard", Level: 30, AC: -100, THAC0: 0}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	m, err := w.SpawnMob(300, 100)
	if err != nil {
		t.Fatal(err)
	}

	baseAC, baseHit, baseDam := m.GetAC(), m.GetHitroll(), m.GetDamroll()
	if baseAC != -100 || baseHit != 20 {
		t.Fatalf("base points = %d/%d/%d, want prototype AC -100 and hitroll 20", baseAC, baseHit, baseDam)
	}

	// A successful berserk: three records, +25 AC / +2 hitroll / +2 damroll.
	m.AddAffect(engine.NewAffectDirect(1, ApplyHitroll, 1, 2, engine.AFFBerserk, "berserk"))
	m.AddAffect(engine.NewAffectDirect(1, ApplyDamroll, 1, 2, engine.AFFBerserk, "berserk"))
	m.AddAffect(engine.NewAffectDirect(1, ApplyAC, 1, 25, engine.AFFBerserk, "berserk"))

	if got := m.GetAC(); got != baseAC+25 {
		t.Fatalf("berserked AC = %d, want %d (APPLY_AC +25 recompute)", got, baseAC+25)
	}
	if got := m.GetHitroll(); got != baseHit+2 {
		t.Fatalf("berserked hitroll = %d, want %d (APPLY_HITROLL +2 recompute)", got, baseHit+2)
	}
	if got := m.GetDamroll(); got != baseDam+2 {
		t.Fatalf("berserked damroll = %d, want %d (APPLY_DAMROLL +2 recompute)", got, baseDam+2)
	}

	// Curse's two records shift hit and dam the other way (src/magic.c:986-994).
	m2, err := w.SpawnMob(300, 100)
	if err != nil {
		t.Fatal(err)
	}
	m2.AddAffect(engine.NewAffectDirect(2, ApplyHitroll, 2, -3, 0, "curse"))
	m2.AddAffect(engine.NewAffectDirect(2, ApplyDamroll, 2, -3, 0, "curse"))
	if got := m2.GetHitroll(); got != 20-3 {
		t.Fatalf("cursed hitroll = %d, want 17", got)
	}
	if got := m2.GetDamroll(); got != -3 {
		t.Fatalf("cursed damroll = %d, want -3", got)
	}
	_ = combat.TYPE_SLASH // keep combat import if assertions evolve
}
