package spells

import (
	"github.com/zax0rz/darkpawns/pkg/dprng"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// MagDamage inflicts damage from a spell, ported from src/magic.c mag_damage().
// Handles die-based damage per spellnum, saving throw halving, and special effects
// (soul leech, backfire chance, reagent bonuses).
func MagDamage(level int, ch, victim interface{}, spellNum, savetype int, world interface{}) {
	if victim == nil || ch == nil {
		return
	}

	isMage := isClassMage(ch)

	// Consume reagents for mage damage spells. In C, has_reagents() is a boolean;
	// the bonus is the full caster level when the reagent is present.
	hasReag := false
	switch spellNum {
	case SpellMagicMissile:
		hasReag = isMage && consumeReagent(ch, SpellMagicMissile, level, "shard of obsidian",
			"Pulling a shard of obsidian from a pocket, you crush it under your heel...",
			"$n pulls something out of $s pocket and crushes it beneath $s heel...")
	case SpellColorSpray:
		hasReag = isMage && consumeReagent(ch, SpellColorSpray, level, "prism",
			"Pulling a prism from a pocket, you crush it under your heel...",
			"$n pulls something out of $s pocket and crushes it beneath $s heel...")
	case SpellFireball:
		hasReag = isMage && consumeReagent(ch, SpellFireball, level, "pinch of ash",
			"Pulling a pinch of ash from a pocket, you cast it about the room...",
			"$n pulls a pinch of ash out of a pocket and casts it about the room.")
	case SpellDisintegrate:
		hasReag = isMage && consumeReagent(ch, SpellDisintegrate, level, "eye of a beholder",
			"Pulling the eye of a beholder from a pocket, you throw it to the ground...",
			"$n pulls a small orb out of a pocket and dashes it to the ground.")
	case SpellSoulLeech, SpellEnergyDrain:
		hasReag = isMage && consumeReagent(ch, SpellEnergyDrain, level, "vampire dust",
			"Pulling the vampire dust from a pocket, you throw it into the air...",
			"$n throws some dust into the air...")
	}

	// Compute base damage dice/flat from the C mag_damage() formula table.
	num, sides, flat, ok := magDamageFormula(level, getLevel(victim), spellNum, isMage, hasReag)
	if !ok {
		return
	}

	dam := dice(num, sides) + flat

	// Backfire chances for mage spells that can turn the spell on the caster.
	switch spellNum {
	case SpellDisintegrate, SpellDisrupt:
		if randBool(51) && !isNPC(ch) {
			sendToCaster(ch, "Your magick backfires!\r\n")
			victim = ch
		}
	}

	// Alignment-based lethal/special cases for dispel evil/good.
	switch spellNum {
	case SpellDispelEvil:
		if isEvil(ch) {
			if !isNPC(ch) {
				victim = ch
				dam = getHP(victim) - 10
			}
		} else if isGood(victim) {
			sendToCaster(ch, "The gods protect $N.\r\n")
			return
		}
	case SpellDispelGood:
		if isGood(ch) {
			if !isNPC(ch) {
				victim = ch
				dam = getHP(victim) - 10
			}
		} else if isEvil(victim) {
			sendToCaster(ch, "The gods protect $N.\r\n")
			return
		}
	}

	// Backfire / recoil effects that change the target.
	if spellNum == SpellPsiblast && randBool(31) && !isNPC(ch) {
		sendToCaster(ch, "Suddenly, your psionic power recoils!\r\n")
		victim = ch
	}

	// Apply saving throw — half damage on success
	if dam > 0 && magSavingThrow(victim, savetype) {
		dam /= 2
		if dam < 1 {
			dam = 1
		}
	}

	// Inflict damage
	if dam > 0 {
		inflictDamage(ch, victim, dam, spellNum, world)
	}

	// Special post-damage effects
	switch spellNum {
	case SpellSoulLeech:
		// Soul leech heals caster for 1/3 of damage dealt
		healAmt := dam / 3
		if healAmt > 0 {
			type healer interface {
				GetHP() int
				GetMaxHP() int
				SetHP(int)
			}
			if h, ok := ch.(healer); ok {
				newHP := h.GetHP() + healAmt
				if newHP > h.GetMaxHP() {
					newHP = h.GetMaxHP()
				}
				h.SetHP(newHP)
			}
		}
	case SpellFireball:
		if randBool(11) {
			sendToZone("A blast of hot air washes over you.", ch, world)
		}
	case SpellCallLightning, SpellLightningBolt:
		if randBool(11) {
			sendToZone("Thunder rumbles through the air.", ch, world)
		}
	}
}

// magDamageFormula returns the base dice parameters and flat bonus for a damage spell.
// It mirrors src/magic.c:mag_damage() dice formulas exactly, with no RNG and no side effects.
// hasReagent should be true when the caster is a mage and consumed the spell's reagent.
// For flat-only spells (e.g. energy drain on a level<=2 victim) it returns (0, 0, flat, true).
// Returns ok=false for spell numbers that are not dice-damage spells in C's mag_damage().
func magDamageFormula(level, victimLevel, spellNum int, isMage, hasReagent bool) (numDice, sidesDice, flat int, ok bool) {
	switch spellNum {
	/* --- MAGE SPELLS --- */
	case SpellMagicMissile:
		bonus := 0
		if isMage && hasReagent {
			bonus = level
		}
		return 4, 3, level + bonus, true
	case SpellChillTouch:
		return 5, 3, level, true
	case SpellBurningHands:
		return 4, 5, level, true
	case SpellShockingGrasp:
		return 4, 7, level, true
	case SpellLightningBolt:
		return 9, 4, level, true
	case SpellColorSpray:
		bonus := 0
		if isMage && hasReagent {
			bonus = level
		}
		return 9, 7, level + bonus, true
	case SpellFireball:
		if isMage {
			bonus := 0
			if hasReagent {
				bonus = level
			}
			return 12, 8, 20 + level*2 + bonus, true
		}
		return 12, 8, level * 2, true
	case SpellDisintegrate:
		if isMage {
			bonus := 0
			if hasReagent {
				bonus = level
			}
			return 18, 8, 3*level + bonus, true
		}
		return 18, 8, level, true
	case SpellDisrupt:
		if isMage {
			return 20, 7, 3 * level, true
		}
		return 20, 7, level, true

	/* --- CLERIC SPELLS --- */
	case SpellDispelEvil:
		return 9, 5, 5 + level + level/2, true
	case SpellDispelGood:
		return 9, 5, 5 + level, true
	case SpellCallLightning:
		return 10, 8, 5 + level, true
	case SpellHarm:
		return 12, 8, level * 2, true

	/* --- NINJA / MAGE --- */
	case SpellSoulLeech, SpellEnergyDrain:
		if victimLevel <= 2 {
			return 0, 0, 100, true
		}
		bonus := 0
		if isMage && hasReagent {
			bonus = level
		}
		return 10, 6, level + bonus, true

	/* --- AREA SPELLS --- */
	case SpellEarthquake:
		return 7, 7, level, true
	case SpellAcidBlast:
		return 4, 3, level, true

	/* --- PSIONIC SPELLS --- */
	case SpellMindPoke:
		return 3, 3, level, true
	case SpellMindAttack:
		return 4, 6, level, true
	case SpellMindBlast:
		return 9, 7, level + level/2, true
	case SpellPsiblast:
		return 15, 13, 3 * level, true

	default:
		// Not a dice-damage spell in C mag_damage() (manual spells, breath weapons, etc.)
		return 0, 0, 0, false
	}
}

// consumeReagent checks for and removes a reagent from the caster's inventory,
// returning true if the reagent was found. It sends flavor messages when consumed.
func consumeReagent(ch interface{}, spellNum, level int, reagentName, casterMsg, roomMsg string) bool {
	reag := checkReagents(ch, spellNum, level, reagentName, casterMsg, roomMsg)
	return reag > 0
}

// woundBroadcaster lets spell damage route wound/position messages through the
// same room broadcaster melee and skills use (game.World.WoundBroadcast →
// woundBroadcast). Kept as a narrow interface so the spells package needn't
// import game.
type woundBroadcaster interface {
	WoundBroadcast(roomVNum int, message, exclude string)
}

// spellDeathPipeline is the melee DeathFunc path (game.World.HandleDeath). Spell
// kills route through it so they award XP/kill-credit and apply the COMBAT death
// penalty (EXP/37, killer-aware corpse, PK bookkeeping), identical to weapon
// kills — see inflictDamage / DP-1022.
type spellDeathPipeline interface {
	HandleDeath(victim, killer combat.Combatant, attackType int)
}

// inflictDamage applies spell damage and, on a lethal blow, drives the death
// pipeline — mirroring the melee/skill damage tail (World.doDamage /
// World.DoSpellDamage) exactly: shared damage() modifier block → apply damage →
// set fighting → update_pos → World.HandleDeath at POS_DEAD.
//
// In C, mag_damage() (src/magic.c:827) calls the same damage() as weapon hits,
// so a spell kill must award XP/kill-credit and apply the COMBAT death penalty
// (EXP/37, killer-aware corpse) through the same HandleDeath path melee uses —
// not HandleNonCombatDeath (EXP/3, killer=nil), which zeroed the caster's kill
// XP and made spell PK ~12x more punishing than the original (DP-1022).
//
// This replaces the earlier combat.TakeDamage(...) + HandleSpellDeath call. Once
// TakeDamage stopped clamping HP at 0 (DP-1021, floor at -11), combat.TakeDamage's
// own award/DieWithKiller block became reachable on the spell path and
// double-handled death alongside HandleSpellDeath. Routing straight to
// HandleDeath keeps a single death authority across melee, skills, and spells.
func inflictDamage(ch, victim interface{}, dam, attackType int, world interface{}) {
	// Send spell flavor text to victim
	singular, _ := MagAttackModifier(attackType)
	victimMsg := "$n " + singular + " you!"
	type sender interface{ SendMessage(string) }
	if s, ok := victim.(sender); ok {
		s.SendMessage(victimMsg)
	}

	// Type-assert to combat.Combatant to run the shared damage tail.
	chCombat, chOk := ch.(combat.Combatant)
	victCombat, victOk := victim.(combat.Combatant)
	if chOk && victOk {
		// Shared damage() modifier block: sanctuary, protect evil/good,
		// race-hate, the 3000 cap, and immortal invulnerability (DP-1025).
		dam = combat.ApplyDamageModifiers(chCombat, victCombat, dam)
		if dam <= 0 {
			// Fully absorbed (immortal victim, or sanctuary on a tiny hit):
			// no damage, no death — matches World.DoSpellDamage.
			return
		}

		victCombat.TakeDamage(dam)
		victCombat.SetFighting(chCombat.GetName())

		// Enter the wounded band or POS_DEAD from the new HP; only run the
		// death pipeline at POS_DEAD (HP <= -11) — fight.c update_pos (DP-1021).
		var wb func(roomVNum int, message, exclude string)
		if b, ok := world.(woundBroadcaster); ok {
			wb = b.WoundBroadcast
		}
		if combat.UpdatePositionAfterDamage(victCombat, wb) == combat.PosDead {
			if dp, ok := world.(spellDeathPipeline); ok {
				dp.HandleDeath(victCombat, chCombat, attackType)
			}
		}
		return
	}

	// Fallback: if ch or victim doesn't implement Combatant, apply raw damage
	// (maintains backward compatibility for non-standard callers)
	type damager interface {
		GetHP() int
		SetHP(int)
	}
	d, ok := victim.(damager)
	if !ok {
		return
	}
	newHP := d.GetHP() - dam
	if newHP < 0 {
		newHP = 0
	}
	d.SetHP(newHP)
}

// --- character trait checks (used in damage formulas) ---

func isNPC(ch interface{}) bool {
	type npcer interface{ IsNPC() bool }
	if n, ok := ch.(npcer); ok {
		return n.IsNPC()
	}
	return false
}

func isEvil(ch interface{}) bool {
	type aligner interface{ GetAlignment() int }
	if a, ok := ch.(aligner); ok {
		return a.GetAlignment() < 0
	}
	return false
}

func isGood(ch interface{}) bool {
	type aligner interface{ GetAlignment() int }
	if a, ok := ch.(aligner); ok {
		return a.GetAlignment() > 0
	}
	return false
}

func getHP(ch interface{}) int {
	type hper interface{ GetHP() int }
	if h, ok := ch.(hper); ok {
		return h.GetHP()
	}
	return 0
}

// zoneMessager is the interface for world to send messages to all players in a zone.
type zoneMessager interface {
	ForEachPlayerInZoneInterface(int, func(interface{}))
}

func sendToZone(msg string, ch interface{}, world interface{}) {
	w, ok := world.(zoneMessager)
	if !ok {
		return
	}
	type roomed interface{ GetRoomVNum() int }
	rg, ok := ch.(roomed)
	if !ok {
		return
	}
	// Get caster's zone from world
	type zoner interface{ GetRoomZone(int) int }
	zw, ok := world.(zoner)
	if !ok {
		return
	}
	zone := zw.GetRoomZone(rg.GetRoomVNum())

	type sender interface{ SendMessage(string) }
	w.ForEachPlayerInZoneInterface(zone, func(p interface{}) {
		if s, ok := p.(sender); ok {
			s.SendMessage(msg)
		}
	})
}

func randBool(denom int) bool {
	if denom <= 1 {
		return true
	}
	// #nosec G404 — game RNG, not cryptographic
	// #nosec G404
	return dprng.Number(0, denom-1) == 0
}
