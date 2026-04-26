package spells

import "math/rand"

// MagDamage inflicts damage from a spell, ported from src/magic.c mag_damage().
// Handles die-based damage per spellnum, saving throw halving, and special effects
// (soul leech, backfire chance, reagent bonuses).
func MagDamage(level int, ch, victim interface{}, spellNum, savetype int, world interface{}) {
	if victim == nil || ch == nil {
		return
	}

	// Character type checks
	isMage := isClassMage(ch)
	dam := 0

	switch spellNum {
	/* --- MAGE SPELLS --- */
	case SpellMagicMissile:
		if isMage {
			reag := checkReagents(ch, SpellMagicMissile, level, "shard of obsidian",
				"Pulling a shard of obsidian from a pocket, you crush it under your heel...",
				"$n pulls something out of $s pocket and crushes it beneath $s heel...")
			dam = dice(4, 3) + reag + level
		} else {
			dam = dice(4, 3) + level
		}

	case SpellChillTouch:
		dam = dice(5, 3) + level

	case SpellBurningHands:
		dam = dice(4, 5) + level

	case SpellShockingGrasp:
		dam = dice(4, 7) + level

	case SpellLightningBolt:
		dam = dice(9, 4) + level

	case SpellColorSpray:
		if isMage {
			reag := checkReagents(ch, SpellColorSpray, level, "prism",
				"Pulling a prism from a pocket, you crush it under your heel...",
				"$n pulls something out of $s pocket and crushes it beneath $s heel...")
			dam = dice(9, 7) + reag + level
		} else {
			dam = dice(9, 7) + level
		}

	case SpellFireball:
		if isMage {
			reag := checkReagents(ch, SpellFireball, level, "pinch of ash",
				"Pulling a pinch of ash from a pocket, you cast it about the room...",
				"$n pulls a pinch of ash out of a pocket and casts it about the room.")
			dam = dice(12, 8) + 20 + level + level + reag
		} else {
			dam = dice(12, 8) + level*2
		}

	case SpellDisintegrate:
		if isMage {
			reag := checkReagents(ch, SpellDisintegrate, level, "eye of a beholder",
				"Pulling the eye of a beholder from a pocket, you throw it to the ground...",
				"$n pulls a small orb out of a pocket and dashes it to the ground.")
			dam = dice(18, 8) + 3*level + reag
		} else {
			dam = dice(18, 8) + level
		}
		if !randBool(51) && !isNPC(ch) {
			sendToCaster(ch, "Your magick backfires!\r\n")
			victim = ch
		}

	case SpellDisrupt:
		if isMage {
			dam = dice(20, 7) + 3*level
		} else {
			dam = dice(20, 7) + level
		}
		if !randBool(51) && !isNPC(ch) {
			sendToCaster(ch, "Your magick backfires!\r\n")
			victim = ch
		}

	/* --- CLERIC SPELLS --- */
	case SpellDispelEvil:
		dam = dice(9, 5) + level + 5 + level/2
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
		dam = dice(9, 5) + level + 5
		if isGood(ch) {
			if !isNPC(ch) {
				victim = ch
				dam = getHP(victim) - 10
			}
		} else if isEvil(victim) {
			sendToCaster(ch, "The gods protect $N.\r\n")
			return
		}

	case SpellCallLightning:
		dam = dice(10, 8) + level + 5

	case SpellHarm:
		dam = dice(12, 8) + level*2

	/* --- NINJA / MAGE --- */
	case SpellSoulLeech, SpellEnergyDrain:
		if getLevel(victim) <= 2 {
			dam = 100
		} else {
			dam = dice(10, 6) + level
		}
		if isMage {
			reag := checkReagents(ch, SpellEnergyDrain, level, "vampire dust",
				"Pulling the vampire dust from a pocket, you throw it into the air...",
				"$n throws some dust into the air...")
			dam += reag
		}

	/* --- AREA SPELLS --- */
	case SpellEarthquake:
		dam = dice(7, 7) + level

	case SpellAcidBlast:
		dam = dice(4, 3) + level

	/* --- PSIONIC SPELLS --- */
	case SpellMindPoke:
		dam = dice(3, 3) + level

	case SpellMindAttack:
		dam = dice(4, 6) + level

	case SpellMindBlast:
		dam = dice(9, 7) + level + level/2

	case SpellPsiblast:
		dam = dice(15, 13) + 3*level
		if !randBool(31) && !isNPC(ch) {
			sendToCaster(ch, "Suddenly, your psionic power recoils!\r\n")
			victim = ch
		}

	/* --- ADDITIONAL SPELLS --- */
	case SpellFlameStrike:
		dam = dice(12, 8) + level*2

	case SpellHellfire:
		dam = dice(10, 8) + level*3

	case SpellMeteorSwarm:
		dam = dice(20, 6) + level*4

	case SpellCalliope:
		dam = dice(8, 10) + level

	case SpellSmokescreen:
		dam = dice(3, 4) + level/2

	case SpellRayOfDisruption:
		dam = dice(6, 6) + level

	case SpellMentalLapse:
		dam = dice(5, 5) + level

	case SpellFireBreath:
		dam = dice(8, 6) + level

	case SpellGasBreath:
		dam = dice(6, 8) + level

	case SpellFrostBreath:
		dam = dice(7, 7) + level

	case SpellAcidBreath:
		dam = dice(9, 5) + level

	case SpellLightningBreath:
		dam = dice(10, 6) + level

	case SpellDragonBreath:
		dam = dice(15, 10) + level*3

	case SpellDrowning:
		dam = dice(8, 4) + level

	case SpellPetrify:
		dam = dice(6, 6) + level
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
		if !randBool(11) {
			sendToZone("A blast of hot air washes over you.", ch, world)
		}
	case SpellCallLightning, SpellLightningBolt:
		if !randBool(11) {
			sendToZone("Thunder rumbles through the air.", ch, world)
		}
	}
}

// inflictDamage applies damage to a victim via damage(ch, victim, dam, attackType).
// Uses the damage system from combat/game — ported from C damage() call.
func inflictDamage(ch, victim interface{}, dam, attackType int, world interface{}) {
	type damager interface {
		GetHP() int
		GetMaxHP() int
		GetName() string
		SetHP(int)
		SendMessage(string)
		IsNPC() bool
	}

	d, ok := victim.(damager)
	if !ok {
		return
	}

	singular, _ := MagAttackModifier(attackType)

	// Notify victim
	victimMsg := "$n " + singular + " you!"
	type sender interface{ SendMessage(string) }
	if s, ok := victim.(sender); ok {
		s.SendMessage(victimMsg)
	}

	// Reduce HP
	newHP := d.GetHP() - dam
	if newHP < 0 {
		newHP = 0
	}
	d.SetHP(newHP)

	_ = dam // use for death check later
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
	return rand.Intn(denom) == 0
}
