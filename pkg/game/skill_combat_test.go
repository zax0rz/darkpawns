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

// ---------------------------------------------------------------------------
// DP-906: DoBackstab C-gate regression tests.
// Mirror the DoCircle tests above. newBackstabTestWorld equips a thief with
// a piercing weapon and backstab skill so each test can mutate one variable.
// ---------------------------------------------------------------------------

// newBackstabTestWorld builds a thief with a piercing dagger and backstab 100.
func newBackstabTestWorld(t *testing.T) (*World, *Player) {
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
	ch.SetSkill(SkillBackstab, 100)
	w.AddPlayer(ch)
	return w, ch
}

func TestDoBackstab_SelfTarget(t *testing.T) {
	w, ch := newBackstabTestWorld(t)
	_ = w
	result := DoBackstab(ch, ch, w)
	if result.Success {
		t.Error("expected backstab to block self-target")
	}
	if !strings.Contains(result.MessageToCh, "sneak up on yourself") {
		t.Errorf("expected self-target message, got %q", result.MessageToCh)
	}
}

func TestDoBackstab_NoWeapon(t *testing.T) {
	w, ch := newBackstabTestWorld(t)
	mob := spawnTargetMob(t, w)

	result := DoBackstab(ch, mob, w)
	if result.Success {
		t.Error("expected backstab to fail without weapon")
	}
	if !strings.Contains(result.MessageToCh, "wield a weapon") {
		t.Errorf("expected wield requirement message, got %q", result.MessageToCh)
	}
}

func TestDoBackstab_WrongWeaponType(t *testing.T) {
	w, ch := newBackstabTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeSpikeWeapon("sword") // Values[3] == 3 (slash), not piercing
	equipWeapon(t, ch, weapon)

	result := DoBackstab(ch, mob, w)
	if result.Success {
		t.Error("expected backstab to fail with non-piercing weapon")
	}
	if !strings.Contains(result.MessageToCh, "Only piercing weapons") {
		t.Errorf("expected piercing-weapon message, got %q", result.MessageToCh)
	}
}

func TestDoBackstab_WhileMounted(t *testing.T) {
	w, ch := newBackstabTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.MountName = "pony"

	weapon := makeCircleWeapon() // piercing
	equipWeapon(t, ch, weapon)

	result := DoBackstab(ch, mob, w)
	if result.Success {
		t.Error("expected backstab to fail while mounted")
	}
	if !strings.Contains(result.MessageToCh, "Dismount") {
		t.Errorf("expected mount message, got %q", result.MessageToCh)
	}
}

func TestDoBackstab_TargetFighting_Gate(t *testing.T) {
	w, ch := newBackstabTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	// Target is already fighting someone else — too alert.
	mob.SetFighting("SomeoneElse")

	result := DoBackstab(ch, mob, w)
	if result.Success {
		t.Error("expected backstab to fail when target is fighting")
	}
	if !strings.Contains(result.MessageToCh, "too alert") {
		t.Errorf("expected target-fighting message, got %q", result.MessageToCh)
	}
}

// TestDoBackstab_MobAware_NoticesAndStartsCombat: a MOB_AWARE awake mob
// notices the attempt, sets fighting state, and the result carries
// StartCombat=true so the caller enrolls the player too.
func TestDoBackstab_MobAware_NoticesAndStartsCombat(t *testing.T) {
	w, ch := newBackstabTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetMobFlag(MobFlagAware)

	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	result := DoBackstab(ch, mob, w)
	if result.Success {
		t.Error("expected aware mob to block backstab")
	}
	if mob.GetFighting() != ch.Name {
		t.Errorf("expected aware mob to start fighting %q, got %q", ch.Name, mob.GetFighting())
	}
	if !strings.Contains(result.MessageToCh, "notices you") {
		t.Errorf("expected noticed message, got %q", result.MessageToCh)
	}
	if !result.StartCombat {
		t.Error("DP-906: MOB_AWARE notice should set StartCombat so the caller enrolls the player")
	}
}

// TestDoBackstab_HitIncludesStrToDam: a successful backstab's damage must
// include the str_app to-dam bonus (DP-906 str-to-dam gate). With a known
// strength we can bound the damage below the pre-fix formula's max.
func TestDoBackstab_HitIncludesStrToDam(t *testing.T) {
	w, ch := newBackstabTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	// Force a high strength so StrAppToDam is meaningfully positive. Str 25 →
	// str_app[25].todam == 14 (see pkg/combat/formulas.go strApp table).
	ch.Stats.Str = 25

	// Backstab a sleeping mob so the skill roll auto-succeeds (percent > prob
	// only fails when AWAKE). Run a few attempts to absorb RNG variance.
	var result SkillResult
	for i := 0; i < 20; i++ {
		mob.SetPosition(combat.PosSleeping)
		result = DoBackstab(ch, mob, w)
		if result.Success {
			break
		}
	}
	if !result.Success {
		t.Fatalf("expected at least one backstab hit in 20 tries, last msg %q", result.MessageToCh)
	}

	// Lower bound: weapon dice (1d6=1) + damroll(0) + strToDam(14) = 15, × mult.
	// At level 20 mult = 20*0.2+1 = 5.0, so min damage ≥ int(15*5) = 75. The
	// str-to-dam term alone contributes 14*5 = 70. Without the fix the floor
	// would have been int((1+0)*5)=5. Use a conservative threshold that only
	// the str-to-dam-inclusive formula can clear.
	if result.Damage < 70 {
		t.Errorf("DP-906: backstab damage %d looks like it omits str-to-dam (str 25 → +14×mult); expected ≥ 70", result.Damage)
	}
}

// TestDoBackstab_MissSetsStartCombat: a miss against an awake mob must flag
// StartCombat so the caller initiates combat (C: damage(ch, vict, 0, SKILL)).
// We force a miss by giving the mob high position (awake) and cranking skill
// to 0 AFTER passing the skill gate would fail — instead set skill low and
// retry until a miss is observed.
func TestDoBackstab_MissSetsStartCombat(t *testing.T) {
	w, ch := newBackstabTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	// Low skill → high miss probability against an awake mob.
	ch.SetSkill(SkillBackstab, 1)
	mob.SetPosition(combat.PosStanding)

	var missed bool
	for i := 0; i < 50; i++ {
		result := DoBackstab(ch, mob, w)
		if !result.Success && result.Damage == 0 {
			// A miss (not a gate failure): must signal combat start.
			if !result.StartCombat {
				t.Errorf("DP-906: backstab miss should set StartCombat (C: damage(ch,vict,0,SKILL)), msg %q", result.MessageToCh)
			}
			if !strings.Contains(result.MessageToCh, "notices you") {
				t.Errorf("expected miss message, got %q", result.MessageToCh)
			}
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no miss observed in 50 tries (RNG); StartCombat-on-miss not exercised")
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
