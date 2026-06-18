package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

// TestActiveAffectsModifyStats is the foundational guard for the timed-affect →
// stat pipeline. Buff/debuff spells and skills (bless, armor, curse, berserk,
// kuji-kiri, …) store their effect in ActiveAffects with a Location + Magnitude
// via AddAffect, but never touch base stats. The stat getters must fold those in
// at read time, or every such buff is silently inert. Mirrors the engine
// APPLY_* / magnitude conventions used by MagAffects.
func TestActiveAffectsModifyStats(t *testing.T) {
	p := NewPlayer(1, "Caster", 3001)
	p.Hitroll = 0
	p.Damroll = 0
	p.AC = 100

	baseHit, baseDam, baseAC := p.GetHitroll(), p.GetDamroll(), p.GetAC()

	// bless: +2 hitroll. armor: -15 AC (lower is better). a +3 damroll buff.
	// (Spell IDs as literals — only used for removal matching here.)
	const spellBless, spellArmor = 35, 1
	p.AddAffect(engine.NewAffect(spellBless, engine.ApplyHitroll, 6, 2, "bless"))
	p.AddAffect(engine.NewAffect(spellArmor, engine.ApplyAC, 24, -15, "armor"))
	p.AddAffect(engine.NewAffect(0, engine.ApplyDamroll, 6, 3, "test buff"))

	if got := p.GetHitroll(); got != baseHit+2 {
		t.Errorf("GetHitroll = %d, want %d (bless +2 not applied)", got, baseHit+2)
	}
	if got := p.GetDamroll(); got != baseDam+3 {
		t.Errorf("GetDamroll = %d, want %d (+3 buff not applied)", got, baseDam+3)
	}
	if got := p.GetAC(); got != baseAC-15 {
		t.Errorf("GetAC = %d, want %d (armor -15 not applied)", got, baseAC-15)
	}
}

// TestActiveAffectsStackAndExpire verifies multiple affects on the same location
// sum, and that removing them (the expiry path's RemoveAffectBySpell) reverts the
// stat — so a buff wearing off restores the original value.
func TestActiveAffectsStackAndExpire(t *testing.T) {
	p := NewPlayer(1, "Caster", 3001)
	p.Hitroll = 5

	const spellBless = 35
	p.AddAffect(engine.NewAffect(spellBless, engine.ApplyHitroll, 6, 2, "bless"))
	p.AddAffect(engine.NewAffect(101, engine.ApplyHitroll, 6, 4, "other"))

	if got := p.GetHitroll(); got != 5+2+4 {
		t.Fatalf("stacked GetHitroll = %d, want %d", got, 11)
	}

	// bless wears off → only the +4 remains
	p.RemoveAffectBySpell(spellBless)
	if got := p.GetHitroll(); got != 5+4 {
		t.Errorf("after bless expiry GetHitroll = %d, want %d", got, 9)
	}

	// other wears off → back to base
	p.RemoveAffectBySpell(101)
	if got := p.GetHitroll(); got != 5 {
		t.Errorf("after all expiry GetHitroll = %d, want base 5", got)
	}
}
