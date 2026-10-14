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
	ApplyNone         = 0
	ApplyStr          = 1
	ApplyDex          = 2
	ApplyInt          = 3
	ApplyWis          = 4
	ApplyCon          = 5
	ApplyCha          = 6
	ApplyMana         = 12
	ApplyHit          = 13
	ApplyMove         = 14
	ApplyAC           = 17
	ApplyHitroll      = 18
	ApplyDamroll      = 19
	ApplySavingPara   = 20
	ApplySavingRod    = 21
	ApplySavingPetri  = 22
	ApplySavingBreath = 23
	ApplySavingSpell  = 24
	ApplyRaceHate     = 25 // Special: src/handler.c:205-238 slot management, not a numeric modifier.
	ApplySpell        = 29
)

// APPLY_RACE_HATE slot mutation is implemented in race_hate.go and runs from
// both player and mobile equipment recomputation. It is a slot operation, not
// an ordinary numeric APPLY_* modifier (src/handler.c:205-238).

// --------------------------------------------------------------------------
// GetMinusDam — damage reduction based on target AC
// Source: src/fight.c:1722
// --------------------------------------------------------------------------

// --------------------------------------------------------------------------
// IsMounted — checks if a character is mounted
// Source: src/utils.c:378
// --------------------------------------------------------------------------

func (p *Player) IsMounted() bool {
	return p.MountName != "" || p.IsAffected(affMounted)
}

func (m *MobInstance) IsMountedMob() bool {
	return m.MountRider != ""
}

// --------------------------------------------------------------------------
// Unmount — dismount a rider from a mount
// Source: src/utils.c:378-385
// --------------------------------------------------------------------------

func Unmount(rider *Player, mount *MobInstance) {
	clearMountedAffects(rider, mount)
}

func (p *Player) Unmount() {
	if p == nil || !p.IsMounted() {
		return
	}
	Unmount(p, nil)
}

// clearMountedAffects mirrors utils.c:378-385. C clears only AFF_MOUNT on
// the supplied rider and mount; master/follower links are separate state.
// MountName is a player-side display cache, so clear it with the rider affect.
func clearMountedAffects(rider Actor, mount *MobInstance) {
	switch actor := rider.(type) {
	case *Player:
		if actor != nil {
			actor.SetAffect(affMounted, false)
			actor.MountName = ""
		}
	case *MobInstance:
		if actor != nil {
			actor.RemoveAffected(affMounted)
		}
	}
	if mount != nil && mount.IsAffected(affMounted) {
		mount.RemoveAffected(affMounted)
	}
}

// --------------------------------------------------------------------------
// GetRider — returns the rider of a mount mob
// Source: src/utils.c:387-392
// --------------------------------------------------------------------------

// --------------------------------------------------------------------------
// CanSpeak — checks if a character is intelligent enough to speak
// Source: src/utils.c:685-687
// --------------------------------------------------------------------------

func (p *Player) CanSpeak() bool {
	return true
}

func (m *MobInstance) CanSpeak() bool {
	if m == nil || m.Proto() == nil {
		return false
	}
	// C is_intelligent() compares the mobile race against the literal
	// intelligent_races[] list (src/constants.c:353-373). This is deliberately
	// not the player-race bitvector in pkg/game/constants.go.
	switch m.Proto().Race {
	case 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 18, 19, 20, 27, 28, 29, 30:
		return true
	default:
		return false
	}
}

// --------------------------------------------------------------------------
// StopFollower — stop following a leader
// Source: src/utils.c:397-440
// --------------------------------------------------------------------------

func (p *Player) StopFollower() {
	if p.GetFollowing() == "" {
		return
	}
	p.SetFollowing("")
	p.SetInGroup(false)
}

// --------------------------------------------------------------------------
// SetHunting — set a mob's hunting target
// Source: src/utils.c:708-729
// --------------------------------------------------------------------------

func (m *MobInstance) Remember(name string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Memory == nil {
		return
	}
	for _, n := range m.Memory {
		if n == name {
			return
		}
	}
	m.Memory = append(m.Memory, name)
}

func (m *MobInstance) SetHunting(target string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Hunting == target {
		return
	}
	m.Hunting = ""
	m.HuntingID = ""
	m.HuntingMobID = 0
	if target != "" {
		m.Hunting = target
	}
}

// --------------------------------------------------------------------------
// Remember / Forget — mob memory
// Source: src/mobact.c:347-395 (remember), src/mobact.c:397-434 (forget)
// --------------------------------------------------------------------------

func (m *MobInstance) Forget(name string) bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.Memory == nil {
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
	TattooDragon = 1
	TattooTribal = 2
	TattooSkull  = 3
	TattooTiger  = 4
	TattooWorm   = 5
	TattooEye    = 6
	TattooSwords = 7
	TattooEagle  = 8
	TattooHeart  = 9
	TattooStar   = 10
	TattooShip   = 11
	TattooSpider = 12
	TattooJyhadi = 13
	TattooMom    = 14
	TattooAngel  = 15
	TattooFox    = 16
	TattooOwl    = 17
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
	case TattooTribal:
		return []TattooBonus{
			{Location: ApplyDex, Modifier: 1},
		}
	case TattooWorm:
		return []TattooBonus{
			{Location: ApplyDamroll, Modifier: 2},
		}
	case TattooSwords:
		return []TattooBonus{
			{Location: ApplyDamroll, Modifier: 1},
			{Location: ApplyHitroll, Modifier: 1},
		}
	case TattooEagle:
		return []TattooBonus{
			{Location: ApplyMove, Modifier: 20},
		}
	case TattooHeart:
		return []TattooBonus{
			{Location: ApplyHit, Modifier: 20},
		}
	case TattooStar:
		return []TattooBonus{
			{Location: ApplyMana, Modifier: 20},
		}
	case TattooSpider:
		return []TattooBonus{
			{Location: ApplyDex, Modifier: 3},
		}
	case TattooJyhadi:
		return []TattooBonus{
			{Location: ApplyDamroll, Modifier: 1},
		}
	case TattooMom:
		return []TattooBonus{
			{Location: ApplyWis, Modifier: 3},
		}
	case TattooTiger:
		return []TattooBonus{
			{Location: ApplyDex, Modifier: 1},
			{Location: ApplyMove, Modifier: 10},
		}
	case TattooFox:
		return []TattooBonus{
			{Location: ApplyInt, Modifier: 1},
		}
	case TattooOwl:
		return []TattooBonus{
			{Location: ApplyWis, Modifier: 1},
		}
	default:
		return nil
	}
}

// TattooAf applies or removes tattoo stat effects on a player.
// Source: src/tattoo.c:104 (tattoo_af struct char_data *ch, bool add).
// C calls affect_modify without bounds (src/tattoo.c:193-195). Abilities
// change only aff_abils; the next affect_total rebuilds from base and Tattoo.
func TattooAf(p *Player, add bool) {
	if p == nil || p.Tattoo == 0 {
		return
	}
	bonuses := GetTattooBonuses(p.Tattoo)
	if len(bonuses) == 0 {
		return
	}

	sign := 1
	if !add {
		sign = -1
	}

	p.mu.Lock()
	stats := p.effectiveAttributesLocked()
	defer func() { p.effectiveAttributes = &stats; p.mu.Unlock(); p.refreshAttributeCapacity() }()
	for _, b := range bonuses {
		delta := b.Modifier * sign
		switch b.Location {
		case ApplyStr:
			stats.Str += delta
		case ApplyDex:
			stats.Dex += delta
		case ApplyInt:
			stats.Int += delta
		case ApplyWis:
			stats.Wis += delta
		case ApplyCon:
			stats.Con += delta
		case ApplyMana:
			p.MaxMana += delta
		case ApplyHit:
			p.MaxHealth += delta
		case ApplyMove:
			p.MaxMove += delta
		case ApplyHitroll:
			p.Hitroll += delta
		case ApplyDamroll:
			p.Damroll += delta
		}
	}
}

// --------------------------------------------------------------------------
// Mount/Hunting fields on Player — add missing fields to Player struct?
// Already checked: Player has MountName, MobInstance has MountRider and Hunting.
// --------------------------------------------------------------------------
