package combat

// GameCallbacks holds the bridge functions between the combat package and the
// game layer. The combat package only sees the Combatant interface; it uses
// these callbacks to query other characters' state and to perform game-layer
// side effects (sending messages, creating corpses, awarding XP, etc.).
//
// This struct is owned by CombatEngine and validated at construction time.
type GameCallbacks struct {
	// Messaging
	Broadcast    func(roomVNum int, msg string, exclude string)
	SendToChar   func(name string, msg string)
	SkillMessage func(dam int, ch, vict string, attackType int, roomVNum int) bool
	BroadChat    func(chName string, msg string)
	Log          func(msg string, level string, minLevel int, toLog bool)

	// Character identity
	GetRace      func(name string) int
	GetRaceHate  func(name string, index int) int
	GetAlignment func(name string) int
	SetAlignment func(name string, val int)
	GetSex       func(name string) int
	GetSkill     func(name string, skillNum int) int

	// Affects
	HasAffect        func(name string, aff int) bool
	HasAffectStr     func(name string, aff string) bool
	RemoveAffect     func(name string, skillNum int)
	RemoveAllAffects func(name string)

	// Player/Mob/Room flags
	HasPlrFlag          func(name string, flag string) bool
	SetPlrFlag          func(name string) bool
	HasPrfFlag          func(name string, flag string) bool
	HasMobFlag          func(name string, flag string) bool
	HasMobVNum          func(name string, vnum int) bool
	MobHasJailGuardSpec func(name string) bool
	HasRoomFlag         func(roomVNum int, flag string) bool
	HasScriptFlag       func(name string, flag string) bool
	IsShopkeeper        func(name string) bool
	GetRoomCombatants   func(roomVNum int) []Combatant
	GetFollowing        func(name string) string
	JailGuardSubdue     func(guardName, victimName string) bool

	// Equipment & mounts
	IsMounted     func(name string) bool
	Dismount      func(name string)
	Unmount       func(name string)
	GetWeaponInfo func(chName string) (wType, damDice, damSize int, isBlessed bool)

	// Room navigation
	GetAdjacentRoom func(roomVNum, door int) int

	// Kill/Death/Stats
	GainExp         func(name string, amount int)
	GetExp          func(name string) int
	GetKills        func(name string) int64
	SetKills        func(name string, kills int64)
	GetDeaths       func(name string) int64
	SetDeaths       func(name string, deaths int64)
	SetLastDeath    func(name string, t int64)
	GetPks          func(name string) int64
	SetPks          func(name string, pks int64)
	GetConstitution func(name string) int
	SetConstitution func(name string, val int)

	// Corpse & extraction
	MakeCorpse     func(victim string, attackType int)
	MakeDust       func(victim string, attackType int)
	ExtractChar    func(name string)
	RunDeathScript func(killer, victim string, roomVNum int)

	// Group/Party
	GetFollowersInRoom       func(name string, roomVNum int) int
	GetMasterInRoom          func(name string, roomVNum int) bool
	GetFellowFollowersInRoom func(name string, roomVNum int) bool
	CountGroupMembers        func(leaderName string, roomVNum int) int
	ApplyToGroupMembers      func(leaderName string, roomVNum int, fn func(name string))

	// Gold
	GetGold func(name string) int
	SetGold func(name string, gold int)

	// Items
	JunkInventoryItems func(chName string)

	// Commands
	PerformCommand func(chName, cmd string)

	// Flee/Retreat
	GetWimpyLev func(name string) int
	DoFlee      func(name string)
	DoRetreat   func(name string)

	// World
	IncreaseMaxStat func(name string, stat string)
	HealAllPlayers  func()
}

// callbacks is the canonical package-level accessor for the active
// GameCallbacks instance. It is set during CombatEngine initialization and is
// used by the legacy fight_core functions that do not yet receive
// *GameCallbacks as a parameter.
var callbacks *GameCallbacks

// SetCallbacks sets the canonical package-level callback accessor.
func SetCallbacks(cb *GameCallbacks) {
	callbacks = cb
}

// GetCallbacks returns the current package-level callback accessor.
func GetCallbacks() *GameCallbacks {
	return callbacks
}

// ---------------------------------------------------------------------------
// GameCallbacks helpers. These read from the active callbacks instance.
// They are nil-safe: if a hook is not wired, they return the appropriate zero
// value and perform no side effects.
// ---------------------------------------------------------------------------

func cbBroadcast(roomVNum int, msg string, exclude string) {
	if cb := callbacks; cb != nil && cb.Broadcast != nil {
		cb.Broadcast(roomVNum, msg, exclude)
	}
}

func cbSendToChar(name string, msg string) {
	if cb := callbacks; cb != nil && cb.SendToChar != nil {
		cb.SendToChar(name, msg)
	}
}

func cbSkillMessage(dam int, ch, vict string, attackType int, roomVNum int) bool {
	if cb := callbacks; cb != nil && cb.SkillMessage != nil {
		return cb.SkillMessage(dam, ch, vict, attackType, roomVNum)
	}
	return false
}

func cbBroadChat(chName string, msg string) {
	if cb := callbacks; cb != nil && cb.BroadChat != nil {
		cb.BroadChat(chName, msg)
	}
}

func cbLog(msg string, level string, minLevel int, toLog bool) {
	if cb := callbacks; cb != nil && cb.Log != nil {
		cb.Log(msg, level, minLevel, toLog)
	}
}

// ---------------------------------------------------------------------------
// Character-state helpers.
// ---------------------------------------------------------------------------

func cbGetRace(name string) int {
	if cb := callbacks; cb != nil && cb.GetRace != nil {
		return cb.GetRace(name)
	}
	return 0
}

func cbGetRaceHate(name string, index int) int {
	if cb := callbacks; cb != nil && cb.GetRaceHate != nil {
		return cb.GetRaceHate(name, index)
	}
	return 0
}

func cbGetAlignment(name string) int {
	if cb := callbacks; cb != nil && cb.GetAlignment != nil {
		return cb.GetAlignment(name)
	}
	return 0
}

func cbSetAlignment(name string, val int) {
	if cb := callbacks; cb != nil && cb.SetAlignment != nil {
		cb.SetAlignment(name, val)
	}
}

func cbGetSex(name string) int {
	if cb := callbacks; cb != nil && cb.GetSex != nil {
		return cb.GetSex(name)
	}
	return 0
}

func cbGetSkill(name string, skillNum int) int {
	if cb := callbacks; cb != nil && cb.GetSkill != nil {
		return cb.GetSkill(name, skillNum)
	}
	return 0
}

func cbHasAffect(name string, aff int) bool {
	if cb := callbacks; cb != nil && cb.HasAffect != nil {
		return cb.HasAffect(name, aff)
	}
	return false
}

func cbHasAffectStr(name string, aff string) bool {
	if cb := callbacks; cb != nil && cb.HasAffectStr != nil {
		return cb.HasAffectStr(name, aff)
	}
	return false
}

func cbRemoveAffect(name string, skillNum int) {
	if cb := callbacks; cb != nil && cb.RemoveAffect != nil {
		cb.RemoveAffect(name, skillNum)
	}
}

func cbRemoveAllAffects(name string) {
	if cb := callbacks; cb != nil && cb.RemoveAllAffects != nil {
		cb.RemoveAllAffects(name)
	}
}

func cbHasPlrFlag(name string, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasPlrFlag != nil {
		return cb.HasPlrFlag(name, flag)
	}
	return false
}

func cbSetPlrFlag(name string) {
	if cb := callbacks; cb != nil && cb.SetPlrFlag != nil {
		cb.SetPlrFlag(name)
	}
}

func cbHasPrfFlag(name string, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasPrfFlag != nil {
		return cb.HasPrfFlag(name, flag)
	}
	return false
}

func cbHasMobFlag(name string, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasMobFlag != nil {
		return cb.HasMobFlag(name, flag)
	}
	return false
}

func cbMobHasJailGuardSpec(name string) bool {
	if cb := callbacks; cb != nil && cb.MobHasJailGuardSpec != nil {
		return cb.MobHasJailGuardSpec(name)
	}
	return false
}

func cbHasRoomFlag(roomVNum int, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasRoomFlag != nil {
		return cb.HasRoomFlag(roomVNum, flag)
	}
	return false
}

func cbHasScriptFlag(name string, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasScriptFlag != nil {
		return cb.HasScriptFlag(name, flag)
	}
	return false
}

func cbIsShopkeeper(name string) bool {
	if cb := callbacks; cb != nil && cb.IsShopkeeper != nil {
		return cb.IsShopkeeper(name)
	}
	return false
}

func cbGetRoomCombatants(roomVNum int) []Combatant {
	if cb := callbacks; cb != nil && cb.GetRoomCombatants != nil {
		return cb.GetRoomCombatants(roomVNum)
	}
	return nil
}

func cbGetFollowing(name string) string {
	if cb := callbacks; cb != nil && cb.GetFollowing != nil {
		return cb.GetFollowing(name)
	}
	return ""
}

func cbJailGuardSubdue(guardName, victimName string) bool {
	if cb := callbacks; cb != nil && cb.JailGuardSubdue != nil {
		return cb.JailGuardSubdue(guardName, victimName)
	}
	return false
}

func cbIsMounted(name string) bool {
	if cb := callbacks; cb != nil && cb.IsMounted != nil {
		return cb.IsMounted(name)
	}
	return false
}

func cbDismount(name string) {
	if cb := callbacks; cb != nil && cb.Dismount != nil {
		cb.Dismount(name)
	}
}

func cbUnmount(name string) {
	if cb := callbacks; cb != nil && cb.Unmount != nil {
		cb.Unmount(name)
	}
}

func cbGetAdjacentRoom(roomVNum, door int) int {
	if cb := callbacks; cb != nil && cb.GetAdjacentRoom != nil {
		return cb.GetAdjacentRoom(roomVNum, door)
	}
	return -1
}

func cbGainExp(name string, amount int) {
	if cb := callbacks; cb != nil && cb.GainExp != nil {
		cb.GainExp(name, amount)
	}
}

func cbGetExp(name string) int {
	if cb := callbacks; cb != nil && cb.GetExp != nil {
		return cb.GetExp(name)
	}
	return 0
}

func cbGetKills(name string) int64 {
	if cb := callbacks; cb != nil && cb.GetKills != nil {
		return cb.GetKills(name)
	}
	return 0
}

func cbSetKills(name string, kills int64) {
	if cb := callbacks; cb != nil && cb.SetKills != nil {
		cb.SetKills(name, kills)
	}
}

func cbGetDeaths(name string) int64 {
	if cb := callbacks; cb != nil && cb.GetDeaths != nil {
		return cb.GetDeaths(name)
	}
	return 0
}

func cbSetDeaths(name string, deaths int64) {
	if cb := callbacks; cb != nil && cb.SetDeaths != nil {
		cb.SetDeaths(name, deaths)
	}
}

func cbSetLastDeath(name string, t int64) {
	if cb := callbacks; cb != nil && cb.SetLastDeath != nil {
		cb.SetLastDeath(name, t)
	}
}

func cbGetPks(name string) int64 {
	if cb := callbacks; cb != nil && cb.GetPks != nil {
		return cb.GetPks(name)
	}
	return 0
}

func cbSetPks(name string, pks int64) {
	if cb := callbacks; cb != nil && cb.SetPks != nil {
		cb.SetPks(name, pks)
	}
}

func cbGetConstitution(name string) int {
	if cb := callbacks; cb != nil && cb.GetConstitution != nil {
		return cb.GetConstitution(name)
	}
	return 0
}

func cbSetConstitution(name string, val int) {
	if cb := callbacks; cb != nil && cb.SetConstitution != nil {
		cb.SetConstitution(name, val)
	}
}

func cbMakeCorpse(victim string, attackType int) {
	if cb := callbacks; cb != nil && cb.MakeCorpse != nil {
		cb.MakeCorpse(victim, attackType)
	}
}

func cbMakeDust(victim string, attackType int) {
	if cb := callbacks; cb != nil && cb.MakeDust != nil {
		cb.MakeDust(victim, attackType)
	}
}

func cbExtractChar(name string) {
	if cb := callbacks; cb != nil && cb.ExtractChar != nil {
		cb.ExtractChar(name)
	}
}

func cbRunDeathScript(killer, victim string, roomVNum int) {
	if cb := callbacks; cb != nil && cb.RunDeathScript != nil {
		cb.RunDeathScript(killer, victim, roomVNum)
	}
}

// ---------------------------------------------------------------------------
// Wimpy helper.
// ---------------------------------------------------------------------------

func cbGetWimpyLev(name string) int {
	if cb := callbacks; cb != nil && cb.GetWimpyLev != nil {
		return cb.GetWimpyLev(name)
	}
	return 0
}

// ---------------------------------------------------------------------------
// Group/Party helpers.
// ---------------------------------------------------------------------------

func cbGetFollowersInRoom(name string, roomVNum int) int {
	if cb := callbacks; cb != nil && cb.GetFollowersInRoom != nil {
		return cb.GetFollowersInRoom(name, roomVNum)
	}
	return 0
}

func cbGetMasterInRoom(name string, roomVNum int) bool {
	if cb := callbacks; cb != nil && cb.GetMasterInRoom != nil {
		return cb.GetMasterInRoom(name, roomVNum)
	}
	return false
}

func cbGetFellowFollowersInRoom(name string, roomVNum int) bool {
	if cb := callbacks; cb != nil && cb.GetFellowFollowersInRoom != nil {
		return cb.GetFellowFollowersInRoom(name, roomVNum)
	}
	return false
}

func cbCountGroupMembers(leaderName string, roomVNum int) int {
	if cb := callbacks; cb != nil && cb.CountGroupMembers != nil {
		return cb.CountGroupMembers(leaderName, roomVNum)
	}
	return 1
}

func cbApplyToGroupMembers(leaderName string, roomVNum int, fn func(name string)) {
	if cb := callbacks; cb != nil && cb.ApplyToGroupMembers != nil {
		cb.ApplyToGroupMembers(leaderName, roomVNum, fn)
	}
}

// ---------------------------------------------------------------------------
// Gold helpers.
// ---------------------------------------------------------------------------

func cbGetGold(name string) int {
	if cb := callbacks; cb != nil && cb.GetGold != nil {
		return cb.GetGold(name)
	}
	return 0
}

func cbSetGold(name string, gold int) {
	if cb := callbacks; cb != nil && cb.SetGold != nil {
		cb.SetGold(name, gold)
	}
}

// ---------------------------------------------------------------------------
// Item helpers.
// ---------------------------------------------------------------------------

func cbJunkInventoryItems(chName string) {
	if cb := callbacks; cb != nil && cb.JunkInventoryItems != nil {
		cb.JunkInventoryItems(chName)
	}
}

// ---------------------------------------------------------------------------
// Command helpers.
// ---------------------------------------------------------------------------

func cbPerformCommand(chName, cmd string) {
	if cb := callbacks; cb != nil && cb.PerformCommand != nil {
		cb.PerformCommand(chName, cmd)
	}
}

// ---------------------------------------------------------------------------
// Flee/Retreat helpers.
// ---------------------------------------------------------------------------

func cbDoFlee(name string) {
	if cb := callbacks; cb != nil && cb.DoFlee != nil {
		cb.DoFlee(name)
	}
}

func cbDoRetreat(name string) {
	if cb := callbacks; cb != nil && cb.DoRetreat != nil {
		cb.DoRetreat(name)
	}
}

// ---------------------------------------------------------------------------
// World helpers.
// ---------------------------------------------------------------------------

func cbIncreaseMaxStat(name string, stat string) {
	if cb := callbacks; cb != nil && cb.IncreaseMaxStat != nil {
		cb.IncreaseMaxStat(name, stat)
	}
}

func cbHealAllPlayers() {
	if cb := callbacks; cb != nil && cb.HealAllPlayers != nil {
		cb.HealAllPlayers()
	}
}
