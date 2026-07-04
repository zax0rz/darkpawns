package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
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

// TestActiveAffectsModifyCoreStats extends TestActiveAffectsModifyStats to the
// six core stat getters (Str/Dex/Int/Wis/Con/Cha) plus max-pool and
// saving-throw getters, which fold ActiveAffects via the same
// sumAffectModsLocked pattern but live in different files
// (player_combat.go, player_social.go, player_stats.go).
func TestActiveAffectsModifyCoreStats(t *testing.T) {
	p := NewPlayer(1, "Caster", 3001)
	p.Stats.Str, p.Stats.Dex, p.Stats.Int, p.Stats.Wis, p.Stats.Con, p.Stats.Cha = 10, 10, 10, 10, 10, 10
	p.MaxHealth, p.MaxMana, p.MaxMove = 50, 50, 50
	p.SavingThrows = [5]int{0, 0, 0, 0, 0}

	p.AddAffect(engine.NewAffectDirect(0, ApplyStr, 6, 2, 0, "strength"))
	p.AddAffect(engine.NewAffectDirect(0, ApplyDex, 6, -1, 0, "clumsy"))
	p.AddAffect(engine.NewAffectDirect(0, ApplyInt, 6, 1, 0, "intellect"))
	p.AddAffect(engine.NewAffectDirect(0, ApplyWis, 6, 1, 0, "wisdom"))
	p.AddAffect(engine.NewAffectDirect(0, ApplyCon, 6, 3, 0, "con"))
	p.AddAffect(engine.NewAffectDirect(0, ApplyCha, 6, -2, 0, "ugly"))
	p.AddAffect(engine.NewAffectDirect(0, ApplyHit, 6, 20, 0, "vigor"))
	p.AddAffect(engine.NewAffectDirect(0, ApplyMana, 6, 15, 0, "clarity"))
	p.AddAffect(engine.NewAffectDirect(0, ApplyMove, 6, 10, 0, "haste"))
	p.AddAffect(engine.NewAffectDirect(0, ApplySavingPara, 6, 5, 0, "iron will"))

	cases := []struct {
		name string
		got  int
		want int
	}{
		{"GetStr", p.GetStr(), 12},
		{"GetDex", p.GetDex(), 9},
		{"GetInt", p.GetInt(), 11},
		{"GetWis", p.GetWis(), 11},
		{"GetCon", p.GetCon(), 13},
		{"GetCha", p.GetCha(), 8},
		{"GetMaxHP", p.GetMaxHP(), 70},
		{"GetMaxMana", p.GetMaxMana(), 65},
		{"GetMaxMove", p.GetMaxMove(), 60},
		{"GetSavingThrow(0)", p.GetSavingThrow(0), 5},
		{"GetSavingThrow(1)", p.GetSavingThrow(1), 0},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d (active affect not folded in)", c.name, c.got, c.want)
		}
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

// TestAffectUpdateDurationZeroExpires verifies that an affect with duration 0
// is removed on the first AffectUpdate tick, matching C magic.c:441-450 (DP-669).
func TestAffectUpdateDurationZeroExpires(t *testing.T) {
	world, err := NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer world.StopAITicker()

	var messages []string
	world.MessageSink = func(_ string, msg []byte) {
		messages = append(messages, string(msg))
	}

	p := NewPlayer(1, "Tester", 3001)
	world.AddPlayer(p)

	const spellID = 0
	af := engine.NewAffect(spellID, engine.ApplyHitroll, 0, 5, "zero-duration buff")
	p.AddAffect(af)

	if len(p.ActiveAffects) != 1 {
		t.Fatalf("expected 1 affect before update, got %d", len(p.ActiveAffects))
	}

	world.AffectUpdate()

	if len(p.ActiveAffects) != 0 {
		t.Errorf("expected duration-0 affect to be removed after AffectUpdate, got %d affects", len(p.ActiveAffects))
	}

	expectedMsg := SpellWearOffMsg(spellID) + "\r\n"
	found := false
	for _, msg := range messages {
		if msg == expectedMsg {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected wear-off message %q, got %v", expectedMsg, messages)
	}
}

// TestAffectUpdateDurationOneTicksDown verifies that an affect with duration 1
// decrements to 0 and survives until the next tick, matching C magic.c:441-450 (DP-669).
func TestAffectUpdateDurationOneTicksDown(t *testing.T) {
	world, err := NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer world.StopAITicker()

	p := NewPlayer(1, "Tester", 3001)
	world.AddPlayer(p)

	af := engine.NewAffect(0, engine.ApplyHitroll, 1, 5, "one-tick buff")
	p.AddAffect(af)

	world.AffectUpdate()

	if len(p.ActiveAffects) != 1 {
		t.Fatalf("expected affect to survive first tick, got %d", len(p.ActiveAffects))
	}
	if got := p.ActiveAffects[0].Duration; got != 0 {
		t.Errorf("expected duration 0 after first tick, got %d", got)
	}

	world.AffectUpdate()

	if len(p.ActiveAffects) != 0 {
		t.Errorf("expected affect removed after second tick, got %d", len(p.ActiveAffects))
	}
}

// TestAffectUpdateDurationNegativeOnePermanent verifies that duration -1
// survives AffectUpdate unchanged, matching C's GOD-only unlimited affect (DP-669).
func TestAffectUpdateDurationNegativeOnePermanent(t *testing.T) {
	world, err := NewWorld(&parser.World{})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	defer world.StopAITicker()

	p := NewPlayer(1, "Tester", 3001)
	world.AddPlayer(p)

	af := engine.NewAffect(0, engine.ApplyHitroll, -1, 5, "permanent buff")
	p.AddAffect(af)

	world.AffectUpdate()

	if len(p.ActiveAffects) != 1 {
		t.Errorf("expected duration -1 affect to survive AffectUpdate, got %d affects", len(p.ActiveAffects))
	}
	if got := p.ActiveAffects[0].Duration; got != -1 {
		t.Errorf("expected duration to remain -1, got %d", got)
	}
}
