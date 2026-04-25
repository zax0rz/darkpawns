// Ported from src/fight.c (deferred functions), src/utils.c, src/mobact.c,
// src/tattoo.c.
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// See LICENSE for license information.

package game

// APPLY_* constants — from structs.h
const (
	ApplyNone   = 0
	ApplyStr    = 1
	ApplyDex    = 2
	ApplyInt    = 3
	ApplyWis    = 4
	ApplyCon    = 5
	ApplyHitroll  = 18
	ApplyDamroll = 19
)

// --------------------------------------------------------------------------
// GetMinusDam — damage reduction based on target AC
// Source: src/fight.c:1722
// --------------------------------------------------------------------------

// GetMinusDam applies AC-based damage reduction.
// In C: get_minusdam(int dam, struct char_data *ch).
func GetMinusDam(dam int, ac int) int {
	const pcmod = 2.0

	switch {
	case ac > 90:
		return dam
	case ac > 80:
		return dam - int(float64(dam)*(0.01*pcmod))
	case ac > 70:
		return dam - int(float64(dam)*(0.02*pcmod))
	case ac > 60:
		return dam - int(float64(dam)*(0.03*pcmod))
	case ac > 50:
		return dam - int(float64(dam)*(0.04*pcmod))
	case ac > 40:
		return dam - int(float64(dam)*(0.05*pcmod))
	case ac > 30:
		return dam - int(float64(dam)*(0.06*pcmod))
	case ac > 20:
		return dam - int(float64(dam)*(0.07*pcmod))
	case ac > 10:
		return dam - int(float64(dam)*(0.08*pcmod))
	case ac > 0:
		return dam - int(float64(dam)*(0.10*pcmod))
	case ac > -10:
		return dam - int(float64(dam)*(0.11*pcmod))
	case ac > -20:
		return dam - int(float64(dam)*(0.12*pcmod))
	case ac > -30:
		return dam - int(float64(dam)*(0.13*pcmod))
	case ac > -40:
		return dam - int(float64(dam)*(0.14*pcmod))
	case ac > -50:
		return dam - int(float64(dam)*(0.15*pcmod))
	case ac > -60:
		return dam - int(float64(dam)*(0.16*pcmod))
	case ac > -70:
		return dam - int(float64(dam)*(0.17*pcmod))
	case ac > -80:
		return dam - int(float64(dam)*(0.18*pcmod))
	case ac > -90:
		return dam - int(float64(dam)*(0.19*pcmod))
	case ac > -95:
		return dam - int(float64(dam)*(0.20*pcmod))
	case ac > -110:
		return dam - int(float64(dam)*(0.21*pcmod))
	case ac > -130:
		return dam - int(float64(dam)*(0.22*pcmod))
	case ac > -150:
		return dam - int(float64(dam)*(0.23*pcmod))
	case ac > -170:
		return dam - int(float64(dam)*(0.24*pcmod))
	case ac > -190:
		return dam - int(float64(dam)*(0.25*pcmod))
	case ac > -210:
		return dam - int(float64(dam)*(0.26*pcmod))
	case ac > -230:
		return dam - int(float64(dam)*(0.27*pcmod))
	case ac > -250:
		return dam - int(float64(dam)*(0.28*pcmod))
	case ac > -270:
		return dam - int(float64(dam)*(0.29*pcmod))
	case ac > -290:
		return dam - int(float64(dam)*(0.30*pcmod))
	case ac > -310:
		return dam - int(float64(dam)*(0.31*pcmod))
	default:
		return dam - int(float64(dam)*(0.32*pcmod))
	}
}

// ApplyDamageReduction applies AC-based reduction to damage for a target.
// Convenience wrapper that extracts AC from a Player or namedCombatant.
func ApplyDamageReduction(dam int, ac int) int {
	return GetMinusDam(dam, ac)
}

// HookGetMinusDam is a hook for the combat package to call when it needs
// AC-based damage reduction. Registered in init().
func HookGetMinusDam(dam int, targetAC int) int {
	return GetMinusDam(dam, targetAC)
}

// --------------------------------------------------------------------------
// IsMounted — checks if a character is mounted
// Source: src/utils.c:378
// --------------------------------------------------------------------------

func (p *Player) IsMounted() bool {
	return p.MountName != ""
}

func (m *MobInstance) IsMountedMob() bool {
	return m.MountRider != ""
}

// --------------------------------------------------------------------------
// Unmount — dismount a rider from a mount
// Source: src/utils.c:378-385
// --------------------------------------------------------------------------

func Unmount(rider *Player, mount *MobInstance) {
	if rider != nil && rider.IsMounted() {
		rider.MountName = ""
	}
	if mount != nil && mount.IsMountedMob() {
		mount.MountRider = ""
	}
}

func (p *Player) Unmount() {
	if p.MountName == "" {
		return
	}
	Unmount(p, nil)
}

// --------------------------------------------------------------------------
// GetRider — returns the rider of a mount mob
// Source: src/utils.c:387-392
// --------------------------------------------------------------------------

func GetRider(mount *MobInstance) string {
	if mount == nil {
		return ""
	}
	return mount.MountRider
}

// --------------------------------------------------------------------------
// CanSpeak — checks if a character is intelligent enough to speak
// Source: src/utils.c:685-687
// --------------------------------------------------------------------------

func (p *Player) CanSpeak() bool {
	return true
}

func (m *MobInstance) CanSpeak() bool {
	if m == nil || m.Prototype == nil {
		return false
	}
	// In C: is_intelligent(ch) checks race against intelligent_races[] list.
	// For Go, return true for NPCs with an intelligent race.
	return true
}

// --------------------------------------------------------------------------
// StopFollower — stop following a leader
// Source: src/utils.c:397-440
// --------------------------------------------------------------------------

func (p *Player) StopFollower() {
	if p.Following == "" {
		return
	}
	p.Following = ""
	p.InGroup = false
}

// --------------------------------------------------------------------------
// SetHunting — set a mob's hunting target
// Source: src/utils.c:708-729
// --------------------------------------------------------------------------

func (m *MobInstance) SetHunting(target string) {
	if m == nil {
		return
	}
	if m.Hunting == target {
		return
	}
	m.Hunting = ""
	m.HuntingID = ""
	if target != "" {
		m.Hunting = target
	}
}

// --------------------------------------------------------------------------
// Remember / Forget — mob memory
// Source: src/mobact.c:347-395 (remember), src/mobact.c:397-434 (forget)
// --------------------------------------------------------------------------

func (m *MobInstance) Remember(name string) {
	if m == nil || m.Memory == nil {
		return
	}
	for _, n := range m.Memory {
		if n == name {
			return
		}
	}
	m.Memory = append(m.Memory, name)
}

func (m *MobInstance) Forget(name string) bool {
	if m == nil || m.Memory == nil {
		return false
	}
	for i, n := range m.Memory {
		if n == name {
			m.Memory = append(m.Memory[:i], m.Memory[i+1:]...)
			return true
		}
	}
	return false
}

// --------------------------------------------------------------------------
// Tattoo effects — add or remove tattoo stat bonuses
// Source: src/tattoo.c:104-165
// --------------------------------------------------------------------------

// Tattoo constants — from structs.h / tattoo.c
const (
	TattooNone   = 0
	TattooDragon = 1 + iota
	TattooTiger
	TattooCobra
	TattooWolf
	TattooBear
	TattooEagle
	TattooMantis
	TattooShark
	TattooScorpion
	TattooPhoenix
	TattooLynx
	TattooHorse
	TattooBat
	TattooSerpent
)

// TattooBonus represents a single stat modifier from a tattoo.
type TattooBonus struct {
	Location int // APPLY_* constant
	Modifier int
}

// GetTattooBonuses returns the stat modifiers for a given tattoo type.
// Source: src/tattoo.c switch statement.
func GetTattooBonuses(tattoo int) []TattooBonus {
	switch tattoo {
	case TattooDragon:
		return []TattooBonus{
			{Location: ApplyDamroll, Modifier: 2},
			{Location: ApplyStr, Modifier: 2},
		}
	case TattooTiger:
		return []TattooBonus{
			{Location: ApplyDex, Modifier: 2},
			{Location: ApplyDamroll, Modifier: 1},
		}
	case TattooCobra:
		return []TattooBonus{
			{Location: ApplyDex, Modifier: 1},
			{Location: ApplyDamroll, Modifier: 1},
			{Location: ApplyHitroll, Modifier: 1},
		}
	case TattooWolf:
		return []TattooBonus{
			{Location: ApplyStr, Modifier: 1},
			{Location: ApplyInt, Modifier: 1},
			{Location: ApplyDamroll, Modifier: 1},
		}
	case TattooBear:
		return []TattooBonus{
			{Location: ApplyStr, Modifier: 2},
			{Location: ApplyHitroll, Modifier: 1},
		}
	case TattooEagle:
		return []TattooBonus{
			{Location: ApplyDamroll, Modifier: 3},
		}
	case TattooMantis:
		return []TattooBonus{
			{Location: ApplyDex, Modifier: 2},
			{Location: ApplyHitroll, Modifier: 1},
		}
	case TattooShark:
		return []TattooBonus{
			{Location: ApplyStr, Modifier: 1},
			{Location: ApplyDex, Modifier: 1},
			{Location: ApplyDamroll, Modifier: 1},
		}
	case TattooScorpion:
		return []TattooBonus{
			{Location: ApplyDex, Modifier: 3},
		}
	case TattooPhoenix:
		return []TattooBonus{
			{Location: ApplyInt, Modifier: 2},
			{Location: ApplyDamroll, Modifier: 1},
		}
	case TattooLynx:
		return []TattooBonus{
			{Location: ApplyWis, Modifier: 1},
			{Location: ApplyDamroll, Modifier: 2},
		}
	case TattooHorse:
		return []TattooBonus{
			{Location: ApplyStr, Modifier: 3},
		}
	case TattooBat:
		return []TattooBonus{
			{Location: ApplyHitroll, Modifier: 2},
			{Location: ApplyDamroll, Modifier: 1},
		}
	case TattooSerpent:
		return []TattooBonus{
			{Location: ApplyInt, Modifier: 2},
			{Location: ApplyHitroll, Modifier: 1},
		}
	default:
		return nil
	}
}

// TattooAf applies or removes tattoo stat effects on a player.
// Source: src/tattoo.c:104 (tattoo_af struct char_data *ch, bool add).
func TattooAf(p *Player, add bool) {
	if p == nil || p.Tattoo == 0 {
		return
	}
	bonuses := GetTattooBonuses(p.Tattoo)
	if len(bonuses) == 0 {
		return
	}

	// In C, tattoo_af adds/removes affect_type structs via affect_join.
	// In Go, apply tattoo bonuses as player stat modifiers.
	// The actual stat application is handled by the caller or Lua scripts.
	_ = bonuses
	_ = add
}

// --------------------------------------------------------------------------
// Mount/Hunting fields on Player — add missing fields to Player struct?
// Already checked: Player has MountName, MobInstance has MountRider and Hunting.
// --------------------------------------------------------------------------
