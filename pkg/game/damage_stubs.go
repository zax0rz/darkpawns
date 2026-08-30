package game

// CRIT-009: Dual hit-resolution paths in Dark Pawns combat
//
// Path 1 — processCombatPair (pkg/combat/engine.go)
//   The main combat tick. Runs every round for each fighting pair.
//   Flow: hit roll → parry check → dodge check → damage → death
//   Used for: standard melee attacks (auto-attack each combat round)
//   parry/dodge: CHECKED (via CheckParry/CheckDodge in formulas.go)
//
// Path 2 — doDamage / DoSpellDamage (this file, pkg/game/damage_stubs.go)
//   Called directly by skills, spells, and special attacks.
//   Flow: hit roll (skill-specific) → damage → death
//   Used for: bash, kick, backstab, spell damage, etc.
//   parry/dodge: NOT CHECKED (intentional)
//
// Why two paths? This matches CircleMUD design:
//   - Skill attacks (bash, kick, trip) have their own hit/miss logic and
//     intentionally bypass parry/dodge — that's the tradeoff for using
//     a limited-use skill vs auto-attack.
//   - Backstab from behind explicitly cannot be parried or dodged.
//   - Spell damage (fireball, etc.) uses saving throws instead of parry/dodge.
//
// Future consideration: Some melee skills (e.g., second attack, third attack)
//   MAY want parry/dodge checks. Currently they go through processCombatPair
//   and are handled correctly. Only manually-invoked skills use path 2.

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// Damage helper stubs for act_offensive.go
// These match the CircleMUD fight.c patterns: damage() and hit_skill()
// ---------------------------------------------------------------------------

// skillToAttackType maps a skill/spell name to the closest TYPE_* / SKILL_*
// numeric attack type used by the corpse-description system. Unknown skills
// fall back to TYPE_SLASH (303) to preserve legacy behavior.
func skillToAttackType(skill string) int {
	switch skill {
	case "backstab":
		return SkillBackstabNum
	case "circle":
		return SkillCircleNum
	case "bash", "kick", "punch", "dragon_kick", "tiger_punch", "headbutt",
		"smackheads", "slug", "serpent_kick":
		return TypeBludgeon // 305
	case "bite":
		return SkillBiteNum // SKILL_BITE (150)
	case "charge":
		return SkillChargeNum // SKILL_CHARGE (147), default corpse wording
	case "disembowel":
		return SkillDisembowelNum // 184
	case "neckbreak":
		return SkillNeckbreakNum // 190
	default:
		return TypeSlash // 303 fallback
	}
}

// combatantFromInterface converts an interface{} attacker to a combat.Combatant.
func combatantFromInterface(attacker interface{}) combat.Combatant {
	if attacker == nil {
		return nil
	}
	if c, ok := attacker.(combat.Combatant); ok {
		return c
	}
	return nil
}

// DoSpellDamage applies damage to a player or mob, handling death.
// Used by damage spells (hellfire, meteor_swarm, etc.) that need to hit any character type.
func (w *World) DoSpellDamage(attacker, victim interface{}, dam int, skill string) bool {
	attackerName := getAttackerName(attacker)
	killer := combatantFromInterface(attacker)
	attackType := skillToAttackType(skill)

	// Route the computed damage through the shared damage() modifier block
	// (sanctuary, protect evil/good, race-hate, immortal immunity, 3000 cap)
	// before applying — the same funnel melee and skills use (DP-1025). A hit
	// reduced to 0 (immortal victim / sanctuary on a tiny hit) deals nothing.
	if victimC := combatantFromInterface(victim); victimC != nil {
		dam = combat.ApplyDamageModifiers(killer, victimC, dam)
	}
	if dam <= 0 {
		return false
	}

	switch v := victim.(type) {
	case *Player:
		v.TakeDamage(dam)
		v.SetFighting(attackerName)
		// Enter the wounded band or POS_DEAD from the new HP; only run the
		// death pipeline at POS_DEAD (HP <= -11) — fight.c update_pos (DP-1021).
		newPos := combat.UpdatePositionAfterDamage(v, w.woundBroadcast)
		if skill == SkillCharge && newPos > combat.PosStunned && dam > v.GetMaxHP()/4 {
			// C damage() burns the pain/scream number(0,2) draw even when the
			// victim is an NPC with no descriptor (fight.c:1580-1585). The
			// charge vehicle reaches this branch and relies on its draw order.
			_ = dprng.Number(0, 2)
		}
		if newPos == combat.PosDead {
			w.HandleDeath(v, killer, attackType)
		}
		return true
	case *MobInstance:
		v.TakeDamage(dam)
		v.SetFighting(attackerName)
		// Enter the wounded band or POS_DEAD from the new HP; only run the
		// death pipeline at POS_DEAD (HP <= -11) — fight.c update_pos (DP-1021).
		newPos := combat.UpdatePositionAfterDamage(v, w.woundBroadcast)
		if skill == SkillCharge && newPos > combat.PosStunned && dam > v.GetMaxHP()/4 {
			// C evaluates this random scream branch for mobs too; act() simply
			// has no descriptor to deliver the victim-directed output to.
			_ = dprng.Number(0, 2)
		}
		if newPos == combat.PosDead {
			w.HandleDeath(v, killer, attackType)
		}
		return true
	default:
		return false
	}
}

// doDamage applies skill/offensive damage to a player or mob and handles death.
// Used by bash, kick, backstab, disembowel, ambush, neckbreak, tiger/dragon
// kick, spec_proc hits, etc. — every skill that goes through the "path 2"
// damage pipeline documented at the top of this file.
//
// DP-901: previously type-asserted vict.(*Player) and silently returned false
// for mobs, so every offensive skill against a mob no-op'd. Now mirrors
// DoSpellDamage: damages both types, sets fighting state, and routes death
// through HandleDeath so XP, kill counters, events, and autoloot fire.
func (w *World) doDamage(ch, vict interface{}, dam int, skill string) bool {
	attackerName := getAttackerName(ch)
	killer := combatantFromInterface(ch)
	attackType := skillToAttackType(skill)

	// Route skill/offensive damage through the shared damage() modifier block
	// (sanctuary, protect evil/good, race-hate, immortal immunity, 3000 cap)
	// before applying — the same funnel melee and spells use (DP-1025).
	if victimC := combatantFromInterface(vict); victimC != nil {
		dam = combat.ApplyDamageModifiers(killer, victimC, dam)
	}

	if dam <= 0 {
		// C: damage(ch, vict, 0, ...) prints the no-damage message and still
		// counts as a hit (used by skills that connect for zero damage).
		switch v := vict.(type) {
		case *Player:
			v.SendMessage(fmt.Sprintf("%s hits you, but it doesn't hurt!\r\n", attackerName))
		case *MobInstance:
			// Mobs don't get a player-style message; the call site reports.
		}
		return false
	}

	switch v := vict.(type) {
	case *Player:
		v.TakeDamage(dam)
		v.SetFighting(attackerName)
		// Enter the wounded band or POS_DEAD from the new HP; only run the
		// death pipeline at POS_DEAD (HP <= -11) — fight.c update_pos (DP-1021).
		if combat.UpdatePositionAfterDamage(v, w.woundBroadcast) == combat.PosDead {
			w.HandleDeath(v, killer, attackType)
		}
		return true
	case *MobInstance:
		v.TakeDamage(dam)
		v.SetFighting(attackerName)
		// Enter the wounded band or POS_DEAD from the new HP; only run the
		// death pipeline at POS_DEAD (HP <= -11) — fight.c update_pos (DP-1021).
		if combat.UpdatePositionAfterDamage(v, w.woundBroadcast) == combat.PosDead {
			w.HandleDeath(v, killer, attackType)
		}
		return true
	default:
		return false
	}
}

// woundBroadcast adapts World's room messaging to the signature that
// combat.UpdatePositionAfterDamage expects (roomVNum, message, exclude).
func (w *World) woundBroadcast(roomVNum int, message, exclude string) {
	w.roomMessageExcludeTwo(roomVNum, message, exclude, "")
}

// WoundBroadcast exposes woundBroadcast across the package boundary so the
// spells package can route spell-damage wound/position messages through the
// exact same room broadcaster melee and skills use (DP-1022). It is the
// exported wrapper the spells package asserts via its woundBroadcaster
// interface; keep its signature aligned with woundBroadcast.
func (w *World) WoundBroadcast(roomVNum int, message, exclude string) {
	w.woundBroadcast(roomVNum, message, exclude)
}

// getAttackerName returns the name of the attacker for messages.
func getAttackerName(ch interface{}) string {
	if p, ok := ch.(*Player); ok {
		return p.GetName()
	}
	if m, ok := ch.(*MobInstance); ok {
		return m.GetName()
	}
	return "someone"
}

// randRange returns a random integer in [min, max].

// executeCommand executes a command string on behalf of a player (e.g. from doOrder).
// Delegates to the session layer via CommandExecFunc if wired.
func (w *World) executeCommand(ch *Player, command string) bool {
	if w.CommandExecFunc != nil {
		return w.CommandExecFunc(ch, command)
	}
	return false
}

// doForced is a stub for perform_act / do_forced — received a forced command string
func (w *World) doForced(ch *Player, command string) bool {
	return w.executeCommand(ch, command)
}

// doBackstab handles the backstab command

// diceRoll rolls N dice of D sides each through the canonical stream.
func diceRoll(n, d int) int {
	return dprng.Dice(n, d)
}
