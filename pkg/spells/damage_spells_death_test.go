package spells

import (
	"strings"
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
	fighting combat.Combatant
	messages []string
}

func (c *spellCombatant) GetName() string                         { return c.name }
func (c *spellCombatant) IsNPC() bool                             { return c.npc }
func (c *spellCombatant) GetFlags() uint64                        { return c.flags }
func (c *spellCombatant) GetRoom() int                            { return c.room }
func (c *spellCombatant) GetLevel() int                           { return c.level }
func (c *spellCombatant) GetHP() int                              { return c.hp }
func (c *spellCombatant) GetMaxHP() int                           { return c.maxHP }
func (c *spellCombatant) GetAC() int                              { return 0 }
func (c *spellCombatant) GetTHAC0() int                           { return 20 }
func (c *spellCombatant) GetDamageRoll() combat.DiceRoll          { return combat.DiceRoll{} }
func (c *spellCombatant) GetPosition() int                        { return c.pos }
func (c *spellCombatant) SetPosition(p int)                       { c.pos = p }
func (c *spellCombatant) GetClass() int                           { return 0 }
func (c *spellCombatant) GetStr() int                             { return 13 }
func (c *spellCombatant) GetStrAdd() int                          { return 0 }
func (c *spellCombatant) GetDex() int                             { return 13 }
func (c *spellCombatant) GetInt() int                             { return 13 }
func (c *spellCombatant) GetWis() int                             { return 13 }
func (c *spellCombatant) GetHitroll() int                         { return 0 }
func (c *spellCombatant) GetDamroll() int                         { return 0 }
func (c *spellCombatant) GetSex() int                             { return 0 }
func (c *spellCombatant) Heal(amount int)                         { c.hp += amount }
func (c *spellCombatant) SetFightingBody(target combat.Combatant) { c.fighting = target }
func (c *spellCombatant) GetFightingBody() combat.Combatant       { return c.fighting }
func (c *spellCombatant) StopFighting()                           { c.fighting = nil }
func (c *spellCombatant) GetFighting() string {
	if c.fighting == nil {
		return ""
	}
	return c.fighting.GetName()
}
func (c *spellCombatant) SendMessage(msg string) { c.messages = append(c.messages, msg) }

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

func (w *spellDeathWorld) WoundBroadcast(roomVNum int, message string, exclude []combat.Combatant) {
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

// gatedSpellWorld refuses or allows damage the way World.DamageRefused does;
// the refusal bytes themselves are covered in pkg/game (DP-1327).
type gatedSpellWorld struct {
	spellDeathWorld
	refuse bool
	asked  int
}

func (w *gatedSpellWorld) DamageRefused(ch, victim combat.Combatant) bool {
	w.asked++
	return w.refuse
}

func TestInflictDamageAsksWorldDamageGate(t *testing.T) {
	for _, refuse := range []bool{true, false} {
		caster := &spellCombatant{name: "Caster", level: 30, hp: 100, maxHP: 100, pos: combat.PosStanding}
		victim := &spellCombatant{name: "Victim", level: 1, hp: 100, maxHP: 100, pos: combat.PosStanding}
		world := &gatedSpellWorld{refuse: refuse}

		dealt := inflictDamage(caster, victim, 50, testSpellNum, world)

		if world.asked != 1 {
			t.Fatalf("refuse=%v: gate asked %d times, want 1", refuse, world.asked)
		}
		wantHP, wantDealt := 50, true
		if refuse {
			wantHP, wantDealt = 100, false
		}
		if victim.hp != wantHP || dealt != wantDealt {
			t.Errorf("refuse=%v: victim HP = %d, dealt = %v; want %d, %v", refuse, victim.hp, dealt, wantHP, wantDealt)
		}
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
	victim := &spellCombatant{name: "Immortal", npc: false, level: lvlImmort, hp: 50, maxHP: 50, pos: combat.PosStanding}
	world := &spellDeathWorld{}

	inflictDamage(caster, victim, 40, testSpellNum, world)

	if len(world.deaths) != 0 {
		t.Errorf("HandleDeath called %d times on an immortal victim; want 0", len(world.deaths))
	}
	if victim.GetHP() != 50 {
		t.Errorf("immortal victim HP = %d, want 50 (damage absorbed)", victim.GetHP())
	}
	// damage() enrolls before immortal absorption (src/fight.c:1443-1445, 1473-1480).
	if victim.GetFighting() != caster.GetName() {
		t.Errorf("immortal victim fighting = %q, want %q even when damage is absorbed", victim.GetFighting(), caster.GetName())
	}
}

// R1/R5h: C damage subtracts HP and updates position before skill_message,
// but emits wound notices and performs death cleanup afterward (fight.c:1484-1583).
func TestSpellMessageSeesPostDamagePosition(t *testing.T) {
	for _, tc := range []struct {
		name                               string
		hp, damage, level, wantHP, wantPos int
	}{
		{"lethal", 1, 50, 5, -11, combat.PosDead},
		{"mortally", 5, 12, 5, -7, combat.PosMortally},
		{"incapacitated", 5, 9, 5, -4, combat.PosIncap},
		{"stunned", 5, 6, 5, -1, combat.PosStunned},
		{"healthy", 100, 10, 5, 90, combat.PosFighting},
		{"zero", 100, 0, 5, 100, combat.PosFighting},
		{"immortal", 100, 50, 31, 100, combat.PosFighting},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := combat.GetCallbacks()
			t.Cleanup(func() { combat.SetCallbacks(old) })
			caster := &spellCombatant{name: "Caster", level: 30, hp: 200, maxHP: 200, pos: combat.PosStanding}
			victim := &spellCombatant{name: "Victim", level: tc.level, hp: tc.hp, maxHP: 100, pos: combat.PosStanding}
			world := &spellDeathWorld{}
			calls := 0
			combat.SetCallbacks(&combat.GameCallbacks{SkillMessage: func(dam int, ch, vict combat.Combatant, attackType, room int) bool {
				calls++
				if vict.GetHP() != tc.wantHP || vict.GetPosition() != tc.wantPos {
					t.Errorf("message sees HP/position %d/%d, want %d/%d", vict.GetHP(), vict.GetPosition(), tc.wantHP, tc.wantPos)
				}
				if len(victim.messages) != 0 || len(world.woundMsgs) != 0 || len(world.deaths) != 0 {
					t.Error("wound/death output preceded skill message")
				}
				if victim.GetFightingBody() != caster {
					t.Error("combat cleanup preceded skill message")
				}
				return true
			}})
			inflictDamage(caster, victim, tc.damage, testSpellNum, world)
			if calls != 1 {
				t.Fatalf("message calls=%d, want 1", calls)
			}
			wantDeaths := 0
			if tc.wantPos == combat.PosDead {
				wantDeaths = 1
			}
			if len(world.deaths) != wantDeaths {
				t.Errorf("death calls=%d, want %d", len(world.deaths), wantDeaths)
			}
		})
	}
}

// The production file-backed selector must consume exactly its one variant
// draw for lethal, ordinary, absorbed and breath damage alike (R3).
type spellMessageRoller struct{ draws int }

func (r *spellMessageRoller) Number(from, to int) int { r.draws++; return from }
func (r *spellMessageRoller) Dice(num, size int) int  { r.draws++; return num }
func (r *spellMessageRoller) IntN(n int) int          { r.draws++; return 0 }
func TestSpellMessageBranchesAndDrawParity(t *testing.T) {
	for _, tc := range []struct {
		name                      string
		hp, damage, level, attack int
		branch                    string
	}{
		{"lethal", 1, 50, 5, 12, "die"},
		{"wounded", 5, 12, 5, 12, "hit"},
		{"zero", 100, 0, 5, 12, "miss"},
		{"immortal", 100, 50, 31, 12, "god"},
		{"breath-zero", 100, 0, 5, SpellFireBreath, "miss"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			old := combat.GetCallbacks()
			t.Cleanup(func() { combat.SetCallbacks(old) })
			caster := &spellCombatant{name: "Caster", level: 30, hp: 200, maxHP: 200, pos: combat.PosStanding}
			victim := &spellCombatant{name: "Victim", level: tc.level, hp: tc.hp, maxHP: 100, pos: combat.PosStanding}
			events := []string{}
			cb := &combat.GameCallbacks{
				GetHP:      func(c combat.Combatant) int { return c.GetHP() },
				GetLevel:   func(c combat.Combatant) int { return c.GetLevel() },
				IsNPC:      func(c combat.Combatant) bool { return c.IsNPC() },
				SendToChar: func(c combat.Combatant, s string) { events = append(events, c.GetName()+":"+s) },
				Broadcast:  func(_ int, s string, _ []combat.Combatant) { events = append(events, "room:"+s) },
			}
			action := func(s string) combat.FightMessageAction {
				return combat.FightMessageAction{Attacker: strings.ToUpper(s[:1]) + s[1:], Victim: strings.ToUpper(s[:1]) + s[1:], Room: strings.ToUpper(s[:1]) + s[1:]}
			}
			combat.SetCallbacks(cb)
			combat.InitFightMessages(cb, combat.FightMessages{tc.attack: {{Die: action("die"), Hit: action("hit"), Miss: action("miss"), God: action("god")}}})
			roller := &spellMessageRoller{}
			combat.WithRoller(roller, func() { inflictDamage(caster, victim, tc.damage, tc.attack, &spellDeathWorld{}) })
			branch := strings.ToUpper(tc.branch[:1]) + tc.branch[1:]
			want := "room:" + branch + ",Caster:" + branch + ",Victim:" + branch
			if strings.Join(events, ",") != want {
				t.Errorf("spell message audiences=%v, want %s", events, want)
			}
			if roller.draws != 1 {
				t.Errorf("selector draws=%d, want 1", roller.draws)
			}
		})
	}
}
