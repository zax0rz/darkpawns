package spells

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// DP-1022 (F2): a spell kill must drive the SAME death pipeline as a weapon
// kill — World.HandleDeath(victim, killer, spellNum) — so the caster earns
// XP/kill-credit and a slain player pays the COMBAT penalty (EXP/37). The old
// path called combat.TakeDamage then HandleSpellDeath → HandleNonCombatDeath
// (killer=nil, EXP/3), zeroing caster XP and making spell PK ~12x too harsh.
//
// These are white-box tests of inflictDamage's Combatant branch: a mock
// combatant (floors HP at -11 like the real *Player/*MobInstance) and a mock
// world recording HandleDeath vs HandleSpellDeath calls.
// ---------------------------------------------------------------------------

// spellCombatant is a minimal combat.Combatant test double.
type spellCombatant struct {
	name     string
	npc      bool
	level    int
	flags    uint64
	hp       int
	maxHP    int
	room     int
	pos      int
	fighting string
	messages []string
}

func (c *spellCombatant) GetName() string                { return c.name }
func (c *spellCombatant) IsNPC() bool                    { return c.npc }
func (c *spellCombatant) GetFlags() uint64               { return c.flags }
func (c *spellCombatant) GetRoom() int                   { return c.room }
func (c *spellCombatant) GetLevel() int                  { return c.level }
func (c *spellCombatant) GetHP() int                     { return c.hp }
func (c *spellCombatant) GetMaxHP() int                  { return c.maxHP }
func (c *spellCombatant) GetAC() int                     { return 0 }
func (c *spellCombatant) GetTHAC0() int                  { return 20 }
func (c *spellCombatant) GetDamageRoll() combat.DiceRoll { return combat.DiceRoll{} }
func (c *spellCombatant) GetPosition() int               { return c.pos }
func (c *spellCombatant) SetPosition(p int)              { c.pos = p }
func (c *spellCombatant) GetClass() int                  { return 0 }
func (c *spellCombatant) GetStr() int                    { return 13 }
func (c *spellCombatant) GetStrAdd() int                 { return 0 }
func (c *spellCombatant) GetDex() int                    { return 13 }
func (c *spellCombatant) GetInt() int                    { return 13 }
func (c *spellCombatant) GetWis() int                    { return 13 }
func (c *spellCombatant) GetHitroll() int                { return 0 }
func (c *spellCombatant) GetDamroll() int                { return 0 }
func (c *spellCombatant) GetSex() int                    { return 0 }
func (c *spellCombatant) Heal(amount int)                { c.hp += amount }
func (c *spellCombatant) SetFighting(target string)      { c.fighting = target }
func (c *spellCombatant) StopFighting()                  { c.fighting = "" }
func (c *spellCombatant) GetFighting() string            { return c.fighting }
func (c *spellCombatant) SendMessage(msg string)         { c.messages = append(c.messages, msg) }

// TakeDamage mirrors *Player/*MobInstance: HP into the wounded band, floored at
// -11 (POS_DEAD threshold, DP-1021).
func (c *spellCombatant) TakeDamage(amount int) {
	c.hp -= amount
	if c.hp < -11 {
		c.hp = -11
	}
}

type deathCall struct {
	victim, killer combat.Combatant
	attackType     int
}

// spellDeathWorld records both the new (HandleDeath) and old (HandleSpellDeath)
// death entry points so tests can assert the pipeline switched.
type spellDeathWorld struct {
	deaths          []deathCall
	spellDeathCalls int
	woundMsgs       []string
}

func (w *spellDeathWorld) HandleDeath(victim, killer combat.Combatant, attackType int) {
	w.deaths = append(w.deaths, deathCall{victim, killer, attackType})
}

// HandleSpellDeath is the retired non-combat bridge; present only so the tests
// can prove inflictDamage no longer routes through it.
func (w *spellDeathWorld) HandleSpellDeath(victim interface{}) { w.spellDeathCalls++ }

func (w *spellDeathWorld) WoundBroadcast(roomVNum int, message, exclude string) {
	w.woundMsgs = append(w.woundMsgs, message)
}

const testSpellNum = 12 // arbitrary spell number, echoed as attackType

func TestInflictDamage_LethalRoutesThroughHandleDeath(t *testing.T) {
	caster := &spellCombatant{name: "Caster", level: 30, hp: 200, maxHP: 200, pos: combat.PosStanding}
	victim := &spellCombatant{name: "Victim", npc: true, level: 5, hp: 1, maxHP: 100, pos: combat.PosStanding}
	world := &spellDeathWorld{}

	inflictDamage(caster, victim, 50, testSpellNum, world)

	if len(world.deaths) != 1 {
		t.Fatalf("HandleDeath calls = %d, want 1 (spell kill must drive the melee death pipeline)", len(world.deaths))
	}
	got := world.deaths[0]
	if got.victim != combat.Combatant(victim) {
		t.Errorf("HandleDeath victim = %v, want the spell victim", got.victim.GetName())
	}
	if got.killer != combat.Combatant(caster) {
		t.Errorf("HandleDeath killer = %v, want the caster (kill-credit)", got.killer)
	}
	if got.attackType != testSpellNum {
		t.Errorf("HandleDeath attackType = %d, want spellNum %d", got.attackType, testSpellNum)
	}
	if world.spellDeathCalls != 0 {
		t.Errorf("HandleSpellDeath called %d times; want 0 (non-combat EXP/3 path retired)", world.spellDeathCalls)
	}
	// (A dead victim's FIGHTING is cleared by update_pos, so fighting-engagement
	// is asserted in the non-lethal case instead.)
	if victim.GetPosition() != combat.PosDead {
		t.Errorf("victim position = %d, want PosDead(%d)", victim.GetPosition(), combat.PosDead)
	}
}

func TestInflictDamage_NonLethalDoesNotDie(t *testing.T) {
	caster := &spellCombatant{name: "Caster", level: 30, hp: 200, maxHP: 200, pos: combat.PosStanding}
	victim := &spellCombatant{name: "Victim", npc: true, level: 20, hp: 100, maxHP: 100, pos: combat.PosStanding}
	world := &spellDeathWorld{}

	inflictDamage(caster, victim, 10, testSpellNum, world)

	if len(world.deaths) != 0 {
		t.Errorf("HandleDeath called %d times on a non-lethal hit; want 0", len(world.deaths))
	}
	if victim.GetHP() != 90 {
		t.Errorf("victim HP = %d, want 90 (100 - 10)", victim.GetHP())
	}
	if victim.GetFighting() != caster.GetName() {
		t.Errorf("victim fighting = %q, want caster (spell engages combat even when non-lethal)", victim.GetFighting())
	}
}

func TestInflictDamageHonorsLowLevelPlayerProtection(t *testing.T) {
	tests := []struct {
		name       string
		casterLvl  int
		victimLvl  int
		victimFlag uint64
		wantHP     int
		wantMsg    string
	}{
		{
			name:      "experienced caster cannot hit protected newbie",
			casterLvl: 30,
			victimLvl: 1,
			wantHP:    100,
			wantMsg:   "Ancient forces protect Victim from your wrath!\r\n",
		},
		{
			name:      "newbie caster cannot attack player",
			casterLvl: 1,
			victimLvl: 30,
			wantHP:    100,
			wantMsg:   "You are not experienced enough to attack Victim!\r\n",
		},
		{
			name:       "outlaw newbie remains attackable",
			casterLvl:  30,
			victimLvl:  1,
			victimFlag: 1,
			wantHP:     50,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			caster := &spellCombatant{name: "Caster", level: tc.casterLvl, hp: 100, maxHP: 100, pos: combat.PosStanding}
			victim := &spellCombatant{name: "Victim", level: tc.victimLvl, flags: tc.victimFlag, hp: 100, maxHP: 100, pos: combat.PosStanding}
			world := &spellDeathWorld{}

			inflictDamage(caster, victim, 50, testSpellNum, world)

			if victim.hp != tc.wantHP {
				t.Errorf("victim HP = %d, want %d", victim.hp, tc.wantHP)
			}
			if tc.wantMsg != "" {
				if len(caster.messages) != 1 || caster.messages[0] != tc.wantMsg {
					t.Errorf("caster messages = %q, want %q", caster.messages, tc.wantMsg)
				}
			}
		})
	}
}

// A victim dropped into the wounded band (HP 0..-10) is NOT dead — only crossing
// POS_DEAD (HP <= -11) kills, matching melee/skills. This pins the threshold
// move from the old GetHP()<=0 spell-kill trigger to update_pos (DP-1021/1022).
func TestInflictDamage_WoundedBandNotDead(t *testing.T) {
	caster := &spellCombatant{name: "Caster", level: 30, hp: 200, maxHP: 200, pos: combat.PosStanding}
	victim := &spellCombatant{name: "Victim", npc: true, level: 10, hp: 5, maxHP: 100, pos: combat.PosStanding}
	world := &spellDeathWorld{}

	inflictDamage(caster, victim, 12, testSpellNum, world) // 5 - 12 = -7

	if len(world.deaths) != 0 {
		t.Errorf("HandleDeath called %d times at HP -7; want 0 (POS_DEAD only at -11)", len(world.deaths))
	}
	if victim.GetHP() != -7 {
		t.Errorf("victim HP = %d, want -7 (wounded band, floored above -11)", victim.GetHP())
	}
	if victim.GetPosition() != combat.PosMortally {
		t.Errorf("victim position = %d, want PosMortally(%d)", victim.GetPosition(), combat.PosMortally)
	}
}

// An immortal (non-NPC, level >= LVL_IMMORT) victim absorbs all spell damage:
// ApplyDamageModifiers zeroes dam, so no damage and no death.
func TestInflictDamage_ImmortalVictimAbsorbs(t *testing.T) {
	caster := &spellCombatant{name: "Caster", level: 30, hp: 200, maxHP: 200, pos: combat.PosStanding}
	victim := &spellCombatant{name: "Immortal", npc: false, level: 100, hp: 50, maxHP: 50, pos: combat.PosStanding}
	world := &spellDeathWorld{}

	inflictDamage(caster, victim, 40, testSpellNum, world)

	if len(world.deaths) != 0 {
		t.Errorf("HandleDeath called %d times on an immortal victim; want 0", len(world.deaths))
	}
	if victim.GetHP() != 50 {
		t.Errorf("immortal victim HP = %d, want 50 (damage absorbed)", victim.GetHP())
	}
	if victim.GetFighting() != "" {
		t.Errorf("immortal victim fighting = %q, want empty (absorbed hit engages nothing)", victim.GetFighting())
	}
}
