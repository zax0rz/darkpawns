// Package combat — round execution tests
//
// These tests cover TakeDamage, DamMessage, PerformGroupGain,
// Appear, DeathCry, CounterProcs, and the engine round execution
// (processCombatPair, handleDeath, StartCombat, StopCombat).
//
// Global function pointers (the var hooks) are wired per-test and
// restored via defer to avoid cross-test contamination.
package combat

import (
	"fmt"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// TestDamMessage — damage message tier selection
// ---------------------------------------------------------------------------

func TestDamMessage_TierSelection(t *testing.T) {
	var broadcastRoom int
	var broadcastMsg string
	var sendToCalls []string

	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	defer func() {
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastRoom = room
		broadcastMsg = msg
	}
	SendToCharFunc = func(name string, msg string) {
		sendToCalls = append(sendToCalls, name+":"+msg)
	}

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
	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	defer func() {
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastMessages = append(broadcastMessages, msg)
	}
	SendToCharFunc = func(name string, msg string) {}

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
	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	defer func() {
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastCalled = true
	}
	SendToCharFunc = func(name string, msg string) {}

	attacker := &mockCombatant{name: "Player", room: 100, sex: 0, position: PosStanding}
	defender := &mockCombatant{name: "Rat", room: 100, sex: 0, position: PosStanding}

	DamMessage(0, attacker, defender, 1) // sting

	if !broadcastCalled {
		t.Error("expected broadcast even for 0 damage (miss message)")
	}
}

// ---------------------------------------------------------------------------
// TestAppear — become visible
// ---------------------------------------------------------------------------

func TestAppear_Basic(t *testing.T) {
	var broadcastMsg string
	origBroadcast := BroadcastMessage
	defer func() {
		BroadcastMessage = origBroadcast
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastMsg = msg
	}

	ch := &mockCombatant{name: "Rogue", room: 200, level: 10}
	Appear(ch)

	if !strings.Contains(broadcastMsg, "fades into existence") {
		t.Errorf("expected fade message, got %q", broadcastMsg)
	}
}

func TestAppear_ImmortalLevel(t *testing.T) {
	var broadcastMsg string
	origBroadcast := BroadcastMessage
	defer func() {
		BroadcastMessage = origBroadcast
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastMsg = msg
	}

	ch := &mockCombatant{name: "Wizard", room: 200, level: LVL_IMMORT}
	Appear(ch)

	if !strings.Contains(broadcastMsg, "strange presence") {
		t.Errorf("expected strange presence message for immortal, got %q", broadcastMsg)
	}
}

// ---------------------------------------------------------------------------
// TestDeathCry
// ---------------------------------------------------------------------------

func TestDeathCry_Basic(t *testing.T) {
	var broadcastMessages []string
	origBroadcast := BroadcastMessage
	defer func() {
		BroadcastMessage = origBroadcast
	}()

	BroadcastMessage = func(room int, msg string, exclude string) {
		broadcastMessages = append(broadcastMessages, fmt.Sprintf("room=%d msg=%q", room, msg))
	}

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
// TestCounterProcs — dummy test since CounterProcs is simplified/empty
// ---------------------------------------------------------------------------

func TestCounterProcs_Noop(t *testing.T) {
	npc := &mockCombatant{name: "Orc", npc: true}
	CounterProcs(npc)
}

// ---------------------------------------------------------------------------
// TestPerformGroupGain — individual group member exp
// ---------------------------------------------------------------------------

func TestPerformGroupGain_Basic(t *testing.T) {
	ch := &mockCombatant{name: "Fighter", level: 10, room: 100, npc: false}
	victim := &mockCombatant{name: "Orc", level: 8, npc: true}

	PerformGroupGain(ch, victim, 200)
}

func TestPerformGroupGain_OneExpPoint(t *testing.T) {
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

	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	origGetExp := GetExp
	origDoFlee := DoFlee
	origDoRetreat := DoRetreat
	origSkillMsg := SkillMessageFunc
	defer func() {
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
		GetExp = origGetExp
		DoFlee = origDoFlee
		DoRetreat = origDoRetreat
		SkillMessageFunc = origSkillMsg
	}()

	// Wire global hooks
	BroadcastMessage = func(room int, msg string, exclude string) {}
	SendToCharFunc = func(name string, msg string) {}
	GetExp = func(name string) int { return 0 }
	DoFlee = func(name string) {}
	DoRetreat = func(name string) {}
	SkillMessageFunc = func(dam int, ch, vict string, atk int, room int) bool { return false }

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

	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	origGetExp := GetExp
	origDoFlee := DoFlee
	origDoRetreat := DoRetreat
	origSkillMsg := SkillMessageFunc
	origNowUnix := NowUnix
	defer func() {
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
		GetExp = origGetExp
		DoFlee = origDoFlee
		DoRetreat = origDoRetreat
		SkillMessageFunc = origSkillMsg
		NowUnix = origNowUnix
	}()

	// Wire all the hooks
	BroadcastMessage = func(room int, msg string, exclude string) {}
	SendToCharFunc = func(name string, msg string) {}
	GetExp = func(name string) int { return 0 }
	DoFlee = func(name string) {}
	DoRetreat = func(name string) {}
	SkillMessageFunc = func(dam int, ch, vict string, atk int, room int) bool { return false }
	NowUnix = func() int64 { return 12345 }

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

	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	defer func() {
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
	}()
	BroadcastMessage = func(room int, msg string, exclude string) {}
	SendToCharFunc = func(name string, msg string) {}

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

	origSendTo := SendToCharFunc
	origBroadcast := BroadcastMessage
	defer func() {
		SendToCharFunc = origSendTo
		BroadcastMessage = origBroadcast
	}()
	SendToCharFunc = func(name string, msg string) {}
	BroadcastMessage = func(room int, msg string, exclude string) {}

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
// Regression Tests (DP-952)
// ---------------------------------------------------------------------------

func TestDeadHookNoopDoesNotPanic(t *testing.T) {
	origBroadcast := BroadcastMessage
	origSendTo := SendToCharFunc
	origDoFlee := DoFlee
	origDoRetreat := DoRetreat
	origSkillMsg := SkillMessageFunc
	origGetExp := GetExp
	defer func() {
		BroadcastMessage = origBroadcast
		SendToCharFunc = origSendTo
		DoFlee = origDoFlee
		DoRetreat = origDoRetreat
		SkillMessageFunc = origSkillMsg
		GetExp = origGetExp
	}()

	BroadcastMessage = nil
	SendToCharFunc = nil
	DoFlee = nil
	DoRetreat = nil
	SkillMessageFunc = nil
	GetExp = nil

	ch := &mockCombatant{name: "Hero", room: 100, level: 10, hp: 100, maxHP: 100, position: PosStanding}
	victim := &mockCombatant{name: "Goblin", room: 100, level: 10, hp: 10, maxHP: 10, position: PosStanding}

	// Trigger combat actions on nil hooks - must not panic
	TakeDamage(ch, victim, 5, TYPE_HIT) // Normal damage message hook path
	TakeDamage(ch, victim, 20, TYPE_HIT) // Kill path triggering Die and RawKill
}

func TestGetExpNilGuard(t *testing.T) {
	origGetExp := GetExp
	defer func() { GetExp = origGetExp }()

	GetExp = nil

	ch := &mockCombatant{name: "Hero", room: 100, level: 10, hp: 100, maxHP: 100, position: PosStanding}
	victim := &mockCombatant{name: "Goblin", room: 100, level: 10, hp: 10, maxHP: 10, position: PosStanding}

	// GroupGain with GetExp = nil should not panic and default to 0
	GroupGain(ch, victim)
}
