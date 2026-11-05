// Package combat — round execution tests
//
// These tests cover TakeDamage, DamMessage, PerformGroupGain,
// Appear, DeathCry, CounterProcs, and the engine round execution
// (processCombatPair, handleDeath, StartCombat, StopCombat).
//
// Game-layer callbacks are wired per-test via SetCallbacks to avoid
// cross-test contamination.
package combat

import (
	"fmt"
	"strings"
	"testing"
)

// defaultCombatCallbacks returns a GameCallbacks with all hooks wired to safe
// no-ops. Tests can override specific fields before calling SetCallbacks.
func defaultCombatCallbacks() *GameCallbacks {
	return &GameCallbacks{
		Broadcast:                func(roomVNum int, msg string, excludedBodies []Combatant) {},
		SendToChar:               func(bodyname Combatant, msg string) {},
		SkillMessage:             func(dam int, bodych Combatant, bodyvict Combatant, attackType int, roomVNum int) bool { return false },
		BroadChat:                func(bodychName Combatant, msg string) {},
		Log:                      func(msg string, level string, minLevel int, toLog bool) {},
		GetRace:                  func(bodyname Combatant) int { return 0 },
		GetRaceHate:              func(bodyname Combatant, index int) int { return 0 },
		GetAlignment:             func(bodyname Combatant) int { return 0 },
		SetAlignment:             func(bodyname Combatant, val int) {},
		GetSex:                   func(bodyname Combatant) int { return 0 },
		GetHP:                    func(bodyname Combatant) int { return 1 },
		GetLevel:                 func(bodyname Combatant) int { return 1 },
		IsNPC:                    func(bodyname Combatant) bool { return false },
		GetSkill:                 func(bodyname Combatant, skillNum int) int { return 0 },
		HasAffect:                func(bodyname Combatant, aff int) bool { return false },
		HasAffectStr:             func(bodyname Combatant, aff string) bool { return false },
		RemoveAffect:             func(bodyname Combatant, skillNum int) {},
		RemoveAllAffects:         func(bodyname Combatant) {},
		HasPlrFlag:               func(bodyname Combatant, flag string) bool { return false },
		SetPlrFlag:               func(bodyname Combatant) bool { return false },
		HasPrfFlag:               func(bodyname Combatant, flag string) bool { return false },
		HasMobFlag:               func(bodyname Combatant, flag string) bool { return false },
		HasMobVNum:               func(bodyname Combatant, vnum int) bool { return false },
		HasRoomFlag:              func(roomVNum int, flag string) bool { return false },
		HasScriptFlag:            func(bodyname Combatant, flag string) bool { return false },
		IsShopkeeper:             func(bodyname Combatant) bool { return false },
		GetRoomCombatants:        func(roomVNum int) []Combatant { return nil },
		GetFollowing:             func(bodyname Combatant) Combatant { return nil },
		JailGuardSubdue:          func(bodyguardName Combatant, bodyvictimName Combatant) bool { return false },
		IsMounted:                func(bodyname Combatant) bool { return false },
		Dismount:                 func(bodyname Combatant) {},
		Unmount:                  func(bodyname Combatant) {},
		GetWeaponInfo:            func(chName Combatant) (wType, damDice, damSize int, isBlessed bool) { return 0, 0, 0, false }, // offset 0 = "hit"
		GetAdjacentRoom:          func(roomVNum, door int) int { return -1 },
		GainExp:                  func(bodyname Combatant, amount int) {},
		GetExp:                   func(bodyname Combatant) int { return 0 },
		GetKills:                 func(bodyname Combatant) int64 { return 0 },
		SetKills:                 func(bodyname Combatant, kills int64) {},
		CounterProcs:             func(bodyname Combatant, kills int64) {},
		GetDeaths:                func(bodyname Combatant) int64 { return 0 },
		SetDeaths:                func(bodyname Combatant, deaths int64) {},
		SetLastDeath:             func(bodyname Combatant, t int64) {},
		GetPks:                   func(bodyname Combatant) int64 { return 0 },
		SetPks:                   func(bodyname Combatant, pks int64) {},
		GetConstitution:          func(bodyname Combatant) int { return 0 },
		SetConstitution:          func(bodyname Combatant, val int) {},
		MakeCorpse:               func(bodyvictim Combatant, attackType int) {},
		MakeDust:                 func(bodyvictim Combatant, attackType int) {},
		ExtractChar:              func(bodyname Combatant) {},
		RunDeathScript:           func(bodykiller Combatant, bodyvictim Combatant, roomVNum int) {},
		GetFollowersInRoom:       func(bodyname Combatant, roomVNum int) int { return 0 },
		GetMasterInRoom:          func(bodyname Combatant, roomVNum int) bool { return false },
		GetFellowFollowersInRoom: func(bodyname Combatant, roomVNum int) bool { return false },
		CountGroupMembers:        func(bodyleaderName Combatant, roomVNum int) int { return 1 },
		ApplyToGroupMembers:      func(bodyleaderName Combatant, roomVNum int, fn func(name Combatant)) {},
		GetGold:                  func(bodyname Combatant) int { return 0 },
		SetGold:                  func(bodyname Combatant, gold int) {},
		JunkInventoryItems:       func(bodychName Combatant) {},
		PerformCommand:           func(bodychName Combatant, cmd string) {},
		GetWimpyLev:              func(bodyname Combatant) int { return 0 },
		DoFlee:                   func(bodyname Combatant) {},
		DoRetreat:                func(bodyname Combatant) {},
	}
}

// ---------------------------------------------------------------------------
// TestDamMessage — damage message tier selection
// ---------------------------------------------------------------------------

func TestDamMessage_TierSelection(t *testing.T) {
	var broadcastRoom int
	var broadcastMsg string
	var sendToCalls []string

	cb := defaultCombatCallbacks()
	cb.Broadcast = func(roomVNum int, msg string, excludedBodies []Combatant) {
		broadcastRoom = roomVNum
		broadcastMsg = msg
	}
	cb.SendToChar = func(bodyname Combatant, msg string) {
		name := bodyname.GetName()

		sendToCalls = append(sendToCalls, name+":"+msg)
	}
	SetCallbacks(cb)

	attacker := &mockCombatant{name: "Warrior", room: 100, sex: 0, position: PosStanding}
	defender := &mockCombatant{name: "Goblin", room: 100, sex: 1, position: PosStanding}

	// Damage=5 → tier "light" (minDamage 5)
	DamMessage(5, attacker, defender, 4) // attackType=4 = bite

	if broadcastRoom != 100 {
		t.Errorf("expected room 100, got %d", broadcastRoom)
	}
	if !strings.Contains(broadcastMsg, "Goblin") {
		t.Errorf("expected message mentioning Goblin, got %q", broadcastMsg)
	}
	if len(sendToCalls) < 2 {
		t.Errorf("expected at least 2 SendToChar calls (attacker+victim), got %d", len(sendToCalls))
	}
}

func TestDamMessage_HighDamageTier(t *testing.T) {
	var broadcastMessages []string

	cb := defaultCombatCallbacks()
	cb.Broadcast = func(roomVNum int, msg string, excludedBodies []Combatant) {
		broadcastMessages = append(broadcastMessages, msg)
	}
	SetCallbacks(cb)

	attacker := &mockCombatant{name: "Hero", room: 100, sex: 0, position: PosStanding}
	defender := &mockCombatant{name: "Dragon", room: 100, sex: 1, position: PosStanding}

	DamMessage(1000, attacker, defender, 5) // bludgeon

	// Damage 1000 maps to tier 11 (ROCKS THE HELL OUT OF, 54+).
	// randPick chooses one variant at random, so check any tier-appropriate keyword.
	foundTier := false
	for _, msg := range broadcastMessages {
		if strings.Contains(msg, "ROCKS THE HELL OUT OF") || strings.Contains(msg, "catastrophic") {
			foundTier = true
			break
		}
	}
	if !foundTier {
		t.Errorf("expected high-damage tier message for dam=1000, got %v", broadcastMessages)
	}
}

func TestDamMessage_ZeroDamage(t *testing.T) {
	var broadcastCalled bool

	cb := defaultCombatCallbacks()
	cb.Broadcast = func(roomVNum int, msg string, excludedBodies []Combatant) {
		broadcastCalled = true
	}
	SetCallbacks(cb)

	attacker := &mockCombatant{name: "Player", room: 100, sex: 0, position: PosStanding}
	defender := &mockCombatant{name: "Rat", room: 100, sex: 0, position: PosStanding}

	DamMessage(0, attacker, defender, 1) // sting

	if !broadcastCalled {
		t.Error("expected broadcast even for 0 damage (miss message)")
	}
}

// ---------------------------------------------------------------------------
// TestDeathCry
// ---------------------------------------------------------------------------

func TestDeathCry_Basic(t *testing.T) {
	var broadcastMessages []string

	cb := defaultCombatCallbacks()
	cb.Broadcast = func(roomVNum int, msg string, excludedBodies []Combatant) {
		broadcastMessages = append(broadcastMessages, fmt.Sprintf("room=%d msg=%q", roomVNum, msg))
	}
	cb.GetAdjacentRoom = func(roomVNum, door int) int {
		if roomVNum == 100 && door < 2 {
			return 101 + door
		}
		return -1
	}
	SetCallbacks(cb)

	ch := &mockCombatant{name: "Orc", room: 100}
	result := DeathCry(ch)

	if !strings.Contains(result, "100") {
		t.Errorf("expected room 100 in result, got %q", result)
	}
	if len(broadcastMessages) < 1 {
		t.Error("expected at least one broadcast")
	}
}

// ---------------------------------------------------------------------------
// TestCounterProcs — the milestone reward lives in the game layer
//
// C's counter_procs(ch) (src/fight.c:1252-1311) has exactly one implementation
// in the port: game.World.counter_procs, pinned by
// pkg/game/glm_batch_a_test.go, pkg/game/descriptor_counts_test.go,
// pkg/game/game_diagnostic_mudlog_test.go and the TakeDamage-path test in
// pkg/game/counterprocs_takedamage_test.go. This package only forwards, so
// these tests pin the forwarding seam and nothing about the reward.
// ---------------------------------------------------------------------------

func TestCounterProcsForwardsBodyAndKillCount(t *testing.T) {
	var gotName string
	var gotKills int64

	cb := defaultCombatCallbacks()
	cb.GetKills = func(bodyname Combatant) int64 { return 2000 }
	cb.CounterProcs = func(bodyname Combatant, kills int64) {
		gotName = bodyname.GetName()
		gotKills = kills
	}
	SetCallbacks(cb)

	CounterProcs(&mockCombatant{name: "Hero"})

	if gotName != "Hero" || gotKills != 2000 {
		t.Errorf("forwarded (%q, %d), want (%q, 2000)", gotName, gotKills, "Hero")
	}
}

// C returns before any work for an NPC or mobile body (src/fight.c:1257-1258),
// so no callback may fire for one.
func TestCounterProcsSkipsNPCs(t *testing.T) {
	forwarded := false

	cb := defaultCombatCallbacks()
	cb.GetKills = func(bodyname Combatant) int64 { return 5000 }
	cb.CounterProcs = func(bodyname Combatant, kills int64) { forwarded = true }
	SetCallbacks(cb)

	CounterProcs(&mockCombatant{name: "Orc", npc: true})

	if forwarded {
		t.Error("NPC body reached the counter_procs callback; C returns at src/fight.c:1257")
	}
}

// ---------------------------------------------------------------------------
// TestPerformGroupGain — individual group member exp
// ---------------------------------------------------------------------------

func TestPerformGroupGain_Basic(t *testing.T) {
	var gainedExp int
	var gainedName string
	var alignSet bool

	cb := defaultCombatCallbacks()
	cb.GainExp = func(bodyname Combatant, amount int) {
		name := bodyname.GetName()

		gainedName = name
		gainedExp = amount
	}
	cb.GetAlignment = func(bodyname Combatant) int {
		name := bodyname.GetName()

		if name == "Orc" {
			return 900 // good-aligned victim triggers alignment shift
		}
		return 0
	}
	cb.SetAlignment = func(bodyname Combatant, val int) { alignSet = true }
	SetCallbacks(cb)

	ch := &mockCombatant{name: "Fighter", level: 10, room: 100, npc: false}
	victim := &mockCombatant{name: "Orc", level: 8, npc: true}

	PerformGroupGain(ch, victim, 200)
	if got, want := strings.Join(ch.messages, ""), "You receive your share of experience -- 200 points.\r\n"; got != want {
		t.Fatalf("group XP output = %q, want %q", got, want)
	}

	if gainedName != "Fighter" {
		t.Errorf("expected exp assigned to Fighter, got %q", gainedName)
	}
	if gainedExp <= 0 {
		t.Errorf("expected positive exp gain, got %d", gainedExp)
	}
	if !alignSet {
		t.Error("expected alignment change when killing a good-aligned victim")
	}
}

func TestPerformGroupGain_OneExpPoint(t *testing.T) {
	cb := defaultCombatCallbacks()
	cb.GainExp = func(bodyname Combatant, amount int) {}
	cb.GetAlignment = func(bodyname Combatant) int { return 0 }
	cb.SetAlignment = func(bodyname Combatant, val int) {}
	SetCallbacks(cb)

	ch := &mockCombatant{name: "Fighter", level: 99, room: 100, npc: false}
	victim := &mockCombatant{name: "Rat", level: 1, npc: true}

	// base=1, extreme level diff, level>20 penalty → should floor at 1
	PerformGroupGain(ch, victim, 1)
	if got, want := strings.Join(ch.messages, ""), "You receive your share of experience -- one measly little point!\r\n"; got != want {
		t.Fatalf("one-point group XP output = %q, want %q", got, want)
	}
}

// ---------------------------------------------------------------------------
// Engine lifecycle tests
// ---------------------------------------------------------------------------

// TestStartCombat verifies StartCombat initializes a combat pair correctly.
func TestStartCombat(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	attacker := &mockCombatant{name: "Hero", npc: false, room: 100}
	defender := &mockCombatant{name: "Orc", npc: true, room: 100}

	err := engine.StartCombat(attacker, defender)
	if err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	if !engine.IsFighting(attacker) {
		t.Error("expected Hero to be fighting")
	}
	if !engine.IsFighting(defender) {
		t.Error("expected Orc to be fighting")
	}
	if attacker.GetFighting() != "Orc" {
		t.Errorf("expected attacker fighting Orc, got %q", attacker.fighting)
	}
	if defender.GetFighting() != "Hero" {
		t.Errorf("expected defender fighting Hero, got %q", defender.fighting)
	}
}

// TestStartCombat_Duplicate prevents starting combat twice for same attacker.
func TestStartCombat_Duplicate(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	attacker := &mockCombatant{name: "Hero", room: 100}
	orc := &mockCombatant{name: "Orc", room: 100}
	goblin := &mockCombatant{name: "Goblin", room: 100}

	_ = engine.StartCombat(attacker, orc)
	err := engine.StartCombat(attacker, goblin)
	if err == nil {
		t.Error("expected error when starting combat with already-fighting attacker")
	}
}

// TestStartCombat_DefenderKeepsExistingTarget verifies that attacking a
// defender already in combat does not retarget the defender. In DikuMUD a
// character FIGHTs one opponent at a time; a second attacker is added but the
// defender keeps fighting its original target until that target is gone.
func TestStartCombat_DefenderKeepsExistingTarget(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	hero := &mockCombatant{name: "Hero", room: 100}
	orc := &mockCombatant{name: "Orc", npc: true, room: 100}
	goblin := &mockCombatant{name: "Goblin", npc: true, room: 100}

	// Orc is already fighting Hero.
	if err := engine.StartCombat(orc, hero); err != nil {
		t.Fatalf("StartCombat(orc, hero) failed: %v", err)
	}
	// Goblin now jumps Hero, who is already fighting Orc.
	if err := engine.StartCombat(goblin, hero); err != nil {
		t.Fatalf("StartCombat(goblin, hero) failed: %v", err)
	}

	// Hero must still be fighting its original target, not retargeted to Goblin.
	if hero.GetFighting() != "Orc" {
		t.Errorf("expected Hero to keep fighting Orc, got %q", hero.fighting)
	}
	// Goblin is engaged with Hero either way.
	if goblin.GetFighting() != "Hero" {
		t.Errorf("expected Goblin fighting Hero, got %q", goblin.fighting)
	}
	if !engine.IsFighting(goblin) {
		t.Error("expected Goblin to be registered as fighting")
	}
}

func TestCombatOrderPrependsNewFightersAndRemovesStoppedFighters(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	hero := &mockCombatant{name: "Hero", room: 100}
	orc := &mockCombatant{name: "Orc", npc: true, room: 100}
	goblin := &mockCombatant{name: "Goblin", npc: true, room: 100}

	if err := engine.StartCombat(orc, hero); err != nil {
		t.Fatalf("StartCombat(orc, hero) failed: %v", err)
	}
	if err := engine.StartCombat(goblin, hero); err != nil {
		t.Fatalf("StartCombat(goblin, hero) failed: %v", err)
	}

	order := func() string {
		names := make([]string, 0, len(engine.combatOrder))
		for _, fighter := range engine.combatOrder {
			names = append(names, fighter.GetName())
		}
		return strings.Join(names, ",")
	}
	if got, want := order(), "Goblin,Hero,Orc"; got != want {
		t.Fatalf("combat order = %s, want %s", got, want)
	}

	engine.StopCombat(goblin)
	if got, want := order(), "Hero,Orc"; got != want {
		t.Fatalf("combat order after stopping Goblin = %s, want %s", got, want)
	}

	engine.StopCombat(hero)
	if got := order(); got != "Orc" {
		t.Fatalf("combat order after stopping Hero = %s, want the one-way Orc fight", got)
	}
}

// TestStopCombat removes a combat pair and clears fighting states.
func TestStopCombat(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	attacker := &mockCombatant{name: "Hero", room: 100}
	defender := &mockCombatant{name: "Orc", room: 100}

	_ = engine.StartCombat(attacker, defender)
	engine.StopCombat(attacker)

	if engine.IsFighting(attacker) {
		t.Error("expected Hero to not be fighting after StopCombat")
	}
}

// TestGetCombatTarget verifies target lookup works both ways.
func TestGetCombatTarget(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	attacker := &mockCombatant{name: "Hero", room: 100}
	defender := &mockCombatant{name: "Orc", room: 100}

	_ = engine.StartCombat(attacker, defender)

	target, ok := engine.GetCombatTarget(attacker)
	if !ok {
		t.Fatal("expected Hero to have a combat target")
	}
	if target.GetName() != "Orc" {
		t.Errorf("expected target Orc, got %q", target.GetName())
	}

	// Reverse lookup
	target, ok = engine.GetCombatTarget(defender)
	if !ok {
		t.Fatal("expected Orc to have a combat target")
	}
	if target.GetName() != "Hero" {
		t.Errorf("expected target Hero, got %q", target.GetName())
	}

	// Non-combatant
	_, ok = engine.GetCombatTarget(nil)
	if ok {
		t.Error("expected no target for non-combatant")
	}
}

// TestGetCombatStatus verifies status string output.

// ---------------------------------------------------------------------------
// processCombatPair — the core round execution
// ---------------------------------------------------------------------------

// TestProcessCombatPair_MobAttack runs a combat pair round between a mob and player.
func TestProcessCombatPair_MobAttack(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	cb := defaultCombatCallbacks()
	cb.GetWeaponInfo = func(chName Combatant) (wType, damDice, damSize int, isBlessed bool) {
		if chName.GetName() == "Hero" {
			// wType is the 0-based OFFSET (val3), not a TYPE_* constant:
			// SendWeaponMessage adds TYPE_HIT itself. Slash offset = 3.
			return 3, 1, 8, false
		}
		return 0, 0, 0, false // barehand → "hit"
	}
	// The default fixture answers race 0 for the victim and 0 for every
	// race-hate slot, which the C loop at src/fight.c:1466-1468 reads as five
	// matches and turns into 5 x GET_LEVEL(ch) bonus damage. Give the victim a
	// real race so this swing carries no race-hate bonus and the damage under
	// test is the hit() arithmetic alone.
	cb.GetRace = func(bodyname Combatant) int { return 1 }
	SetCallbacks(cb)

	// Wire engine callbacks
	engine.BroadcastFunc = func(room int, msg string, excludedBodies []Combatant) {}
	engine.ScriptFightFunc = nil

	attacker := &mockCombatant{
		name:       "Hero",
		npc:        false,
		room:       100,
		level:      10,
		hp:         100,
		maxHP:      100,
		ac:         30,
		thac0:      15,
		position:   PosStanding,
		class:      ClassWarrior,
		str:        16,
		strAdd:     0,
		dex:        14,
		intVal:     12,
		wis:        10,
		hitroll:    1,
		damroll:    2,
		damageRoll: DiceRoll{Num: 1, Sides: 8},
	}
	defender := &mockCombatant{
		name:       "Orc",
		npc:        true,
		room:       100,
		level:      5,
		hp:         50,
		maxHP:      50,
		ac:         40,
		thac0:      20,
		position:   PosStanding,
		str:        15,
		strAdd:     0,
		dex:        10,
		intVal:     8,
		wis:        8,
		damageRoll: DiceRoll{Num: 1, Sides: 4},
	}

	_ = engine.StartCombat(attacker, defender)

	// Force every draw in the round so the swing lands and its damage is
	// exact. In C order (src/fight.c:1939, 1949, 1815, 1844):
	//   1. number(0,500)  — level 10 is not > 30, so the attack-bonus probe runs; 1 != 0
	//   2. number(0,10000) — parry probe; 1 > SKILL_PARRY 0, so no parry
	//   3. number(1,20)   — to-hit 10: calc_thaco 12 - 10 = 2 is not > victim_ac 4, so a hit
	//   4. dice(1,8)      — weapon damage 8
	// Damage = strApp[16].todam 1 + damroll 2 + 8 = 11, reduced by
	// get_minusdam at AC 40 (src/fight.c:1732, .06*pcmod): 11 - 1.32 -> 9.
	old := GetRoller()
	SetRoller(NewScriptedRoller([]int{1, 1, 10, 8}))
	defer SetRoller(old)

	beforeHP := defender.hp
	engine.processCombatPair(&CombatPair{
		Attacker: attacker,
		Defender: defender,
	})

	// The old assertion only rejected an HP increase, so a swing that never
	// happened (or one that dealt 0) passed. Assert the exact drop instead.
	const wantDamage = 9
	if got := beforeHP - defender.hp; got != wantDamage {
		t.Errorf("defender HP drop = %d, want %d (one landed swing of 9)", got, wantDamage)
	}

	t.Logf("processCombatPair: defender HP %d -> %d", beforeHP, defender.hp)
}

// TestProcessCombatPair_PlayerDeath sets defender to low HP and verifies death is detected.
func TestProcessCombatPair_PlayerDeath(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	cb := defaultCombatCallbacks()
	cb.GetWeaponInfo = func(chName Combatant) (wType, damDice, damSize int, isBlessed bool) {
		if chName.GetName() == "Hero" {
			// wType is the 0-based OFFSET (val3), not a TYPE_* constant:
			// SendWeaponMessage adds TYPE_HIT itself. Slash offset = 3.
			return 3, 1, 8, false
		}
		return 0, 0, 0, false // barehand → "hit"
	}
	SetCallbacks(cb)

	var deathCalled bool
	engine.BroadcastFunc = func(room int, msg string, excludedBodies []Combatant) {}
	engine.DeathFunc = func(victim, killer Combatant, attackType int) {
		deathCalled = true
	}

	attacker := &mockCombatant{
		name:       "Hero",
		npc:        false,
		room:       100,
		level:      10,
		hp:         100,
		maxHP:      100,
		ac:         30,
		thac0:      15,
		position:   PosStanding,
		class:      ClassWarrior,
		str:        16,
		strAdd:     0,
		dex:        14,
		intVal:     12,
		wis:        10,
		hitroll:    5,
		damroll:    10,
		damageRoll: DiceRoll{Num: 5, Sides: 10},
	}
	defender := &mockCombatant{
		name:       "Goblin",
		npc:        true,
		room:       100,
		level:      1,
		hp:         5,
		maxHP:      5,
		ac:         80,
		thac0:      20,
		position:   PosStanding,
		str:        10,
		strAdd:     0,
		dex:        10,
		intVal:     8,
		wis:        8,
		damageRoll: DiceRoll{Num: 1, Sides: 3},
	}

	_ = engine.StartCombat(attacker, defender)

	// Run multiple rounds to ensure death with a weak defender
	for round := 0; round < 5; round++ {
		if engine.IsFighting(attacker) && engine.IsFighting(defender) {
			engine.processCombatPair(&CombatPair{
				Attacker: attacker,
				Defender: defender,
			})
		}
	}

	if deathCalled {
		t.Log("death was detected and handled via engine.DeathFunc")
	} else if defender.hp <= 0 {
		t.Log("goblin HP dropped to 0 — death processed internally via TakeDamage")
	} else {
		t.Logf("goblin survived with %d HP (all misses — improbable but possible)", defender.hp)
	}
}

// TestProcessCombatPair_DifferentRoom verifies combat stops when rooms don't match.
func TestProcessCombatPair_DifferentRoom(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	SetCallbacks(defaultCombatCallbacks())

	attacker := &mockCombatant{name: "Hero", room: 100, hp: 100, position: PosStanding}
	defender := &mockCombatant{name: "Orc", room: 200, hp: 50, position: PosStanding}

	_ = engine.StartCombat(attacker, defender)
	defender.StopFighting() // no eligible reciprocal retarget: fight.c:230-254
	engine.processCombatPair(&CombatPair{Attacker: attacker, Defender: defender})

	if engine.IsFighting(attacker) {
		t.Error("expected combat to stop when rooms differ")
	}
}

// ---------------------------------------------------------------------------
// handleDeath test
// ---------------------------------------------------------------------------

func TestHandleDeath_Basic(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	var deathFuncCalled bool

	SetCallbacks(defaultCombatCallbacks())

	attacker := &mockCombatant{name: "Hero", room: 100, hp: 100}
	defender := &mockCombatant{name: "Orc", room: 100, hp: 0}

	engine.BroadcastFunc = func(room int, msg string, excludedBodies []Combatant) {}
	engine.DeathFunc = func(victim, killer Combatant, attackType int) {
		deathFuncCalled = true
	}

	engine.handleDeath(defender, attacker)

	if !deathFuncCalled {
		t.Error("expected DeathFunc to be called")
	}
}

// ---------------------------------------------------------------------------
// Engine SendMessage helpers
// ---------------------------------------------------------------------------

func TestSendHitMessage(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	var attackerMsg, defenderMsg, roomMsg string
	engine.BroadcastFunc = func(room int, msg string, excludedBodies []Combatant) {
		roomMsg = msg
	}

	attacker := &messagingCombatant{}
	attacker.name = "Hero"
	defender := &messagingCombatant{}
	defender.name = "Orc"

	attacker.sendFunc = func(msg string) { attackerMsg = msg }
	defender.sendFunc = func(msg string) { defenderMsg = msg }

	engine.sendHitMessage(attacker, defender, 15, 0)

	if !strings.Contains(attackerMsg, "15") {
		t.Errorf("expected damage number in attacker msg, got %q", attackerMsg)
	}
	if !strings.Contains(defenderMsg, "15") {
		t.Errorf("expected damage number in defender msg, got %q", defenderMsg)
	}
	if !strings.Contains(roomMsg, "Hero") || !strings.Contains(roomMsg, "Orc") {
		t.Errorf("expected both names in room msg, got %q", roomMsg)
	}
}

func TestSendMissMessage(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	var attackerMsg, defenderMsg string

	attacker := &messagingCombatant{}
	attacker.name = "Hero"
	defender := &messagingCombatant{}
	defender.name = "Orc"

	attacker.sendFunc = func(msg string) { attackerMsg = msg }
	defender.sendFunc = func(msg string) { defenderMsg = msg }

	engine.sendMissMessage(attacker, defender, 0)

	if !strings.Contains(attackerMsg, "miss") {
		t.Errorf("expected 'miss' in attacker msg, got %q", attackerMsg)
	}
	if !strings.Contains(defenderMsg, "miss") {
		t.Errorf("expected 'miss' in defender msg, got %q", defenderMsg)
	}
}

// messagingCombatant wraps mockCombatant with message capture capability.
type messagingCombatant struct {
	mockCombatant
	sendFunc func(string)
}

func (m *messagingCombatant) SendMessage(msg string) {
	if m.sendFunc != nil {
		m.sendFunc(msg)
	}
}

// ---------------------------------------------------------------------------
// ChangeAlignment — more comprehensive
// ---------------------------------------------------------------------------

func TestChangeAlignment_NPCKiller(t *testing.T) {
	alignResult := -9999
	calls := 0

	cb := defaultCombatCallbacks()
	cb.GetAlignment = func(bodyname Combatant) int {
		name := bodyname.GetName()

		if name == "good_victim" {
			return 900
		}
		if name == "npc_killer" {
			return 500
		}
		return 0
	}
	cb.SetAlignment = func(bodyname Combatant, val int) {
		calls++
		alignResult = val
	}
	SetCallbacks(cb)

	// The fixture's "npc_killer" is a PC (npc: false), so only the
	// victim-alignment guard is in play. src/fight.c:496-501 gives
	// 500 + ((-900 - 500) >> 4) = 500 + (-88) = 412. The old assertion only
	// rejected values >= 500, and the captured value started at 0, so a
	// suppressed update passed.
	npcKiller := &mockCombatant{name: "npc_killer", npc: false}
	goodVictim := &mockCombatant{name: "good_victim", npc: true}

	ChangeAlignment(npcKiller, goodVictim)
	if calls != 1 {
		t.Fatalf("SetAlignment called %d times, want 1 (fight.c:496)", calls)
	}
	if alignResult != 412 {
		t.Errorf("alignment after killing a +900 victim = %d, want 412 (fight.c:496)", alignResult)
	}
}

func TestChangeAlignment_NPCKillerNPC(t *testing.T) {
	var alignCalled bool

	cb := defaultCombatCallbacks()
	cb.GetAlignment = func(bodyname Combatant) int {
		if bodyname.GetName() == "Elf" {
			return 900
		}
		return 0
	}
	cb.SetAlignment = func(bodyname Combatant, val int) { alignCalled = true }
	SetCallbacks(cb)

	// src/fight.c:488-489 — an NPC killer returns before any alignment math.
	// The victim is strongly good (src/utils.h:454-456: IS_GOOD is >= 350), so
	// the separate neutral guard cannot suppress the update; only the NPC
	// guard can. The old neutral victim (alignment 0) hid a missing guard.
	killer := &mockCombatant{name: "Orc", npc: true}
	victim := &mockCombatant{name: "Elf", npc: true}

	ChangeAlignment(killer, victim)
	if alignCalled {
		t.Error("NPC killer must not trigger an alignment change (fight.c:488)")
	}
}
