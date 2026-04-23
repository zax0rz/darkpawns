package unit

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// mockCombatant implements combat.Combatant for testing
type mockCombatant struct {
	name     string
	isNPC    bool
	room     int
	level    int
	hp       int
	maxHP    int
	ac       int
	thac0    int
	damRoll  combat.DiceRoll
	pos      int
	class    int
	str      int
	strAdd   int
	dex      int
	_int     int
	wis      int
	hitroll  int
	damroll  int
	fighting string
}

func (m *mockCombatant) GetName() string             { return m.name }
func (m *mockCombatant) IsNPC() bool                 { return m.isNPC }
func (m *mockCombatant) GetRoom() int                { return m.room }
func (m *mockCombatant) GetLevel() int               { return m.level }
func (m *mockCombatant) GetHP() int                  { return m.hp }
func (m *mockCombatant) GetMaxHP() int               { return m.maxHP }
func (m *mockCombatant) GetAC() int                  { return m.ac }
func (m *mockCombatant) GetTHAC0() int               { return m.thac0 }
func (m *mockCombatant) GetDamageRoll() combat.DiceRoll { return m.damRoll }
func (m *mockCombatant) GetPosition() int            { return m.pos }
func (m *mockCombatant) GetClass() int               { return m.class }
func (m *mockCombatant) GetStr() int                 { return m.str }
func (m *mockCombatant) GetStrAdd() int              { return m.strAdd }
func (m *mockCombatant) GetDex() int                 { return m.dex }
func (m *mockCombatant) GetInt() int                 { return m._int }
func (m *mockCombatant) GetWis() int                 { return m.wis }
func (m *mockCombatant) GetHitroll() int             { return m.hitroll }
func (m *mockCombatant) GetDamroll() int             { return m.damroll }
func (m *mockCombatant) TakeDamage(amount int)       { m.hp -= amount }
func (m *mockCombatant) Heal(amount int)             { m.hp += amount }
func (m *mockCombatant) SetFighting(target string)   { m.fighting = target }
func (m *mockCombatant) StopFighting()               { m.fighting = "" }
func (m *mockCombatant) GetFighting() string         { return m.fighting }
func (m *mockCombatant) SendMessage(msg string)      {}

func TestCalculateHitChance(t *testing.T) {
	// Level 5 warrior vs level 5 warrior — equal combat
	attacker := &mockCombatant{
		name:  "Attacker",
		level: 5,
		class: combat.ClassWarrior,
		str:   15, dex: 15, _int: 10, wis: 10,
		pos: combat.PosStanding,
	}
	defender := &mockCombatant{
		name:  "Defender",
		level: 5,
		class: combat.ClassWarrior,
		ac:    50,
		str:   15, dex: 15, _int: 10, wis: 10,
		pos: combat.PosStanding,
	}

	// Run multiple times to exercise the hit logic
	hits := 0
	runs := 100
	for i := 0; i < runs; i++ {
		if combat.CalculateHitChance(attacker, defender) {
			hits++
		}
	}

	// Should register some hits — even a weak THAC0 (17) vs decent AC (50/10=5 + dex -1 = 4)
	// calc_thaco = 17 - 0 - 0 - 0 - 0 = 17
	// victimAC = 50/10 + dexApp[15].Defensive = 5 + (-1) = 4
	// miss if (17 - diceroll) > 4, meaning diceroll < 13 = miss
	// So roughly 60% hit rate
	if hits == 0 {
		t.Error("CalculateHitChance() should produce some hits")
	}
}

func TestCalculateDamage(t *testing.T) {
	attacker := &mockCombatant{
		name:   "Attacker",
		level:  5,
		class:  combat.ClassWarrior,
		str:    15,
		damRoll: combat.DiceRoll{Num: 1, Sides: 8},
		pos:    combat.PosStanding,
	}
	defender := &mockCombatant{
		name: "Defender",
		ac:   50,
		pos:  combat.PosFighting,
	}

	damage := combat.CalculateDamage(attacker, defender, combat.DiceRoll{}, combat.AttackNormal)

	if damage < 1 {
		t.Errorf("CalculateDamage() = %d, expected at least 1", damage)
	}
}

func TestRollDice(t *testing.T) {
	result := combat.RollDice(2, 6)
	if result < 2 || result > 12 {
		t.Errorf("RollDice(2, 6) = %d, expected between 2 and 12", result)
	}
}
