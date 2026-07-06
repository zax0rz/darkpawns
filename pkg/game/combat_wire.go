package game

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// WireCombatCallbacks builds a combat.GameCallbacks populated with game-layer
// lookups and simple mutators. Hooks that have no straightforward production
// implementation remain nil so that behavior is unchanged from the legacy
// package-level variables they replace.
func (w *World) WireCombatCallbacks() *combat.GameCallbacks {
	cb := &combat.GameCallbacks{}

	// -------------------------------------------------------------------------
	// Character identity
	// -------------------------------------------------------------------------
	cb.GetRace = func(name string) int {
		if p, ok := w.GetPlayer(name); ok {
			return p.GetRace()
		}
		if m := w.GetMobByName(name); m != nil && m.Prototype != nil {
			return m.Prototype.Race
		}
		return 0
	}

	cb.GetAlignment = func(name string) int {
		if p, ok := w.GetPlayer(name); ok {
			return p.GetAlignment()
		}
		if m := w.GetMobByName(name); m != nil {
			return m.GetAlignment()
		}
		return 0
	}

	cb.SetAlignment = func(name string, val int) {
		if p, ok := w.GetPlayer(name); ok {
			p.SetAlignment(val)
		}
	}

	cb.GetSex = func(name string) int {
		if p, ok := w.GetPlayer(name); ok {
			return p.GetSex()
		}
		if m := w.GetMobByName(name); m != nil {
			return m.GetSex()
		}
		return 0
	}

	cb.GetSkill = func(name string, skillNum int) int {
		if p, ok := w.GetPlayer(name); ok {
			return p.GetSkill(name)
		}
		return 0
	}

	// -------------------------------------------------------------------------
	// Affects
	// -------------------------------------------------------------------------
	cb.HasAffect = func(name string, aff int) bool {
		if p, ok := w.GetPlayer(name); ok {
			return p.IsAffected(aff)
		}
		if m := w.GetMobByName(name); m != nil {
			return m.HasAffect(aff)
		}
		return false
	}

	cb.HasAffectStr = func(name string, aff string) bool {
		bit := affectStringToBit(aff)
		if bit < 0 {
			return false
		}
		if p, ok := w.GetPlayer(name); ok {
			return p.IsAffected(bit)
		}
		if m := w.GetMobByName(name); m != nil {
			return m.HasAffect(bit)
		}
		return false
	}

	cb.RemoveAffect = func(name string, skillNum int) {
		if p, ok := w.GetPlayer(name); ok {
			p.RemoveAffectBySpell(skillNum)
		}
		if m := w.GetMobByName(name); m != nil {
			m.RemoveAffectBySpell(skillNum)
		}
	}

	// -------------------------------------------------------------------------
	// Player/Mob/Room flags
	// -------------------------------------------------------------------------
	cb.HasPlrFlag = func(name string, flag string) bool {
		p, ok := w.GetPlayer(name)
		if !ok {
			return false
		}
		return hasPlrFlag(p, flag)
	}

	cb.SetPlrFlag = func(name string) bool {
		p, ok := w.GetPlayer(name)
		if !ok {
			return false
		}
		p.SetPlrFlag(PlrOutlaw, true)
		return true
	}

	cb.HasPrfFlag = func(name string, flag string) bool {
		p, ok := w.GetPlayer(name)
		if !ok {
			return false
		}
		bit, ok := prfFlagMap[strings.ToLower(flag)]
		if !ok {
			return false
		}
		return p.GetFlags()&(1<<uint(bit)) != 0
	}

	cb.HasMobFlag = func(name string, flag string) bool {
		m := w.GetMobByName(name)
		if m == nil {
			return false
		}
		return m.HasFlag(flag)
	}

	cb.HasMobVNum = func(name string, vnum int) bool {
		m := w.GetMobByName(name)
		if m == nil || m.Prototype == nil {
			return false
		}
		return m.Prototype.VNum == vnum
	}

	cb.HasRoomFlag = func(roomVNum int, flag string) bool {
		room, ok := w.GetRoom(roomVNum)
		if !ok {
			return false
		}
		return hasRoomFlag(room, flag)
	}

	cb.HasScriptFlag = func(name string, flag string) bool {
		m := w.GetMobByName(name)
		if m == nil {
			return false
		}
		return m.HasScript(strings.ToLower(strings.TrimPrefix(flag, "MS_")))
	}

	// -------------------------------------------------------------------------
	// Equipment & mounts
	// -------------------------------------------------------------------------
	cb.IsMounted = func(name string) bool {
		if p, ok := w.GetPlayer(name); ok {
			return p.IsMounted()
		}
		return false
	}

	cb.Dismount = func(name string) {
		p, ok := w.GetPlayer(name)
		if !ok || !p.IsMounted() {
			return
		}
		roomVNum := p.GetRoom()
		for _, m := range w.GetMobsInRoom(roomVNum) {
			if m.GetMountRider() == name {
				m.SetMountRider("")
				break
			}
		}
		p.SetAffect(affMounted, false)
		p.SetFollowing("")
	}

	cb.Unmount = func(name string) {
		if p, ok := w.GetPlayer(name); ok {
			p.Unmount()
		}
	}

	// -------------------------------------------------------------------------
	// Room navigation
	// -------------------------------------------------------------------------
	cb.GetAdjacentRoom = func(roomVNum, door int) int {
		room, ok := w.GetRoom(roomVNum)
		if !ok {
			return -1
		}
		dirs := []string{"north", "east", "south", "west", "up", "down"}
		if door < 0 || door >= len(dirs) {
			return -1
		}
		exit, ok := room.Exits[dirs[door]]
		if !ok {
			return -1
		}
		return exit.ToRoom
	}

	// -------------------------------------------------------------------------
	// Kill/Death/Stats
	// -------------------------------------------------------------------------
	cb.GainExp = func(name string, amount int) {
		if p, ok := w.GetPlayer(name); ok {
			p.AddExp(amount)
		}
	}

	cb.GetExp = func(name string) int {
		if p, ok := w.GetPlayer(name); ok {
			return p.GetExp()
		}
		return 0
	}

	cb.GetKills = func(name string) int64 {
		if p, ok := w.GetPlayer(name); ok {
			return int64(p.Kills)
		}
		return 0
	}

	cb.SetKills = func(name string, kills int64) {
		if p, ok := w.GetPlayer(name); ok {
			p.Kills = int(kills)
		}
	}

	cb.GetDeaths = func(name string) int64 {
		if p, ok := w.GetPlayer(name); ok {
			return int64(p.Deaths)
		}
		return 0
	}

	cb.SetDeaths = func(name string, deaths int64) {
		if p, ok := w.GetPlayer(name); ok {
			p.Deaths = int(deaths)
		}
	}

	cb.SetLastDeath = func(name string, t int64) {
		if p, ok := w.GetPlayer(name); ok {
			p.SetLastDeath(t)
		}
	}

	cb.GetPks = func(name string) int64 {
		if p, ok := w.GetPlayer(name); ok {
			return int64(p.PKs)
		}
		return 0
	}

	cb.SetPks = func(name string, pks int64) {
		if p, ok := w.GetPlayer(name); ok {
			p.PKs = int(pks)
		}
	}

	cb.GetConstitution = func(name string) int {
		if p, ok := w.GetPlayer(name); ok {
			return p.GetCon()
		}
		return 0
	}

	cb.SetConstitution = func(name string, val int) {
		if p, ok := w.GetPlayer(name); ok {
			p.mu.Lock()
			p.Stats.Con = val
			p.mu.Unlock()
		}
	}

	// -------------------------------------------------------------------------
	// Corpse & extraction — left nil for PR2. These are invoked by the legacy
	// combat.RawKill path; the active engine path uses CombatEngine.DeathFunc,
	// which already routes through World.HandleDeath. Wiring these in PR2 would
	// change behavior because the legacy hooks were nil in production. They will
	// be wired to game-layer helpers in PR3.
	// -------------------------------------------------------------------------

	return cb
}

// affectStringToBit maps the AFF_* string constants used by the combat package
// to the C-style bit positions defined in affects_constants.go.
func affectStringToBit(aff string) int {
	switch strings.ToUpper(aff) {
	case "AFF_GROUP":
		return affGroup
	case "AFF_WEREWOLF":
		return affWerewolf
	case "AFF_VAMPIRE":
		return affVampire
	case "AFF_FLESH_ALTER", "AFF_FLESHALTER":
		return affFleshAlter
	case "AFF_HASTE":
		return affHaste
	case "AFF_SLOW":
		return affSlow
	default:
		return -1
	}
}

// prfFlagMap maps PRF flag names to their bit positions.
var prfFlagMap = map[string]int{
	"autogold":  PrfAutoGold,
	"autosplit": PrfAutoSplit,
	"autoloot":  PrfAutoLoot,
	"summon":    PrfSummonable,
	"nohassle":  PrfNohassle,
	"brief":     PrfBrief,
	"compact":   PrfCompact,
	"notell":    PrfNotell,
	"noauction": PrfNoAuctions,
	"deaf":      PrfDeaf,
	"nogossip":  PrfNoGossip,
	"nogratz":   PrfNoGratz,
	"nowiz":     PrfNowiz,
	"quest":     PrfQuest,
	"roomflags": PrfRoomFlags,
	"norepeat":  PrfNoRepeat,
	"holylight": PrfHolyLight,
	"nonewbie":  PrfNoNewbie,
	"noctell":   PrfNoCTell,
	"nobroad":   PrfNoBroad,
}
