// Package spells provides spell constants and casting functions for Dark Pawns MUD.
// Based on original spells.h and spell constants from globals.lua.
package spells

import "github.com/zax0rz/darkpawns/pkg/engine"

// Spell constants from spells.h and globals.lua
const (
	// Core spells referenced in combat AI scripts
	SpellMagicMissile  = 32
	SpellBurningHands  = 5
	SpellLightningBolt = 30
	SpellFireball      = 26
	SpellHellfire      = 58
	SpellColorSpray    = 10
	SpellDisrupt       = 92
	SpellDisintegrate  = 93
	SpellFlamestrike   = 96
	SpellAcidBlast     = 75
	SpellTeleport      = 2
	SpellDispelEvil    = 22
	SpellDispelGood    = 46
	SpellHeal          = 28
	SpellVitality      = 67
	SpellCureLight     = 16
	SpellHarm          = 27
	SpellBlindness     = 4
	SpellCurse         = 17
	SpellPoison        = 33
	SpellEarthquake    = 23
	SpellDivineInt     = 81
	SpellMindBar       = 82
	SpellShockingGrasp = 37
	SpellChillTouch    = 8
	SpellEnergyDrain   = 21
	SpellSoulLeech     = 94
	SpellPsiblast      = 100
	SpellPetrify       = 104
	SpellDrowning      = 24
	SpellCallLightning = 15
	SpellMeteorSwarm   = 41
	SpellSleep         = 38
	SpellCharm         = 7
	SpellBless         = 3
	SpellGreatPercept  = 70
	SpellChangeDensity = 74
	SpellSanctuary     = 36
	SpellRemovePoison  = 43
	SpellEnchantArmor  = 59
	SpellEnchantWeapon = 24
	SpellIdentify      = 60
	SpellWordOfRecall  = 42
	SpellInvulnerability = 66
	SpellFireBreath    = 202
	SpellGasBreath     = 203
	SpellFrostBreath   = 204
	SpellAcidBreath    = 205
	SpellLightningBreath = 206

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
