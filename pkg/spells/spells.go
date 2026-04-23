// Package spells provides spell constants and casting functions for Dark Pawns MUD.
// Based on original spells.h and spell constants from globals.lua.
package spells

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

// Cast simulates casting a spell. For now, it's a stub that logs the action.
// Based on lua_spell() function that would be called from Lua scripts.
func Cast(_ interface{}, _ interface{}, _ int, _ bool) {
	// TODO: Implement actual spell casting logic
	// For now, just log that a spell was cast
	// caster and target would be ScriptablePlayer/ScriptableMob interfaces
	// spellNum is one of the SPELL_* constants above
	// aggressive indicates if it's an offensive spell
}
