package spells

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/engine"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// mockSpellsChar satisfies all local interfaces in affect_spells.go
type mockSpellsChar struct {
	name          string
	npc           bool
	level         int
	class         int
	sex           int
	roomVNum      int
	position      int
	hp            int
	maxHP         int
	move          int
	maxMove       int
	dex           int
	aff           uint64
	flags         uint64
	mobFlags      uint64
	inGroup       bool
	following     string
	messages      []string
	activeAffects []*engine.Affect
	inventory     *mockInventory
}

func (m *mockSpellsChar) GetName() string  { return m.name }
func (m *mockSpellsChar) IsNPC() bool      { return m.npc }
func (m *mockSpellsChar) GetFlags() uint64 { return m.flags }
func (m *mockSpellsChar) HasSpellAffect(n int) bool {
	for _, aff := range m.activeAffects {
		if aff.SpellID == n {
			return true
		}
	}
	return false
}
func (m *mockSpellsChar) GetLevel() int          { return m.level }
func (m *mockSpellsChar) GetClass() int          { return m.class }
func (m *mockSpellsChar) GetSex() int            { return m.sex }
func (m *mockSpellsChar) GetRoomVNum() int       { return m.roomVNum }
func (m *mockSpellsChar) GetPosition() int       { return m.position }
func (m *mockSpellsChar) SetPosition(pos int)    { m.position = pos }
func (m *mockSpellsChar) GetHP() int             { return m.hp }
func (m *mockSpellsChar) GetMaxHP() int          { return m.maxHP }
func (m *mockSpellsChar) SetHP(hp int)           { m.hp = hp }
func (m *mockSpellsChar) SetHealth(hp int)       { m.hp = hp }
func (m *mockSpellsChar) GetMove() int           { return m.move }
func (m *mockSpellsChar) GetMaxMove() int        { return m.maxMove }
func (m *mockSpellsChar) SetMove(mv int)         { m.move = mv }
func (m *mockSpellsChar) SendMessage(msg string) { m.messages = append(m.messages, msg) }
func (m *mockSpellsChar) AddAffect(aff *engine.Affect) {
	m.activeAffects = append(m.activeAffects, aff)
}

func (m *mockSpellsChar) RemoveAffectBySpell(spellNum int) {
	filtered := m.activeAffects[:0]
	for _, aff := range m.activeAffects {
		if aff.SpellID != spellNum {
			filtered = append(filtered, aff)
		}
	}
	m.activeAffects = filtered
}
func (m *mockSpellsChar) GetDex() int                { return m.dex }
func (m *mockSpellsChar) IsInGroup() bool            { return m.inGroup }
func (m *mockSpellsChar) GetFollowing() string       { return m.following }
func (m *mockSpellsChar) IsAffected(bit int) bool    { return m.aff&(1<<bit) != 0 }
func (m *mockSpellsChar) HasMobFlag(bit uint64) bool { return m.flags&bit != 0 }
func (m *mockSpellsChar) GetMobFlags() uint64        { return m.mobFlags }
func (m *mockSpellsChar) GetCha() int                { return 18 }
func (m *mockSpellsChar) GetInventory() ReagentInventory {
	if m.inventory == nil {
		return nil
	}
	return m.inventory
}

func TestMagAffects_ChillTouch(t *testing.T) {
	ch := &mockSpellsChar{level: 10, class: 0}
	victim := &mockSpellsChar{level: 5}

	MagAffects(10, ch, victim, SpellChillTouch, int(SaveSpell), nil)

	if len(victim.activeAffects) != 1 {
		t.Fatalf("expected 1 affect on victim, got %d", len(victim.activeAffects))
	}
	aff := victim.activeAffects[0]
	if aff.SpellID != SpellChillTouch {
		t.Errorf("SpellID = %d, want %d", aff.SpellID, SpellChillTouch)
	}
	if aff.Location != engine.ApplyStr {
		t.Errorf("Location = %d, want %d (ApplyStr)", aff.Location, engine.ApplyStr)
	}
	if aff.Magnitude != -1 {
		t.Errorf("Magnitude = %d, want -1", aff.Magnitude)
	}
}

type summonTestObject struct {
	vnum  int
	timer int
}

func (o *summonTestObject) GetVNum() int       { return o.vnum }
func (o *summonTestObject) GetTimer() int      { return o.timer }
func (o *summonTestObject) SetTimer(timer int) { o.timer = timer }

type summonTestWorld struct {
	items []interface{}
	rooms map[int]*parser.Room
}

func (w *summonTestWorld) GetItemsInRoomI(int) []interface{}    { return w.items }
func (w *summonTestWorld) GetRoomInWorld(vnum int) *parser.Room { return w.rooms[vnum] }

func TestSummonManualDispatchRequiresGlyphBeforeOtherGates(t *testing.T) {
	caster := &mockSpellsChar{name: "Mage", roomVNum: 1001}
	victim := &mockSpellsChar{name: "Target", npc: true, level: 1}

	ExecuteManualSpell(SpellSummon, 20, caster, victim, nil, "", &summonTestWorld{})

	if len(caster.messages) != 1 || caster.messages[0] != "You lack the proper glyphs to perform this magick.\r\n" {
		t.Fatalf("caster messages = %#v, want the exact glyph refusal", caster.messages)
	}
	if len(victim.messages) != 0 {
		t.Fatalf("victim messages = %#v, want none", victim.messages)
	}
}

func TestSummonManualDispatchConsumesCircleThenRejectsNoSummonMob(t *testing.T) {
	oldSaveRoll := magSummonSaveRoll
	magSummonSaveRoll = func(interface{}, int) bool { return false }
	t.Cleanup(func() { magSummonSaveRoll = oldSaveRoll })

	circle := &summonTestObject{vnum: CocVnum, timer: 2}
	caster := &mockSpellsChar{name: "Mage", roomVNum: 1001}
	victim := &mockSpellsChar{name: "Guard", npc: true, level: 1, mobFlags: 1 << 14}
	world := &summonTestWorld{
		items: []interface{}{circle},
		rooms: map[int]*parser.Room{
			1001: {VNum: 1001, Exits: map[string]parser.Exit{"north": {Direction: "north", ToRoom: 1002}}},
		},
	}

	ExecuteManualSpell(SpellSummon, 20, caster, victim, nil, "", world)

	if circle.timer != 1 {
		t.Fatalf("circle timer = %d, want 1 after C circle_check use", circle.timer)
	}
	if len(caster.messages) != 1 || caster.messages[0] != "You failed.\r\n" {
		t.Fatalf("caster messages = %#v, want only You failed", caster.messages)
	}
	if len(victim.messages) != 0 {
		t.Fatalf("victim messages = %#v, want none", victim.messages)
	}
}

func TestSummonManualDispatchRequiresCasterSummonableForPvP(t *testing.T) {
	circle := &summonTestObject{vnum: CocVnum, timer: 2}
	caster := &mockSpellsChar{name: "Mage", level: combat.LVL_IMMORT - 1, roomVNum: 1001}
	victim := &mockSpellsChar{name: "Target", level: 1}
	world := &summonTestWorld{items: []interface{}{circle}}

	ExecuteManualSpell(SpellSummon, 20, caster, victim, nil, "", world)

	if circle.timer != 1 {
		t.Fatalf("circle timer = %d, want 1 before the player target gate", circle.timer)
	}
	if len(caster.messages) != 1 || caster.messages[0] != "Sorry, but you need to be summonable to summon other players.\r\n" {
		t.Fatalf("caster messages = %#v, want exact player-summon refusal", caster.messages)
	}
	if len(victim.messages) != 0 {
		t.Fatalf("victim messages = %#v, want none", victim.messages)
	}
}

func TestSummonManualDispatchHighLevelVictimGetsCNotice(t *testing.T) {
	circle := &summonTestObject{vnum: CocVnum, timer: 2}
	caster := &mockSpellsChar{name: "Mage", level: 40, roomVNum: 1001}
	victim := &mockSpellsChar{name: "Target", level: combat.LVL_IMMORT}
	world := &summonTestWorld{
		items: []interface{}{circle},
		rooms: map[int]*parser.Room{1001: {VNum: 1001, Name: "Summon Test Room"}},
	}

	ExecuteManualSpell(SpellSummon, 40, caster, victim, nil, "", world)

	if len(caster.messages) != 1 || caster.messages[0] != "You failed.\r\n" {
		t.Fatalf("caster messages = %#v, want only You failed", caster.messages)
	}
	if len(victim.messages) != 1 || victim.messages[0] != "Mage just tried to summon you to: Summon Test Room.\r\nHe failed.\r\n" {
		t.Fatalf("victim messages = %#v, want the C room and pronoun notice", victim.messages)
	}
}

func TestSummonManualDispatchAllowsSummonableMortalCaster(t *testing.T) {
	oldSaveRoll := magSummonSaveRoll
	magSummonSaveRoll = func(interface{}, int) bool { return false }
	t.Cleanup(func() { magSummonSaveRoll = oldSaveRoll })

	circle := &summonTestObject{vnum: CocVnum, timer: 2}
	flags := uint64(1) << prfSummonableBit
	caster := &mockSpellsChar{name: "Mage", level: combat.LVL_IMMORT - 1, roomVNum: 1001, flags: flags}
	victim := &mockSpellsChar{name: "Target", level: 1, roomVNum: 1002, flags: flags, position: int(PosStanding)}
	world := &summonTestWorld{
		items: []interface{}{circle},
		rooms: map[int]*parser.Room{
			1001: {VNum: 1001, Exits: map[string]parser.Exit{"north": {Direction: "north", ToRoom: 1002}}},
		},
	}

	ExecuteManualSpell(SpellSummon, 20, caster, victim, nil, "", world)

	if len(caster.messages) != 0 {
		t.Fatalf("caster messages = %#v, want no invented success line", caster.messages)
	}
	if len(victim.messages) != 1 || victim.messages[0] != "Mage has summoned you!\r\n" {
		t.Fatalf("victim messages = %#v, want the C success line", victim.messages)
	}
}

func TestSummonManualDispatchRejectsNonOutlawUnSummonablePC(t *testing.T) {
	oldSaveRoll := magSummonSaveRoll
	magSummonSaveRoll = func(interface{}, int) bool { return false }
	t.Cleanup(func() { magSummonSaveRoll = oldSaveRoll })

	circle := &summonTestObject{vnum: CocVnum, timer: 2}
	caster := &mockSpellsChar{name: "Mage", level: combat.LVL_IMMORT - 1, roomVNum: 1001, flags: uint64(1) << prfSummonableBit}
	victim := &mockSpellsChar{name: "Target", level: 1}
	world := &summonTestWorld{
		items: []interface{}{circle},
		rooms: map[int]*parser.Room{
			1001: {VNum: 1001, Exits: map[string]parser.Exit{"north": {Direction: "north", ToRoom: 1002}}},
		},
	}

	ExecuteManualSpell(SpellSummon, 20, caster, victim, nil, "", world)

	if circle.timer != 1 {
		t.Fatalf("circle timer = %d, want 1 before the non-outlaw target refusal", circle.timer)
	}
	if len(caster.messages) != 1 || caster.messages[0] != "You failed.\r\n" {
		t.Fatalf("caster messages = %#v, want only You failed", caster.messages)
	}
	if len(victim.messages) != 0 {
		t.Fatalf("victim messages = %#v, want none", victim.messages)
	}
}

func TestSummonManualDispatchSleepingVictimDreamOutput(t *testing.T) {
	oldSaveRoll := magSummonSaveRoll
	magSummonSaveRoll = func(interface{}, int) bool { return false }
	t.Cleanup(func() { magSummonSaveRoll = oldSaveRoll })

	circle := &summonTestObject{vnum: CocVnum, timer: 2}
	flags := uint64(1) << prfSummonableBit
	caster := &mockSpellsChar{name: "Mage", level: combat.LVL_IMMORT - 1, roomVNum: 1001, flags: flags}
	victim := &mockSpellsChar{name: "Target", level: 1, roomVNum: 1002, flags: flags, position: int(PosSleeping)}
	world := &summonTestWorld{
		items: []interface{}{circle},
		rooms: map[int]*parser.Room{
			1001: {VNum: 1001, Exits: map[string]parser.Exit{"north": {Direction: "north", ToRoom: 1002}}},
		},
	}

	ExecuteManualSpell(SpellSummon, 20, caster, victim, nil, "", world)

	if len(caster.messages) != 0 {
		t.Fatalf("caster messages = %#v, want no success message", caster.messages)
	}
	if len(victim.messages) != 1 || victim.messages[0] != "You have a strange dream about falling...\r\n" {
		t.Fatalf("sleeping victim messages = %#v, want the C dream line", victim.messages)
	}
}

func TestSummonManualDispatchHonorsNoCharmFlag(t *testing.T) {
	oldSaveRoll := magSummonSaveRoll
	magSummonSaveRoll = func(interface{}, int) bool { return false }
	t.Cleanup(func() { magSummonSaveRoll = oldSaveRoll })

	caster := &mockSpellsChar{name: "Mage", npc: true, roomVNum: 1001}
	victim := &mockSpellsChar{name: "Guard", npc: true, roomVNum: 1002, mobFlags: 1 << mobNoCharmBit}
	world := &summonTestWorld{
		rooms: map[int]*parser.Room{
			1001: {VNum: 1001, Name: "Summon Test Room", Exits: map[string]parser.Exit{"north": {Direction: "north", ToRoom: 1002}}},
		},
	}

	ExecuteManualSpell(SpellSummon, 20, caster, victim, nil, "", world)

	if len(caster.messages) != 1 || caster.messages[0] != "You failed.\r\n" {
		t.Fatalf("caster messages = %#v, want only You failed", caster.messages)
	}
	if len(victim.messages) != 1 || victim.messages[0] != "Mage just tried to summon you to: Summon Test Room.\r\nHe failed.\r\n" {
		t.Fatalf("victim messages = %#v, want the C room and pronoun notice", victim.messages)
	}
}

func TestSummonMobFlagPositionsMatchC(t *testing.T) {
	mob := &mockSpellsChar{mobFlags: (1 << mobNoCharmBit) | (1 << mobNoSummonBit)}
	if !hasSummonMobFlag(mob, mobNoCharmBit) || !hasSummonMobFlag(mob, mobNoSummonBit) {
		t.Fatalf("C summon flags were not read: flags=%#x", mob.mobFlags)
	}
}

func TestMagAffects_Bless(t *testing.T) {
	ch := &mockSpellsChar{level: 20}
	victim := &mockSpellsChar{level: 10}

	MagAffects(20, ch, victim, SpellBless, int(SaveSpell), nil)

	// Bless applies two affects: hitroll +2 and saving spell -2.
	if len(victim.activeAffects) != 2 {
		t.Fatalf("expected 2 affects on victim for Bless, got %d", len(victim.activeAffects))
	}
	aff1 := victim.activeAffects[0]
	if aff1.Location != engine.ApplyHitroll || aff1.Magnitude != 2 {
		t.Errorf("unexpected bless hitroll affect: %+v", aff1)
	}
	aff2 := victim.activeAffects[1]
	if aff2.Location != engine.ApplySavingSpell || aff2.Magnitude != -2 {
		t.Errorf("unexpected bless save affect: %+v", aff2)
	}
}

func TestMagAffects_Armor(t *testing.T) {
	ch := &mockSpellsChar{level: 10}
	victim := &mockSpellsChar{level: 10}

	MagAffects(10, ch, victim, SpellArmor, int(SaveSpell), nil)

	if len(victim.activeAffects) != 1 {
		t.Fatalf("expected 1 affect on victim, got %d", len(victim.activeAffects))
	}
	aff := victim.activeAffects[0]
	if aff.SpellID != SpellArmor || aff.Location != engine.ApplyAC || aff.Magnitude != -15 {
		t.Errorf("unexpected armor affect: %+v", aff)
	}
}

func TestCallMagicArmorSpokenCastAppliesAffect(t *testing.T) {
	caster := &mockSpellsChar{name: "Caster", level: 31, position: int(PosFighting)}
	victim := &mockSpellsChar{name: "Victim", level: 10, position: int(PosStanding)}

	if !CallMagic(caster, victim, nil, SpellArmor, 31, CastSpell, nil) {
		t.Fatal("CallMagic did not dispatch holy ward")
	}
	if len(victim.activeAffects) != 1 {
		t.Fatalf("expected one armor affect, got %d", len(victim.activeAffects))
	}
	aff := victim.activeAffects[0]
	if aff.SpellID != SpellArmor || aff.Location != engine.ApplyAC || aff.Magnitude != -15 {
		t.Errorf("unexpected armor affect from spoken dispatch: %+v", aff)
	}
}

func TestCallMagicPerceptionSpellsApplyAffects(t *testing.T) {
	cases := []struct {
		name     string
		spellNum int
		flag     uint64
	}{
		{"detect alignment", SpellDetectAlign, engine.AFFDetectAlign},
		{"detect invisibility", SpellDetectInvis, engine.AFFDetectInvisible},
		{"detect magic", SpellDetectMagic, engine.AFFDetectMagic},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caster := &mockSpellsChar{name: "Caster", level: 31, position: int(PosStanding)}

			if !CallMagic(caster, caster, nil, tc.spellNum, 31, CastSpell, nil) {
				t.Fatal("CallMagic did not dispatch spell")
			}
			if len(caster.activeAffects) != 1 {
				t.Fatalf("expected one perception affect, got %d", len(caster.activeAffects))
			}
			aff := caster.activeAffects[0]
			if aff.SpellID != tc.spellNum || aff.Location != engine.ApplyNone || aff.Duration != 43 || aff.Flags != tc.flag {
				t.Errorf("unexpected perception affect: %+v", aff)
			}
			if len(caster.messages) != 1 || caster.messages[0] != "Your eyes tingle.\r\n" {
				t.Errorf("self-cast messages = %q, want the C recipient line", caster.messages)
			}
		})
	}
}

// forceMagAffectsSave makes the mag_affects saving throw deterministic for the
// duration of a test: saved=true means the victim resists, saved=false means the
// spell lands. Restores the real (RNG-backed) roll via t.Cleanup.
func forceMagAffectsSave(t *testing.T, saved bool) {
	t.Helper()
	prev := magAffectsSaveRoll
	magAffectsSaveRoll = func(interface{}, int) bool { return saved }
	t.Cleanup(func() { magAffectsSaveRoll = prev })
}

func TestMagAffects_Sleep(t *testing.T) {
	forceMagAffectsSave(t, false)                     // save fails -> Sleep lands
	ch := &mockSpellsChar{level: 30, flags: 1}        // PLR_OUTLAW
	victim := &mockSpellsChar{level: 30, position: 8} // standing = 8

	MagAffects(30, ch, victim, SpellSleep, int(SaveSpell), nil)

	if len(victim.activeAffects) != 1 {
		t.Fatalf("expected 1 affect on victim, got %d", len(victim.activeAffects))
	}
	aff := victim.activeAffects[0]
	if aff.SpellID != SpellSleep {
		t.Errorf("SpellID = %d, want %d", aff.SpellID, SpellSleep)
	}
	if victim.position != 4 { // PosSleeping = 4
		t.Errorf("victim position = %d, want 4 (PosSleeping)", victim.position)
	}
}

func TestMagAffects_Sleep_SavedResists(t *testing.T) {
	forceMagAffectsSave(t, true)                      // save succeeds -> Sleep is resisted
	ch := &mockSpellsChar{level: 30, flags: 1}        // PLR_OUTLAW
	victim := &mockSpellsChar{level: 30, position: 8} // standing = 8

	MagAffects(30, ch, victim, SpellSleep, int(SaveSpell), nil)

	if len(victim.activeAffects) != 0 {
		t.Fatalf("expected no affect on a successful save, got %d", len(victim.activeAffects))
	}
	if victim.position != 8 { // unchanged: still standing
		t.Errorf("victim position = %d, want 8 (standing, unaffected)", victim.position)
	}
}

func TestMagAffects_Poison(t *testing.T) {
	forceMagAffectsSave(t, false) // save fails -> Poison lands
	ch := &mockSpellsChar{level: 20}
	victim := &mockSpellsChar{level: 10}

	MagAffects(20, ch, victim, SpellPoison, int(SaveSpell), nil)

	if len(victim.activeAffects) != 3 {
		t.Fatalf("expected 3 affects on victim for Poison, got %d", len(victim.activeAffects))
	}
	// Direct, Str, and Hitroll affects
	var foundStr, foundHitroll, foundDirect bool
	for _, aff := range victim.activeAffects {
		if aff.SpellID == SpellPoison {
			switch aff.Location {
			case engine.ApplyStr:
				foundStr = true
			case engine.ApplyHitroll:
				foundHitroll = true
			case engine.ApplyNone:
				foundDirect = true
			}
		}
	}

	if !foundStr || !foundHitroll || !foundDirect {
		t.Errorf("missing expected poison affects: str=%t, hitroll=%t, direct=%t", foundStr, foundHitroll, foundDirect)
	}
}

func TestMagAffects_Metalskin(t *testing.T) {
	ch := &mockSpellsChar{
		level: 30,
		inventory: &mockInventory{
			items: []*mockItem{
				{shortDesc: "chunk of iron"},
			},
		},
	}
	victim := &mockSpellsChar{level: 10}

	MagAffects(30, ch, victim, SpellMetalskin, int(SaveSpell), nil)

	// Metalskin applies two affects: AFF_METALSKIN flag and AC modifier.
	if len(victim.activeAffects) != 2 {
		t.Fatalf("expected 2 affects for Metalskin, got %d", len(victim.activeAffects))
	}
	aff1 := victim.activeAffects[0]
	if aff1.SpellID != SpellMetalskin || aff1.Flags != engine.AFFMetalskin {
		t.Errorf("unexpected metalskin flag affect: %+v", aff1)
	}
	aff2 := victim.activeAffects[1]
	if aff2.SpellID != SpellMetalskin || aff2.Location != engine.ApplyAC {
		t.Errorf("unexpected metalskin AC affect: %+v", aff2)
	}
}

func TestMagPoints(t *testing.T) {
	tests := []struct {
		spell            int
		initialHP        int
		maxHP            int
		initialMove      int
		maxMove          int
		wantHPIncrease   bool
		wantMoveIncrease bool
		self             bool
		expected         string
	}{
		{spell: SpellCureLight, initialHP: 50, maxHP: 100, wantHPIncrease: true, expected: "better"},
		{spell: SpellCureCritic, initialHP: 50, maxHP: 100, wantHPIncrease: true, expected: "lot better"},
		{spell: SpellVitality, initialHP: 50, maxHP: 150, initialMove: 50, maxMove: 150, wantHPIncrease: true, wantMoveIncrease: true, expected: "vitalized"},
		{spell: SpellInvigorate, initialHP: 50, maxHP: 50, initialMove: 50, maxMove: 150, wantMoveIncrease: true, expected: "invigorated"},
		{spell: SpellCellAdjustment, initialHP: 50, maxHP: 150, wantHPIncrease: true, expected: "warm feeling"},
		{spell: SpellLayHands, initialHP: 50, maxHP: 150, self: true, wantHPIncrease: true, expected: "mend beneath your hands"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			ch := &mockSpellsChar{name: "Castero", level: 20}
			victim := &mockSpellsChar{name: "Barnaby", hp: tt.initialHP, maxHP: tt.maxHP, move: tt.initialMove, maxMove: tt.maxMove}
			if tt.self {
				victim = ch
				victim.hp, victim.maxHP = tt.initialHP, tt.maxHP
				victim.move, victim.maxMove = tt.initialMove, tt.maxMove
			}

			MagPoints(20, ch, victim, tt.spell, int(SaveSpell), nil)

			if tt.wantHPIncrease && victim.hp <= tt.initialHP {
				t.Errorf("HP did not increase: got %d from %d", victim.hp, tt.initialHP)
			}
			if !tt.wantHPIncrease && victim.hp != tt.initialHP {
				t.Errorf("HP changed to %d, want unchanged %d", victim.hp, tt.initialHP)
			}
			if victim.hp > tt.maxHP {
				t.Errorf("HP overflowed: %d, max is %d", victim.hp, tt.maxHP)
			}
			if tt.wantMoveIncrease && victim.move <= tt.initialMove {
				t.Errorf("movement did not increase: got %d from %d", victim.move, tt.initialMove)
			}
			if !tt.wantMoveIncrease && victim.move != tt.initialMove {
				t.Errorf("movement changed to %d, want unchanged %d", victim.move, tt.initialMove)
			}
			if victim.move > victim.maxMove {
				t.Errorf("movement overflowed: %d, max is %d", victim.move, victim.maxMove)
			}

			// Verify text message
			foundMsg := false
			for _, m := range victim.messages {
				if strings.Contains(m, tt.expected) {
					foundMsg = true
					break
				}
			}
			if !foundMsg {
				t.Errorf("expected msg containing %q, got messages: %v", tt.expected, victim.messages)
			}
		})
	}
}

// TestMagUnaffects pins mag_unaffects (magic.c:1828-1876): the cured pair for
// the vision arm is BLINDNESS and SMOKESCREEN, unaffected victims answer
// NOEFFECT to the caster (SILENT for heal/mass-heal, which ride along other
// routines), and the removed affect yields "Your vision returns!".
func victimText(m *mockSpellsChar) string {
	if len(m.messages) == 0 {
		return ""
	}
	return m.messages[0]
}

func TestMagUnaffects(t *testing.T) {
	newVictim := func() *mockSpellsChar {
		return &mockSpellsChar{
			activeAffects: []*engine.Affect{
				engine.NewAffect(SpellBlindness, engine.ApplyHitroll, 5, -2, "blind"),
				engine.NewAffect(SpellPoison, engine.ApplyStr, 5, -2, "poison"),
				engine.NewAffect(SpellCurse, engine.ApplyHitroll, 5, -2, "curse"),
			},
		}
	}

	t.Run("cures", func(t *testing.T) {
		ch := &mockSpellsChar{}
		victim := newVictim()

		MagUnaffects(20, ch, victim, SpellCureBlind, nil)
		for _, aff := range victim.activeAffects {
			if aff.SpellID == SpellBlindness {
				t.Error("SpellBlindness was not removed")
			}
		}
		if got := victimText(victim); got != "Your vision returns!\r\n" {
			t.Errorf("vision to_vict = %q, want C's 'Your vision returns!'", got)
		}

		MagUnaffects(20, ch, victim, SpellRemovePoison, nil)
		for _, aff := range victim.activeAffects {
			if aff.SpellID == SpellPoison {
				t.Error("SpellPoison was not removed")
			}
		}

		MagUnaffects(20, ch, victim, SpellRemoveCurse, nil)
		for _, aff := range victim.activeAffects {
			if aff.SpellID == SpellCurse {
				t.Error("SpellCurse was not removed")
			}
		}
	})

	t.Run("mass heal on unblinded drinker is silent", func(t *testing.T) {
		ch := &mockSpellsChar{}
		victim := &mockSpellsChar{} // no affects
		MagUnaffects(20, ch, victim, SpellMassHeal, nil)
		if got := victimText(victim); got != "" {
			t.Errorf("victim output = %q, want silence (guard returns before bytes)", got)
		}
		if got := victimText(ch); got != "" {
			t.Errorf("caster output = %q, want silence (heal/mass-heal never NOEFFECT)", got)
		}
	})

	t.Run("cure blind on unblinded victim answers caster NOEFFECT", func(t *testing.T) {
		ch := &mockSpellsChar{}
		victim := &mockSpellsChar{}
		MagUnaffects(20, ch, victim, SpellCureBlind, nil)
		if got := victimText(ch); got != "Nothing seems to happen.\r\n" {
			t.Errorf("caster output = %q, want NOEFFECT", got)
		}
		if got := victimText(victim); got != "" {
			t.Errorf("victim output = %q, want none", got)
		}
	})
}

func TestMagUnaffectsActivityOutputSites(t *testing.T) {
	t.Run("no-effect message goes only to caster", func(t *testing.T) {
		caster := &mockSpellsChar{}
		target := &mockSpellsChar{}
		MagUnaffects(20, caster, target, SpellCureBlind, nil)
		if got := victimText(caster); got != "Nothing seems to happen.\r\n" {
			t.Fatalf("no-effect caster output = %q, want exact C message", got)
		}
		if got := victimText(target); got != "" {
			t.Fatalf("no-effect target output = %q, want silence", got)
		}
	})

	t.Run("cured victim and room witness", func(t *testing.T) {
		caster := newAffectMale("Caster", 8008)
		victim := newAffectMale("Victim", 8008)
		witness := newAffectMale("Witness", 8008)
		victim.affects = []*engine.Affect{
			engine.NewAffect(SpellBlindness, engine.ApplyHitroll, 5, -2, "blind"),
		}
		world := newAffectMsgWorld(caster, victim, witness)

		MagUnaffects(20, caster, victim, SpellCureBlind, world)
		if got := strings.Join(victim.messages, ""); got != "Your vision returns!\r\n" {
			t.Fatalf("cured-victim output = %q, want exact C message", got)
		}
		if got := strings.Join(witness.messages, ""); got != "There's a momentary gleam in Victim's eyes.\r\n" {
			t.Fatalf("room-witness output = %q, want exact C act", got)
		}
		if got := strings.Join(caster.messages, ""); got != "There's a momentary gleam in Victim's eyes.\r\n" {
			t.Fatalf("room-caster output = %q, want exact C act", got)
		}
	})

	t.Run("removed poison output reaches victim and room witness", func(t *testing.T) {
		caster := newAffectMale("Caster", 8008)
		victim := newAffectMale("Victim", 8008)
		witness := newAffectMale("Witness", 8008)
		victim.affects = []*engine.Affect{
			engine.NewAffect(SpellPoison, engine.ApplyStr, 5, -2, "poison"),
		}
		world := newAffectMsgWorld(caster, victim, witness)

		MagUnaffects(20, caster, victim, SpellRemovePoison, world)
		if got := strings.Join(victim.messages, ""); got != "A warm feeling runs through your body!\r\n" {
			t.Fatalf("cured-victim output = %q, want exact C message", got)
		}
		if got := strings.Join(witness.messages, ""); got != "Victim looks better.\r\n" {
			t.Fatalf("room-witness output = %q, want exact C act", got)
		}
		if got := strings.Join(caster.messages, ""); got != "Victim looks better.\r\n" {
			t.Fatalf("room-caster output = %q, want exact C act", got)
		}
	})
}

// mockCorpse satisfies the object interfaces MagSummons uses for C's explicit
// animate-dead container target.
type mockCorpse struct {
	keywords string
	objType  int
	values   [4]int
}

func (m *mockCorpse) GetKeywords() string { return m.keywords }
func (m *mockCorpse) GetObjType() int     { return m.objType }
func (m *mockCorpse) GetValue(idx int) int {
	if idx < 0 || idx >= len(m.values) {
		return 0
	}
	return m.values[idx]
}

// mockAnimateWorld is a minimal world for testing Animate Dead.
type mockAnimateWorld struct {
	items             []interface{}
	players           []interface{}
	removed           bool
	spawnErr          error
	spawnedVNum       int
	spawnedRoom       int
	canRaise          bool
	canRaiseMsg       string
	canRaiseCalled    bool
	charmFollowCalled bool
	charmed           bool
	following         string
}

func (w *mockAnimateWorld) GetItemsInRoomI(roomVNum int) []interface{} { return w.items }
func (w *mockAnimateWorld) RemoveItemFromRoomI(item interface{}, roomVNum int) {
	w.removed = true
}

func (w *mockAnimateWorld) SpawnMobWithLevelI(vnum, roomVNum, level int) (interface{}, error) {
	w.spawnedVNum = vnum
	w.spawnedRoom = roomVNum
	if w.spawnErr != nil {
		return nil, w.spawnErr
	}
	return &mockSpellsChar{name: "zombie"}, nil
}

func (w *mockAnimateWorld) CanRaiseUndeadI(ch interface{}) (bool, string) {
	w.canRaiseCalled = true
	return w.canRaise, w.canRaiseMsg
}

func (w *mockAnimateWorld) ForEachPlayerInRoomInterface(room int, fn func(interface{})) {
	for _, player := range w.players {
		fn(player)
	}
}

func (w *mockAnimateWorld) ForEachMobInRoomInterface(int, func(interface{})) {}

func (w *mockAnimateWorld) CharmAndFollowI(mob, leader interface{}) {
	w.charmFollowCalled = true
	if m, ok := mob.(*mockSpellsChar); ok {
		m.aff |= 1 << 21 // affCharm bit
		w.charmed = true
	}
	if l, ok := leader.(*mockSpellsChar); ok {
		w.following = l.name
	}
}

// forceAnimateDeadRoll makes the SPELL_ANIMATE_DEAD pfail roll deterministic for
// the duration of a test: val < 8 always fails the roll, val >= 8 always passes.
// Restores the real (rand-backed) roll via t.Cleanup.
func forceAnimateDeadRoll(t *testing.T, val int) {
	t.Helper()
	prev := animateDeadPfailRoll
	animateDeadPfailRoll = func() int { return val }
	t.Cleanup(func() { animateDeadPfailRoll = prev })
}

func TestMagSummons_AnimateDead_KeepsCorpseOnSpawnFailure(t *testing.T) {
	forceAnimateDeadRoll(t, 101) // pass the pfail so we reach the spawn attempt
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	corpse := &mockCorpse{keywords: "corpse", objType: 15, values: [4]int{0, 0, 0, 1}}
	world := &mockAnimateWorld{
		items:    []interface{}{corpse},
		spawnErr: fmt.Errorf("spawn failed"),
		canRaise: true,
	}

	MagSummons(10, caster, corpse, SpellAnimateDead, world)

	if world.removed {
		t.Error("corpse was removed even though spawn failed")
	}
	if world.spawnedVNum != 10 {
		t.Errorf("expected zombie vnum 10, got %d", world.spawnedVNum)
	}
}

func TestMagSummons_AnimateDead_InvalidTarget(t *testing.T) {
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	roomCorpse := &mockCorpse{keywords: "corpse", objType: 15, values: [4]int{0, 0, 0, 1}}
	invalidTarget := &mockCorpse{keywords: "bread", objType: 19}
	world := &mockAnimateWorld{items: []interface{}{roomCorpse}, canRaise: true}

	MagSummons(10, caster, invalidTarget, SpellAnimateDead, world)

	if !reflect.DeepEqual(caster.messages, []string{"There is no corpse!\r\n\r\n"}) {
		t.Errorf("invalid-target output = %q, want C's exact no-corpse line", caster.messages)
	}
	if world.spawnedVNum != 0 || world.removed || world.canRaiseCalled {
		t.Errorf("invalid target passed C's early gate: spawn=%d removed=%t raise-check=%t", world.spawnedVNum, world.removed, world.canRaiseCalled)
	}
}

func TestMagSummons_AnimateDead_PfailKeepsCorpseNoSpawn(t *testing.T) {
	forceAnimateDeadRoll(t, 0) // 0 < 8 -> the pfail roll fails
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	corpse := &mockCorpse{keywords: "corpse", objType: 15, values: [4]int{0, 0, 0, 1}}
	world := &mockAnimateWorld{
		items:    []interface{}{corpse},
		canRaise: true,
	}

	MagSummons(10, caster, corpse, SpellAnimateDead, world)

	if world.spawnedVNum != 0 {
		t.Errorf("pfail should abort before spawn, but spawned vnum %d", world.spawnedVNum)
	}
	if world.removed {
		t.Error("pfail should not remove the corpse")
	}
	// C sends mag_summon_fail_msgs[fmsg] with fmsg still 0: a bare CRLF
	// (src/magic.c:1719-1722).
	if !reflect.DeepEqual(caster.messages, []string{"\r\n"}) {
		t.Errorf("pfail message = %q, want C's bare CRLF", caster.messages)
	}
}

func TestMagSummons_AnimateDead_GiddyBlocked(t *testing.T) {
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	corpse := &mockCorpse{keywords: "corpse", objType: 15, values: [4]int{0, 0, 0, 1}}
	world := &mockAnimateWorld{
		items:       []interface{}{corpse},
		canRaise:    false,
		canRaiseMsg: "You are too giddy to have any followers!\r\n",
	}

	MagSummons(10, caster, corpse, SpellAnimateDead, world)

	if world.spawnedVNum != 0 {
		t.Error("expected no spawn when caster is charmed")
	}
	if world.removed {
		t.Error("corpse should not be removed when blocked")
	}
	if !reflect.DeepEqual(caster.messages, []string{"You are too giddy to have any followers!\r\n"}) {
		t.Errorf("giddy output = %q, want C's exact refusal", caster.messages)
	}
}

func TestMagSummons_AnimateDead_FollowerCapBlocked(t *testing.T) {
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	corpse := &mockCorpse{keywords: "corpse", objType: 15, values: [4]int{0, 0, 0, 1}}
	world := &mockAnimateWorld{
		items:       []interface{}{corpse},
		canRaise:    false,
		canRaiseMsg: "You can't have any more followers!\r\n",
	}

	MagSummons(10, caster, corpse, SpellAnimateDead, world)

	if world.spawnedVNum != 0 {
		t.Error("expected no spawn at follower cap")
	}
	if world.removed {
		t.Error("corpse should not be removed when blocked")
	}
	if !reflect.DeepEqual(caster.messages, []string{"You can't have any more followers!\r\n"}) {
		t.Errorf("follower-cap output = %q, want C's exact refusal", caster.messages)
	}
}

func TestMagSummons_AnimateDead_Success(t *testing.T) {
	forceAnimateDeadRoll(t, 101) // pass the pfail so the spawn path runs
	caster := &mockSpellsChar{name: "Necro", level: 10, class: 0, roomVNum: 100}
	corpse := &mockCorpse{keywords: "corpse", objType: 15, values: [4]int{0, 0, 0, 1}}
	observer := &mockSpellsChar{name: "Observer", roomVNum: 100}
	world := &mockAnimateWorld{
		items:    []interface{}{corpse},
		players:  []interface{}{caster, observer},
		canRaise: true,
	}

	MagSummons(10, caster, corpse, SpellAnimateDead, world)
	if world.spawnedVNum != 10 {
		t.Fatalf("expected zombie vnum 10 spawned, got %d", world.spawnedVNum)
	}

	if !world.removed {
		t.Error("corpse should be removed on successful spawn")
	}
	if !world.charmFollowCalled {
		t.Error("CharmAndFollowI should be called on successful spawn")
	}
	if !world.charmed {
		t.Error("spawned mob should be charmed")
	}
	if world.following != caster.name {
		t.Errorf("expected mob to follow %q, got %q", caster.name, world.following)
	}
	foundSuccessMsg := false
	for _, m := range caster.messages {
		if strings.Contains(m, "stands with a life of it's own") {
			foundSuccessMsg = true
			break
		}
	}
	if !foundSuccessMsg {
		t.Errorf("expected success message, got %v", caster.messages)
	}
	// act("\r\n", ...) appends its own CRLF (src/magic.c:1729).
	if got := observer.messages; !reflect.DeepEqual(got, []string{"\r\n\r\n"}) {
		t.Errorf("room message = %q, want C's blank act line plus act's CRLF", got)
	}
}
