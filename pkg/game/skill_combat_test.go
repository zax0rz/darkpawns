package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newSpikeTestWorld builds a tiny world with one non-peaceful room.
func newSpikeTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Test Room", Zone: 1},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	ch := NewPlayer(1, "Hunter", 1001)
	ch.Level = 10
	ch.Class = ClassWarrior
	ch.Race = RaceHuman
	ch.Inventory = NewInventory()
	ch.Equipment = NewEquipment()
	w.AddPlayer(ch)
	return w, ch
}

// makeSpikeWeapon creates a wieldable weapon whose short description contains keyword.
func makeSpikeWeapon(keyword string) *ObjectInstance {
	return &ObjectInstance{
		Prototype: &parser.Obj{
			VNum:      1,
			Keywords:  keyword + " weapon",
			ShortDesc: "a sharp " + keyword,
			TypeFlag:  5, // ITEM_WEAPON
			WearFlags: [4]int{1 << 13, 0, 0, 0}, // ITEM_WEAR_WIELD
			Values:    [4]int{0, 1, 4, 3},
		},
	}
}

func equipWeapon(t *testing.T, ch *Player, weapon *ObjectInstance) {
	t.Helper()
	ch.Inventory.Items = append(ch.Inventory.Items, weapon)
	if err := ch.Equipment.Equip(weapon, ch.Inventory); err != nil {
		t.Fatalf("equip weapon: %v", err)
	}
}

func TestDoSpike_RequiresWieldedWeapon(t *testing.T) {
	w, ch := newSpikeTestWorld(t)
	victim := NewPlayer(2, "Wolf", 1001)
	victim.SetAffect(affWerewolf, true)
	w.AddPlayer(victim)

	result := DoSpike(ch, victim, 0, w)
	if result.Success {
		t.Error("expected spike to fail without wielded weapon")
	}
	if !strings.Contains(result.MessageToCh, "wield") {
		t.Errorf("expected wield requirement message, got %q", result.MessageToCh)
	}
}

func TestDoSpike_RequiresCorrectAffect(t *testing.T) {
	w, ch := newSpikeTestWorld(t)
	victim := NewPlayer(2, "Villager", 1001)
	w.AddPlayer(victim)

	weapon := makeSpikeWeapon("spike")
	equipWeapon(t, ch, weapon)

	result := DoSpike(ch, victim, 0, w)
	if result.Success {
		t.Error("expected spike to fail on non-werewolf")
	}
	if !strings.Contains(result.MessageToCh, "werewolves") {
		t.Errorf("expected werewolf-only message, got %q", result.MessageToCh)
	}
}

func TestDoSpike_BlocksPeacefulRoom(t *testing.T) {
	w, ch := newSpikeTestWorld(t)
	victim := NewPlayer(2, "Wolf", 1001)
	victim.SetAffect(affWerewolf, true)
	w.AddPlayer(victim)

	weapon := makeSpikeWeapon("spike")
	equipWeapon(t, ch, weapon)

	room := w.GetRoomInWorld(1001)
	room.Flags = []string{"peaceful"}

	result := DoSpike(ch, victim, 0, w)
	if result.Success {
		t.Error("expected spike to fail in peaceful room")
	}
	if !strings.Contains(result.MessageToCh, "holy place") {
		t.Errorf("expected peaceful room message, got %q", result.MessageToCh)
	}
}

func TestDoSpike_BlocksSelfTarget(t *testing.T) {
	w, ch := newSpikeTestWorld(t)
	ch.SetAffect(affWerewolf, true)

	weapon := makeSpikeWeapon("spike")
	equipWeapon(t, ch, weapon)

	result := DoSpike(ch, ch, 0, w)
	if result.Success {
		t.Error("expected spike to block self-target")
	}
	if !strings.Contains(result.MessageToCh, "suicide") {
		t.Errorf("expected self-target message, got %q", result.MessageToCh)
	}
}

func TestDoSpike_BlocksOwnKind(t *testing.T) {
	w, ch := newSpikeTestWorld(t)
	ch.Flags |= 1 << PlrWerewolf

	victim := NewPlayer(2, "Wolf", 1001)
	victim.SetAffect(affWerewolf, true)
	w.AddPlayer(victim)

	weapon := makeSpikeWeapon("spike")
	equipWeapon(t, ch, weapon)

	result := DoSpike(ch, victim, 0, w)
	if result.Success {
		t.Error("expected spike to block attacking own kind")
	}
	if !strings.Contains(result.MessageToCh, "own kind") {
		t.Errorf("expected own-kind message, got %q", result.MessageToCh)
	}
}

func TestDoSpike_KillsWerewolf(t *testing.T) {
	w, ch := newSpikeTestWorld(t)
	ch.Level = 50 // guaranteed success vs low-level victim

	victim := NewPlayer(2, "Wolf", 1001)
	victim.Level = 5
	victim.SetAffect(affWerewolf, true)
	w.AddPlayer(victim)

	weapon := makeSpikeWeapon("spike")
	equipWeapon(t, ch, weapon)

	// Stub RawKill dependencies to avoid side effects.
	origExtract := combat.ExtractChar
	combat.ExtractChar = func(name string) {}
	defer func() { combat.ExtractChar = origExtract }()
	origMakeCorpse := combat.MakeCorpseFunc
	combat.MakeCorpseFunc = func(name string, attackType int) {}
	defer func() { combat.MakeCorpseFunc = origMakeCorpse }()

	result := DoSpike(ch, victim, 0, w)
	if !result.Success {
		t.Errorf("expected spike success, got %q", result.MessageToCh)
	}
	if !strings.Contains(result.MessageToCh, "drive") {
		t.Errorf("expected success message, got %q", result.MessageToCh)
	}
}

func TestDoStake_KillsVampire(t *testing.T) {
	w, ch := newSpikeTestWorld(t)
	ch.Level = 50

	victim := NewPlayer(2, "Vamp", 1001)
	victim.Level = 5
	victim.SetAffect(affVampire, true)
	w.AddPlayer(victim)

	weapon := makeSpikeWeapon("stake")
	equipWeapon(t, ch, weapon)

	origExtract := combat.ExtractChar
	combat.ExtractChar = func(name string) {}
	defer func() { combat.ExtractChar = origExtract }()
	origMakeCorpse := combat.MakeCorpseFunc
	combat.MakeCorpseFunc = func(name string, attackType int) {}
	defer func() { combat.MakeCorpseFunc = origMakeCorpse }()

	result := DoSpike(ch, victim, 1, w)
	if !result.Success {
		t.Errorf("expected stake success, got %q", result.MessageToCh)
	}
}

// newCircleTestWorld builds a minimal world for circle/charge tests.
func newCircleTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Test Room", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 2001, ShortDesc: "a training dummy"},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	ch := NewPlayer(1, "Rogue", 1001)
	ch.Level = 20
	ch.Class = ClassThief
	ch.SetSkill(SkillCircle, 100)
	w.AddPlayer(ch)
	return w, ch
}

// makeCircleWeapon creates a wielded piercing weapon (TYPE_PIERCE - TYPE_HIT = 11).
func makeCircleWeapon() *ObjectInstance {
	return &ObjectInstance{
		Prototype: &parser.Obj{
			VNum:      1,
			Keywords:  "dagger piercing",
			ShortDesc: "a slim dagger",
			TypeFlag:  5,                       // ITEM_WEAPON
			WearFlags: [4]int{1 << 13, 0, 0, 0}, // ITEM_WEAR_WIELD
			Values:    [4]int{0, 1, 6, 11},
		},
	}
}

func TestDoCircle_NoSkill(t *testing.T) {
	w, ch := newCircleTestWorld(t)
	ch.SetSkill(SkillCircle, 0)
	mob := spawnTargetMob(t, w)

	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	result := DoCircle(ch, mob)
	if result.Success {
		t.Error("expected circle to fail without skill")
	}
	if !strings.Contains(result.MessageToCh, "make a circle") {
		t.Errorf("expected no-skill message, got %q", result.MessageToCh)
	}
}

func TestDoCircle_SelfTarget(t *testing.T) {
	w, ch := newCircleTestWorld(t)
	_ = w
	result := DoCircle(ch, ch)
	if result.Success {
		t.Error("expected circle to block self-target")
	}
	if !strings.Contains(result.MessageToCh, "stab yourself") {
		t.Errorf("expected self-target message, got %q", result.MessageToCh)
	}
}

func TestDoCircle_NoWeapon(t *testing.T) {
	w, ch := newCircleTestWorld(t)
	mob := spawnTargetMob(t, w)
	_ = w

	result := DoCircle(ch, mob)
	if result.Success {
		t.Error("expected circle to fail without weapon")
	}
	if !strings.Contains(result.MessageToCh, "wield a weapon") {
		t.Errorf("expected wield requirement message, got %q", result.MessageToCh)
	}
}

func TestDoCircle_WrongWeaponType(t *testing.T) {
	w, ch := newCircleTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeSpikeWeapon("sword") // Values[3] == 3 (slash)
	equipWeapon(t, ch, weapon)

	result := DoCircle(ch, mob)
	if result.Success {
		t.Error("expected circle to fail with non-piercing weapon")
	}
	if !strings.Contains(result.MessageToCh, "Only piercing weapons") {
		t.Errorf("expected piercing-weapon message, got %q", result.MessageToCh)
	}
}

func TestDoCircle_WhileMounted(t *testing.T) {
	w, ch := newCircleTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.MountName = "pony"

	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	result := DoCircle(ch, mob)
	if result.Success {
		t.Error("expected circle to fail while mounted")
	}
	if !strings.Contains(result.MessageToCh, "Dismount") {
		t.Errorf("expected mount message, got %q", result.MessageToCh)
	}
}

func TestDoCircle_MobAware(t *testing.T) {
	w, ch := newCircleTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetMobFlag(MobFlagAware)

	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	result := DoCircle(ch, mob)
	if result.Success {
		t.Error("expected aware mob to block circle")
	}
	if mob.GetFighting() != ch.Name {
		t.Errorf("expected aware mob to start fighting %q, got %q", ch.Name, mob.GetFighting())
	}
	if !strings.Contains(result.MessageToCh, "notices you") {
		t.Errorf("expected noticed message, got %q", result.MessageToCh)
	}
}

func TestDoCircle_Success(t *testing.T) {
	w, ch := newCircleTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	result := DoCircle(ch, mob)
	if !result.Success {
		t.Errorf("expected circle success, got %q", result.MessageToCh)
	}
	if result.Damage <= 0 {
		t.Errorf("expected positive damage, got %d", result.Damage)
	}
	if result.WaitCh != 3 {
		t.Errorf("expected wait 3, got %d", result.WaitCh)
	}
}

// newChargeTestWorld builds a minimal world for charge tests.
func newChargeTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Test Room", Zone: 1},
		},
		Mobs: []parser.Mob{
			{VNum: 2001, ShortDesc: "a training dummy"},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	ch := NewPlayer(1, "Warrior", 1001)
	ch.Level = 30
	ch.Class = ClassWarrior
	ch.SetSkill(SkillCharge, 100)
	w.AddPlayer(ch)
	return w, ch
}

// makeChargeWeapon creates a wielded sword (type 3) or lance (type 12).
func makeChargeWeapon(weaponType int) *ObjectInstance {
	return &ObjectInstance{
		Prototype: &parser.Obj{
			VNum:      1,
			Keywords:  "charge weapon",
			ShortDesc: "a heavy weapon",
			TypeFlag:  5,                       // ITEM_WEAPON
			WearFlags: [4]int{1 << 13, 0, 0, 0}, // ITEM_WEAR_WIELD
			Values:    [4]int{0, 1, 8, weaponType},
		},
	}
}

func TestDoCharge_NoSkill(t *testing.T) {
	w, ch := newChargeTestWorld(t)
	ch.SetSkill(SkillCharge, 0)
	mob := spawnTargetMob(t, w)

	weapon := makeChargeWeapon(3)
	equipWeapon(t, ch, weapon)

	result := DoCharge(ch, mob)
	if result.Success {
		t.Error("expected charge to fail without skill")
	}
	if !strings.Contains(result.MessageToCh, "couldn't charge") {
		t.Errorf("expected no-skill message, got %q", result.MessageToCh)
	}
}

func TestDoCharge_SelfTarget(t *testing.T) {
	_, ch := newChargeTestWorld(t)
	result := DoCharge(ch, ch)
	if result.Success {
		t.Error("expected charge to block self-target")
	}
	if !strings.Contains(result.MessageToCh, "ground") {
		t.Errorf("expected self-target message, got %q", result.MessageToCh)
	}
}

func TestDoCharge_NoWeapon(t *testing.T) {
	w, ch := newChargeTestWorld(t)
	mob := spawnTargetMob(t, w)

	result := DoCharge(ch, mob)
	if result.Success {
		t.Error("expected charge to fail without weapon")
	}
	if !strings.Contains(result.MessageToCh, "barehanded") {
		t.Errorf("expected barehanded message, got %q", result.MessageToCh)
	}
}

func TestDoCharge_WrongWeaponType(t *testing.T) {
	w, ch := newChargeTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeCircleWeapon() // piercing, type 11
	equipWeapon(t, ch, weapon)

	result := DoCharge(ch, mob)
	if result.Success {
		t.Error("expected charge to fail with non-sword/lance weapon")
	}
	if !strings.Contains(result.MessageToCh, "sword or a lance") {
		t.Errorf("expected weapon-type message, got %q", result.MessageToCh)
	}
}

func TestDoCharge_MountedBonusDamage(t *testing.T) {
	w, ch := newChargeTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.MountName = "warhorse"

	weapon := makeChargeWeapon(12) // lance
	equipWeapon(t, ch, weapon)

	result := DoCharge(ch, mob)
	if !result.Success {
		t.Errorf("expected mounted charge success, got %q", result.MessageToCh)
	}
	if result.Damage < 50 {
		t.Errorf("expected mounted bonus damage >= 50, got %d", result.Damage)
	}
	if result.SelfStumble {
		t.Error("mounted charge should not stumble")
	}
}

func TestDoCharge_MobNobashPenalty(t *testing.T) {
	w, ch := newChargeTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetMobFlag(MobFlagNobash)

	weapon := makeChargeWeapon(3) // sword
	equipWeapon(t, ch, weapon)

	result := DoCharge(ch, mob)
	// With a +25 penalty and skill 100 it may still succeed, but the path
	// should at least run without panic and produce a valid result.
	if result.MessageToCh == "" {
		t.Error("expected a message from charge")
	}
}

func TestDoCharge_Success(t *testing.T) {
	w, ch := newChargeTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeChargeWeapon(3) // sword
	equipWeapon(t, ch, weapon)

	result := DoCharge(ch, mob)
	if !result.Success {
		t.Errorf("expected charge success, got %q", result.MessageToCh)
	}
	if result.Damage <= 0 {
		t.Errorf("expected positive damage, got %d", result.Damage)
	}
	if result.WaitCh != 2 {
		t.Errorf("expected wait 2, got %d", result.WaitCh)
	}
}
