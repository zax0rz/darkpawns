package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// mobWearOffFixture builds a one-mob world whose switched body captures the
// mobile's own output, the way MobInstance.SendMessage delivers it.
func mobWearOffFixture(t *testing.T) (*World, *MobInstance, func() []string) {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Test Room"}},
		Mobs:  []parser.Mob{{VNum: 9001, Keywords: "watcher", ShortDesc: "a watcher", LongDesc: "A watcher is here.", Level: 30}},
	})
	if err != nil {
		t.Fatal(err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)

	owner := NewPlayer(1, "Controller", 1001)
	if err := w.AddPlayer(owner); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}
	mob, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	mob.CurrentHP, mob.MaxHP = 100, 100

	var out []string
	w.MobileMessageSink = func(body *MobInstance, msg []byte) {
		if body == mob {
			out = append(out, string(msg))
		}
	}
	w.MobileDescriptorOwner = func(body *MobInstance) *Player {
		if body == mob {
			return owner
		}
		return nil
	}
	return w, mob, func() []string { s := out; out = nil; return s }
}

func TestMetalskinAppliesACOnceAndWearsOffOnce(t *testing.T) {
	w, mob, drain := mobWearOffFixture(t)
	baseAC := mob.GetAC()
	mob.AddAffect(engine.NewAffectDirect(spells.SpellMetalskin, engine.ApplyAC, 0, -30, engine.AFFMetalskin, "metalskin"))
	if got := mob.GetAC(); got != baseAC-30 {
		t.Fatalf("metalskin AC = %d, want %d", got, baseAC-30)
	}

	w.AffectUpdate()
	if got := mob.GetAC(); got != baseAC {
		t.Fatalf("AC after metalskin wore off = %d, want %d", got, baseAC)
	}
	if got := drain(); len(got) != 1 || got[0] != "Your skin softens.\r\n" {
		t.Fatalf("metalskin wear-off = %q, want exactly one line", got)
	}
}

// TestMobWearOffDeliveredInListOrder pins C's affect_update order: it walks
// ch->affected newest-first (src/magic.c:435), so the most recently added
// affect's wear-off speaks first, and the line reaches the switched body's
// descriptor via send_to_char (src/magic.c:448-449).
func TestMobWearOffDeliveredInListOrder(t *testing.T) {
	w, mob, drain := mobWearOffFixture(t)

	mob.AddAffect(engine.NewAffectDirect(spells.SpellArmor, engine.ApplyAC, 0, -15, 0, "armor"))
	mob.AddAffect(engine.NewAffectDirect(spells.SpellBless, engine.ApplyHitroll, 0, 2, 0, "bless"))

	w.AffectUpdate()

	got := drain()
	want := []string{"You feel less righteous.\r\n", "You feel less protected.\r\n"}
	if len(got) != len(want) {
		t.Fatalf("wear-off lines = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("wear-off line %d = %q, want %q", i, got[i], want[i])
		}
	}
	if remaining := mob.OrderedAffects(); len(remaining) != 0 {
		t.Fatalf("expired affects left on the mob: %d", len(remaining))
	}
}

// TestMobWearOffMultiRecordSpeaksOnce pins C's last-record condition
// (src/magic.c:439-444): a spell stored as two records (bless, like poison and
// curse) sends its wear-off once, from the oldest expiring record.
func TestMobWearOffMultiRecordSpeaksOnce(t *testing.T) {
	w, mob, drain := mobWearOffFixture(t)

	mob.AddAffect(engine.NewAffectDirect(spells.SpellBless, engine.ApplyHitroll, 0, 2, 0, "bless"))
	mob.AddAffect(engine.NewAffectDirect(spells.SpellBless, engine.ApplySavingSpell, 0, -2, 0, "bless"))

	w.AffectUpdate()

	got := drain()
	if len(got) != 1 || got[0] != "You feel less righteous.\r\n" {
		t.Fatalf("two-record bless wear-off = %q, want exactly one line", got)
	}
}
