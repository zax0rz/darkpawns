package spells

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// DP-1468 / DP-1458: a spell has two levels — the cast level (the `level`
// parameter; an item cast passes the object's level, src/spell_parser.c:602)
// and the caster's own level GET_LEVEL(ch). These tests hold the cast level
// different from the caster's level and assert the sites where C reads the
// caster's level. Every test here fails if its site is reverted to the cast
// level.

// C src/magic.c:1823 — lay hands heals dice(3, GET_LEVEL(ch)), so a wand of
// lay hands (spell level 20) wielded by a level-30 user rolls 3d30, not 3d20.
// The dice sides are asserted at the formula level in
// TestMagPointsFormula_GoldenAgainstCSource ("lay hands wand L20 by L30
// caster"); this test proves the wired path reads the caster's level too.
func TestLayHandsItemCastRollsCasterLevelDice(t *testing.T) {
	caster := &mockSpellsChar{name: "ItemMage", level: 30}
	formula, ok := magPointsFormula(20, getLevel(caster), SpellLayHands, false)
	if !ok {
		t.Fatal("magPointsFormula(SpellLayHands) returned ok=false")
	}
	if formula.hitNum != 3 || formula.hitSides != 30 {
		t.Fatalf("lay hands item cast hit dice = %dd%d, want 3d30 (GET_LEVEL(ch), src/magic.c:1823)", formula.hitNum, formula.hitSides)
	}
}

// C src/spells.c:1094 — meteor swarm damage is
// GET_LEVEL(ch)*6 + number(-10, GET_LEVEL(ch)*3): one number() draw whose
// bounds move with the caster's level. Replaying the same seeded draw proves
// the damage equals the caster-level expression exactly (R3: the draw count
// is unchanged; only the bounds move).
func TestMeteorSwarmItemCastDamagesWithCasterLevel(t *testing.T) {
	const (
		castLevel   = 20
		casterLevel = 30
	)
	caster := &mockSpellsChar{name: "ItemMage", level: casterLevel, roomVNum: 100}
	victim := &mockSpellsChar{name: "Victim", level: 30, maxHP: 5000, hp: 5000}
	world := &mockAreaWorld{room: &parser.Room{}, chars: []interface{}{caster, victim}}

	dprng.ResetStream(7)
	wantRoll := dprng.Number(-10, casterLevel*3)
	dprng.ResetStream(7)
	castMeteorSwarm(castLevel, caster, world)

	if len(world.calls) != 1 {
		t.Fatalf("meteor swarm damage calls = %d, want 1", len(world.calls))
	}
	want := casterLevel*6 + wantRoll
	if got := world.calls[0].Dam; got != want {
		t.Fatalf("meteor swarm item-cast dam = %d, want %d = %d*6 + draw(-10,%d) from GET_LEVEL(ch) (src/spells.c:1094)",
			got, want, casterLevel, casterLevel*3)
	}
}

// C src/spells.c:1018 — the circle's timer is GET_LEVEL(ch)/2 + number(-2,1).
// A level-30 user of a level-20 staff draws 15 + [-2,1] = [13,16]; the cast
// level would give [8,11]. The ranges are disjoint, so any draw discriminates.
func TestCircleOfSummoningItemCastTimerUsesCasterLevel(t *testing.T) {
	for seed := uint32(1); seed <= 8; seed++ {
		caster := &mockSpellsChar{name: "ItemMage", level: 30, roomVNum: 8105}
		world := &cocOutputWorld{}

		dprng.ResetStream(seed)
		ExecuteManualSpell(SpellCoC, 20, caster, nil, nil, "", world)

		if world.object == nil {
			t.Fatalf("seed %d: COC object was not spawned", seed)
		}
		timer := world.object.timer
		if timer < 13 || timer > 16 {
			t.Fatalf("seed %d: COC item-cast timer = %d, want in [13,16] = GET_LEVEL(30)/2 + number(-2,1) (src/spells.c:1018)", seed, timer)
		}
	}
}

// C src/spells.c:1136 — mirror image spawns the clone at a literal level 1
// (DP-1458), whatever the cast or caster level is.
func TestMirrorImageItemCastClonesSpawnAtLevelOne(t *testing.T) {
	caster := &mockSpellsChar{name: "ItemMage", level: 30, roomVNum: 8105}
	world := &mirrorImageOutputWorld{}

	ExecuteManualSpell(SpellMirrorImage, 20, caster, nil, nil, "", world)

	if world.vnum != 69 || world.level != 1 {
		t.Fatalf("mirror image item-cast clone = {vnum:%d level:%d}, want {vnum:69 level:1} (literal 1, src/spells.c:1136)", world.vnum, world.level)
	}
}

// C src/magic.c:1124 — flamestrike's duration is (GET_LEVEL(ch)*.17), the one
// mag_affects duration that reads the caster's level. A level-30 user of a
// level-20 wand gets 5 hours, not 3.
func TestFlameStrikeItemCastDurationUsesCasterLevel(t *testing.T) {
	caster := &mockSpellsChar{name: "ItemMage", level: 30}
	victim := &mockSpellsChar{name: "Victim", level: 30, maxHP: 500, hp: 500}

	magAffectsApply(20, caster, victim, SpellFlameStrike, false, 0, nil)

	if len(victim.activeAffects) != 1 {
		t.Fatalf("flamestrike affects = %d, want 1", len(victim.activeAffects))
	}
	if got := victim.activeAffects[0].Duration; got != 5 {
		t.Fatalf("flamestrike item-cast duration = %d, want 5 = int(30*.17) from GET_LEVEL(ch) (src/magic.c:1124)", got)
	}
}

// C src/spells.c:878 — zen heals 2*GET_LEVEL(ch).
func TestZenItemCastHealsWithCasterLevel(t *testing.T) {
	caster := &mockSpellsChar{name: "ItemMage", level: 30, hp: 10, maxHP: 100, position: int(PosStanding)}

	castZen(20, caster, caster, &zenSpellOutputWorld{})

	if caster.hp != 70 {
		t.Fatalf("zen item-cast HP = %d, want 70 = 10 + 2*GET_LEVEL(30) (src/spells.c:878)", caster.hp)
	}
}

// C src/spells.c:877-878 — on the staff fan-out path (mag_objectmagic walks
// the room, src/spell_parser.c:806-810) victim != ch, and zen ASSIGNS the
// user's hit points from the victim: GET_HIT(ch) = MIN(GET_MAX_HIT(victim),
// GET_HIT(victim)+2*GET_LEVEL(ch)). The cap is the victim's max, so a healthy
// user is dropped to a frail target's HP — DP-1461's staff-olc-16 witness:
// the staff left the user at 21/5000 because the last fan-out target sat at
// 21/21. Fails if the port reverts to additively healing the user capped at
// the user's own max.
func TestZenStaffCastAssignsCasterHPFromVictim(t *testing.T) {
	caster := &mockSpellsChar{name: "ItemMage", level: 30, hp: 4880, maxHP: 5000, position: int(PosStanding)}

	frail := &mockSpellsChar{name: "ItemBuilder", level: 35, hp: 21, maxHP: 21}
	castZen(20, caster, frail, &zenSpellOutputWorld{})
	if caster.hp != 21 {
		t.Fatalf("zen staff-cast HP = %d, want 21 = MIN(21, 21+2*30) assigned from the victim (src/spells.c:878)", caster.hp)
	}
	if caster.position != int(PosStunned) {
		t.Fatalf("zen staff-cast position = %d, want stunned (src/spells.c:877)", caster.position)
	}

	hale := &mockSpellsChar{name: "ItemPeer", level: 30, hp: 100, maxHP: 5000}
	castZen(20, caster, hale, &zenSpellOutputWorld{})
	if caster.hp != 160 {
		t.Fatalf("zen staff-cast HP = %d, want 160 = MIN(5000, 100+2*30) (src/spells.c:878)", caster.hp)
	}
}

// C src/spells.c:1178,1212 — divine intervention's angels spawn at
// GET_LEVEL(ch)/2 and carry the charm for (GET_LEVEL(ch)/2)-2 hours.
func TestDivineIntItemCastScalesWithCasterLevel(t *testing.T) {
	oldSave := magDivineIntSaveRoll
	magDivineIntSaveRoll = func(interface{}, int) bool { return true }
	t.Cleanup(func() { magDivineIntSaveRoll = oldSave })

	caster := &alignedSpellProbe{mockSpellsChar: &mockSpellsChar{name: "ItemMage", level: 30}, alignment: 1000}
	world := &divineIntOutputWorld{}

	ExecuteManualSpell(SpellDivineInt, 20, caster, nil, nil, "", world)

	if len(world.spawnLvls) != 2 {
		t.Fatalf("divine int angel spawns = %d, want 2 (alignment 1000)", len(world.spawnLvls))
	}
	for i, lvl := range world.spawnLvls {
		if lvl != 15 {
			t.Fatalf("angel %d item-cast spawn level = %d, want 15 = GET_LEVEL(30)/2 (src/spells.c:1212)", i, lvl)
		}
	}
	for _, angel := range world.angels {
		if len(angel.activeAffects) != 1 {
			t.Fatalf("angel charm affects = %d, want 1", len(angel.activeAffects))
		}
		if got := angel.activeAffects[0].Duration; got != 13 {
			t.Fatalf("angel charm duration = %d, want 13 = GET_LEVEL(30)/2-2 (src/spells.c:1178)", got)
		}
	}
}

// C src/spells.c:1071-1072 — the conjured elemental spawns at
// (GET_LEVEL(ch)/2)+3.
func TestConjureElementalItemCastSpawnsWithCasterLevel(t *testing.T) {
	caster := &mockSpellsChar{name: "ItemMage", level: 30, roomVNum: 100}
	world := &conjureElemWorld{items: []interface{}{&conjureElemItem{vnum: 81}}}

	castConjureElemental(20, caster, world)

	if len(world.spawnLvls) != 1 {
		t.Fatalf("elemental spawns = %d, want 1", len(world.spawnLvls))
	}
	if world.spawnLvls[0] != 18 {
		t.Fatalf("elemental item-cast level = %d, want 18 = GET_LEVEL(30)/2+3 (src/spells.c:1072)", world.spawnLvls[0])
	}
}

// C src/magic.c:1620 — mag_areas passes GET_LEVEL(ch) as mag_damage's level
// for every target, so an item-cast area spell damages with the user's level.
// Replaying the same seeded stream through MagDamage directly proves the
// level MagAreas handed over: the damage equals the GET_LEVEL(ch) replay and
// differs from a cast-level replay by acid blast's flat level term.
func TestMagAreasItemCastDamagesWithCasterLevel(t *testing.T) {
	newWorld := func() (*mockAreaWorld, *mockSpellsChar, *mockSpellsChar) {
		caster := &mockSpellsChar{name: "ItemMage", class: 3, level: 30, roomVNum: 100}
		victim := &mockSpellsChar{name: "Victim", level: 30, maxHP: 5000, hp: 5000}
		world := &mockAreaWorld{room: &parser.Room{}, chars: []interface{}{caster, victim}}
		return world, caster, victim
	}

	areaWorld, caster, areaVictim := newWorld()
	dprng.ResetStream(11)
	MagAreas(20, caster, SpellAcidBlast, int(SaveSpell), areaWorld)

	_, _, casterLevelVictim := newWorld()
	dprng.ResetStream(11)
	MagDamage(30, caster, casterLevelVictim, SpellAcidBlast, 1, nil)

	_, _, castLevelVictim := newWorld()
	dprng.ResetStream(11)
	MagDamage(20, caster, castLevelVictim, SpellAcidBlast, 1, nil)

	areaDam := 5000 - areaVictim.hp
	casterLevelDam := 5000 - casterLevelVictim.hp
	castLevelDam := 5000 - castLevelVictim.hp
	if areaDam == 0 || casterLevelDam == 0 || castLevelDam == 0 {
		t.Fatalf("damage: area=%d casterLevel=%d castLevel=%d, want all nonzero",
			areaDam, casterLevelDam, castLevelDam)
	}
	if areaDam != casterLevelDam {
		t.Fatalf("mag_areas item-cast dam = %d, want %d = the GET_LEVEL(ch) replay (src/magic.c:1620)", areaDam, casterLevelDam)
	}
	if casterLevelDam == castLevelDam {
		t.Fatalf("replays are not level-distinguishing (both %d); pick another spell or seed", casterLevelDam)
	}
}

// --- mocks ---

type conjureElemItem struct {
	vnum int
}

func (i *conjureElemItem) GetVNum() int                 { return i.vnum }
func (i *conjureElemItem) GetName() string              { return "a chunk of earth" }
func (i *conjureElemItem) ExtractFromRoom(roomVNum int) {}

type conjureElemWorld struct {
	items     []interface{}
	spawnLvls []int
	mob       *mockSpellsChar
}

func (w *conjureElemWorld) GetItemsInRoomI(roomVNum int) []interface{} { return w.items }
func (w *conjureElemWorld) SpawnMobWithLevelI(vnum, roomVNum, level int) (interface{}, error) {
	w.spawnLvls = append(w.spawnLvls, level)
	w.mob = &mockSpellsChar{name: "an earth elemental", npc: true, level: level, roomVNum: roomVNum}
	return w.mob, nil
}
func (w *conjureElemWorld) AddFollowerQuiet(ch, leader interface{}) {}
