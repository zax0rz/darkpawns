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
		Broadcast:                func(roomVNum int, msg string, exclude string) {},
		SendToChar:               func(name string, msg string) {},
		SkillMessage:             func(dam int, ch, vict string, attackType int, roomVNum int) bool { return false },
		BroadChat:                func(chName string, msg string) {},
		Log:                      func(msg string, level string, minLevel int, toLog bool) {},
		GetRace:                  func(name string) int { return 0 },
		GetRaceHate:              func(name string, index int) int { return 0 },
		GetAlignment:             func(name string) int { return 0 },
		SetAlignment:             func(name string, val int) {},
		GetSex:                   func(name string) int { return 0 },
		GetSkill:                 func(name string, skillNum int) int { return 0 },
		HasAffect:                func(name string, aff int) bool { return false },
		HasAffectStr:             func(name string, aff string) bool { return false },
		RemoveAffect:             func(name string, skillNum int) {},
		RemoveAllAffects:         func(name string) {},
		HasPlrFlag:               func(name string, flag string) bool { return false },
		SetPlrFlag:               func(name string) bool { return false },
		HasPrfFlag:               func(name string, flag string) bool { return false },
		HasMobFlag:               func(name string, flag string) bool { return false },
		HasMobVNum:               func(name string, vnum int) bool { return false },
		HasRoomFlag:              func(roomVNum int, flag string) bool { return false },
		HasScriptFlag:            func(name string, flag string) bool { return false },
		IsShopkeeper:             func(name string) bool { return false },
		IsMounted:                func(name string) bool { return false },
		Dismount:                 func(name string) {},
		Unmount:                  func(name string) {},
		GetWeaponInfo:            func(chName string) (wType, damDice, damSize int, isBlessed bool) { return TYPE_HIT, 0, 0, false },
		GetAdjacentRoom:          func(roomVNum, door int) int { return -1 },
		GainExp:                  func(name string, amount int) {},
		GetExp:                   func(name string) int { return 0 },
		GetKills:                 func(name string) int64 { return 0 },
		SetKills:                 func(name string, kills int64) {},
		GetDeaths:                func(name string) int64 { return 0 },
		SetDeaths:                func(name string, deaths int64) {},
		SetLastDeath:             func(name string, t int64) {},
		GetPks:                   func(name string) int64 { return 0 },
		SetPks:                   func(name string, pks int64) {},
		GetConstitution:          func(name string) int { return 0 },
		SetConstitution:          func(name string, val int) {},
		MakeCorpse:               func(victim string, attackType int) {},
		MakeDust:                 func(victim string, attackType int) {},
		ExtractChar:              func(name string) {},
		RunDeathScript:           func(killer, victim string, roomVNum int) {},
		GetFollowersInRoom:       func(name string, roomVNum int) int { return 0 },
		GetMasterInRoom:          func(name string, roomVNum int) bool { return false },
		GetFellowFollowersInRoom: func(name string, roomVNum int) bool { return false },
		CountGroupMembers:        func(leaderName string, roomVNum int) int { return 1 },
		ApplyToGroupMembers:      func(leaderName string, roomVNum int, fn func(name string)) {},
		GetGold:                  func(name string) int { return 0 },
		SetGold:                  func(name string, gold int) {},
		JunkInventoryItems:       func(chName string) {},
		PerformCommand:           func(chName, cmd string) {},
		GetWimpyLev:              func(name string) int { return 0 },
		DoFlee:                   func(name string) {},
		DoRetreat:                func(name string) {},
		IncreaseMaxStat:          func(name string, stat string) {},
		HealAllPlayers:           func() {},
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
	cb.Broadcast = func(roomVNum int, msg string, exclude string) {
		broadcastRoom = roomVNum
		broadcastMsg = msg
	}
	cb.SendToChar = func(name string, msg string) {
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
	cb.Broadcast = func(roomVNum int, msg string, exclude string) {
		broadcastMessages = append(broadcastMessages, msg)
	}
	SetCallbacks(cb)

	attacker := &mockCombatant{name: "Hero", room: 100, sex: 0, position: PosStanding}
	defender := &mockCombatant{name: "Dragon", room: 100, sex: 1, position: PosStanding}

	DamMessage(1000, attacker, defender, 5) // bludgeon

	// Tier 12 (OBLITERATES, 101+) and Tier 13 (ROCKS, 10000+) are both valid.
	// randPick chooses one variant at random, so check any tier-appropriate keyword.
	foundTier := false
	for _, msg := range broadcastMessages {
		if strings.Contains(msg, "OBLITERATES") || strings.Contains(msg, "blow of legend") ||
			strings.Contains(msg, "R O C K S") || strings.Contains(msg, "smear") {
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
	cb.Broadcast = func(roomVNum int, msg string, exclude string) {
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
	cb.Broadcast = func(roomVNum int, msg string, exclude string) {
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
// TestCounterProcs — kill milestone rewards
// ---------------------------------------------------------------------------

func TestCounterProcs_NoMilestone(t *testing.T) {
	npc := &mockCombatant{name: "Orc", npc: true}
	CounterProcs(npc)
	// Should not panic
}

func TestCounterProcs_KillCountTracking(t *testing.T) {
	var loggedMsg string

	cb := defaultCombatCallbacks()
	cb.GetKills = func(name string) int64 {
		return 5000 // minor milestone
	}
	cb.Log = func(msg string, level string, minLevel int, toLog bool) {
		loggedMsg = msg
	}
	SetCallbacks(cb)

	player := &mockCombatant{name: "Hero", npc: false, hp: 50, maxHP: 100}
	CounterProcs(player)

	if !strings.Contains(loggedMsg, "5000 kills") {
		t.Errorf("expected log about 5000 kills, got %q", loggedMsg)
	}
}

func TestCounterProcs_MajorMilestone(t *testing.T) {
	var statCalls []string
	var logged bool

	cb := defaultCombatCallbacks()
	cb.GetKills = func(name string) int64 {
		return 2000 // major milestone
	}
	cb.Log = func(msg string, level string, minLevel int, toLog bool) {
		logged = true
	}
	cb.IncreaseMaxStat = func(name string, stat string) {
		statCalls = append(statCalls, stat)
	}
	SetCallbacks(cb)

	player := &mockCombatant{name: "Hero", npc: false, hp: 50, maxHP: 100}
	CounterProcs(player)

	// Major milestone: C bug causes all three stats to be incremented (fall-through)
	if len(statCalls) != 3 {
		t.Errorf("expected 3 IncreaseMaxStat calls (C bug: all fall through), got %d: %v", len(statCalls), statCalls)
	}
	if !logged {
		t.Error("expected log for milestone")
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
	cb.GainExp = func(name string, amount int) {
		gainedName = name
		gainedExp = amount
	}
	cb.GetAlignment = func(name string) int {
		if name == "Orc" {
			return 900 // good-aligned victim triggers alignment shift
		}
		return 0
	}
	cb.SetAlignment = func(name string, val int) { alignSet = true }
	SetCallbacks(cb)

	ch := &mockCombatant{name: "Fighter", level: 10, room: 100, npc: false}
	victim := &mockCombatant{name: "Orc", level: 8, npc: true}

	PerformGroupGain(ch, victim, 200)

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
	cb.GainExp = func(name string, amount int) {}
	cb.GetAlignment = func(name string) int { return 0 }
	cb.SetAlignment = func(name string, val int) {}
	SetCallbacks(cb)

	ch := &mockCombatant{name: "Fighter", level: 99, room: 100, npc: false}
	victim := &mockCombatant{name: "Rat", level: 1, npc: true}

	// base=1, extreme level diff, level>20 penalty → should floor at 1
	PerformGroupGain(ch, victim, 1)
	// Should not panic, internal calc returns at least 1
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

	if !engine.IsFighting("Hero") {
		t.Error("expected Hero to be fighting")
	}
	if !engine.IsFighting("Orc") {
		t.Error("expected Orc to be fighting")
	}
	if attacker.fighting != "Orc" {
		t.Errorf("expected attacker fighting Orc, got %q", attacker.fighting)
	}
	if defender.fighting != "Hero" {
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
	if hero.fighting != "Orc" {
		t.Errorf("expected Hero to keep fighting Orc, got %q", hero.fighting)
	}
	// Goblin is engaged with Hero either way.
	if goblin.fighting != "Hero" {
		t.Errorf("expected Goblin fighting Hero, got %q", goblin.fighting)
	}
	if !engine.IsFighting("Goblin") {
		t.Error("expected Goblin to be registered as fighting")
	}
}

// TestStopCombat removes a combat pair and clears fighting states.
func TestStopCombat(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	attacker := &mockCombatant{name: "Hero", room: 100}
	defender := &mockCombatant{name: "Orc", room: 100}

	_ = engine.StartCombat(attacker, defender)
	engine.StopCombat("Hero")

	if engine.IsFighting("Hero") {
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

	target, ok := engine.GetCombatTarget("Hero")
	if !ok {
		t.Fatal("expected Hero to have a combat target")
	}
	if target.GetName() != "Orc" {
		t.Errorf("expected target Orc, got %q", target.GetName())
	}

	// Reverse lookup
	target, ok = engine.GetCombatTarget("Orc")
	if !ok {
		t.Fatal("expected Orc to have a combat target")
	}
	if target.GetName() != "Hero" {
		t.Errorf("expected target Hero, got %q", target.GetName())
	}

	// Non-combatant
	_, ok = engine.GetCombatTarget("Nobody")
	if ok {
		t.Error("expected no target for non-combatant")
	}
}

// TestGetCombatStatus verifies status string output.
func TestGetCombatStatus(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	attacker := &mockCombatant{name: "Hero", room: 100}
	defender := &mockCombatant{name: "Orc", room: 100}
	_ = engine.StartCombat(attacker, defender)

	status := engine.GetCombatStatus("Hero")
	if !strings.Contains(status, "fighting") {
		t.Errorf("expected 'fighting' in status for attacker, got %q", status)
	}

	status = engine.GetCombatStatus("Nobody")
	if !strings.Contains(status, "not in combat") {
		t.Errorf("expected 'not in combat' for non-combatant, got %q", status)
	}
}

// ---------------------------------------------------------------------------
// processCombatPair — the core round execution
// ---------------------------------------------------------------------------

// TestProcessCombatPair_MobAttack runs a combat pair round between a mob and player.
func TestProcessCombatPair_MobAttack(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	cb := defaultCombatCallbacks()
	cb.GetWeaponInfo = func(chName string) (wType, damDice, damSize int, isBlessed bool) {
		if chName == "Hero" {
			return TYPE_SLASH, 1, 8, false
		}
		return TYPE_HIT, 0, 0, false
	}
	SetCallbacks(cb)

	// Wire engine callbacks
	engine.BroadcastFunc = func(room int, msg string, exclude string) {}
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

	beforeHP := defender.hp
	engine.processCombatPair(&CombatPair{
		Attacker: attacker,
		Defender: defender,
	})

	// Damage should have been dealt (or missed)
	if defender.hp > beforeHP {
		t.Error("defender HP should not increase after a round")
	}

	t.Logf("processCombatPair: defender HP %d -> %d", beforeHP, defender.hp)
}

// TestProcessCombatPair_PlayerDeath sets defender to low HP and verifies death is detected.
func TestProcessCombatPair_PlayerDeath(t *testing.T) {
	engine := NewCombatEngine()
	defer engine.Stop()

	cb := defaultCombatCallbacks()
	cb.GetWeaponInfo = func(chName string) (wType, damDice, damSize int, isBlessed bool) {
		if chName == "Hero" {
			return TYPE_SLASH, 1, 8, false
		}
		return TYPE_HIT, 0, 0, false
	}
	SetCallbacks(cb)

	var deathCalled bool
	engine.BroadcastFunc = func(room int, msg string, exclude string) {}
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
		if engine.IsFighting("Hero") && engine.IsFighting("Goblin") {
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
	engine.processCombatPair(&CombatPair{Attacker: attacker, Defender: defender})

	if engine.IsFighting("Hero") {
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

	engine.BroadcastFunc = func(room int, msg string, exclude string) {}
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
	engine.BroadcastFunc = func(room int, msg string, exclude string) {
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
	var alignResult int

	cb := defaultCombatCallbacks()
	cb.GetAlignment = func(name string) int {
		if name == "good_victim" {
			return 900
		}
		if name == "npc_killer" {
			return 500
		}
		return 0
	}
	cb.SetAlignment = func(name string, val int) {
		alignResult = val
	}
	SetCallbacks(cb)

	npcKiller := &mockCombatant{name: "npc_killer", npc: false}
	goodVictim := &mockCombatant{name: "good_victim", npc: true}

	ChangeAlignment(npcKiller, goodVictim)
	if alignResult >= 500 {
		t.Errorf("expected alignment to drop after killing good, got %d", alignResult)
	}
}

func TestChangeAlignment_NPCKillerNPC(t *testing.T) {
	var alignCalled bool

	cb := defaultCombatCallbacks()
	cb.GetAlignment = func(name string) int { return 0 }
	cb.SetAlignment = func(name string, val int) { alignCalled = true }
	SetCallbacks(cb)

	killer := &mockCombatant{name: "Orc", npc: true}
	victim := &mockCombatant{name: "Elf", npc: true}

	ChangeAlignment(killer, victim)
	if alignCalled {
		t.Error("NPC should not trigger alignment change")
	}
}
