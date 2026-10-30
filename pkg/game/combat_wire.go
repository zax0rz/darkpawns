package game

import (
	"sort"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/parser"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// WireCombatCallbacks builds a combat.GameCallbacks populated with game-layer
// lookups and simple mutators. Hooks that have no straightforward production
// implementation remain nil so that behavior is unchanged from the legacy
// package-level variables they replace.
func (w *World) WireCombatCallbacks() *combat.GameCallbacks {
	cb := &combat.GameCallbacks{}
	pre := w.damageBeforeMessageCallbacks()
	cb.DamageNewbieExp = pre.DamageNewbieExp
	cb.StopFighting = pre.StopFighting
	cb.NeutralRescue = pre.NeutralRescue
	cb.RangedHunt = func(attacker, defender combat.Combatant) {
		if mob, ok := attacker.(*MobInstance); ok && mob.HasMobFlag(MobFlagHunter) {
			// damage()'s hunter arm: set_hunting(ch, victim) with the body
			// the mob just hit (src/fight.c:1453-1455).
			if prey, ok := defender.(Actor); ok {
				w.SetHuntingTarget(mob, prey)
			}
		}
	}
	cb.DoFlee = func(body combat.Combatant) {
		if mob, ok := body.(*MobInstance); ok {
			// fight.c:1599 calls do_flee(victim) directly for a low-HP
			// MOB_WIMPY. That ACMD skips command_interpreter's preamble.
			w.NPCCommandDirect(mob, "flee", "")
		}
	}

	// -------------------------------------------------------------------------
	// Character identity
	// -------------------------------------------------------------------------
	cb.GetRace = func(name combat.Combatant) int {
		if p, ok := combatPlayer(name); ok {
			return p.GetRace()
		}
		if m := combatMob(name); m != nil && m.Proto() != nil {
			return m.Proto().Race
		}
		return 0
	}

	cb.GetRaceHate = func(name combat.Combatant, index int) int {
		if index < 0 || index >= 5 {
			return -1
		}
		if p, ok := combatPlayer(name); ok {
			p.mu.RLock()
			defer p.mu.RUnlock()
			return p.RaceHates[index]
		}
		if m := combatMob(name); m != nil {
			m.mu.RLock()
			defer m.mu.RUnlock()
			return m.RaceHates[index]
		}
		return -1
	}

	cb.GetAlignment = func(name combat.Combatant) int {
		if p, ok := combatPlayer(name); ok {
			return p.GetAlignment()
		}
		if m := combatMob(name); m != nil {
			return m.GetAlignment()
		}
		return 0
	}

	cb.SetAlignment = func(name combat.Combatant, val int) {
		if p, ok := combatPlayer(name); ok {
			p.SetAlignment(val)
		}
	}

	cb.GetSex = func(name combat.Combatant) int {
		if p, ok := combatPlayer(name); ok {
			return p.GetSex()
		}
		if m := combatMob(name); m != nil {
			return m.GetSex()
		}
		return 0
	}

	cb.GetHP = func(name combat.Combatant) int {
		if p, ok := combatPlayer(name); ok {
			return p.GetHP()
		}
		if m := combatMob(name); m != nil {
			return m.GetHP()
		}
		return 1
	}

	cb.GetLevel = func(name combat.Combatant) int {
		if p, ok := combatPlayer(name); ok {
			return p.GetLevel()
		}
		if m := combatMob(name); m != nil {
			return m.GetLevel()
		}
		return 0
	}

	cb.IsNPC = func(name combat.Combatant) bool {
		if _, ok := combatPlayer(name); ok {
			return false
		}
		return combatMob(name) != nil
	}

	cb.GetSkill = func(name combat.Combatant, skillNum int) int {
		if p, ok := combatPlayer(name); ok {
			return p.GetSkill(combatSkillName(skillNum))
		}
		return 0
	}

	// COLOR_LEV(ch): PRF_COLOR_1 counts 1, PRF_COLOR_2 counts 2. NPCs have no
	// player_specials prefs, so C's shared dummy_mob reports 0 (db.c:1281).
	cb.GetColorLevel = func(name combat.Combatant) int {
		if p, ok := combatPlayer(name); ok {
			return colorLevel(p)
		}
		return 0
	}

	// -------------------------------------------------------------------------
	// Affects
	// -------------------------------------------------------------------------
	cb.HasAffect = func(name combat.Combatant, aff int) bool {
		if p, ok := combatPlayer(name); ok {
			return p.IsAffected(aff)
		}
		if m := combatMob(name); m != nil {
			return m.HasAffect(aff)
		}
		return false
	}

	cb.HasAffectStr = func(name combat.Combatant, aff string) bool {
		bit := affectStringToBit(aff)
		if bit < 0 {
			return false
		}
		if p, ok := combatPlayer(name); ok {
			return p.IsAffected(bit)
		}
		if m := combatMob(name); m != nil {
			return m.HasAffect(bit)
		}
		return false
	}

	cb.RemoveAffect = func(name combat.Combatant, skillNum int) {
		if p, ok := combatPlayer(name); ok {
			// fight.c passes AFF_HIDE here and clears the bitmask directly;
			// RemoveAffectBySpell alone only removes timed spell records.
			p.RemoveAffectBit(skillNum)
			p.RemoveAffectBySpell(skillNum)
		}
		if m := combatMob(name); m != nil {
			m.ClearAffect(skillNum)
			m.RemoveAffectBySpell(skillNum)
		}
	}

	// src/fight.c:544-571: remove affects, tattoo, nightbreed, then memories.
	cb.RemoveAllAffects = func(name combat.Combatant) {
		if p, ok := combatPlayer(name); ok {
			p.removeRawKillAffects()
		}
	}
	cb.RemoveTattoo = func(name combat.Combatant) {
		if p, ok := combatPlayer(name); ok {
			removeRawKillTattoo(p)
		}
	}
	cb.ClearNightbreed = func(name combat.Combatant) {
		if p, ok := combatPlayer(name); ok {
			clearRawKillNightbreed(p)
		}
	}
	cb.ForgetVictim = func(body combat.Combatant) { w.forgetRawKillBody(body) }

	// -------------------------------------------------------------------------
	// Player/Mob/Room flags
	// -------------------------------------------------------------------------
	cb.HasPlrFlag = func(name combat.Combatant, flag string) bool {
		p, ok := combatPlayer(name)
		if !ok {
			return false
		}
		return hasPlrFlag(p, flag)
	}

	cb.SetPlrFlag = func(name combat.Combatant) bool {
		p, ok := combatPlayer(name)
		if !ok {
			return false
		}
		p.SetPlrFlag(PlrOutlaw, true)
		return true
	}

	cb.HasPrfFlag = func(name combat.Combatant, flag string) bool {
		p, ok := combatPlayer(name)
		if !ok {
			return false
		}
		bit, ok := prfFlagMap[strings.ToLower(flag)]
		if !ok {
			return false
		}
		return p.GetFlags()&(1<<uint(bit)) != 0
	}

	cb.HasMobFlag = func(name combat.Combatant, flag string) bool {
		m := combatMob(name)
		if m == nil {
			return false
		}
		return m.HasFlag(flag)
	}

	cb.IsShopkeeper = func(name combat.Combatant) bool {
		return IsShopkeeperMob(w, combatMob(name))
	}
	cb.CanSee = w.combatCanSee

	// attitude_loot's brag gate reads can_speak(ch) before drawing anything
	// (src/fight.c:1168-1169). can_speak is is_intelligent(ch)
	// (src/utils.c:685-687), which is true for every player.
	cb.CanSpeak = func(name combat.Combatant) bool {
		if p, ok := combatPlayer(name); ok {
			return p.CanSpeak()
		}
		if m := combatMob(name); m != nil {
			return m.CanSpeak()
		}
		return false
	}

	// The kill brag is do_gen_comm(ch, buf2, 0, SCMD_GOSSIP) (src/fight.c:1246):
	// route the built argument through the direct command body; the actor has
	// no descriptor, so its TO_CHAR echo remains absent.
	cb.BroadChat = func(chName combat.Combatant, msg string) {
		if mob := combatMob(chName); mob != nil {
			w.NPCCommandDirect(mob, "gossip", msg)
		}
	}

	cb.DamageRefused = w.DamageRefused

	cb.HasMobVNum = func(name combat.Combatant, vnum int) bool {
		m := combatMob(name)
		if m == nil || m.Proto() == nil {
			return false
		}
		return m.Proto().VNum == vnum
	}

	cb.MobHasJailGuardSpec = func(name combat.Combatant) bool {
		m := combatMob(name)
		if m == nil || m.Proto() == nil {
			return false
		}
		switch MobSpecAssign[m.Proto().VNum] {
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

	cb.HasScriptFlag = func(name combat.Combatant, flag string) bool {
		m := combatMob(name)
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
		sort.SliceStable(chars, func(i, j int) bool { return combatRoomSequence(chars[i]) > combatRoomSequence(chars[j]) })
		return chars
	}

	cb.GetFollowing = w.combatFollowingBody

	cb.JailGuardSubdue = func(guardName, victimName combat.Combatant) bool {
		victim, ok := combatPlayer(victimName)
		if !ok {
			return false
		}
		guard := combatMob(guardName)
		if guard == nil {
			return false
		}

		// C stops the victim's combat before the jail messages. If the victim's
		// opponent reciprocates the target, stop that side as well; the combat
		// engine also removes its pair after this callback returns.
		if victim.IsFighting() {
			if opponent := victim.GetFightingBody(); opponent != nil && opponent.GetFightingBody() == victim {
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
			w.unmountActor(victim)
		}
		if guard.HasMobFlag(MobFlagMemory) || guard.HasFlag("MOB_MEMORY") {
			guard.Forget(victim.GetName())
		}
		if guard.GetHunting() == victim.GetName() {
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
	cb.IsMounted = func(name combat.Combatant) bool {
		if p, ok := combatPlayer(name); ok {
			return p.IsMounted()
		}
		return false
	}

	cb.Dismount = func(name combat.Combatant) {
		p, ok := combatPlayer(name)
		if !ok || !p.IsMounted() {
			return
		}
		if mount := w.riddenMount(p); mount != nil {
			mount.SetMountRider("")
		}
		p.SetAffect(affMounted, false)
		p.SetFollowing("")
	}

	cb.Unmount = func(name combat.Combatant) {
		if p, ok := combatPlayer(name); ok {
			mount := w.riddenMount(p)
			w.clearMountedPair(p, mount)
			if mount != nil {
				mount.RemoveAffected(affMounted)
			}
		}
	}

	// GetWeaponInfo returns the wielded weapon's message attack-type and blessed
	// status for the named attacker — fight.c:1792-1806 one_hit w_type / ITEM_BLESS.
	// The wType return is the 0-based OFFSET (C's GET_OBJ_VAL(wielded,3), e.g. 11 for a
	// piercing dagger, 3 for slash) that SendWeaponMessage adds TYPE_HIT to.
	// damDice/damSize are unused by the current message path (left zero) — only
	// wType and isBlessed are consumed by performOneHit. For mobs, wType is the
	// parsed BareHandAttack field copied by read_mobile into mob_specials, which
	// is C's attack_type fallback when no weapon is wielded.
	cb.GetWeaponInfo = func(body combat.Combatant) (wType, damDice, damSize int, isBlessed bool) {
		if p, ok := combatPlayer(body); ok && p.Equipment != nil {
			if weapon, wielded := p.Equipment.GetItemInSlot(SlotWield); wielded && weapon != nil && weapon.Prototype != nil && weapon.GetTypeFlag() == ITEM_WEAPON {
				// Values[3] holds the weapon attack type (pierce=11, slash=3,
				// bludgeon=5, …) — the offset into attack_hit_text, NOT a
				// TYPE_* constant. fight.c:1795 w_type = val3 + TYPE_HIT, and
				// SendWeaponMessage performs that +TYPE_HIT itself.
				return weapon.Prototype.Values[3], 0, 0, weapon.HasExtraFlag(0, itemExtraBless)
			}
			return 0, 0, 0, false // barehand → "hit"
		}
		if m, ok := body.(*MobInstance); ok && m != nil {
			if weapon := m.Equipped(mobWearWield); weapon != nil && weapon.Prototype != nil && weapon.GetTypeFlag() == ITEM_WEAPON {
				return weapon.GetValue(3), 0, 0, weapon.HasExtraFlag(0, itemExtraBless)
			}
			if m.Proto() != nil {
				return m.Proto().BareHandAttack, 0, 0, false
			}
		}
		return 0, 0, 0, false // mob / unknown → "hit"
	}

	cb.GetWeaponDescription = func(name combat.Combatant) string {
		if p, ok := combatPlayer(name); ok && p.Equipment != nil {
			if weapon, wielded := p.Equipment.GetItemInSlot(SlotWield); wielded && weapon != nil {
				return weapon.GetShortDesc()
			}
		}
		return ""
	}

	cb.GetDrunk = func(body combat.Combatant) int {
		if p, ok := combatPlayer(body); ok {
			return p.GetCondition(CondDrunk)
		}
		return 0
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
		// CAN_GO (src/utils.h:565-567): the exit exists, leads somewhere,
		// and is not closed. combat.DeathCry walks every direction through
		// this callback, and C's death_cry muffles the cry behind closed
		// doors (src/fight.c:520-527).
		if !ok || exit.ToRoom <= 0 || exit.ExitInfo&parser.ExitClosed != 0 {
			return -1
		}
		return exit.ToRoom
	}

	// -------------------------------------------------------------------------
	// Kill/Death/Stats
	// -------------------------------------------------------------------------
	cb.GainExp = func(name combat.Combatant, amount int) {
		if p, ok := combatPlayer(name); ok {
			p.AddExp(amount)
		}
	}

	cb.GetExp = func(name combat.Combatant) int {
		if p, ok := combatPlayer(name); ok {
			return p.GetExp()
		}
		return 0
	}

	cb.GetKills = func(name combat.Combatant) int64 {
		if p, ok := combatPlayer(name); ok {
			p.mu.RLock()
			defer p.mu.RUnlock()
			return int64(p.Kills)
		}
		if name != nil && name.IsNPC() {
			return w.mobKillCount.Load()
		}
		return 0
	}

	cb.SetKills = func(name combat.Combatant, kills int64) {
		if p, ok := combatPlayer(name); ok {
			p.mu.Lock()
			p.Kills = int(kills)
			p.mu.Unlock()
		} else if name != nil && name.IsNPC() {
			w.mobKillCount.Store(kills)
		}
	}

	cb.GetDeaths = func(name combat.Combatant) int64 {
		if p, ok := combatPlayer(name); ok {
			p.mu.RLock()
			defer p.mu.RUnlock()
			return int64(p.Deaths)
		}
		return 0
	}

	cb.SetDeaths = func(name combat.Combatant, deaths int64) {
		if p, ok := combatPlayer(name); ok {
			p.mu.Lock()
			p.Deaths = int(deaths)
			p.mu.Unlock()
		}
	}

	cb.SetLastDeath = func(name combat.Combatant, t int64) {
		if p, ok := combatPlayer(name); ok {
			p.SetLastDeath(t)
		}
	}

	cb.GetPks = func(name combat.Combatant) int64 {
		if p, ok := combatPlayer(name); ok {
			p.mu.RLock()
			defer p.mu.RUnlock()
			return int64(p.PKs)
		}
		return 0
	}

	cb.SetPks = func(name combat.Combatant, pks int64) {
		if p, ok := combatPlayer(name); ok {
			p.mu.Lock()
			p.PKs = int(pks)
			p.mu.Unlock()
		}
	}

	cb.GetConstitution = func(name combat.Combatant) int {
		if p, ok := combatPlayer(name); ok {
			p.mu.RLock()
			defer p.mu.RUnlock()
			return p.Stats.Con
		}
		return 0
	}

	cb.SetConstitution = func(name combat.Combatant, val int) {
		if p, ok := combatPlayer(name); ok {
			p.mu.Lock()
			p.Stats.Con = val
			p.mu.Unlock()
			p.AffectTotal()
		}
	}

	// -------------------------------------------------------------------------
	// Corpse & extraction — wired to game-layer helpers for the legacy RawKill
	// path. The active engine path still uses CombatEngine.DeathFunc.
	// -------------------------------------------------------------------------
	cb.RawKillNPC = w.RawKillCombatant
	cb.ObserveDeath = func(victim, killer combat.Combatant) {
		if p, ok := combatPlayer(victim); ok && p.GetFlags()&(1<<uint(plrExtractBit)) == 0 {
			w.recordPlayerDeath(p, killer, "unknown")
		}
	}
	// C's death syslog runs inside damage()'s tail (fight.c:1681). Combat has no
	// MudLog of its own, so wire both halves: the level string maps to the game's
	// MudLog type, and the room vnum resolves to the name C prints.
	cb.Log = func(msg, level string, minLevel int, toLog bool) {
		typ := MudlogBrief
		switch level {
		case "NRM":
			typ = MudlogNormal
		case "CMP":
			typ = MudlogComplete
		}
		MudLog(msg, typ, minLevel, toLog)
	}
	cb.RoomName = func(vnum int) string {
		if room, ok := w.GetRoom(vnum); ok {
			return room.Name
		}
		return ""
	}
	cb.MakeCorpse = func(victim combat.Combatant, attackType int) {
		if p, ok := combatPlayer(victim); ok {
			w.makeRawKillBody(p, attackType, false)
		}
	}

	cb.MakeDust = func(name combat.Combatant, attackType int) {
		if p, ok := combatPlayer(name); ok {
			w.makeRawKillBody(p, attackType, true)
		}
	}

	cb.ExtractChar = func(name combat.Combatant) {
		if p, ok := combatPlayer(name); ok {
			if p.GetFlags()&(1<<uint(plrExtractBit)) == 0 {
				w.observePlayerDeath(p)
			}
			ExtractChar(p)
		}
	}

	cb.RunDeathScript = func(killer, victim combat.Combatant, roomVNum int) {
		w.FireMobDeathScript(victim, killer, roomVNum)
	}

	// -------------------------------------------------------------------------
	// Group/Party
	// -------------------------------------------------------------------------
	cb.GetFollowersInRoom = func(name combat.Combatant, roomVNum int) int {
		return len(w.combatFollowersInRoom(name, roomVNum))
	}

	cb.GetMasterInRoom = func(name combat.Combatant, roomVNum int) bool {
		p, ok := combatPlayer(name)
		if !ok {
			return false
		}
		master, ok := combatPlayer(w.combatFollowingBody(p))
		return ok && master.GetRoom() == roomVNum
	}

	cb.GetFellowFollowersInRoom = func(name combat.Combatant, roomVNum int) bool {
		p, ok := combatPlayer(name)
		if !ok {
			return false
		}
		leader := w.combatFollowingBody(p)
		if leader == nil {
			return false
		}
		for _, follower := range w.combatFollowersInRoom(leader, roomVNum) {
			if follower != p {
				return true
			}
		}
		return false
	}

	cb.CountGroupMembers = func(leaderName combat.Combatant, roomVNum int) int {
		members := w.combatGroupMembers(leaderName)
		count := 0
		for _, m := range members {
			if m.GetRoom() == roomVNum {
				count++
			}
		}
		return count
	}

	cb.ApplyToGroupMembers = func(leaderName combat.Combatant, roomVNum int, fn func(name combat.Combatant)) {
		for _, m := range w.combatGroupMembers(leaderName) {
			if m.GetRoom() == roomVNum {
				fn(m)
			}
		}
	}

	// -------------------------------------------------------------------------
	// Gold
	// -------------------------------------------------------------------------
	cb.GetGold = func(name combat.Combatant) int {
		if p, ok := combatPlayer(name); ok {
			return p.GetGold()
		}
		return 0
	}

	cb.SetGold = func(name combat.Combatant, gold int) {
		if p, ok := combatPlayer(name); ok {
			p.SetGold(gold)
		}
	}

	// -------------------------------------------------------------------------
	// Items
	// -------------------------------------------------------------------------
	cb.JunkInventoryItems = func(chName combat.Combatant) {
		p, ok := combatPlayer(chName)
		if !ok {
			return
		}
		w.junkCheapItems(p)
	}

	// -------------------------------------------------------------------------
	// Commands
	// -------------------------------------------------------------------------
	cb.PerformCommand = func(chName combat.Combatant, cmd string) {
		p, ok := combatPlayer(chName)
		if !ok {
			return
		}
		if w.CommandExecFunc != nil {
			w.CommandExecFunc(p, cmd)
		}
	}

	// fight.c:1457 stop_follower(victim) when the attacker is the victim's
	// master — the game layer owns the act audiences of the charm trio.
	cb.StopFollowerOfMaster = func(victimName, masterName combat.Combatant) {
		if p, ok := combatPlayer(victimName); ok {
			StopFollower(w, p)
			return
		}
		if mob := combatMob(victimName); mob != nil {
			StopFollowerMob(w, mob)
		}
	}

	// -------------------------------------------------------------------------
	// Flee/Retreat — wired by the session layer via Manager.SetFleeHooks because
	// they need access to player sessions. Leave nil here so SetFleeHooks can
	// install its callbacks without conflict.
	// -------------------------------------------------------------------------
	cb.GetWimpyLev = func(name combat.Combatant) int {
		if p, ok := combatPlayer(name); ok {
			p.mu.RLock()
			defer p.mu.RUnlock()
			return p.WimpLevel
		}
		return 0
	}

	// -------------------------------------------------------------------------
	// World
	// -------------------------------------------------------------------------
	cb.IncreaseMaxStat = func(name combat.Combatant, stat string) {
		p, ok := combatPlayer(name)
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
		for _, p := range w.playingPlayers() {
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

// combatPlayer/combatMob type-check a retained body; they never search a registry.
func combatPlayer(body combat.Combatant) (*Player, bool) {
	p, ok := body.(*Player)
	return p, ok && p != nil
}
func combatMob(body combat.Combatant) *MobInstance { m, _ := body.(*MobInstance); return m }

func combatRoomSequence(body combat.Combatant) uint64 {
	switch body := body.(type) {
	case *Player:
		return body.GetRoomEntrySequence()
	case *MobInstance:
		return body.GetRoomEntrySequence()
	}
	return 0
}

func (w *World) combatFollowingBody(body combat.Combatant) combat.Combatant {
	var name string
	switch body := body.(type) {
	case *Player:
		body.mu.RLock()
		retained := body.followingBody
		name = body.Following
		body.mu.RUnlock()
		if retained != nil {
			return retained
		}
	case *MobInstance:
		body.mu.RLock()
		retained := body.followingBody
		name = body.Following
		body.mu.RUnlock()
		if retained != nil {
			return retained
		}
	default:
		return nil
	}
	// Existing name-backed legacy edges designate a uniquely held player only.
	if p, ok := w.GetPlayer(name); ok {
		return p
	}
	return nil
}

// FollowingActor returns the body that actor follows, as an Actor: the exact
// retained leader when the follow link carries one — SetFollowingBody keeps
// duplicate NPCs apart (pkg/game/follow.go:395) — and otherwise the uniquely
// named player.
//
// C holds a live ch->master pointer, so a name is never how the leader is
// resolved. Resolving a mobile by name would match a substring over a
// map-ordered slice and could pick a different duplicate from run to run (R3).
// Returns nil when nothing is followed.
func (w *World) FollowingActor(body combat.Combatant) Actor {
	if body == nil {
		return nil
	}
	return asActor(w.combatFollowingBody(body))
}

func (w *World) combatFollowersInRoom(body combat.Combatant, room int) []combat.Combatant {
	var followers []combat.Combatant
	for _, p := range w.GetAllPlayers() {
		if p.GetRoom() == room && w.combatFollowingBody(p) == body {
			followers = append(followers, p)
		}
	}
	for _, m := range w.GetAllMobs() {
		if m.GetRoom() == room && w.combatFollowingBody(m) == body {
			followers = append(followers, m)
		}
	}
	return followers
}

// combatGroupedBody reports a body's AFF_GROUP state for players and mobiles
// alike. C's group code never filters on IS_NPC (src/fight.c:637-657, 708-848).
func combatGroupedBody(body combat.Combatant) bool {
	switch b := body.(type) {
	case *Player:
		return b.IsInGroup()
	case *MobInstance:
		return b.IsInGroup()
	}
	return false
}

// combatGroupMembers returns body's group the way group_gain() builds it
// (src/fight.c:716-727): the leader first, then every grouped follower. C
// counts players and mobiles identically here, so the list holds both; the
// caller applies C's in-room filter. Followers keep C's follower-list order
// (newest first; add_follower prepends, src/utils.c).
func (w *World) combatGroupMembers(body combat.Combatant) []combat.Combatant {
	leader := body
	if master := w.combatFollowingBody(body); master != nil {
		leader = master
	}
	var members []combat.Combatant
	if combatGroupedBody(leader) {
		members = append(members, leader)
	}
	followers := make([]combat.Combatant, 0, 4)
	for _, p := range w.GetAllPlayers() {
		if p != leader && w.combatFollowingBody(p) == leader && combatGroupedBody(p) {
			followers = append(followers, p)
		}
	}
	for _, m := range w.GetAllMobs() {
		if m != leader && w.combatFollowingBody(m) == leader && combatGroupedBody(m) {
			followers = append(followers, m)
		}
	}
	sort.SliceStable(followers, func(i, j int) bool {
		left, right := actorFollowingSequence(followers[i]), actorFollowingSequence(followers[j])
		if left != right {
			return left > right
		}
		return strings.ToLower(followers[i].GetName()) < strings.ToLower(followers[j].GetName())
	})
	return append(members, followers...)
}

// FollowingBody returns the retained master or the unique player-name fallback.
func (w *World) FollowingBody(body combat.Combatant) combat.Combatant {
	return w.combatFollowingBody(body)
}
