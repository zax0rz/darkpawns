package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1438: the caught-steal retaliation is C's hit(vict, ch,
// TYPE_UNDEFINED) (act.other.c:528-529; kender_steal reaches it by
// delegating to do_steal's subcmd arm, spec_procs2.c:626). The mob's first
// swing consumes hit()'s draws and runs damage()'s gates against the
// pre-enrolment state before anyone enrols. The old plain StartCombat
// enrolled without a swing — zero draws, no gate — and both tests fail on
// that shape (R5h).
func stealRetaliationWorld(t *testing.T, peaceful bool) (*World, *combat.CombatEngine, *Player, *MobInstance) {
	t.Helper()
	flags := []string{}
	if peaceful {
		flags = []string{"peaceful"}
	}
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Vault", Flags: flags}},
		Mobs: []parser.Mob{{
			VNum: 2001, Keywords: "mob twothousandone", ShortDesc: "the mob",
			Level: 11, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: 99},
		}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(string, []byte) {}
	p := NewPlayer(1, "Filch", 1001)
	p.Level = 12 // both level gates clear: attacker and victim are above 10
	p.Stats.Str = 15
	p.Stats.Dex = 12
	p.CopyBaseAttributes()
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	mob, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	oldCB := combat.GetCallbacks()
	ce := combat.NewCombatEngine()
	ce.SetCallbacks(w.WireCombatCallbacks())
	w.SetCombatEngine(ce)
	t.Cleanup(func() { ce.Stop(); combat.SetCallbacks(oldCB) })
	return w, ce, p, mob
}

// stealMissSeed finds a stream seed whose draws are: a caught steal
// (percent > skill 0) and then a natural-1 to-hit miss, so the retaliation
// consumes exactly number(1,101) + number(1,20) and no damage dice.
func stealMissSeed(t *testing.T, p *Player) uint32 {
	t.Helper()
	pp := dexAppSkill(p.GetDex()).PPocket
	for seed := uint32(1); seed < 8192; seed++ {
		dprng.ResetStream(seed)
		percent := dprng.Number(1, 101) - pp
		if percent <= p.GetSkill(SkillSteal) {
			continue
		}
		dprng.ResetStream(seed)
		dprng.Number(1, 101)
		if dprng.Number(1, 20) == 1 {
			return seed
		}
	}
	t.Fatal("no seed found with a caught steal and a missed retaliation")
	return 0
}

func TestKenderStealCaughtRetaliationDrawsAndEnrols(t *testing.T) {
	w, ce, p, mob := stealRetaliationWorld(t, false)
	obj := newFlagItem(w, 7103, 1)
	obj.SetWeight(0)
	mob.AddToInventory(obj)

	seed := stealMissSeed(t, p)
	dprng.ResetStream(seed)
	w.kenderStealItem(p, mob, obj)
	gotNext := dprng.Number(0, 999)

	// Reference: the percent draw, then the missed to-hit roll.
	dprng.ResetStream(seed)
	dprng.Number(1, 101)
	dprng.Number(1, 20)
	wantNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("retaliation changed the draw stream: next=%d, want %d (number(1,101) + number(1,20) consumed)", gotNext, wantNext)
	}
	if !ce.IsFighting(mob) || mob.GetFightingBody() != p {
		t.Fatal("landed retaliation did not enrol the mob")
	}
	if p.GetFightingBody() != mob {
		t.Fatal("landed retaliation did not enrol the thief")
	}
}

func TestKenderStealCaughtRetaliationRefusedInPeacefulRoom(t *testing.T) {
	w, ce, p, mob := stealRetaliationWorld(t, true)
	obj := newFlagItem(w, 7103, 1)
	obj.SetWeight(0)
	mob.AddToInventory(obj)

	seed := stealMissSeed(t, p)
	dprng.ResetStream(seed)
	w.kenderStealItem(p, mob, obj)
	gotNext := dprng.Number(0, 999)

	// kender_steal forces the peaceful-room failure (spec_procs2.c:610-613)
	// after its percent draw, then hit()'s roll misses: two draws total.
	dprng.ResetStream(seed)
	dprng.Number(1, 101)
	dprng.Number(1, 20)
	wantNext := dprng.Number(0, 999)
	if gotNext != wantNext {
		t.Fatalf("refused retaliation changed the draw stream: next=%d, want %d (number(1,101) + number(1,20) consumed)", gotNext, wantNext)
	}
	if ce.IsFighting(mob) || ce.IsFighting(p) || mob.GetFightingBody() != nil || p.GetFightingBody() != nil {
		t.Fatal("refused retaliation enrolled anyone")
	}
}
