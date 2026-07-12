package combat

import (
	"strings"
	"testing"
)

type msgMockCombatant struct {
	mockCombatant
	messages []string
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

	// Run processCombatPair. Loop to handle potential natural 1 misses.
	for i := 0; i < 10 && !deathFuncCalled; i++ {
		defender.hp = 1
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
// TestMain (formulas_test.go) wires GetSkill to return 50 for everyone and
// GetWeaponInfo to grant every non-"unarmed_guy" a weapon — which means the
// global parry/dodge checks would otherwise consume ScriptedRoller values and
// randomly negate hits. Making these mocks NPCs (npc: true) short-circuits
// CheckParry at its first line (formulas.go:641), and the dodge probe is fed
// a failing value by the test's ScriptedRoller.
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
// Determinism: a ScriptedRoller forces every d20 to 20 (natural-20 auto-hit)
// and every dodge probe to 101 (dodge fails, since skill is 50). NPCs skip
// the parry check entirely. The remaining values feed the attacks-per-round
// probe and the damage dice. The pattern repeats for each combatant's edge.
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
	// Per edge: Number(0,500)=500 [no bonus attack], Number(1,20)=20 [auto-hit],
	// Number(1,101)=101 [dodge fails], Dice(1,8)=8 [damage die].
	SetRoller(NewScriptedRoller([]int{
		500, 20, 101, 8, // attacker → defender
		500, 20, 101, 8, // defender → attacker
		500, 20, 101, 8, 500, 20, 101, 8, // headroom
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
	// 10 rounds × 2 edges × 4 values = 80; supply extra headroom. ScriptedRoller
	// wraps if it runs short, so this is belt-and-suspenders.
	script := make([]int, 0, 100)
	for i := 0; i < 25; i++ {
		script = append(script, 500, 20, 101, 8) // one edge: probe, hit, dodge-fail, dmg
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
