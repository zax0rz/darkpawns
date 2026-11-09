package game

// Tests for C's create_mobile (src/new_cmds2.c:588-620), the one shared spawn
// path CreateMobileI, and spell_mirror_image's world-side effects
// (src/spells.c:1132-1164). DP-1487.

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// createMobileTestWorld builds a world with the clone prototype (vnum 69), a
// watcher prototype (vnum 2001), the caster's room (1001) and create_mobile's
// hunting room (18201).
func createMobileTestWorld(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Lab", Zone: 1},
			{VNum: huntingStartRoom, Name: "Temple", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 69, Keywords: "clone", ShortDesc: "a clone", LongDesc: "A clone is standing here.\r\n", Level: 40, Exp: 100, AC: 50, Sex: 1, HP: parser.DiceRoll{Num: 5, Sides: 5}},
			{VNum: 2001, Keywords: "watcher", ShortDesc: "a watcher", Level: 3, HP: parser.DiceRoll{Num: 2, Sides: 5}},
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

// TestCreateMobileIAppliesCreateMobileScaling proves the shared path writes
// every field C's create_mobile writes (src/new_cmds2.c:592-607): level, XP 0,
// damroll, NDD, SDD, hitroll, AC, max hit and hit, and the zeroed carry counts.
func TestCreateMobileIAppliesCreateMobileScaling(t *testing.T) {
	w := createMobileTestWorld(t)

	mob, err := w.CreateMobileI(nil, 69, 1001, 20, false)
	if err != nil {
		t.Fatalf("CreateMobileI: %v", err)
	}

	if got := mob.GetLevel(); got != 20 {
		t.Errorf("level = %d, want 20 (GET_LEVEL)", got)
	}
	// The prototype has Exp 100; C overwrites it with 0.
	if got := mob.GetExp(); got != 0 {
		t.Errorf("exp = %d, want 0 (GET_EXP 0)", got)
	}
	if got := mob.GetAC(); got != 100-10*20 {
		t.Errorf("ac = %d, want %d (100 - 10*level)", got, 100-10*20)
	}
	if got := mob.GetMaxHP(); got != 10*20+10 {
		t.Errorf("max hp = %d, want %d (10*level+10)", got, 10*20+10)
	}
	if got := mob.GetHP(); got != 10*20+10 {
		t.Errorf("hp = %d, want %d (GET_HIT = GET_MAX_HIT)", got, 10*20+10)
	}
	if got := mob.GetHitroll(); got != 20 {
		t.Errorf("hitroll = %d, want 20 (GET_HITROLL = level)", got)
	}
	if got := mob.GetDamrollPoint(); got != 14 {
		t.Errorf("damroll = %d, want 14 ((level+1)/1.50)", got)
	}
	if got := mob.GetRoom(); got != 1001 {
		t.Errorf("room = %d, want 1001 (char_to_room ch->in_room)", got)
	}
	if mob.Runtime.CarriedNumber != 0 || mob.Runtime.CarriedWeight != 0 {
		t.Errorf("carry counts = {n:%d w:%d}, want 0/0", mob.Runtime.CarriedNumber, mob.Runtime.CarriedWeight)
	}
}

// TestSpawnMobWithLevelIAppliesCreateMobileScaling is the DP-1487 regression:
// the spell-layer spawn path must apply create_mobile's stats too. Before the
// fix it only called SetLevel, so a mirror clone kept the prototype's XP and
// combat stats (the `XP: [100]` vs `[0]` census symptom).
func TestSpawnMobWithLevelIAppliesCreateMobileScaling(t *testing.T) {
	w := createMobileTestWorld(t)

	spawned, err := w.SpawnMobWithLevelI(69, 1001, 1)
	if err != nil {
		t.Fatalf("SpawnMobWithLevelI: %v", err)
	}
	mob, ok := spawned.(*MobInstance)
	if !ok {
		t.Fatalf("SpawnMobWithLevelI returned %T, want *MobInstance", spawned)
	}
	if got := mob.GetExp(); got != 0 {
		t.Errorf("spell spawn exp = %d, want 0 (create_mobile overwrites prototype XP)", got)
	}
	if got := mob.GetLevel(); got != 1 {
		t.Errorf("spell spawn level = %d, want 1", got)
	}
	if got := mob.GetMaxHP(); got != 20 {
		t.Errorf("spell spawn max hp = %d, want 20 (10*1+10)", got)
	}
	if got := mob.GetAC(); got != 90 {
		t.Errorf("spell spawn ac = %d, want 90 (100-10*1)", got)
	}
}

// TestCreateMobileIHuntingPlacesAtTempleAndHunts covers create_mobile's hunting
// arm (src/new_cmds2.c:609-613): the spawned mobile lands in room 18201 and
// hunts the caster.
func TestCreateMobileIHuntingPlacesAtTempleAndHunts(t *testing.T) {
	w := createMobileTestWorld(t)
	caster := NewPlayer(7, "Summoner", 1001)
	if err := w.AddPlayer(caster); err != nil {
		t.Fatalf("AddPlayer caster: %v", err)
	}

	mob, err := w.CreateMobileI(caster, 69, 1001, 25, true)
	if err != nil {
		t.Fatalf("CreateMobileI hunting: %v", err)
	}
	if got := mob.GetRoom(); got != huntingStartRoom {
		t.Errorf("hunting spawn room = %d, want %d (real_room(18201))", got, huntingStartRoom)
	}
	if got := mob.GetHunting(); got != caster.GetName() {
		t.Errorf("hunting spawn target = %q, want %q (set_hunting)", got, caster.GetName())
	}
}

// mirrorTestCaster implements the identity surface ConfigureMirrorClone needs.
type mirrorTestCaster struct {
	name, title, description string
	sex                      int
}

func (c mirrorTestCaster) GetName() string        { return c.name }
func (c mirrorTestCaster) GetTitle() string       { return c.title }
func (c mirrorTestCaster) GetDescription() string { return c.description }
func (c mirrorTestCaster) GetSex() int            { return c.sex }

// TestConfigureMirrorCloneCopiesIdentity proves src/spells.c:1145-1150: SET_NAME
// (short_descr on a mobile) and short_descr become the caster's name, the long
// description is "<name> <title>\r\n", the description and sex are copied, and
// the prototype keywords ("clone") are left alone.
func TestConfigureMirrorCloneCopiesIdentity(t *testing.T) {
	w := createMobileTestWorld(t)
	clone, err := w.SpawnMob(69, 1001)
	if err != nil {
		t.Fatalf("SpawnMob clone: %v", err)
	}
	caster := mirrorTestCaster{name: "ItemMage", title: "the Warrior", description: "A silver-haired figure.", sex: SexFemale}

	w.ConfigureMirrorClone(clone, caster)

	if got := clone.GetName(); got != "ItemMage" {
		t.Errorf("clone name = %q, want %q", got, "ItemMage")
	}
	if proto := clone.Proto(); proto == nil || proto.Keywords != "clone" {
		t.Errorf("clone keywords changed, want untouched %q", "clone")
	}
	if got := clone.GetLongDesc(); got != "ItemMage the Warrior\r\n" {
		t.Errorf("clone long desc = %q, want %q", got, "ItemMage the Warrior\r\n")
	}
	if proto := clone.Proto(); proto == nil || proto.DetailedDesc != caster.description {
		t.Errorf("clone description = %q, want %q", caster.description, caster.description)
	}
	if got := clone.GetSex(); got != SexFemale {
		t.Errorf("clone sex = %d, want %d", got, SexFemale)
	}
}

// TestMirrorImageHandoffMovesMemoryAndHunting proves src/spells.c:1155-1162: a
// MOB_MEMORY watcher that remembered the caster now remembers the clone, and a
// watcher hunting the caster now hunts the clone. The clone is given a distinct
// name so the retarget is observable — in play the clone carries the caster's
// name, so the hand-off is a same-name transfer. Reverting either branch leaves
// the watcher pointing at the caster and fails this test.
func TestMirrorImageHandoffMovesMemoryAndHunting(t *testing.T) {
	w := createMobileTestWorld(t)
	caster := mirrorTestCaster{name: "ItemMage", title: "the Warrior", description: "A silver-haired figure.", sex: SexMale}

	watcher, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob watcher: %v", err)
	}
	watcher.SetMobFlag(MobFlagMemory)
	watcher.Remember(caster.GetName())
	watcher.SetHunting(caster.GetName())

	// A second mob with no MOB_MEMORY flag must be left untouched.
	ignorant, err := w.SpawnMob(2001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob ignorant: %v", err)
	}
	ignorant.Remember(caster.GetName())

	clone, err := w.SpawnMob(69, 1001)
	if err != nil {
		t.Fatalf("SpawnMob clone: %v", err)
	}
	clone.SetName("a clone") // distinct from the caster so the hand-off is visible

	w.MirrorImageHandoff(clone, caster)

	mem := watcher.GetMemory()
	if len(mem) != 1 || mem[0] != "a clone" {
		t.Errorf("watcher memory = %v, want [a clone] (forget caster, remember clone)", mem)
	}
	if got := watcher.GetHunting(); got != "a clone" {
		t.Errorf("watcher hunting = %q, want %q (set_hunting clone)", got, "a clone")
	}
	if mem := ignorant.GetMemory(); len(mem) != 1 || mem[0] != caster.GetName() {
		t.Errorf("non-memory watcher memory = %v, want untouched [%s]", mem, caster.GetName())
	}
}
