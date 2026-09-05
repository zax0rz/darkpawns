package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
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
			TypeFlag:  5,                        // ITEM_WEAPON
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
	orig := combat.GetCallbacks()
	defer combat.SetCallbacks(orig)
	combat.SetCallbacks(&combat.GameCallbacks{
		ExtractChar: func(name string) {},
		MakeCorpse:  func(name string, attackType int) {},
	})

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

	orig := combat.GetCallbacks()
	defer combat.SetCallbacks(orig)
	combat.SetCallbacks(&combat.GameCallbacks{
		ExtractChar: func(name string) {},
		MakeCorpse:  func(name string, attackType int) {},
	})

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
	ch.Stats.Str = 12 // str_app[12].ToDam == 0; avoids skewing damage assertions
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
			TypeFlag:  5,                        // ITEM_WEAPON
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
	if !strings.Contains(result.MessageToCh, "notices you") {
		t.Errorf("expected noticed message, got %q", result.MessageToCh)
	}
	// C leaves the game-layer branch via hit(vict, ch); the command caller
	// performs that synchronous opener and enrolls both fighters.
	if !result.RetaliateHit || result.StartCombat {
		t.Errorf("MOB_AWARE result = retaliate %v, start %v; want true, false", result.RetaliateHit, result.StartCombat)
	}

	// The same early branch still emits the notice when the mob is already
	// fighting, but C skips hit(vict, ch) in that case.
	mob.SetFighting("TheTank")
	busyResult := DoCircle(ch, mob)
	if busyResult.RetaliateHit || busyResult.StartCombat {
		t.Errorf("already-fighting aware result = retaliate %v, start %v; want false, false", busyResult.RetaliateHit, busyResult.StartCombat)
	}
}

func TestDoCircle_Success(t *testing.T) {
	w, ch := newCircleTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	// A passed circle roll still runs hit()'s ordinary THAC0 d20. Sleeping
	// makes that C path deterministic while retaining the circle probability
	// draw and the exact skill-message result contract.
	mob.SetPosition(combat.PosSleeping)
	result := DoCircle(ch, mob)
	if !result.Success {
		t.Errorf("expected circle success, got %#v", result)
	}
	if result.Damage <= 0 || result.SkillMsgType != SkillCircleNum || !result.SkillMsgAfterDamage {
		t.Errorf("circle hit result = %#v; want positive set-173 after-damage result", result)
	}
	if result.WaitCh != 3 {
		t.Errorf("expected wait 3, got %d", result.WaitCh)
	}
}

// TestDoCircle_MissPullsAggro — C: new_cmds.c:2457. A botched circle against
// a target that's fighting someone else pulls the mob's aggro onto the circler
// (stop_fighting + hit(vict, ch)), and the circler enters combat too
// (damage(ch, vict, 0, SKILL_CIRCLE)). The result carries both the ordering
// and caller-side combat work; the game layer does not invent output.
func TestDoCircle_MissPullsAggro(t *testing.T) {
	w, ch := newCircleTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	// Mob is tanked on someone else; circler is assisting (not the mob's target).
	mob.SetFighting("TheTank")
	// Low skill so a miss is near-certain; awake mob so the AWAKE gate applies.
	ch.SetSkill(SkillCircle, 1)
	mob.SetPosition(combat.PosStanding)

	var missed bool
	for i := 0; i < 50; i++ {
		mob.SetFighting("TheTank") // reset each attempt
		result := DoCircle(ch, mob)
		if !result.Success && result.Damage == 0 {
			// Miss: C stops the old engagement before the synchronous retaliation;
			// sendSkillResult then emits set 173 and starts the circle fight.
			if mob.GetFighting() != "" {
				t.Errorf("missed circle should stop the old target engagement, got %q", mob.GetFighting())
			}
			if !result.RetaliateHit || !result.RetaliateHitBeforeSkillMessage || !result.StartCombat {
				t.Errorf("missed circle result = %#v; want retaliation-before-message plus start", result)
			}
			if result.SkillMsgType != SkillCircleNum || result.MessageToCh != "" {
				t.Errorf("missed circle message contract = set %d, ch %q; want set %d and no literal", result.SkillMsgType, result.MessageToCh, SkillCircleNum)
			}
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no miss observed in 50 tries (RNG); miss-pulls-aggro not exercised")
	}
}

// TestDoCircle_HitIncludesStrToDam — the damage formula must include the
// str_app to-dam bonus (DP-circle-fidelity gap #4). With high strength the
// damage floor rises above what weapon-dice + damroll alone could produce.
func TestDoCircle_HitIncludesStrToDam(t *testing.T) {
	w, ch := newCircleTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeCircleWeapon()
	equipWeapon(t, ch, weapon)

	// Str 25 → str_app[25].todam == 14. Circle uses backstab_mult(level)/3;
	// at level 20 mult = (20*0.2+1)/3 = 5/3 = 1. The str-to-dam term adds 14*1.
	// Without the fix the floor is weapon(1d6=1)+damroll(0) = 1; with it, ≥15.
	ch.Stats.Str = 25

	var hit bool
	for i := 0; i < 20; i++ {
		mob.SetPosition(combat.PosSleeping) // sleeping → auto-hit
		result := DoCircle(ch, mob)
		if result.Success {
			if result.Damage < 15 {
				t.Errorf("circle damage %d looks like it omits str-to-dam (str 25 → +14); expected ≥ 15", result.Damage)
			}
			hit = true
			break
		}
	}
	if !hit {
		t.Fatalf("expected at least one circle hit in 20 tries")
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
	if !strings.Contains(result.MessageToCh, "notices you") {
		t.Errorf("expected noticed message, got %q", result.MessageToCh)
	}
	// C act.offensive.c:216 hit(vict, ch): the aware mob swings back at once.
	// RetaliateHit makes the caller (sendSkillResult) enroll target->ch and run
	// one synchronous swing from the target; the game layer no longer sets the
	// mob fighting directly (that belongs to the combat engine at the caller).
	if !result.RetaliateHit {
		t.Error("MOB_AWARE notice should set RetaliateHit so the caller runs the guard's hit(vict, ch)")
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
// StartCombat so the caller initiates combat (C: damage(ch, vict, 0, SKILL)),
// and must route its message through the skill_message path (SkillMsgType=131)
// rather than a hardcoded string (R4). We force a miss by giving the mob high
// position (awake) and low skill, retrying until a miss is observed.
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
			// A miss (not a gate failure): must signal combat start and route
			// the message through the skill_message path (set 131), carrying NO
			// hardcoded MessageToCh (R4 — the old invented "notices you" string
			// is gone).
			if !result.StartCombat {
				t.Errorf("DP-906: backstab miss should set StartCombat (C: damage(ch,vict,0,SKILL)), SkillMsgType %d", result.SkillMsgType)
			}
			if result.SkillMsgType != SkillBackstabNum {
				t.Errorf("miss should route via skill_message (SkillMsgType=%d), got %d", SkillBackstabNum, result.SkillMsgType)
			}
			if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
				t.Errorf("miss should carry no hardcoded messages (R4), got ch=%q vict=%q room=%q",
					result.MessageToCh, result.MessageToVict, result.MessageToRoom)
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
			TypeFlag:  5,                        // ITEM_WEAPON
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

	// Deterministic RNG: success roll (low) + weapon die (low). Wrapped so the
	// global math/rand/v2 source (unseedable) can't flake the test under -race.
	combat.WithRoller(combat.NewScriptedRoller([]int{1, 1}), func() {
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
	})
}

func TestDoCharge_MobNobashPenalty(t *testing.T) {
	w, ch := newChargeTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetMobFlag(MobFlagNobash)
	ch.SetSkill(SkillCharge, 35)

	weapon := makeChargeWeapon(3) // sword
	equipWeapon(t, ch, weapon)

	combat.WithRoller(combat.NewScriptedRoller([]int{1}), func() {
		result := DoCharge(ch, mob)
		if result.Success {
			t.Error("expected MOB_NOBASH penalty to force this skill-35 charge to fail")
		}
		if result.SkillMsgType != SkillChargeNum || !result.StartCombat || !result.SelfStumble {
			t.Errorf("charge result = success %v, skill message %d, start combat %v, stumble %v; want false, %d, true, true", result.Success, result.SkillMsgType, result.StartCombat, result.SelfStumble, SkillChargeNum)
		}
	})
}

func TestDoCharge_Success(t *testing.T) {
	w, ch := newChargeTestWorld(t)
	mob := spawnTargetMob(t, w)

	weapon := makeChargeWeapon(3) // sword
	equipWeapon(t, ch, weapon)

	// Deterministic RNG: success roll (low) + weapon die (low). Wrapped so the
	// global math/rand/v2 source (unseedable) can't flake the test under -race.
	combat.WithRoller(combat.NewScriptedRoller([]int{1, 1}), func() {
		result := DoCharge(ch, mob)
		if !result.Success {
			t.Errorf("expected charge success, got %q", result.MessageToCh)
		}
		if result.Damage <= 0 {
			t.Errorf("expected positive damage, got %d", result.Damage)
		}
		if result.SkillMsgType != SkillChargeNum || !result.StartCombat {
			t.Errorf("charge result = skill message %d, start combat %v; want %d, true", result.SkillMsgType, result.StartCombat, SkillChargeNum)
		}
		if result.DamageSkill != SkillCharge {
			t.Errorf("damage skill = %q, want %q", result.DamageSkill, SkillCharge)
		}
		if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillCharge {
			t.Errorf("deferred improvement = %v, want [%q]", result.DeferredImprove, SkillCharge)
		}
		if result.WaitCh != 2 {
			t.Errorf("expected wait 2, got %d", result.WaitCh)
		}
	})
}

// ---------------------------------------------------------------------------
// DP-931/DP-933: DoBash C-gate regression tests.
// Source: src/act.offensive.c:419-490 (do_bash). Gate order in C: skill ->
// peaceful room -> self-target -> target-sitting -> mounted -> move points ->
// roll (with NOBASH force-fail and sleeping/immortal-caster auto-success
// overrides on percent) -> unconditional WAIT_STATE(ch, PULSE_VIOLENCE*2)
// after either branch (since ch is always a player, !IS_NPC(ch) is always true).
// ---------------------------------------------------------------------------

func newBashTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	w, ch := newCombatTestWorld(t)
	ch.Level = 20
	ch.SetSkill(SkillBash, 100)
	ch.Move = 100
	return w, ch
}

func TestDoBash_BlocksPeacefulRoom(t *testing.T) {
	w, ch := newBashTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)

	room := w.GetRoomInWorld(1001)
	room.Flags = []string{"peaceful"}

	result := DoBash(ch, mob, w)
	if result.Success {
		t.Error("expected bash to fail in peaceful room")
	}
	if !strings.Contains(result.MessageToCh, "peaceful") {
		t.Errorf("expected peaceful-room message, got %q", result.MessageToCh)
	}
}

func TestDoBash_BlocksSelfTarget(t *testing.T) {
	w, ch := newBashTestWorld(t)
	_ = w

	result := DoBash(ch, ch, w)
	if result.Success {
		t.Error("expected bash to block self-target")
	}
	if !strings.Contains(result.MessageToCh, "funny today") {
		t.Errorf("expected self-target message, got %q", result.MessageToCh)
	}
}

func TestDoBash_BlocksMounted(t *testing.T) {
	w, ch := newBashTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.MountName = "pony"

	result := DoBash(ch, mob, w)
	if result.Success {
		t.Error("expected bash to fail while mounted")
	}
	if !strings.Contains(result.MessageToCh, "Dismount") {
		t.Errorf("expected mount message, got %q", result.MessageToCh)
	}
}

// TestDoBash_NobashMobForcesFail: MOB_NOBASH forces percent=101 (always fails)
// unless the caster is an immortal. act.offensive.c:478.
func TestDoBash_NobashMobForcesFail(t *testing.T) {
	w, ch := newBashTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	mob.SetMobFlag(MobFlagNobash)

	for i := 0; i < 20; i++ {
		result := DoBash(ch, mob, w)
		if result.Success {
			t.Fatal("DP-931/933: MOB_NOBASH mob should never be bashable by a mortal")
		}
	}
}

// TestDoBash_ImmortalCasterAutoSucceeds: GET_LEVEL(ch) >= LEVEL_IMMORT forces
// percent=0 (always succeeds), even with zero bash skill. act.offensive.c:481.
func TestDoBash_ImmortalCasterAutoSucceeds(t *testing.T) {
	w, ch := newBashTestWorld(t)
	ch.SetSkill(SkillBash, 1) // low skill — would normally miss almost always
	ch.Level = LVL_IMMORT
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)

	for i := 0; i < 20; i++ {
		ch.Move = 100
		mob.SetPosition(combat.PosFighting)
		result := DoBash(ch, mob, w)
		if !result.Success {
			t.Fatalf("DP-931/933: immortal caster should auto-succeed bash, got %q", result.MessageToCh)
		}
	}
}

// TestDoBash_WaitStateAlwaysTwo: C applies `if (!IS_NPC(ch)) WAIT_STATE(ch,
// PULSE_VIOLENCE*2)` unconditionally AFTER the success/fail branch (overwriting
// whatever the branch set) — since ch is always a player here, WaitCh must be
// 2 on both outcomes. act.offensive.c:488-489.
func TestDoBash_WaitStateAlwaysTwo(t *testing.T) {
	w, ch := newBashTestWorld(t)
	mob := spawnTargetMob(t, w)

	// Immortal-caster override guarantees success deterministically.
	ch.Level = LVL_IMMORT
	mob.SetPosition(combat.PosFighting)
	result := DoBash(ch, mob, w)
	if !result.Success {
		t.Fatalf("expected bash success (immortal auto-success), got %q", result.MessageToCh)
	}
	if result.WaitCh != 2 {
		t.Errorf("expected WaitCh 2 on success, got %d", result.WaitCh)
	}

	// Low skill against a mortal target guarantees a miss (percent range
	// 11-111 vs prob 1 almost always exceeds prob).
	ch.Level = 20
	ch.SetSkill(SkillBash, 1)
	var missResult SkillResult
	var missed bool
	for i := 0; i < 50; i++ {
		ch.Move = 100
		mob.SetPosition(combat.PosFighting)
		missResult = DoBash(ch, mob, w)
		if !missResult.Success {
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no miss observed in 50 tries (RNG)")
	}
	if missResult.WaitCh != 2 {
		t.Errorf("expected WaitCh 2 on failure (unconditional post-branch WAIT_STATE), got %d", missResult.WaitCh)
	}
}

// ---------------------------------------------------------------------------
// DP-932: DoRescue C-gate + combat-engine-wiring regression tests.
// Source: src/act.offensive.c:499-581 (do_rescue). Gate order: skill -> self
// -> already-fighting-target -> mounted -> peaceful-room (outlaws exempt) ->
// find attacker -> roll -> on success: stop_fighting all three, then
// bidirectionally interpose ch between attacker and victim.
// ---------------------------------------------------------------------------

// newRescueTestWorld builds a rescuer (ch), a victim (player), and an
// attacker (mob) already fighting the victim via the real combat engine.
func newRescueTestWorld(t *testing.T) (w *World, ch *Player, victim *Player, attacker *MobInstance, ce *combat.CombatEngine) {
	t.Helper()
	w, ch = newCombatTestWorld(t)
	// DoRescue rolls number(1,101) and fails when the roll > skill (faithful to
	// C: even a 100-skill rescue has a ~1% base fail chance). Set 101 so the roll
	// can never exceed it — these tests assert rescue *wiring*, not the dice, and
	// skill=100 flaked ~1% of runs on a roll of 101.
	ch.SetSkill(SkillRescue, 101)

	victim = NewPlayer(2, "Victim", 1001)
	if err := w.AddPlayer(victim); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	attacker = spawnTargetMob(t, w)

	ce = combat.NewCombatEngine()
	if err := ce.StartCombat(attacker, victim); err != nil {
		t.Fatalf("StartCombat setup: %v", err)
	}
	return w, ch, victim, attacker, ce
}

func TestDoRescue_BlocksMounted(t *testing.T) {
	w, ch, victim, _, ce := newRescueTestWorld(t)
	_ = w
	ch.MountName = "pony"

	result := DoRescue(ch, victim, w, ce)
	if result.Success {
		t.Error("expected rescue to fail while mounted")
	}
	if !strings.Contains(result.MessageToCh, "Dismount") {
		t.Errorf("expected mount message, got %q", result.MessageToCh)
	}
}

func TestDoRescue_BlocksPeacefulRoomForNonOutlaw(t *testing.T) {
	w, ch, victim, _, ce := newRescueTestWorld(t)
	room := w.GetRoomInWorld(1001)
	room.Flags = []string{"peaceful"}

	result := DoRescue(ch, victim, w, ce)
	if result.Success {
		t.Error("expected rescue to fail in peaceful room for a non-outlaw")
	}
	if !strings.Contains(result.MessageToCh, "peaceful") {
		t.Errorf("expected peaceful-room message, got %q", result.MessageToCh)
	}
}

func TestDoRescue_AllowsPeacefulRoomForOutlaw(t *testing.T) {
	w, ch, victim, _, ce := newRescueTestWorld(t)
	room := w.GetRoomInWorld(1001)
	room.Flags = []string{"peaceful"}
	ch.SetPlrFlag(PlrOutlaw, true)

	result := DoRescue(ch, victim, w, ce)
	if !result.Success {
		t.Errorf("expected outlaw to bypass peaceful-room gate, got %q", result.MessageToCh)
	}
}

// TestDoRescue_WiresCombatEngine: DP-932's core finding — a successful rescue
// must actually redirect combat via the engine (attacker turns onto the
// rescuer, victim is freed), not just return cosmetic success messages.
func TestDoRescue_WiresCombatEngine(t *testing.T) {
	w, ch, victim, attacker, ce := newRescueTestWorld(t)

	result := DoRescue(ch, victim, w, ce)
	if !result.Success {
		t.Fatalf("expected rescue success, got %q", result.MessageToCh)
	}

	if victim.GetFighting() != "" {
		t.Errorf("DP-932: victim should be freed from combat, still fighting %q", victim.GetFighting())
	}
	if attacker.GetFighting() != ch.Name {
		t.Errorf("DP-932: attacker should now be fighting the rescuer %q, got %q", ch.Name, attacker.GetFighting())
	}
	if ch.GetFighting() != attacker.GetName() {
		t.Errorf("DP-932: rescuer should now be fighting the attacker %q, got %q", attacker.GetName(), ch.GetFighting())
	}
	if result.WaitCh != 0 {
		t.Errorf("DP-932: C sets no WaitCh on a successful rescue, got %d", result.WaitCh)
	}
	if result.WaitTarget != 2 {
		t.Errorf("expected WaitTarget 2 (2*PULSE_VIOLENCE on victim), got %d", result.WaitTarget)
	}
}

// ---------------------------------------------------------------------------
// DP-934: DoKick C-gate regression tests.
// Source: src/act.offensive.c:587-634 (do_kick) — sparser than bash: only a
// self-target check, a mount check, and an unconditional
// WAIT_STATE(ch, PULSE_VIOLENCE+2) applied AFTER the hit/miss branch (so both
// outcomes get WaitCh=3). No peaceful-room, NOBASH, or sleeping overrides.
// ---------------------------------------------------------------------------

func newKickTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	w, ch := newCombatTestWorld(t)
	ch.Level = 20
	ch.SetSkill(SkillKick, 100)
	return w, ch
}

func TestDoKick_BlocksSelfTarget(t *testing.T) {
	_, ch := newKickTestWorld(t)

	result := DoKick(ch, ch)
	if result.Success {
		t.Error("expected kick to block self-target")
	}
	if !strings.Contains(result.MessageToCh, "funny today") {
		t.Errorf("expected self-target message, got %q", result.MessageToCh)
	}
}

func TestDoKick_BlocksMounted(t *testing.T) {
	w, ch := newKickTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.MountName = "pony"

	result := DoKick(ch, mob)
	if result.Success {
		t.Error("expected kick to fail while mounted")
	}
	if !strings.Contains(result.MessageToCh, "Dismount") {
		t.Errorf("expected mount message, got %q", result.MessageToCh)
	}
}

// TestDoKick_WaitStateAlwaysThree: C's WAIT_STATE(ch, PULSE_VIOLENCE+2) sits
// outside the hit/miss if/else, so WaitCh must be 3 on both outcomes.
func TestDoKick_WaitStateAlwaysThree(t *testing.T) {
	w, ch := newKickTestWorld(t)
	mob := spawnTargetMob(t, w)

	// percent ranges up to 115, so skill 100 doesn't guarantee a hit on the
	// first roll; retry like the rest of this file's RNG-backed tests do.
	var result SkillResult
	var hit bool
	for i := 0; i < 20; i++ {
		result = DoKick(ch, mob)
		if result.Success {
			hit = true
			break
		}
	}
	if !hit {
		t.Fatalf("expected kick success with skill 100 vs AC 0 within 20 tries, last msg %q", result.MessageToCh)
	}
	if result.WaitCh != 3 {
		t.Errorf("expected WaitCh 3 on hit, got %d", result.WaitCh)
	}

	ch.SetSkill(SkillKick, 1)
	var missResult SkillResult
	var missed bool
	for i := 0; i < 50; i++ {
		missResult = DoKick(ch, mob)
		if !missResult.Success {
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no miss observed in 50 tries (RNG)")
	}
	if missResult.WaitCh != 3 {
		t.Errorf("expected WaitCh 3 on miss, got %d", missResult.WaitCh)
	}
}

// ---------------------------------------------------------------------------
// DP-937: DoTrip C-gate regression tests.
// Source: src/new_cmds.c:735-815 (do_trip). Gate order: skill -> peaceful ->
// mount -> self -> flying-target -> sleeping-target -> roll (with
// immortal-victim and NOBASH force-fail overrides) -> level-gap penalty.
// On success only the victim gets a wait state (WaitTarget=1); C sets no
// WaitCh at all on a successful trip.
// ---------------------------------------------------------------------------

func newTripTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	w, ch := newCombatTestWorld(t)
	ch.Level = 20
	ch.SetSkill(SkillTrip, 100)
	return w, ch
}

func TestDoTrip_BlocksPeacefulRoom(t *testing.T) {
	w, ch := newTripTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	room := w.GetRoomInWorld(1001)
	room.Flags = []string{"peaceful"}

	result := DoTrip(ch, mob, w)
	if result.Success {
		t.Error("expected trip to fail in peaceful room")
	}
	if !strings.Contains(result.MessageToCh, "peaceful") {
		t.Errorf("expected peaceful-room message, got %q", result.MessageToCh)
	}
}

func TestDoTrip_BlocksMounted(t *testing.T) {
	w, ch := newTripTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	ch.MountName = "pony"

	result := DoTrip(ch, mob, w)
	if result.Success {
		t.Error("expected trip to fail while mounted")
	}
	if !strings.Contains(result.MessageToCh, "Dismount") {
		t.Errorf("expected mount message, got %q", result.MessageToCh)
	}
}

func TestDoTrip_BlocksSelfTarget(t *testing.T) {
	w, ch := newTripTestWorld(t)

	result := DoTrip(ch, ch, w)
	if result.Success {
		t.Error("expected trip to block self-target")
	}
	if !strings.Contains(result.MessageToCh, "shoe laces") {
		t.Errorf("expected self-target message, got %q", result.MessageToCh)
	}
}

func TestDoTrip_BlocksFlyingTarget(t *testing.T) {
	w, ch := newTripTestWorld(t)
	victim := NewPlayer(2, "Bird", 1001)
	victim.SetAffect(affFly, true)
	w.AddPlayer(victim)

	result := DoTrip(ch, victim, w)
	if result.Success {
		t.Error("DP-937: flying target should be untrippable")
	}
	if !strings.Contains(result.MessageToCh, "FLYING") {
		t.Errorf("expected flying message, got %q", result.MessageToCh)
	}
}

// TestDoTrip_ImmortalVictimForcesFail: GET_LEVEL(victim) >= LEVEL_IMMORT
// forces percent=101 (always fails), regardless of skill. new_cmds.c:775.
func TestDoTrip_ImmortalVictimForcesFail(t *testing.T) {
	w, ch := newTripTestWorld(t)
	victim := NewPlayer(2, "God", 1001)
	victim.Level = LVL_IMMORT
	w.AddPlayer(victim)

	for i := 0; i < 20; i++ {
		result := DoTrip(ch, victim, w)
		if result.Success {
			t.Fatal("DP-937: mortals should never be able to trip an immortal")
		}
	}
}

// TestDoTrip_NobashMobForcesFail: MOB_NOBASH forces percent=101 for NPC
// victims. new_cmds.c:776.
func TestDoTrip_NobashMobForcesFail(t *testing.T) {
	w, ch := newTripTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	mob.SetMobFlag(MobFlagNobash)

	for i := 0; i < 20; i++ {
		result := DoTrip(ch, mob, w)
		if result.Success {
			t.Fatal("DP-937: MOB_NOBASH mob should never be trippable")
		}
	}
}

// TestDoTrip_SuccessSetsWaitTargetNotWaitCh: C only calls WAIT_STATE on the
// victim on a successful trip (WAIT_STATE(victim, PULSE_VIOLENCE)); ch gets no
// wait state at all on success. new_cmds.c:803-808.
func TestDoTrip_SuccessSetsWaitTargetNotWaitCh(t *testing.T) {
	w, ch := newTripTestWorld(t)
	mob := spawnTargetMob(t, w)

	// percent ranges up to 121, so skill 100 doesn't guarantee a hit on the
	// first roll; retry like the rest of this file's RNG-backed tests do.
	var result SkillResult
	var hit bool
	for i := 0; i < 20; i++ {
		result = DoTrip(ch, mob, w)
		if result.Success {
			hit = true
			break
		}
	}
	if !hit {
		t.Fatalf("expected trip success with skill 100 vs equal level within 20 tries, last msg %q", result.MessageToCh)
	}
	if result.WaitTarget != 1 {
		t.Errorf("expected WaitTarget 1, got %d", result.WaitTarget)
	}
	if result.WaitCh != 0 {
		t.Errorf("DP-937: C sets no WaitCh on a successful trip, got %d", result.WaitCh)
	}
}

// ---------------------------------------------------------------------------
// DP-936: DoHeadbutt full rewrite regression tests.
// Source: src/new_cmds.c:368-460 (do_headbutt). This is a from-scratch port
// replacing a fabricated formula: damage is flat GET_LEVEL(ch) (not
// skill-based), the success roll is 1-121 (not 1-101), a successful headbutt
// costs the caster HP as recoil (level/4, or level/3 wearing a helm), an HP
// gate refuses the attempt if the recoil could be fatal, and NOBASH/sleeping
// targets auto-succeed (percent=0) rather than the mob being unbashable.
// ---------------------------------------------------------------------------

func newHeadbuttTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	w, ch := newCombatTestWorld(t)
	ch.Level = 40
	ch.SetSkill(SkillHeadbutt, 100)
	ch.SetHP(200)
	ch.Move = 100
	return w, ch
}

func makeHeadItem() *ObjectInstance {
	return &ObjectInstance{
		Prototype: &parser.Obj{
			VNum:      1,
			Keywords:  "helm",
			ShortDesc: "an iron helm",
			TypeFlag:  9, // ITEM_ARMOR (any nonzero, unused by headbutt logic)
			WearFlags: [4]int{1 << 4, 0, 0, 0},
		},
	}
}

func TestDoHeadbutt_BlocksPeacefulRoom(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	mob := spawnTargetMob(t, w)
	room := w.GetRoomInWorld(1001)
	room.Flags = []string{"peaceful"}

	result := DoHeadbutt(ch, mob, w)
	if result.Success {
		t.Error("expected headbutt to fail in peaceful room")
	}
	if !strings.Contains(result.MessageToCh, "Gods prevent") {
		t.Errorf("expected the C peaceful-room headbutt message, got %q", result.MessageToCh)
	}
}

func TestDoHeadbutt_NoSkillMessage(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	ch.SetSkill(SkillHeadbutt, 0)
	mob := spawnTargetMob(t, w)

	result := DoHeadbutt(ch, mob, w)
	if result.Success {
		t.Error("expected headbutt to fail without skill")
	}
	if !strings.Contains(result.MessageToCh, "qualified to headbutt") {
		t.Errorf("expected C's exact no-skill message, got %q", result.MessageToCh)
	}
}

func TestDoHeadbutt_BlocksMounted(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.MountName = "pony"

	result := DoHeadbutt(ch, mob, w)
	if result.Success {
		t.Error("expected headbutt to fail while mounted")
	}
	if !strings.Contains(result.MessageToCh, "Dismount") {
		t.Errorf("expected mount message, got %q", result.MessageToCh)
	}
}

func TestDoHeadbutt_BlocksSelfTarget(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)

	result := DoHeadbutt(ch, ch, w)
	if result.Success {
		t.Error("expected headbutt to block self-target")
	}
	if !strings.Contains(result.MessageToCh, "wall") {
		t.Errorf("expected self-target message, got %q", result.MessageToCh)
	}
}

// TestDoHeadbutt_BlocksImmortalTarget: a mortal cannot headbutt a non-NPC
// immortal — new_cmds.c:407-414. The attacker is thrown to the ground.
func TestDoHeadbutt_BlocksImmortalTarget(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	god := NewPlayer(2, "God", 1001)
	god.Level = LVL_IMMORT
	w.AddPlayer(god)

	result := DoHeadbutt(ch, god, w)
	if result.Success {
		t.Error("expected headbutt to block a non-NPC immortal target")
	}
	if !strings.Contains(result.MessageToCh, "dare you") {
		t.Errorf("expected immortal-target message, got %q", result.MessageToCh)
	}
	if !result.SelfStumble {
		t.Error("expected caster to be thrown down (SelfStumble) attempting to headbutt a god")
	}
}

// TestDoHeadbutt_HPGateRefuses: GET_LEVEL(ch)/2 > GET_HIT(ch) refuses the
// attempt outright — new_cmds.c:429-433.
func TestDoHeadbutt_HPGateRefuses(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	ch.SetHP(ch.GetLevel()/2 - 1) // just under the threshold
	mob := spawnTargetMob(t, w)

	result := DoHeadbutt(ch, mob, w)
	if result.Success {
		t.Error("expected headbutt to be refused when recoil could be fatal")
	}
	if !strings.Contains(result.MessageToCh, "could kill you") {
		t.Errorf("expected HP-gate message, got %q", result.MessageToCh)
	}
}

// TestDoHeadbutt_DamageIsFlatLevel: damage = GET_LEVEL(ch), not a skill-based
// formula. Force a hit via a sleeping target (percent=0 override).
func TestDoHeadbutt_DamageIsFlatLevel(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosSleeping)

	result := DoHeadbutt(ch, mob, w)
	if !result.Success {
		t.Fatalf("expected headbutt to auto-hit a sleeping target, got %q", result.MessageToCh)
	}
	if result.Damage != ch.GetLevel() {
		t.Errorf("DP-936: expected damage == level (%d), got %d", ch.GetLevel(), result.Damage)
	}
}

// TestDoHeadbutt_SelfRecoilNoHelm: a successful headbutt costs the caster
// level/4 HP when not wearing a helm — new_cmds.c:443-445.
func TestDoHeadbutt_SelfRecoilNoHelm(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	ch.SetHP(200)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosSleeping)

	result := DoHeadbutt(ch, mob, w)
	if !result.Success {
		t.Fatalf("expected headbutt success, got %q", result.MessageToCh)
	}
	wantHP := 200 - ch.GetLevel()/4
	if ch.GetHP() != wantHP {
		t.Errorf("DP-936: expected caster HP %d after no-helm recoil, got %d", wantHP, ch.GetHP())
	}
}

// TestDoHeadbutt_SelfRecoilWithHelm: wearing WEAR_HEAD reduces recoil to
// level/3 instead of level/4 — new_cmds.c:439-442.
func TestDoHeadbutt_SelfRecoilWithHelm(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	ch.SetHP(200)
	equipWeapon(t, ch, makeHeadItem())
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosSleeping)

	result := DoHeadbutt(ch, mob, w)
	if !result.Success {
		t.Fatalf("expected headbutt success, got %q", result.MessageToCh)
	}
	wantHP := 200 - ch.GetLevel()/3
	if ch.GetHP() != wantHP {
		t.Errorf("DP-936: expected caster HP %d after helmed recoil, got %d", wantHP, ch.GetHP())
	}
}

// TestDoHeadbutt_NobashMobAutoSucceeds: unlike bash/trip, C's MOB_NOBASH
// override on headbutt sets percent=0 (always SUCCEEDS), not force-fail —
// new_cmds.c:428 (`if (MOB_FLAGGED(victim, MOB_NOBASH)) percent = 0;`).
func TestDoHeadbutt_NobashMobAutoSucceeds(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	ch.Level = 20                 // below LVL_IMMORT, so only the NOBASH override is in play
	ch.SetSkill(SkillHeadbutt, 1) // low skill — would normally miss almost always
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosFighting)
	mob.SetMobFlag(MobFlagNobash)

	for i := 0; i < 20; i++ {
		ch.SetHP(200)
		result := DoHeadbutt(ch, mob, w)
		if !result.Success {
			t.Fatalf("DP-936: MOB_NOBASH should auto-succeed headbutt (C quirk), got %q", result.MessageToCh)
		}
	}
}

// TestDoHeadbutt_TargetSitsOnHit: a successful headbutt sits the victim
// (TargetFalls), not a stun — new_cmds.c:448-452.
func TestDoHeadbutt_TargetSitsOnHit(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosSleeping)

	result := DoHeadbutt(ch, mob, w)
	if !result.Success {
		t.Fatalf("expected headbutt success, got %q", result.MessageToCh)
	}
	if !result.TargetFalls {
		t.Error("DP-936: expected TargetFalls (C sits the victim), not a stun")
	}
	if result.StunTarget {
		t.Error("DP-936: C headbutt sits the victim, it does not stun")
	}
}

// TestDoHeadbutt_WaitStateAlwaysThree: WAIT_STATE(ch, PULSE_VIOLENCE*3) sits
// outside the hit/miss if/else — new_cmds.c:459.
// TestDoHeadbutt_ImprovesSkillTwice pins the DP-1168 fix: C do_headbutt calls
// improve_skill TWICE on a player's success (new_cmds.c:450 unconditional + :457
// under `if(!subcmd)`, and a player is always subcmd==0). Both calls run AFTER
// damage() — after the skill_message dice — so DP-1212 defers them to
// sendSkillResult via SkillResult.DeferredImprove; this test runs them the way
// the sender does. On success with the stat gate forced to pass, the skill
// must gain TWO independent number(1,3) increments.
func TestDoHeadbutt_ImprovesSkillTwice(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t) // level 40 → percent=0 → guaranteed hit
	w.StopAITicker()                 // quiesce the shared stream during draw-count assertions
	ch.Stats.Int = 100
	ch.Stats.Wis = 100          // WIS+INT=200 >= any number(1,200): improve gate always passes
	mob := spawnTargetMob(t, w) // mob 2001 has no HP dice → spawn draws nothing
	mob.SetPosition(combat.PosFighting)

	// DoHeadbutt draws number(1,121) (skill roll, line 388) and defers the two
	// improves; run them (as sendSkillResult does), each number(1,200) gate +
	// number(1,3) increment. Derive both increments from the seeded stream
	// (ResetStream, not Seed — Seed keeps C's carry so it can't reproduce a
	// start state).
	dprng.ResetStream(1)
	dprng.Number(1, 121) // skill roll (percent, then overwritten to 0 for the L40 caster — draw still happens)
	dprng.Number(1, 200) // improveSkill #1 gate (passes)
	inc1 := dprng.Number(1, 3)
	dprng.Number(1, 200) // improveSkill #2 gate (passes)
	inc2 := dprng.Number(1, 3)

	ch.SetSkill(SkillHeadbutt, 50)
	dprng.ResetStream(1)
	result := DoHeadbutt(ch, mob, w)
	if !result.Success {
		t.Fatalf("expected headbutt success (level-40 caster), got %q", result.MessageToCh)
	}
	if len(result.DeferredImprove) != 2 {
		t.Fatalf("headbutt success DeferredImprove = %v, want two entries (C improves twice)", result.DeferredImprove)
	}
	for _, skill := range result.DeferredImprove {
		improveSkill(ch, skill)
	}
	if got, want := ch.GetSkill(SkillHeadbutt), 50+inc1+inc2; got != want {
		t.Errorf("headbutt skill after success = %d, want %d (two improve_skill calls: 50 + %d + %d); "+
			"a single call would give 50 + %d = %d", got, want, inc1, inc2, inc1, 50+inc1)
	}
}

func TestDoHeadbutt_WaitStateAlwaysThree(t *testing.T) {
	w, ch := newHeadbuttTestWorld(t)
	mob := spawnTargetMob(t, w)
	mob.SetPosition(combat.PosSleeping)

	result := DoHeadbutt(ch, mob, w)
	if !result.Success {
		t.Fatalf("expected headbutt success, got %q", result.MessageToCh)
	}
	if result.WaitCh != 3 {
		t.Errorf("expected WaitCh 3 on hit, got %d", result.WaitCh)
	}

	ch.SetHP(200)
	ch.Level = 20 // below LVL_IMMORT, so the immortal-caster auto-success override doesn't apply
	ch.SetSkill(SkillHeadbutt, 1)
	mob.SetPosition(combat.PosFighting)
	var missResult SkillResult
	var missed bool
	for i := 0; i < 50; i++ {
		ch.SetHP(200)
		missResult = DoHeadbutt(ch, mob, w)
		if !missResult.Success {
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no miss observed in 50 tries (RNG)")
	}
	if missResult.WaitCh != 3 {
		t.Errorf("expected WaitCh 3 on miss, got %d", missResult.WaitCh)
	}
	if missResult.Damage != 0 || ch.GetHP() != 200 {
		t.Errorf("DP-936: a miss must not apply the fabricated self-stun/recoil path, HP=%d dam=%d", ch.GetHP(), missResult.Damage)
	}
}

// ---------------------------------------------------------------------------
// DP-935: DoDragonKick C-gate regression tests.
// Source: src/act.offensive.c:636-690 (do_dragon_kick) — same shape as kick:
// self-target + mount gates, unconditional WAIT_STATE(ch, PULSE_VIOLENCE+2)
// after the hit/miss branch.
// ---------------------------------------------------------------------------

func newDragonKickTestWorld(t *testing.T) (*World, *Player) {
	t.Helper()
	w, ch := newCombatTestWorld(t)
	ch.Level = 20
	ch.SetSkill(SkillDragonKick, 100)
	ch.Move = 100
	return w, ch
}

func TestDoDragonKick_BlocksSelfTarget(t *testing.T) {
	_, ch := newDragonKickTestWorld(t)

	result := DoDragonKick(ch, ch)
	if result.Success {
		t.Error("expected dragon kick to block self-target")
	}
	if !strings.Contains(result.MessageToCh, "funny today") {
		t.Errorf("expected self-target message, got %q", result.MessageToCh)
	}
}

func TestDoDragonKick_BlocksMounted(t *testing.T) {
	w, ch := newDragonKickTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.MountName = "pony"

	result := DoDragonKick(ch, mob)
	if result.Success {
		t.Error("expected dragon kick to fail while mounted")
	}
	if !strings.Contains(result.MessageToCh, "Dismount") {
		t.Errorf("expected mount message, got %q", result.MessageToCh)
	}
}

func TestDoDragonKick_WaitStateAlwaysThree(t *testing.T) {
	w, ch := newDragonKickTestWorld(t)
	mob := spawnTargetMob(t, w)

	// percent ranges up to 111, so skill 100 doesn't guarantee a hit on the
	// first roll; retry like the rest of this file's RNG-backed tests do.
	var result SkillResult
	var hit bool
	for i := 0; i < 20; i++ {
		ch.Move = 100
		result = DoDragonKick(ch, mob)
		if result.Success {
			hit = true
			break
		}
	}
	if !hit {
		t.Fatalf("expected dragon kick success with skill 100 vs AC 0 within 20 tries, last msg %q", result.MessageToCh)
	}
	if result.WaitCh != 3 {
		t.Errorf("expected WaitCh 3 on hit, got %d", result.WaitCh)
	}
	if result.SkillMsgType != SkillDragonKickNum {
		t.Errorf("hit SkillMsgType = %d, want C dragon-kick set %d", result.SkillMsgType, SkillDragonKickNum)
	}
	if result.DamageSkill != SkillDragonKick || !result.StartCombat {
		t.Errorf("hit damage contract = skill %q, StartCombat %v; want %q, true", result.DamageSkill, result.StartCombat, SkillDragonKick)
	}
	if wantDamage := ch.GetLevel() * 3 / 2; result.Damage != wantDamage {
		t.Errorf("hit damage = %d, want C integer conversion of level*1.5 = %d", result.Damage, wantDamage)
	}
	if len(result.DeferredImprove) != 1 || result.DeferredImprove[0] != SkillDragonKick {
		t.Errorf("hit DeferredImprove = %v, want [%q] after damage/message path", result.DeferredImprove, SkillDragonKick)
	}
	if result.MessageToCh != "" || result.MessageToVict != "" || result.MessageToRoom != "" {
		t.Errorf("hit should use C skill_message, got literal messages ch=%q vict=%q room=%q", result.MessageToCh, result.MessageToVict, result.MessageToRoom)
	}

	ch.SetSkill(SkillDragonKick, 1)
	var missResult SkillResult
	var missed bool
	for i := 0; i < 50; i++ {
		ch.Move = 100
		missResult = DoDragonKick(ch, mob)
		if !missResult.Success {
			missed = true
			break
		}
	}
	if !missed {
		t.Skip("no miss observed in 50 tries (RNG)")
	}
	if missResult.WaitCh != 3 {
		t.Errorf("expected WaitCh 3 on miss, got %d", missResult.WaitCh)
	}
	if missResult.SkillMsgType != SkillDragonKickNum || !missResult.StartCombat {
		t.Errorf("miss damage contract = set %d, StartCombat %v; want set %d, true", missResult.SkillMsgType, missResult.StartCombat, SkillDragonKickNum)
	}
	if missResult.MessageToCh != "" || missResult.MessageToVict != "" || missResult.MessageToRoom != "" {
		t.Errorf("miss should use C skill_message, got literal messages ch=%q vict=%q room=%q", missResult.MessageToCh, missResult.MessageToVict, missResult.MessageToRoom)
	}
}

func TestDoDragonKick_BlocksInsufficientMove(t *testing.T) {
	w, ch := newDragonKickTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.Move = 9

	result := DoDragonKick(ch, mob)
	if result.Success {
		t.Fatal("dragon kick should fail below the ten-move gate")
	}
	if result.MessageToCh != "You're too exhausted!" {
		t.Errorf("insufficient move message = %q, want exact C text", result.MessageToCh)
	}
	if ch.Move != 9 {
		t.Errorf("insufficient move changed move from 9 to %d", ch.Move)
	}
}

func TestDoDragonKick_MissDrawOrder(t *testing.T) {
	w, ch := newDragonKickTestWorld(t)
	mob := spawnTargetMob(t, w)
	ch.SetSkill(SkillDragonKick, 1)

	cb, _, _, teardown := wireKickMessages(t, ch.Name)
	defer teardown()
	messages := loadMessagesFile(t)
	variants, ok := messages.Variants(SkillDragonKickNum)
	if !ok {
		t.Fatalf("set %d (Dragon Kick) not in messages file", SkillDragonKickNum)
	}

	const seed = 7
	dprng.ResetStream(seed)
	result := DoDragonKick(ch, mob)
	if result.Success || result.SkillMsgType != SkillDragonKickNum {
		t.Fatalf("seed %d did not produce the expected dragon-kick miss: %+v", seed, result)
	}
	if !cb.SkillMessage(0, ch.Name, mob.GetName(), SkillDragonKickNum, ch.GetRoom()) {
		t.Fatal("SkillMessage did not handle the Dragon Kick set")
	}

	dprng.ResetStream(seed)
	dprng.Number(1, 101)
	dprng.Dice(1, len(variants))
	wantNext := dprng.Number(0, 999)

	dprng.ResetStream(seed)
	DoDragonKick(ch, mob)
	cb.SkillMessage(0, ch.Name, mob.GetName(), SkillDragonKickNum, ch.GetRoom())
	if got := dprng.Number(0, 999); got != wantNext {
		t.Fatalf("dragon-kick miss draw order wrong: next=%d want=%d; expected percent then set-%d dice", got, wantNext, SkillDragonKickNum)
	}
}
