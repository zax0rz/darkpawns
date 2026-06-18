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

	// Test hit message
	attacker.messages = nil
	defender.messages = nil
	ce.sendHitMessage(attacker, defender, 10)

	if len(attacker.messages) != 1 || !strings.HasSuffix(attacker.messages[0], "\r\n") {
		t.Errorf("attacker hit message does not end with \\r\\n: %q", attacker.messages)
	}
	if len(defender.messages) != 1 || !strings.HasSuffix(defender.messages[0], "\r\n") {
		t.Errorf("defender hit message does not end with \\r\\n: %q", defender.messages)
	}

	// Test miss message
	attacker.messages = nil
	defender.messages = nil
	ce.sendMissMessage(attacker, defender)

	if len(attacker.messages) != 1 || !strings.HasSuffix(attacker.messages[0], "\r\n") {
		t.Errorf("attacker miss message does not end with \\r\\n: %q", attacker.messages)
	}
	if len(defender.messages) != 1 || !strings.HasSuffix(defender.messages[0], "\r\n") {
		t.Errorf("defender miss message does not end with \\r\\n: %q", defender.messages)
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
