// Package spells provides spell constants and casting functions for Dark Pawns MUD.
// Based on original spells.h and spell constants from globals.lua.
//
// Implementation status:
//
//	Implemented (in Cast() switch → ApplySpellAffects):
//	  SPELL_BLINDNESS, SPELL_CURSE, SPELL_POISON, SPELL_SLEEP, SPELL_SANCTUARY
//
//	TODO: Damage & effect spells — Cast() default case is a stub:
//	  SPELL_MAGIC_MISSILE, SPELL_BURNING_HANDS, SPELL_LIGHTNING_BOLT, SPELL_FIREBALL,
//	  SPELL_HELLFIRE, SPELL_COLOR_SPRAY, SPELL_DISRUPT, SPELL_DISINTEGRATE,
//	  SPELL_FLAMESTRIKE, SPELL_ACID_BLAST, SPELL_TELEPORT, SPELL_DISPEL_EVIL,
//	  SPELL_DISPEL_GOOD, SPELL_HEAL, SPELL_VITALITY, SPELL_CURE_LIGHT,
//	  SPELL_HARM, SPELL_EARTHQUAKE, SPELL_DIVINE_INT, SPELL_MIND_BAR,
//	  SPELL_SHOCKING_GRASP, SPELL_CHILL_TOUCH, SPELL_ENERGY_DRAIN,
//	  SPELL_SOUL_LEECH, SPELL_PSIBLAST, SPELL_PETRIFY, SPELL_DROWNING,
//	  SPELL_CALL_LIGHTNING, SPELL_METEOR_SWARM, SPELL_CHARM, SPELL_BLESS,
//	  SPELL_GREAT_PERCEPT, SPELL_CHANGE_DENSITY, SPELL_REMOVE_POISON,
//	  SPELL_ENCHANT_ARMOR, SPELL_ENCHANT_WEAPON, SPELL_IDENTIFY,
//	  SPELL_WORD_OF_RECALL, SPELL_INVULNERABILITY, SPELL_FIRE_BREATH,
//	  SPELL_GAS_BREATH, SPELL_FROST_BREATH, SPELL_ACID_BREATH,
//	  SPELL_LIGHTNING_BREATH, SPELL_CONJURE_ELEMENTAL, SPELL_SOBRIETY,
//	  SPELL_GROUP_HEAL, SPELL_MASS_HEAL, SPELL_HASTE, SPELL_SLOW,
//	  SPELL_INVISIBLE, SPELL_GROUP_INVIS, SPELL_PROTECT_EVIL,
//	  SPELL_PROTECT_GOOD, SPELL_REMOVE_CURSE, SPELL_DETECT_ALIGNMENT,
//	  SPELL_DETECT_INVIS, SPELL_DETECT_MAGIC, SPELL_DETECT_POISON,
//	  SPELL_STRENGTH, SPELL_SUMMON, SPELL_LOCATE_OBJECT, SPELL_SENSE_LIFE,
//	  SPELL_HOLY_SHIELD, SPELL_GROUP_RECALL, SPELL_INFRAVISION,
//	  SPELL_WATERWALK, SPELL_FLY, SPELL_LEVITATE, SPELL_METALSKIN,
//	  SPELL_INVIGORATE, SPELL_LESSER_PERCEPTION, SPELL_MIND_ATTACK,
//	  SPELL_ADRENALINE, SPELL_PSYSHIELD, SPELL_DOMINATE,
//	  SPELL_CELL_ADJUSTMENT, SPELL_ZEN, SPELL_MIRROR_IMAGE,
//	  SPELL_CONSUME_DENSITY, SPELL_MINDBLAST, SPELL_CHAMELEON,
//	  SPELL_MINDPOKE, SPELL_DREAM_TRAVEL, SPELL_CALL_OF_CHAOS,
//	  SPELL_WATER_BREATHE, SPELL_MASS_DOMINATE, SPELL_CALLIOPE,
//	  SPELL_MIND_SIGHT, SPELL_SOUL_LEECH, SPELL_FLAMESTRIKE,
//	  SPELL_CONJURE_ELEMENTAL, SPELL_CURE_CRITICAL, SPELL_CURE_BLIND,
//	  SPELL_CREATE_FOOD, SPELL_CREATE_WATER, SPELL_CONTROL_WEATHER,
//	  SPELL_CLONE, SPELL_ARMOR
//
// TODO: Skills (fighter.lua) — called but not yet ported to Go:
//
//	SKILL_HEADBUTT, SKILL_PARRY, SKILL_BASH, SKILL_BERSERK,
//	SKILL_KICK, SKILL_TRIP
package spells

import "github.com/zax0rz/darkpawns/pkg/engine"

// Spell constants from spells.h and globals.lua
const (
	// Core spells referenced in combat AI scripts
	SpellMagicMissile    = 32
	SpellBurningHands    = 5
	SpellLightningBolt   = 30
	SpellFireball        = 26
	SpellHellfire        = 58
	SpellColorSpray      = 10
	SpellDisrupt         = 92
	SpellDisintegrate    = 93
	SpellFlamestrike     = 96
	SpellAcidBlast       = 75
	SpellTeleport        = 2
	SpellDispelEvil      = 22
	SpellDispelGood      = 46
	SpellHeal            = 28
	SpellVitality        = 67
	SpellCureLight       = 16
	SpellHarm            = 27
	SpellBlindness       = 4
	SpellCurse           = 17
	SpellPoison          = 33
	SpellEarthquake      = 23
	SpellDivineInt       = 81
	SpellIntellect       = 81
	SpellMindBar         = 82
	SpellShockingGrasp   = 37
	SpellChillTouch      = 8
	SpellEnergyDrain     = 21
	SpellSoulLeech       = 83
	SpellPsiblast        = 100
	SpellPetrify         = 104
	SpellDrowning        = 103
	SpellCallLightning   = 6
	SpellMeteorSwarm     = 41
	SpellSleep           = 38
	SpellCharm           = 7
	SpellBless           = 3
	SpellGreatPercept    = 70
	SpellChangeDensity   = 74
	SpellSanctuary       = 36
	SpellRemovePoison    = 43
	SpellEnchantArmor    = 59
	SpellEnchantWeapon   = 24
	SpellIdentify        = 60
	SpellWordOfRecall    = 42
	SpellInvulnerability = 66
	SpellFireBreath      = 202
	SpellGasBreath       = 203
	SpellFrostBreath     = 204
	SpellAcidBreath      = 205
	SpellLightningBreath = 206

	// Additional spell constants from spells.h needed by spell system
	SpellSummon           = 40
	SpellLocateObject     = 31
	SpellDetectPoison     = 21
	SpellCreateWater      = 13
	SpellLycanthropy      = 54
	SpellVampirism        = 55
	SpellSobriety         = 56
	SpellZen              = 78
	SpellMirrorImage      = 79
	SpellGate             = 87
	SpellMindsight        = 84
	SpellCalliope         = 94
	SpellCoC              = 101
	SpellConjureElemental = 105
	SpellControlWeather   = 11
	SpellMentalLapse      = 90

	// Additional missing spell constants from C
	SpellArmor        = 1  /* Reserved Skill[] DO NOT CHANGE */
	SpellCureBlind    = 14 /* Reserved Skill[] DO NOT CHANGE */
	SpellCureCritic   = 15 /* Reserved Skill[] DO NOT CHANGE */
	SpellDetectInvis  = 19 /* Reserved Skill[] DO NOT CHANGE */
	SpellDetectMagic  = 20 /* Reserved Skill[] DO NOT CHANGE */
	SpellInvisible    = 29 /* Reserved Skill[] DO NOT CHANGE */
	SpellRemoveCurse  = 35 /* Reserved Skill[] DO NOT CHANGE */
	SpellInfravision  = 50 /* Reserved Skill[] DO NOT CHANGE */
	SpellMassHeal     = 52
	SpellFly          = 53
	SpellHaste        = 97
	SpellSlow         = 98
	SpellSmokescreen  = 91
	SpellWaterBreathe = 102

	// Extended spell constants (for damage_spells.go, may not exist in C directly)
	SpellMindPoke        = 185
	SpellMindAttack      = 186
	SpellMindBlast       = 187
	SpellFlameStrike     = 188
	SpellRayOfDisruption = 189
	// Breath weapon aliases matching C constants
	SpellDragonBreath = 207

	// Skill constants referenced in fighter.lua
	SkillHeadbutt = 141
	SkillParry    = 172
	SkillBash     = 132
	SkillBerserk  = 171
	SkillKick     = 134
	SkillTrip     = 144

	// Item type constants
	ItemStaff = 4
)

// Cast executes a spell. For non-damage affect spells (blindness, curse, poison,
// sleep, sanctuary), it routes through ApplySpellAffects to create and apply an
// engine.Affect to the target.
//
// caster and target are the involved entities (must implement engine.Affectable).
// spellNum is one of the Spell* constants above.
// casterLevel scales duration and magnitude of affects.
// aggressive indicates if it's an offensive spell (not yet used).
func Cast(caster interface{}, target interface{}, spellNum int, casterLevel int, am *engine.AffectManager) {
	// Route non-damage affect spells through ApplySpellAffects
	switch spellNum {
	case SpellBlindness, SpellCurse, SpellPoison, SpellSleep, SpellSanctuary:
		targetAffectable, ok := target.(engine.Affectable)
		if !ok {
			return
		}
		ApplySpellAffects(targetAffectable, spellNum, casterLevel, am)
	default:
		// TODO: Implement damage spells and other spell types
	}
}
