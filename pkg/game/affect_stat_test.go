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

// TestAffectBitMapping verifies that every C bit position maps correctly to engine.AFF* flags and back,
// and that IsAffected(0) (blindness in C) returns true for blind only, not for sneak (C bit 18) or hide (C bit 19).
func TestAffectBitMapping(t *testing.T) {
	// 1. Verify bidirectionality of mapping tables
	for cBit, engFlag := range AffBitToEngineFlag {
		mappedCBit, ok := EngineFlagToAffBit[engFlag]
		if !ok {
			t.Errorf("Engine flag %d has no mapping back to C bit", engFlag)
		}
		if mappedCBit != cBit {
			t.Errorf("Mapping mismatch: C bit %d -> engine flag %d -> C bit %d", cBit, engFlag, mappedCBit)
		}
	}

	for engFlag, cBit := range EngineFlagToAffBit {
		mappedEngFlag, ok := AffBitToEngineFlag[cBit]
		if !ok {
			t.Errorf("C bit %d has no mapping to engine flag", cBit)
		}
		if mappedEngFlag != engFlag {
			t.Errorf("Mapping mismatch: engine flag %d -> C bit %d -> engine flag %d", engFlag, cBit, mappedEngFlag)
		}
	}

	// 2. Verify player IsAffected and SetAffect (specifically checking the bit 0 vs bit 18 collision fix)
	p := NewPlayer(1, "TestSubject", 3001)

	// Set raw blind (C bit 0)
	p.SetAffect(affBlind, true)
	if !p.IsAffected(affBlind) {
		t.Error("Player should be affected by blind")
	}
	if p.IsAffected(affSneak) {
		t.Error("Player should NOT be affected by sneak when only blind is set")
	}
	if p.IsAffected(affHide) {
		t.Error("Player should NOT be affected by hide when only blind is set")
	}

	// Clear blind, set sneak (C bit 18)
	p.SetAffect(affBlind, false)
	p.SetAffect(affSneak, true)
	if p.IsAffected(affBlind) {
		t.Error("Player should NOT be affected by blind when only sneak is set")
	}
	if !p.IsAffected(affSneak) {
		t.Error("Player should be affected by sneak")
	}

	// 3. Verify that IsAffected checks ActiveAffects as well
	p.SetAffect(affSneak, false)
	// Apply invisibility spell (which sets engine.AFFInvisible -> C bit 1)
	p.AddAffect(engine.NewAffectDirect(100, engine.ApplyNone, 5, 0, engine.AFFInvisible, "invis spell"))
	if !p.IsAffected(affInvisible) {
		t.Error("Player should be affected by invisible via ActiveAffects")
	}
	if p.IsAffected(affBlind) {
		t.Error("Player should NOT be affected by blind via ActiveAffects of invisibility")
	}
}
