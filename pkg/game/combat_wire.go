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

	cb.GetRaceHate = func(name string, index int) int {
		if index < 0 || index >= 5 {
			return -1
		}
		if p, ok := w.GetPlayer(name); ok {
			p.mu.RLock()
			defer p.mu.RUnlock()
			return p.RaceHates[index]
		}
		if m := w.GetMobByName(name); m != nil {
			m.mu.RLock()
			defer m.mu.RUnlock()
			return m.RaceHates[index]
		}
		return -1
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
	// Corpse & extraction — wired to game-layer helpers for the legacy RawKill
	// path. The active engine path still uses CombatEngine.DeathFunc.
	// -------------------------------------------------------------------------
	cb.MakeCorpse = func(victim string, attackType int) {
		if p, ok := w.GetPlayer(victim); ok {
			_ = MakeCorpse(p)
		}
	}

	cb.ExtractChar = func(name string) {
		if p, ok := w.GetPlayer(name); ok {
			ExtractChar(p)
		}
	}

	cb.RunDeathScript = func(killer, victim string, roomVNum int) {
		w.FireMobDeathScript(victim, killer, roomVNum)
	}

	// -------------------------------------------------------------------------
	// Group/Party
	// -------------------------------------------------------------------------
	cb.GetFollowersInRoom = func(name string, roomVNum int) int {
		return len(w.GetFollowersInRoom(name, roomVNum))
	}

	cb.GetMasterInRoom = func(name string, roomVNum int) bool {
		p, ok := w.GetPlayer(name)
		if !ok {
			return false
		}
		masterName := p.GetFollowing()
		if masterName == "" {
			return false
		}
		master, ok := w.GetPlayer(masterName)
		if !ok {
			return false
		}
		return master.GetRoom() == roomVNum
	}

	cb.GetFellowFollowersInRoom = func(name string, roomVNum int) bool {
		p, ok := w.GetPlayer(name)
		if !ok {
			return false
		}
		leaderName := p.GetFollowing()
		if leaderName == "" {
			return false
		}
		for _, follower := range w.GetFollowers(leaderName) {
			if follower.GetName() != name && follower.GetRoom() == roomVNum {
				return true
			}
		}
		return false
	}

	cb.CountGroupMembers = func(leaderName string, roomVNum int) int {
		members := w.GetGroupMembers(leaderName)
		count := 0
		for _, m := range members {
			if m.GetRoom() == roomVNum {
				count++
			}
		}
		return count
	}

	cb.ApplyToGroupMembers = func(leaderName string, roomVNum int, fn func(name string)) {
		for _, m := range w.GetGroupMembers(leaderName) {
			if m.GetRoom() == roomVNum {
				fn(m.GetName())
			}
		}
	}

	// -------------------------------------------------------------------------
	// Gold
	// -------------------------------------------------------------------------
	cb.GetGold = func(name string) int {
		if p, ok := w.GetPlayer(name); ok {
			return p.GetGold()
		}
		return 0
	}

	cb.SetGold = func(name string, gold int) {
		if p, ok := w.GetPlayer(name); ok {
			p.SetGold(gold)
		}
	}

	// -------------------------------------------------------------------------
	// Items
	// -------------------------------------------------------------------------
	cb.JunkInventoryItems = func(chName string) {
		p, ok := w.GetPlayer(chName)
		if !ok {
			return
		}
		w.junkCheapItems(p)
	}

	// -------------------------------------------------------------------------
	// Commands
	// -------------------------------------------------------------------------
	cb.PerformCommand = func(chName, cmd string) {
		p, ok := w.GetPlayer(chName)
		if !ok {
			return
		}
		if w.CommandExecFunc != nil {
			w.CommandExecFunc(p, cmd)
		}
	}

	// -------------------------------------------------------------------------
	// Flee/Retreat — wired by the session layer via Manager.SetFleeHooks because
	// they need access to player sessions. Leave nil here so SetFleeHooks can
	// install its callbacks without conflict.
	// -------------------------------------------------------------------------
	cb.GetWimpyLev = func(name string) int {
		if p, ok := w.GetPlayer(name); ok {
			return p.WimpLevel
		}
		return 0
	}

	// -------------------------------------------------------------------------
	// World
	// -------------------------------------------------------------------------
	cb.IncreaseMaxStat = func(name string, stat string) {
		p, ok := w.GetPlayer(name)
		if !ok {
			return
		}
		switch strings.ToLower(stat) {
		case "hp", "hit", "health":
			p.SetMaxHP(p.GetMaxHP() + 1)
		case "mana":
			p.SetMaxMana(p.GetMaxMana() + 1)
		case "move", "movement":
			p.SetMaxMove(p.GetMaxMove() + 1)
		}
	}

	cb.HealAllPlayers = func() {
		for _, p := range w.AllPlayers() {
			p.SetHP(p.GetMaxHP())
		}
	}

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
