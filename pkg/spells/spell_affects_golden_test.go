package spells

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
)

// mockAffectVictim records every affect applied to it and satisfies the
// interfaces MagAffects/magAffectsApply require for the no-save test path.
type mockAffectVictim struct {
	affects     []*engine.Affect
	position    int
	alignment   int
	npc         bool
	hasMobFlags uint64
}

func (m *mockAffectVictim) AddAffect(a *engine.Affect) {
	if a != nil {
		m.affects = append(m.affects, a)
	}
}

func (m *mockAffectVictim) SetPosition(pos int)         { m.position = pos }
func (m *mockAffectVictim) GetAlignment() int           { return m.alignment }
func (m *mockAffectVictim) IsNPC() bool                 { return m.npc }
func (m *mockAffectVictim) HasMobFlag(flag uint64) bool { return m.hasMobFlags&flag != 0 }
func (m *mockAffectVictim) GetLevel() int               { return 20 }
func (m *mockAffectVictim) GetClass() int               { return 0 }

// mockAffectCaster supplies level/class for duration/magnitude calculations.
type mockAffectCaster struct {
	level int
	class int
}

func (m *mockAffectCaster) GetLevel() int { return m.level }
func (m *mockAffectCaster) GetClass() int { return m.class }

// affectExpectation describes one expected affect in the mock victim.
type affectExpectation struct {
	spellID   int
	location  int
	duration  int
	magnitude int
	flags     uint64
}

func expectAffects(t *testing.T, name string, victim *mockAffectVictim, wants []affectExpectation) {
	t.Helper()
	if len(victim.affects) != len(wants) {
		t.Fatalf("%s: got %d affects, want %d", name, len(victim.affects), len(wants))
	}
	for i, want := range wants {
		got := victim.affects[i]
		if got.SpellID != want.spellID {
			t.Errorf("%s affect[%d] SpellID = %d, want %d", name, i, got.SpellID, want.spellID)
		}
		if got.Location != want.location {
			t.Errorf("%s affect[%d] Location = %d, want %d", name, i, got.Location, want.location)
		}
		if got.Duration != want.duration {
			t.Errorf("%s affect[%d] Duration = %d, want %d", name, i, got.Duration, want.duration)
		}
		if got.Magnitude != want.magnitude {
			t.Errorf("%s affect[%d] Magnitude = %d, want %d", name, i, got.Magnitude, want.magnitude)
		}
		if got.Flags != want.flags {
			t.Errorf("%s affect[%d] Flags = %d, want %d", name, i, got.Flags, want.flags)
		}
	}
}

func TestMagAffectsApply_GoldenAgainstCSource(t *testing.T) {
	ch := &mockAffectCaster{level: 20, class: 0}

	t.Run("chill touch no save", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellChillTouch, false, 0, nil)
		expectAffects(t, "chill touch", v, []affectExpectation{
			{SpellChillTouch, engine.ApplyStr, 4, -1, engine.AFFNone},
		})
	})

	t.Run("chill touch saved", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellChillTouch, true, 0, nil)
		expectAffects(t, "chill touch saved", v, []affectExpectation{
			{SpellChillTouch, engine.ApplyStr, 1, -1, engine.AFFNone},
		})
	})

	t.Run("bless", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellBless, false, 0, nil)
		expectAffects(t, "bless", v, []affectExpectation{
			{SpellBless, engine.ApplyHitroll, 6, 2, engine.AFFNone},
			{SpellBless, engine.ApplySavingSpell, 6, -2, engine.AFFNone},
		})
	})

	t.Run("armor", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellArmor, false, 0, nil)
		expectAffects(t, "armor", v, []affectExpectation{
			{SpellArmor, engine.ApplyAC, 24, -15, engine.AFFNone},
		})
	})

	t.Run("blindness no reagent", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellBlindness, false, 0, nil)
		expectAffects(t, "blindness", v, []affectExpectation{
			{SpellBlindness, engine.ApplyHitroll, 2, -4, engine.AFFNone},
			{SpellBlindness, engine.ApplyNone, 2, 40, engine.AFFBlind},
		})
	})

	t.Run("blindness with reagent", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellBlindness, false, 1, nil)
		expectAffects(t, "blindness reagent", v, []affectExpectation{
			{SpellBlindness, engine.ApplyHitroll, 2, -5, engine.AFFNone},
			{SpellBlindness, engine.ApplyNone, 3, 40, engine.AFFBlind},
		})
	})

	t.Run("curse", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellCurse, false, 0, nil)
		wantDur := 1 + (20 >> 1)
		expectAffects(t, "curse", v, []affectExpectation{
			{SpellCurse, engine.ApplyNone, wantDur, -3, engine.AFFCurse},
			{SpellCurse, engine.ApplyDamroll, wantDur, -3, engine.AFFNone},
			{SpellCurse, engine.ApplyHitroll, wantDur, -3, engine.AFFNone},
		})
	})

	t.Run("invisible", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellInvisible, false, 0, nil)
		expectAffects(t, "invisible", v, []affectExpectation{
			{SpellInvisible, engine.ApplyNone, 12 + 20/4, 0, engine.AFFInvisible},
		})
	})

	t.Run("sanctuary", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellSanctuary, false, 0, nil)
		expectAffects(t, "sanctuary", v, []affectExpectation{
			{SpellSanctuary, engine.ApplyNone, 4, 0, engine.AFFSanctuary},
		})
	})

	t.Run("sleep", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellSleep, false, 0, nil)
		expectAffects(t, "sleep", v, []affectExpectation{
			{SpellSleep, engine.ApplyNone, 4 + 20/4, 0, engine.AFFSleep},
		})
		if v.position != int(PosSleeping) {
			t.Errorf("sleep position = %d, want %d", v.position, int(PosSleeping))
		}
	})

	t.Run("flamestrike", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellFlameStrike, false, 0, nil)
		wantDur := 3
		expectAffects(t, "flamestrike", v, []affectExpectation{
			{SpellFlameStrike, engine.ApplyNone, wantDur, 0, engine.AFFFlaming},
		})
	})

	t.Run("poison", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellPoison, false, 0, nil)
		wantDur := (20 / 2) - 2
		expectAffects(t, "poison", v, []affectExpectation{
			{SpellPoison, engine.ApplyNone, wantDur, -2, engine.AFFPoison},
			{SpellPoison, engine.ApplyStr, wantDur, -2, engine.AFFNone},
			{SpellPoison, engine.ApplyHitroll, wantDur, -2, engine.AFFNone},
		})
	})

	t.Run("haste", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellHaste, false, 0, nil)
		expectAffects(t, "haste", v, []affectExpectation{
			{SpellHaste, engine.ApplyNone, 20, 0, engine.AFFHaste},
		})
	})

	t.Run("slow", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellSlow, false, 0, nil)
		expectAffects(t, "slow", v, []affectExpectation{
			{SpellSlow, engine.ApplyNone, 20, 0, engine.AFFSlow},
		})
	})

	t.Run("detect magic", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellDetectMagic, false, 0, nil)
		expectAffects(t, "detect magic", v, []affectExpectation{
			{SpellDetectMagic, engine.ApplyNone, 12 + 20, 0, engine.AFFDetectMagic},
		})
	})

	t.Run("detect invis", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellDetectInvis, false, 0, nil)
		expectAffects(t, "detect invis", v, []affectExpectation{
			{SpellDetectInvis, engine.ApplyNone, 12 + 20, 0, engine.AFFDetectInvisible},
		})
	})

	t.Run("infravision", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellInfravision, false, 0, nil)
		expectAffects(t, "infravision", v, []affectExpectation{
			{SpellInfravision, engine.ApplyNone, 12 + 20, 0, engine.AFFInfrared},
		})
	})

	t.Run("water breathe", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellWaterBreathe, false, 0, nil)
		expectAffects(t, "water breathe", v, []affectExpectation{
			{SpellWaterBreathe, engine.ApplyNone, 20, 0, engine.AFFWaterBreathing},
		})
	})

	t.Run("detect align", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellDetectAlign, false, 0, nil)
		expectAffects(t, "detect align", v, []affectExpectation{
			{SpellDetectAlign, engine.ApplyNone, 12 + 20, 0, engine.AFFDetectAlign},
		})
	})

	t.Run("dream travel", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellDreamTravel, false, 0, nil)
		expectAffects(t, "dream travel", v, []affectExpectation{
			{SpellDreamTravel, engine.ApplyNone, 6, 0, engine.AFFDream},
		})
	})

	t.Run("levitate", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellLevitate, false, 0, nil)
		expectAffects(t, "levitate", v, []affectExpectation{
			{SpellLevitate, engine.ApplyNone, 20, 0, engine.AFFFlying},
		})
	})

	t.Run("fly", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellFly, false, 0, nil)
		expectAffects(t, "fly", v, []affectExpectation{
			{SpellFly, engine.ApplyNone, 20, 0, engine.AFFFlying},
		})
	})

	t.Run("prot from evil", func(t *testing.T) {
		v := &mockAffectVictim{alignment: 0}
		magAffectsApply(20, ch, v, SpellProtFromEvil, false, 0, nil)
		expectAffects(t, "prot from evil", v, []affectExpectation{
			{SpellProtFromEvil, engine.ApplyNone, 24, 0, engine.AFFProtectionEvil},
		})
	})

	t.Run("prot from good", func(t *testing.T) {
		v := &mockAffectVictim{alignment: 0}
		magAffectsApply(20, ch, v, SpellProtFromGood, false, 0, nil)
		expectAffects(t, "prot from good", v, []affectExpectation{
			{SpellProtFromGood, engine.ApplyNone, 24, 0, engine.AFFProtectionGood},
		})
	})

	t.Run("strength", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellStrength, false, 0, nil)
		expectAffects(t, "strength", v, []affectExpectation{
			{SpellStrength, engine.ApplyStr, (20 >> 1) + 4, 2, engine.AFFNone},
		})
	})

	t.Run("adrenaline", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, v, v, SpellAdrenaline, false, 0, nil)
		expectAffects(t, "adrenaline", v, []affectExpectation{
			{SpellStrength, engine.ApplyStr, (20 >> 1) + 4, 3, engine.AFFNone},
		})
	})

	t.Run("sense life", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellSenseLife, false, 0, nil)
		expectAffects(t, "sense life", v, []affectExpectation{
			{SpellSenseLife, engine.ApplyNone, 20, 0, engine.AFFSenseLife},
		})
	})

	t.Run("waterwalk", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellWaterwalk, false, 0, nil)
		// C's duration expression "4+reag?20:0" parses as (4+reag)?20:0 —
		// always 20 (magic.c:1279 quirk kept).
		expectAffects(t, "waterwalk", v, []affectExpectation{
			{SpellWaterwalk, engine.ApplyNone, 20, 0, engine.AFFWaterwalk},
		})
	})

	t.Run("change density", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellChangeDensity, false, 0, nil)
		// Shares C's waterwalk arm, including the always-20 duration quirk.
		expectAffects(t, "change density", v, []affectExpectation{
			{SpellChangeDensity, engine.ApplyNone, 20, 0, engine.AFFWaterwalk},
		})
	})

	t.Run("chameleon", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellChameleon, false, 0, nil)
		expectAffects(t, "chameleon", v, []affectExpectation{
			{SpellChameleon, engine.ApplyNone, 20, 0, engine.AFFHide},
		})
	})

	t.Run("metalskin", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellMetalskin, false, 0, nil)
		wantMag := -(15 + 20/2)
		expectAffects(t, "metalskin", v, []affectExpectation{
			{SpellMetalskin, engine.ApplyNone, 5, wantMag, engine.AFFMetalskin},
			{SpellMetalskin, engine.ApplyAC, 5, wantMag, engine.AFFNone},
		})
	})

	t.Run("invulnerability", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellInvulnerability, false, 0, nil)
		expectAffects(t, "invulnerability", v, []affectExpectation{
			{SpellInvulnerability, engine.ApplyNone, 7, -100, engine.AFFInvuln},
			{SpellInvulnerability, engine.ApplySavingSpell, 7, -7, engine.AFFNone},
		})
	})

	t.Run("psyshield", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellPsyshield, false, 0, nil)
		expectAffects(t, "psyshield", v, []affectExpectation{
			{SpellPsyshield, engine.ApplyAC, 20 / 2, -15, engine.AFFNone},
		})
	})

	t.Run("great percept", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellGreatPercept, false, 0, nil)
		wantDur := 20/2 + 4
		expectAffects(t, "great percept", v, []affectExpectation{
			{SpellGreatPercept, engine.ApplyNone, wantDur, 0, engine.AFFDetectInvisible},
			{SpellGreatPercept, engine.ApplyNone, wantDur, 0, engine.AFFSenseLife},
		})
	})

	t.Run("less percept", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellLessPercept, false, 0, nil)
		wantDur := 20/2 + 4
		expectAffects(t, "less percept", v, []affectExpectation{
			{SpellLessPercept, engine.ApplyNone, wantDur, 0, engine.AFFDetectAlign},
			{SpellLessPercept, engine.ApplyNone, wantDur, 0, engine.AFFInfrared},
		})
	})

	t.Run("intellect", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellIntellect, false, 0, nil)
		expectAffects(t, "intellect", v, []affectExpectation{
			{SpellIntellect, engine.ApplyInt, 8, 1, engine.AFFNone},
		})
	})

	t.Run("mind bar", func(t *testing.T) {
		v := &mockAffectVictim{}
		magAffectsApply(20, ch, v, SpellMindBar, false, 0, nil)
		expectAffects(t, "mind bar", v, []affectExpectation{
			{SpellMindBar, engine.ApplyNone, (20 / 2) - 2, -18, engine.AFFMindBar},
		})
	})
}
