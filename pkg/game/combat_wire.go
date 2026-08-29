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

	cb.GetHP = func(name string) int {
		if p, ok := w.GetPlayer(name); ok {
			return p.GetHP()
		}
		if m := w.GetMobByName(name); m != nil {
			return m.GetHP()
		}
		return 1
	}

	cb.GetLevel = func(name string) int {
		if p, ok := w.GetPlayer(name); ok {
			return p.GetLevel()
		}
		if m := w.GetMobByName(name); m != nil {
			return m.GetLevel()
		}
		return 0
	}

	cb.IsNPC = func(name string) bool {
		if _, ok := w.GetPlayer(name); ok {
			return false
		}
		return w.GetMobByName(name) != nil
	}

	cb.GetSkill = func(name string, skillNum int) int {
		if p, ok := w.GetPlayer(name); ok {
			return p.GetSkill(combatSkillName(skillNum))
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
			// fight.c passes AFF_HIDE here and clears the bitmask directly;
			// RemoveAffectBySpell alone only removes timed spell records.
			p.RemoveAffectBit(skillNum)
			p.RemoveAffectBySpell(skillNum)
		}
		if m := w.GetMobByName(name); m != nil {
			m.ClearAffect(skillNum)
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

	cb.MobHasJailGuardSpec = func(name string) bool {
		m := w.GetMobByName(name)
		if m == nil || m.Prototype == nil {
			return false
		}
		switch MobSpecAssign[m.Prototype.VNum] {
		case "take_to_jail", "wall_guard_ns":
			return true
		default:
			return false
		}
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

	cb.GetRoomCombatants = func(roomVNum int) []combat.Combatant {
		players := w.GetPlayersInRoom(roomVNum)
		mobs := w.GetMobsInRoom(roomVNum)
		chars := make([]combat.Combatant, 0, len(players)+len(mobs))
		for _, p := range players {
			chars = append(chars, p)
		}
		for _, m := range mobs {
			chars = append(chars, m)
		}
		return chars
	}

	cb.GetFollowing = func(name string) string {
		if p, ok := w.GetPlayer(name); ok {
			return p.GetFollowing()
		}
		for _, m := range w.GetAllMobs() {
			if m.GetName() == name {
				return m.GetFollowing()
			}
		}
		return ""
	}

	cb.JailGuardSubdue = func(guardName, victimName string) bool {
		victim, ok := w.GetPlayer(victimName)
		if !ok {
			return false
		}
		var guard *MobInstance
		for _, m := range w.GetMobsInRoom(victim.GetRoom()) {
			if m.GetName() == guardName {
				guard = m
				break
			}
		}
		if guard == nil {
			return false
		}

		// C stops the victim's combat before the jail messages. If the victim's
		// opponent reciprocates the target, stop that side as well; the combat
		// engine also removes its pair after this callback returns.
		if victim.IsFighting() {
			fightingName := victim.GetFighting()
			if opponent := cityguardCombatantByName(w, victim.GetRoom(), fightingName); opponent != nil && opponent.GetFighting() == victimName {
				switch opponent := opponent.(type) {
				case *Player:
					opponent.StopFighting()
				case *MobInstance:
					opponent.StopFighting()
				}
			}
			victim.StopFighting()
		}
		victim.SetHP(1)
		if victim.IsMounted() {
			victim.Unmount()
		}
		if guard.HasMobFlag(MobFlagMemory) || guard.HasFlag("MOB_MEMORY") {
			guard.Forget(victimName)
		}
		if guard.GetHunting() == victimName {
			guard.ClearHunting()
		}

		Act(w, true, guard, victim, nil, nil,
			"$n grabs $N by the collar, and quickly beats $M into submission.\r\nJerking $M to $S feet, $n carts $N off to jail.", "", ToNotVict)
		Act(w, true, guard, victim, nil, nil,
			"$n grabs you by the collar and quickly beats you into submission.", "", ToVict)
		sendToChar(victim, "Jerking you to your feet, he carts you off to jail...")
		victim.SetRoom(8118)
		w.lookAtRoom(victim, false)
		jailTimer := victim.GetLevel() / 2
		if jailTimer < 2 {
			jailTimer = 2
		}
		victim.JailTimer = jailTimer
		return true
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

	// GetWeaponInfo returns the wielded weapon's message attack-type for the
	// named attacker — fight.c:1792-1806 one_hit w_type derivation. The wType
	// return is the 0-based OFFSET (C's GET_OBJ_VAL(wielded,3), e.g. 11 for a
	// piercing dagger, 3 for slash) that SendWeaponMessage adds TYPE_HIT to.
	// damDice/damSize/isBlessed are unused by the current message path (left
	// zero) — only wType is consumed by performOneHit. For mobs, wType is the
	// parsed BareHandAttack field copied by read_mobile into mob_specials, which
	// is C's attack_type fallback when no weapon is wielded.
	cb.GetWeaponInfo = func(name string) (wType, damDice, damSize int, isBlessed bool) {
		if p, ok := w.GetPlayer(name); ok && p.Equipment != nil {
			if weapon, wielded := p.Equipment.GetItemInSlot(SlotWield); wielded && weapon != nil && weapon.Prototype != nil {
				// Values[3] holds the weapon attack type (pierce=11, slash=3,
				// bludgeon=5, …) — the offset into attack_hit_text, NOT a
				// TYPE_* constant. fight.c:1795 w_type = val3 + TYPE_HIT, and
				// SendWeaponMessage performs that +TYPE_HIT itself.
				return weapon.Prototype.Values[3], 0, 0, false
			}
			return 0, 0, 0, false // barehand → "hit"
		}
		if m := w.GetMobByName(name); m != nil && m.Prototype != nil {
			return m.Prototype.BareHandAttack, 0, 0, false
		}
		return 0, 0, 0, false // mob / unknown → "hit"
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

	// fight.c:1457 stop_follower(victim) when the attacker is the victim's
	// master — the game layer owns the act audiences of the charm trio.
	cb.StopFollowerOfMaster = func(victimName, masterName string) {
		if p, ok := w.GetPlayer(victimName); ok {
			StopFollower(w, p)
			return
		}
		for _, m := range w.GetAllMobs() {
			if m.GetName() == victimName {
				StopFollowerMob(w, m)
				return
			}
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
