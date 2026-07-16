package combat

import (
	"strings"
	"sync"
	"testing"
	"time"
)

type msgMockCombatant struct {
	mockCombatant
	messages []string
}

func TestCombatEngineStopWaitsForInFlightTick(t *testing.T) {
	ce := NewCombatEngine()
	ce.tickEvery = time.Millisecond

	entered := make(chan struct{})
	release := make(chan struct{})
	var enteredOnce sync.Once
	ce.OnRoundEnd = func() {
		enteredOnce.Do(func() { close(entered) })
		<-release
	}
	ce.Start()

	select {
	case <-entered:
	case <-time.After(time.Second):
		close(release)
		ce.Stop()
		t.Fatal("combat ticker did not enter a round")
	}

	stopped := make(chan struct{})
	go func() {
		ce.Stop()
		close(stopped)
	}()

	select {
	case <-stopped:
		t.Error("Stop returned while a combat tick was still running")
	case <-time.After(25 * time.Millisecond):
	}

	close(release)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("Stop did not return after the in-flight tick exited")
	}
}

func (m *msgMockCombatant) SendMessage(msg string) {
	m.messages = append(m.messages, msg)
}

func TestCombatMessages_HaveNewlines(t *testing.T) {
	attacker := &msgMockCombatant{mockCombatant: mockCombatant{name: "Attacker"}}
	defender := &msgMockCombatant{mockCombatant: mockCombatant{name: "Defender"}}

	ce := NewCombatEngine()
	var broadcasts []string
	ce.BroadcastFunc = func(roomVNum int, message string, exclude string) {
		broadcasts = append(broadcasts, message)
	}

	// Test hit message
	attacker.messages = nil
	defender.messages = nil
	broadcasts = nil
	ce.sendHitMessage(attacker, defender, 10, 0)

	if len(attacker.messages) != 1 || !strings.HasSuffix(attacker.messages[0], "\r\n") {
		t.Errorf("attacker hit message does not end with \\r\\n: %q", attacker.messages)
	}
	if len(defender.messages) != 1 || !strings.HasSuffix(defender.messages[0], "\r\n") {
		t.Errorf("defender hit message does not end with \\r\\n: %q", defender.messages)
	}
	if len(broadcasts) != 1 || !strings.HasSuffix(broadcasts[0], "\r\n") {
		t.Errorf("room hit broadcast does not end with \\r\\n: %q", broadcasts)
	}

	// Test miss message
	attacker.messages = nil
	defender.messages = nil
	broadcasts = nil
	ce.sendMissMessage(attacker, defender, 0)

	if len(attacker.messages) != 1 || !strings.HasSuffix(attacker.messages[0], "\r\n") {
		t.Errorf("attacker miss message does not end with \\r\\n: %q", attacker.messages)
	}
	if len(defender.messages) != 1 || !strings.HasSuffix(defender.messages[0], "\r\n") {
		t.Errorf("defender miss message does not end with \\r\\n: %q", defender.messages)
	}
	if len(broadcasts) != 1 || !strings.HasSuffix(broadcasts[0], "\r\n") {
		t.Errorf("room miss broadcast does not end with \\r\\n: %q", broadcasts)
	}
}

func TestSendHitMessageUsesMessageFunc(t *testing.T) {
	ce := NewCombatEngine()
	var gotAttacker, gotDefender Combatant
	var gotDam, gotAttackType int
	ce.MessageFunc = func(attacker, defender Combatant, dam int, atkType int) bool {
		gotAttacker = attacker
		gotDefender = defender
		gotDam = dam
		gotAttackType = atkType
		return true
	}

	atk := &mockCombatant{name: "Alice", room: 1}
	def := &mockCombatant{name: "Bob", room: 1}
	ce.sendHitMessage(atk, def, 25, 5)

	if gotAttacker != atk {
		t.Errorf("MessageFunc got attacker %v, want %v", gotAttacker, atk)
	}
	if gotDefender != def {
		t.Errorf("MessageFunc got defender %v, want %v", gotDefender, def)
	}
	if gotDam != 25 {
		t.Errorf("MessageFunc got dam=%d, want 25", gotDam)
	}
	if gotAttackType != 5 {
		t.Errorf("MessageFunc got attackType=%d, want 5", gotAttackType)
	}
}

func TestSendMissMessageUsesMessageFunc(t *testing.T) {
	ce := NewCombatEngine()
	var gotDam, gotAttackType int
	ce.MessageFunc = func(attacker, defender Combatant, dam int, atkType int) bool {
		gotDam = dam
		gotAttackType = atkType
		return true
	}

	atk := &mockCombatant{name: "Alice", room: 1}
	def := &mockCombatant{name: "Bob", room: 1}
	ce.sendMissMessage(atk, def, 7)

	if gotDam != 0 {
		t.Errorf("MessageFunc miss got dam=%d, want 0", gotDam)
	}
	if gotAttackType != 7 {
		t.Errorf("MessageFunc miss got attackType=%d, want 7", gotAttackType)
	}
}

func TestPerformInitialAttackResolvesExactlyOneSynchronousHit(t *testing.T) {
	ce := NewCombatEngine()
	attacker := &mockCombatant{
		name: "Attacker", room: 1, level: 1, hp: 20, maxHP: 20,
		position: PosStanding, class: ClassWarrior, str: 10, dex: 10, intVal: 10, wis: 10,
	}
	defender := &mockCombatant{
		name: "Defender", room: 1, level: 1, hp: 20, maxHP: 20,
		position: PosStanding, class: ClassWarrior, str: 10, dex: 10, intVal: 10, wis: 10,
	}
	if err := ce.StartCombat(attacker, defender); err != nil {
		t.Fatalf("StartCombat() error = %v", err)
	}

	messageCalls := 0
	ce.MessageFunc = func(gotAttacker, gotDefender Combatant, damage, attackType int) bool {
		messageCalls++
		if gotAttacker != attacker || gotDefender != defender || damage != 0 {
			t.Errorf("initial miss callback = (%v, %v, %d), want attacker, defender, 0", gotAttacker, gotDefender, damage)
		}
		return true
	}

	roller := NewScriptedRoller([]int{1, 99}) // natural 1: guaranteed miss
	WithRoller(roller, func() {
		if err := ce.PerformInitialAttack(attacker, defender); err != nil {
			t.Fatalf("PerformInitialAttack() error = %v", err)
		}
	})

	if got, want := messageCalls, 1; got != want {
		t.Fatalf("initial attack message calls = %d, want %d", got, want)
	}
	if got, want := roller.Index, 1; got != want {
		t.Fatalf("initial miss draws = %d, want one to-hit draw", got)
	}
	if defender.GetHP() != 20 {
		t.Fatalf("defender HP after forced miss = %d, want 20", defender.GetHP())
	}
}

func TestSendHitMessageFallsBackWhenNil(t *testing.T) {
	ce := NewCombatEngine()
	atk := &msgMockCombatant{mockCombatant: mockCombatant{name: "Alice"}}
	def := &msgMockCombatant{mockCombatant: mockCombatant{name: "Bob"}}
	ce.sendHitMessage(atk, def, 25, 5)

	if len(atk.messages) != 1 || !strings.Contains(atk.messages[0], "You hit Bob for 25 damage") {
		t.Errorf("fallback hit message wrong: %q", atk.messages)
	}
}

func TestSendMissMessageFallsBackWhenNil(t *testing.T) {
	ce := NewCombatEngine()
	atk := &msgMockCombatant{mockCombatant: mockCombatant{name: "Alice"}}
	def := &msgMockCombatant{mockCombatant: mockCombatant{name: "Bob"}}
	ce.sendMissMessage(atk, def, 5)

	if len(atk.messages) != 1 || !strings.Contains(atk.messages[0], "You miss Bob") {
		t.Errorf("fallback miss message wrong: %q", atk.messages)
	}
}

// TestShopkeeperProtection_RemovesCombatPair is the DP-923 regression:
// shopkeeper protection must stop combat for both attacker and defender,
// matching C fight.c:1359-1366.
func TestShopkeeperProtection_RemovesCombatPair(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	attacker := &mockCombatant{name: "Attacker", hp: 100, room: 1}
	defender := &mockCombatant{name: "Shopkeeper", hp: 100, room: 1}

	ce := NewCombatEngine()
	ce.SetCallbacks(&GameCallbacks{
		IsShopkeeper: func(name string) bool { return name == "Shopkeeper" },
	})
	if err := ce.StartCombat(attacker, defender); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	ce.processCombatPair(ce.combatPairs[CombatPairKey{Attacker: "Attacker", Target: "Shopkeeper"}])

	if ce.IsFighting("Attacker") {
		t.Error("expected Attacker to be removed from combat after shopkeeper protection")
	}
	if ce.IsFighting("Shopkeeper") {
		t.Error("expected Shopkeeper to be removed from combat after shopkeeper protection")
	}
	if attacker.GetFighting() != "" {
		t.Errorf("expected Attacker.StopFighting called, still fighting %q", attacker.GetFighting())
	}
	if defender.GetFighting() != "" {
		t.Errorf("expected Shopkeeper.StopFighting called, still fighting %q", defender.GetFighting())
	}
}

type fixedRoller struct {
	number int
}

func (r fixedRoller) Number(_, _ int) int { return r.number }
func (r fixedRoller) Dice(_, _ int) int   { return 1 }
func (r fixedRoller) IntN(int) int        { return 0 }

func TestMobRedirect_JailGuardSubduesInsteadOfDamaging(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	guard := &mockCombatant{
		name:       "jail guard",
		npc:        true,
		room:       8117,
		level:      30,
		hp:         80,
		maxHP:      100,
		position:   PosFighting,
		damageRoll: DiceRoll{Num: 10, Sides: 10},
	}
	victim := &mockCombatant{
		name:     "Thief",
		room:     8117,
		level:    20,
		hp:       50,
		maxHP:    50,
		position: PosFighting,
	}

	subdued := false
	ce := NewCombatEngine()
	ce.SetCallbacks(&GameCallbacks{
		MobHasJailGuardSpec: func(name string) bool { return name == "jail guard" },
		HasAffect:           func(name string, aff int) bool { return false },
		JailGuardSubdue: func(guardName, victimName string) bool {
			subdued = guardName == "jail guard" && victimName == "Thief"
			return subdued
		},
	})
	if err := ce.StartCombat(guard, victim); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	ce.processCombatPair(ce.combatPairs[CombatPairKey{Attacker: "jail guard", Target: "Thief"}])

	if !subdued {
		t.Fatal("expected jail guard subdue callback")
	}
	if victim.GetHP() != 50 {
		t.Errorf("expected no live-path melee damage after subdue, hp=%d", victim.GetHP())
	}
	if ce.IsFighting("jail guard") || ce.IsFighting("Thief") {
		t.Fatal("expected jail guard intercept to clear combat")
	}
}

func TestMobRedirect_NonJailMobDoesNotSubdue(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	attacker := &mockCombatant{
		name:       "angry mob",
		npc:        true,
		room:       8117,
		level:      30,
		hp:         80,
		maxHP:      100,
		position:   PosFighting,
		damageRoll: DiceRoll{Num: 10, Sides: 10},
	}
	// HP set high so the attacker's 10d10 round cannot kill the victim and end
	// combat — otherwise the "combat continues" assertion below is nondeterministic.
	victim := &mockCombatant{
		name:     "Thief",
		room:     8117,
		level:    20,
		hp:       1000,
		maxHP:    1000,
		position: PosFighting,
	}

	subdued := false
	ce := NewCombatEngine()
	ce.SetCallbacks(&GameCallbacks{
		MobHasJailGuardSpec: func(name string) bool { return false },
		HasAffect:           func(name string, aff int) bool { return false },
		JailGuardSubdue: func(guardName, victimName string) bool {
			subdued = true
			return true
		},
	})
	if err := ce.StartCombat(attacker, victim); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	ce.processCombatPair(ce.combatPairs[CombatPairKey{Attacker: "angry mob", Target: "Thief"}])

	if subdued {
		t.Fatal("expected JailGuardSubdue not to fire for non-jail mob")
	}
	if !ce.IsFighting("angry mob") || !ce.IsFighting("Thief") {
		t.Fatal("expected combat to continue normally")
	}
}

func TestMobRedirect_CharmedPetRetargetsToMaster(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	attacker := &mockCombatant{name: "angry mob", npc: true, room: 10, level: 15, hp: 50, maxHP: 50, position: PosFighting}
	pet := &mockCombatant{name: "charmed pet", npc: true, room: 10, level: 5, hp: 50, maxHP: 50, position: PosFighting}
	master := &mockCombatant{name: "Master", room: 10, level: 20, hp: 50, maxHP: 50, position: PosFighting}

	ce := NewCombatEngine()
	ce.SetCallbacks(&GameCallbacks{
		HasAffect: func(name string, aff int) bool {
			return name == "charmed pet" && aff == AFF_CHARM
		},
		GetFollowing: func(name string) string {
			if name == "charmed pet" {
				return "Master"
			}
			return ""
		},
		GetRoomCombatants: func(roomVNum int) []Combatant {
			return []Combatant{attacker, pet, master}
		},
	})
	if err := ce.StartCombat(attacker, pet); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	WithRoller(fixedRoller{number: 0}, func() {
		ce.processCombatPair(ce.combatPairs[CombatPairKey{Attacker: "angry mob", Target: "charmed pet"}])
	})

	if attacker.GetFighting() != "Master" {
		t.Fatalf("expected attacker retargeted to Master, got %q", attacker.GetFighting())
	}
	if _, ok := ce.combatPairs[CombatPairKey{Attacker: "angry mob", Target: "Master"}]; !ok {
		t.Fatal("expected combat pair redirected to Master")
	}
	if pet.GetHP() != 50 {
		t.Errorf("expected charmed pet to take no damage during redirect, hp=%d", pet.GetHP())
	}
}

func TestMobRedirect_HighLevelSwitcheroo(t *testing.T) {
	orig := GetCallbacks()
	defer SetCallbacks(orig)

	dragon := &mockCombatant{name: "dragon", npc: true, room: 20, level: 30, hp: 200, maxHP: 200, position: PosFighting}
	tank := &mockCombatant{name: "Tank", room: 20, level: 20, hp: 100, maxHP: 100, position: PosFighting}
	rogue := &mockCombatant{name: "Rogue", room: 20, level: 20, hp: 100, maxHP: 100, position: PosFighting, fighting: "dragon"}

	ce := NewCombatEngine()
	ce.SetCallbacks(&GameCallbacks{
		GetRoomCombatants: func(roomVNum int) []Combatant {
			return []Combatant{dragon, tank, rogue}
		},
	})
	if err := ce.StartCombat(dragon, tank); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	WithRoller(fixedRoller{number: 0}, func() {
		ce.processCombatPair(ce.combatPairs[CombatPairKey{Attacker: "dragon", Target: "Tank"}])
	})

	if dragon.GetFighting() != "Rogue" {
		t.Fatalf("expected high-level mob retargeted to Rogue, got %q", dragon.GetFighting())
	}
	if _, ok := ce.combatPairs[CombatPairKey{Attacker: "dragon", Target: "Rogue"}]; !ok {
		t.Fatal("expected combat pair redirected to Rogue")
	}
	if tank.GetHP() != 100 {
		t.Errorf("expected original defender to take no damage during switcheroo, hp=%d", tank.GetHP())
	}
}

func TestHandleDeath_PassesAttackType(t *testing.T) {
	attacker := &mockCombatant{
		name:     "Attacker",
		npc:      false,
		level:    30,
		hp:       100,
		maxHP:    100,
		position: PosStanding,
	}
	defender := &mockCombatant{
		name:     "Defender",
		npc:      true,
		level:    1,
		hp:       1,
		maxHP:    10,
		position: PosSleeping,
	}

	ce := NewCombatEngine()
	err := ce.StartCombat(attacker, defender)
	if err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	deathFuncCalled := false
	receivedAttackType := -1
	ce.DeathFunc = func(victim, killer Combatant, attackType int) {
		deathFuncCalled = true
		receivedAttackType = attackType
	}

	pairKey := CombatPairKey{Attacker: "Attacker", Target: "Defender"}
	pair := ce.combatPairs[pairKey]

	// Run processCombatPair until a blow lands. Reset the defender to just
	// above the POS_DEAD floor (-11, DP-1021) each iteration so ANY landed hit
	// is lethal — the loop only needs to survive natural-1 misses, not also
	// roll >=12 damage. Resetting to hp=1 made the kill depend on the damage
	// roll crossing -11, which flaked when 10 straight swings rolled low.
	for i := 0; i < 10 && !deathFuncCalled; i++ {
		defender.hp = -10
		ce.processCombatPair(pair)
	}

	if !deathFuncCalled {
		t.Fatal("DeathFunc was not called")
	}

	if receivedAttackType != int(AttackNormal) {
		t.Errorf("expected attackType %d, got %d", int(AttackNormal), receivedAttackType)
	}
}

// dp900Fighter returns a mock combatant tuned to land hits deterministically.
//
// TestMain (formulas_test.go) wires GetSkill to return values for parry tests.
// Making these mocks NPCs (npc: true) short-circuits player parry; because they
// do not have AFF_DODGE, the dodge path does not consume roller values either.
//
// Level 10 keeps GetAttacksPerRound at its baseline 1 attack without firing
// the level-gated bonus-attack d100 rolls (formulas.go:578-602), so the only
// roller call before the hit roll is the single Number(0,500) probe at line 602.
func dp900Fighter(name string) *mockCombatant {
	return &mockCombatant{
		name:       name,
		npc:        true,
		hp:         100,
		maxHP:      100,
		room:       1,
		level:      10,
		thac0:      1,
		ac:         10,
		hitroll:    50,
		intVal:     25,
		wis:        25,
		damageRoll: DiceRoll{Num: 1, Sides: 8, Plus: 5},
		position:   PosStanding,
	}
}

// TestPerformRound_DefenderRetaliates is the DP-900 regression: previously
// PerformRound only processed pair.Attacker, so defenders never swung. With the
// fix it iterates both sides of every pair, so both the attacker and the
// defender deal damage in the same round.
//
// Determinism: a ScriptedRoller forces every d20 to 20 (natural-20 auto-hit).
// The remaining values feed the attacks-per-round probe and the damage dice.
// The pattern repeats for each combatant's edge.
func TestPerformRound_DefenderRetaliates(t *testing.T) {
	attacker := dp900Fighter("Attacker")
	defender := dp900Fighter("Defender")

	ce := NewCombatEngine()
	if err := ce.StartCombat(attacker, defender); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	// Both combatants must be flagged fighting for PerformRound to swing them.
	if defender.GetFighting() == "" {
		t.Fatalf("DP-900 precondition: defender should have FIGHTING set after StartCombat, got %q", defender.GetFighting())
	}

	old := GetRoller()
	// Per edge: Number(0,900)=900 [no bonus attack], Number(1,20)=20 [auto-hit],
	// Dice(1,8)=8 [damage die].
	SetRoller(NewScriptedRoller([]int{
		900, 20, 8, // attacker → defender
		900, 20, 8, // defender → attacker
		900, 20, 8, 900, 20, 8, // headroom
	}))
	defer SetRoller(old)

	ce.PerformRound()

	if defender.GetHP() >= 100 {
		t.Errorf("DP-900: defender should have taken damage from attacker, HP still %d", defender.GetHP())
	}
	if attacker.GetHP() >= 100 {
		t.Errorf("DP-900: attacker should have taken retaliation damage from defender, HP still %d", attacker.GetHP())
	}
}

// TestPerformRound_BothSidesDealDamageOverRounds runs several rounds and
// confirms both combatants bleed HP — the core symptom from the Fable repro
// ("player HP never moves"). Same deterministic roller as the single-round
// test, sized to cover 10 rounds × 2 edges.
func TestPerformRound_BothSidesDealDamageOverRounds(t *testing.T) {
	p := dp900Fighter("Player")
	m := dp900Fighter("Mob")

	ce := NewCombatEngine()
	if err := ce.StartCombat(p, m); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	old := GetRoller()
	// 10 rounds × 2 edges × 3 values = 60; supply extra headroom. ScriptedRoller
	// wraps if it runs short, so this is belt-and-suspenders.
	script := make([]int, 0, 100)
	for i := 0; i < 25; i++ {
		script = append(script, 900, 20, 8) // one edge: probe, hit, dmg
	}
	SetRoller(NewScriptedRoller(script))
	defer SetRoller(old)

	for i := 0; i < 10; i++ {
		if p.GetHP() <= 0 || m.GetHP() <= 0 {
			break
		}
		ce.PerformRound()
	}

	if p.GetHP() >= 100 {
		t.Errorf("DP-900: player should have lost HP over 10 rounds, still %d", p.GetHP())
	}
	if m.GetHP() >= 100 {
		t.Errorf("DP-900: mob should have lost HP over 10 rounds, still %d", m.GetHP())
	}
}

func TestProcessCombatPair_ParryReducesAttackCountOncePerRound(t *testing.T) {
	attacker := &mockCombatant{
		name:       "orc",
		npc:        true,
		hp:         100,
		maxHP:      100,
		room:       1,
		level:      11, // C NPC baseline: 2 attacks
		thac0:      1,
		ac:         10,
		hitroll:    50,
		damageRoll: DiceRoll{Num: 1, Sides: 1},
		position:   PosStanding,
		fighting:   "parry_warrior",
	}
	defender := &mockCombatant{
		name:     "parry_warrior",
		npc:      false,
		hp:       100,
		maxHP:    100,
		room:     1,
		level:    20,
		thac0:    1,
		ac:       10,
		hitroll:  50,
		dex:      10,
		position: PosStanding,
		fighting: "orc",
	}

	ce := NewCombatEngine()
	hits := 0
	ce.OnCombatAction = func(attacker Combatant, defender Combatant, attackType string, damage int, outcome string, targetCount int) {
		if outcome == "hit" {
			hits++
		}
	}

	old := GetRoller()
	// NPC bonus attack probe fails, parry succeeds, then the single remaining
	// hit lands and rolls 1 damage. If parry still negated individual hits or
	// fired per-hit, this would not be exactly one hit.
	SetRoller(NewScriptedRoller([]int{900, 80, 20, 1, 20, 1}))
	defer SetRoller(old)

	ce.processCombatPair(&CombatPair{Attacker: attacker, Defender: defender})

	if hits != 1 {
		t.Fatalf("hits after successful parry = %d, want 1", hits)
	}
	if got := defender.GetHP(); got != 99 {
		t.Fatalf("defender HP after parried two-attack round = %d, want 99", got)
	}
}

// TestPerformRound_IgnoresNonFightingCombatant confirms a combatant whose
// FIGHTING target was cleared (e.g. fled) does not get a phantom attack — the
// fix resolves each fighter's actual FIGHTING target rather than blindly
// processing every pair edge.
func TestPerformRound_IgnoresNonFightingCombatant(t *testing.T) {
	attacker := dp900Fighter("Attacker")
	defender := dp900Fighter("Defender")

	ce := NewCombatEngine()
	if err := ce.StartCombat(attacker, defender); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	// Defender "flees" — clears its own FIGHTING but the pair still exists.
	defender.SetFighting("")

	startHP := attacker.GetHP()
	ce.PerformRound()

	if attacker.GetHP() != startHP {
		t.Errorf("DP-900: attacker should not be hit by a non-fighting defender, HP %d → %d", startHP, attacker.GetHP())
	}
}

// -----------------------------------------------------------------------------
// DP-1032: mob wait-state enforcement in combat rounds
// -----------------------------------------------------------------------------

type waitStateMockCombatant struct {
	mockCombatant
	waitState int
}

func (m *waitStateMockCombatant) GetWaitState() int      { return m.waitState }
func (m *waitStateMockCombatant) SetWaitState(ticks int) { m.waitState = ticks }
func (m *waitStateMockCombatant) DecrementWaitState() {
	if m.waitState > 0 {
		m.waitState--
	}
}

func TestProcessCombatPair_MobWithWaitSkipsAttack(t *testing.T) {
	attacker := &waitStateMockCombatant{
		mockCombatant: mockCombatant{name: "Orc", npc: true, room: 1, position: PosSitting, fighting: "Hero", hp: 100, maxHP: 100, level: 10, ac: 10},
		waitState:     2,
	}
	defender := &mockCombatant{name: "Hero", room: 1, position: PosFighting, hp: 100, maxHP: 100, ac: 10}

	ce := NewCombatEngine()
	if err := ce.StartCombat(attacker, defender); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	ce.BroadcastFunc = func(roomVNum int, message string, exclude string) {}

	// processCombatPair will skip attacks when wait is active, so defender HP
	// should stay at 100.
	ce.processCombatPair(ce.combatPairs[CombatPairKey{Attacker: "Orc", Target: "Hero"}])

	if attacker.waitState != 1 {
		t.Errorf("expected wait state decremented to 1, got %d", attacker.waitState)
	}
	if defender.hp != 100 {
		t.Errorf("expected defender HP unchanged while mob waits, got %d", defender.hp)
	}
	if attacker.GetPosition() != PosSitting {
		t.Errorf("expected attacker still sitting while wait active, got %d", attacker.GetPosition())
	}
}

func TestProcessCombatPair_MobStandsWhenWaitExpires(t *testing.T) {
	attacker := &waitStateMockCombatant{
		mockCombatant: mockCombatant{name: "Orc", npc: true, room: 1, position: PosSitting, fighting: "Hero", hp: 100, maxHP: 100, level: 10, ac: 10},
		waitState:     1,
	}
	defender := &mockCombatant{name: "Hero", room: 1, position: PosFighting, hp: 100, maxHP: 100, ac: 10}

	ce := NewCombatEngine()
	if err := ce.StartCombat(attacker, defender); err != nil {
		t.Fatalf("StartCombat failed: %v", err)
	}

	var broadcasts []string
	ce.BroadcastFunc = func(roomVNum int, message string, exclude string) {
		broadcasts = append(broadcasts, message)
	}

	ce.processCombatPair(ce.combatPairs[CombatPairKey{Attacker: "Orc", Target: "Hero"}])

	if attacker.waitState != 0 {
		t.Errorf("expected wait state 0 after expiry, got %d", attacker.waitState)
	}
	if attacker.GetPosition() != PosFighting {
		t.Errorf("expected attacker to stand up when wait expires, got %d", attacker.GetPosition())
	}
	if len(broadcasts) == 0 || !strings.Contains(broadcasts[0], "scrambles") {
		t.Errorf("expected stand-up room broadcast, got %v", broadcasts)
	}
}
