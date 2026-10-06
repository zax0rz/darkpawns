package combat

// GameCallbacks holds the bridge functions between the combat package and the
// game layer. The combat package only sees the Combatant interface; it uses
// these callbacks to query other characters' state and to perform game-layer
// side effects (sending messages, creating corpses, awarding XP, etc.).
//
// This struct is owned by CombatEngine and validated at construction time.
type GameCallbacks struct {
	// Messaging
	Broadcast  func(roomVNum int, msg string, exclude []Combatant)
	SendToChar func(name Combatant, msg string)
	// SendText retains the ordinary text-event route for formerly direct body messages.
	SendText     func(body Combatant, msg string)
	SkillMessage func(dam int, ch, vict Combatant, attackType int, roomVNum int) bool
	// SendRaw delivers control bytes (a bare color code) with no line ending,
	// matching C's send_to_char(CCYEL(...)) which writes the escape and stops.
	// skill_message brackets its attacker/victim lines with these at C_CMP
	// (fight.c:1049-1054, 1064-1069, 1080-1085).
	SendRaw   func(name Combatant, msg string)
	BroadChat func(chName Combatant, msg string)
	Log       func(msg string, level string, minLevel int, toLog bool)

	// Character identity
	GetRace      func(name Combatant) int
	GetRaceHate  func(name Combatant, index int) int
	GetAlignment func(name Combatant) int
	SetAlignment func(name Combatant, val int)
	GetSex       func(name Combatant) int
	GetHP        func(name Combatant) int
	GetLevel     func(name Combatant) int
	IsNPC        func(name Combatant) bool
	GetSkill     func(name Combatant, skillNum int) int
	// GetColorLevel returns COLOR_LEV(ch) (screen.h:44): PRF_COLOR_1 counts 1
	// and PRF_COLOR_2 counts 2. C's C_CMP color gate needs level 3. NPCs have
	// no player_specials preferences and always report 0 (db.c:1281 dummy_mob).
	GetColorLevel func(name Combatant) int

	// Affects
	HasAffect        func(name Combatant, aff int) bool
	HasAffectStr     func(name Combatant, aff string) bool
	RemoveAffect     func(name Combatant, skillNum int)
	RemoveAllAffects func(name Combatant)
	RemoveTattoo     func(name Combatant)
	ClearNightbreed  func(name Combatant)
	ForgetVictim     func(name Combatant)

	// Player/Mob/Room flags
	HasPlrFlag          func(name Combatant, flag string) bool
	SetPlrFlag          func(name Combatant) bool
	HasPrfFlag          func(name Combatant, flag string) bool
	HasMobFlag          func(name Combatant, flag string) bool
	HasMobVNum          func(name Combatant, vnum int) bool
	MobHasJailGuardSpec func(name Combatant) bool
	HasRoomFlag         func(roomVNum int, flag string) bool
	HasScriptFlag       func(name Combatant, flag string) bool
	IsShopkeeper        func(name Combatant) bool
	// DamageRefused is damage()'s protection block (fight.c:1318-1368),
	// owned by the game layer so every damage seam emits the same refusal
	// bytes. True means C's damage() returned FALSE before hurting victim.
	DamageRefused     func(ch, victim Combatant) bool
	GetRoomCombatants func(roomVNum int) []Combatant
	GetFollowing      func(name Combatant) Combatant
	JailGuardSubdue   func(guardName, victimName Combatant) bool

	// Equipment & mounts
	IsMounted     func(name Combatant) bool
	Dismount      func(name Combatant)
	Unmount       func(name Combatant)
	GetWeaponInfo func(chName Combatant) (wType, damDice, damSize int, isBlessed bool)
	// GetWeaponDescription returns the wielded object's short description for
	// C act()'s $p substitution in skill/fight messages.
	GetWeaponDescription func(chName Combatant) string

	// Character conditions
	GetDrunk func(name Combatant) int

	// Room navigation
	GetAdjacentRoom func(roomVNum, door int) int

	// Kill/Death/Stats
	GainExp         func(name Combatant, amount int)
	GetExp          func(name Combatant) int
	GetKills        func(name Combatant) int64
	SetKills        func(name Combatant, kills int64)
	GetDeaths       func(name Combatant) int64
	SetDeaths       func(name Combatant, deaths int64)
	SetLastDeath    func(name Combatant, t int64)
	GetPks          func(name Combatant) int64
	SetPks          func(name Combatant, pks int64)
	GetConstitution func(name Combatant) int
	SetConstitution func(name Combatant, val int)

	// Corpse & extraction
	RawKillNPC     func(victim Combatant, attackType int)
	MakeCorpse     func(victim Combatant, attackType int)
	MakeDust       func(victim Combatant, attackType int)
	ExtractChar    func(name Combatant)
	RunDeathScript func(killer, victim Combatant, roomVNum int)

	// Group/Party
	GetFollowersInRoom       func(name Combatant, roomVNum int) int
	GetMasterInRoom          func(name Combatant, roomVNum int) bool
	GetFellowFollowersInRoom func(name Combatant, roomVNum int) bool
	CountGroupMembers        func(leaderName Combatant, roomVNum int) int
	ApplyToGroupMembers      func(leaderName Combatant, roomVNum int, fn func(name Combatant))

	// Gold
	GetGold func(name Combatant) int
	SetGold func(name Combatant, gold int)

	// Items
	JunkInventoryItems func(chName Combatant)

	// Commands
	PerformCommand func(chName Combatant, cmd string)

	// Follower break: fight.c:1457 stop_follower(victim) when the attacker is
	// the victim's own master — the charm branch renders the "jerk"/"hates
	// your guts!" trio with proper act audiences.
	StopFollowerOfMaster func(victimName, masterName Combatant)

	// Flee/Retreat
	GetWimpyLev func(name Combatant) int
	DoFlee      func(name Combatant)
	DoRetreat   func(name Combatant)

	// World
	IncreaseMaxStat func(name Combatant, stat string)
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

func cbBroadcast(roomVNum int, msg string, exclude []Combatant) {
	if cb := callbacks; cb != nil && cb.Broadcast != nil {
		cb.Broadcast(roomVNum, msg, exclude)
	}
}

func cbSendToChar(name Combatant, msg string) {
	if cb := callbacks; cb != nil && cb.SendToChar != nil {
		cb.SendToChar(name, msg)
	}
}

// cbSendRaw delivers control bytes with no line ending (C's bare color
// send_to_char calls). An empty message writes nothing, matching
// send_to_char("") when the recipient's color level is below the gate.
func cbSendRaw(name Combatant, msg string) {
	if msg == "" {
		return
	}
	if cb := callbacks; cb != nil && cb.SendRaw != nil {
		cb.SendRaw(name, msg)
	}
}

// cbGetColorLevel returns COLOR_LEV(name); unwired or unknown names report 0,
// which is also what C's shared dummy_mob preferences report for every NPC.
func cbGetColorLevel(name Combatant) int {
	if cb := callbacks; cb != nil && cb.GetColorLevel != nil {
		return cb.GetColorLevel(name)
	}
	return 0
}

func cbSkillMessage(dam int, ch, vict Combatant, attackType int, roomVNum int) bool {
	if cb := callbacks; cb != nil && cb.SkillMessage != nil {
		return cb.SkillMessage(dam, ch, vict, attackType, roomVNum)
	}
	return false
}

// EmitSkillMessage is the exported entry point the spells package uses to route
// a spell's damage message through the same skill_message path C's damage()
// uses (fight.c: !IS_WEAPON(attacktype) → skill_message). It draws Dice(1,N)
// from the shared roller and emits the char/vict/room text for the attack type,
// or returns false when no message set exists.
func EmitSkillMessage(dam int, ch, vict Combatant, attackType int, roomVNum int) bool {
	return cbSkillMessage(dam, ch, vict, attackType, roomVNum)
}

// cbWeaponInfo returns the wielded weapon's message attack-type OFFSET for the
// named attacker (fight.c:1792-1806 one_hit w_type derivation). The value is
// C's val3 (e.g. 11 for pierce, 3 for slash) — the 0-based offset that
// SendWeaponMessage adds TYPE_HIT to. It is NOT a TYPE_* constant and NOT
// TYPE_HIT+offset. Returns 0 ("hit") when unwired, barehand, or for mobs with
// no accessible attack_type.
func cbWeaponInfo(chName Combatant) int {
	if cb := callbacks; cb != nil && cb.GetWeaponInfo != nil {
		wType, _, _, _ := cb.GetWeaponInfo(chName)
		return wType
	}
	return 0
}

func cbWeaponBlessed(chName Combatant) bool {
	if cb := callbacks; cb != nil && cb.GetWeaponInfo != nil {
		_, _, _, isBlessed := cb.GetWeaponInfo(chName)
		return isBlessed
	}
	return false
}

func cbDrunk(chName Combatant) int {
	if cb := callbacks; cb != nil && cb.GetDrunk != nil {
		return cb.GetDrunk(chName)
	}
	return 0
}

func cbWeaponDescription(chName Combatant) string {
	if cb := callbacks; cb != nil && cb.GetWeaponDescription != nil {
		return cb.GetWeaponDescription(chName)
	}
	return ""
}

func cbBroadChat(chName Combatant, msg string) {
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

func cbGetRace(name Combatant) int {
	if cb := callbacks; cb != nil && cb.GetRace != nil {
		return cb.GetRace(name)
	}
	return 0
}

func cbGetRaceHate(name Combatant, index int) int {
	if cb := callbacks; cb != nil && cb.GetRaceHate != nil {
		return cb.GetRaceHate(name, index)
	}
	return 0
}

func cbGetAlignment(name Combatant) int {
	if cb := callbacks; cb != nil && cb.GetAlignment != nil {
		return cb.GetAlignment(name)
	}
	return 0
}

func cbSetAlignment(name Combatant, val int) {
	if cb := callbacks; cb != nil && cb.SetAlignment != nil {
		cb.SetAlignment(name, val)
	}
}

func cbGetSex(name Combatant) int {
	if cb := callbacks; cb != nil && cb.GetSex != nil {
		return cb.GetSex(name)
	}
	return 0
}

func cbGetHP(name Combatant) int {
	if cb := callbacks; cb != nil && cb.GetHP != nil {
		return cb.GetHP(name)
	}
	return 1
}

func cbGetLevel(name Combatant) int {
	if cb := callbacks; cb != nil && cb.GetLevel != nil {
		return cb.GetLevel(name)
	}
	return 0
}

func cbIsNPC(name Combatant) bool {
	if cb := callbacks; cb != nil && cb.IsNPC != nil {
		return cb.IsNPC(name)
	}
	return false
}

func cbGetSkill(name Combatant, skillNum int) int {
	if cb := callbacks; cb != nil && cb.GetSkill != nil {
		return cb.GetSkill(name, skillNum)
	}
	return 0
}

func cbHasAffect(name Combatant, aff int) bool {
	if cb := callbacks; cb != nil && cb.HasAffect != nil {
		return cb.HasAffect(name, aff)
	}
	return false
}

func cbHasAffectStr(name Combatant, aff string) bool {
	if cb := callbacks; cb != nil && cb.HasAffectStr != nil {
		return cb.HasAffectStr(name, aff)
	}
	return false
}

func cbRemoveAffect(name Combatant, skillNum int) {
	if cb := callbacks; cb != nil && cb.RemoveAffect != nil {
		cb.RemoveAffect(name, skillNum)
	}
}

func cbRemoveAllAffects(name Combatant) {
	if cb := callbacks; cb != nil && cb.RemoveAllAffects != nil {
		cb.RemoveAllAffects(name)
	}
}

func cbHasPlrFlag(name Combatant, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasPlrFlag != nil {
		return cb.HasPlrFlag(name, flag)
	}
	return false
}

func cbSetPlrFlag(name Combatant) {
	if cb := callbacks; cb != nil && cb.SetPlrFlag != nil {
		cb.SetPlrFlag(name)
	}
}

func cbHasPrfFlag(name Combatant, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasPrfFlag != nil {
		return cb.HasPrfFlag(name, flag)
	}
	return false
}

func cbHasMobFlag(name Combatant, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasMobFlag != nil {
		return cb.HasMobFlag(name, flag)
	}
	return false
}

func cbMobHasJailGuardSpec(name Combatant) bool {
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

func cbHasScriptFlag(name Combatant, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasScriptFlag != nil {
		return cb.HasScriptFlag(name, flag)
	}
	return false
}

func cbIsShopkeeper(name Combatant) bool {
	if cb := callbacks; cb != nil && cb.IsShopkeeper != nil {
		return cb.IsShopkeeper(name)
	}
	return false
}

// cbDamageRefused asks the game layer's damage() protection gate. Without a
// game layer (package tests) it falls back to the silent subset the combat
// package can see for itself.
func cbDamageRefused(ch, victim Combatant) bool {
	if cb := callbacks; cb != nil && cb.DamageRefused != nil {
		return cb.DamageRefused(ch, victim)
	}
	return defaultDamageRefused(ch, victim)
}

func cbGetRoomCombatants(roomVNum int) []Combatant {
	if cb := callbacks; cb != nil && cb.GetRoomCombatants != nil {
		return cb.GetRoomCombatants(roomVNum)
	}
	return nil
}

func cbGetFollowing(name Combatant) Combatant {
	if cb := callbacks; cb != nil && cb.GetFollowing != nil {
		return cb.GetFollowing(name)
	}
	return nil
}

func cbJailGuardSubdue(guardName, victimName Combatant) bool {
	if cb := callbacks; cb != nil && cb.JailGuardSubdue != nil {
		return cb.JailGuardSubdue(guardName, victimName)
	}
	return false
}

func cbIsMounted(name Combatant) bool {
	if cb := callbacks; cb != nil && cb.IsMounted != nil {
		return cb.IsMounted(name)
	}
	return false
}

func cbDismount(name Combatant) {
	if cb := callbacks; cb != nil && cb.Dismount != nil {
		cb.Dismount(name)
	}
}

func cbUnmount(name Combatant) {
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

func cbGainExp(name Combatant, amount int) {
	if cb := callbacks; cb != nil && cb.GainExp != nil {
		cb.GainExp(name, amount)
	}
}

func cbGetExp(name Combatant) int {
	if cb := callbacks; cb != nil && cb.GetExp != nil {
		return cb.GetExp(name)
	}
	return 0
}

func cbGetKills(name Combatant) int64 {
	if cb := callbacks; cb != nil && cb.GetKills != nil {
		return cb.GetKills(name)
	}
	return 0
}

func cbSetKills(name Combatant, kills int64) {
	if cb := callbacks; cb != nil && cb.SetKills != nil {
		cb.SetKills(name, kills)
	}
}

func cbGetDeaths(name Combatant) int64 {
	if cb := callbacks; cb != nil && cb.GetDeaths != nil {
		return cb.GetDeaths(name)
	}
	return 0
}

func cbSetDeaths(name Combatant, deaths int64) {
	if cb := callbacks; cb != nil && cb.SetDeaths != nil {
		cb.SetDeaths(name, deaths)
	}
}

func cbSetLastDeath(name Combatant, t int64) {
	if cb := callbacks; cb != nil && cb.SetLastDeath != nil {
		cb.SetLastDeath(name, t)
	}
}

func cbGetPks(name Combatant) int64 {
	if cb := callbacks; cb != nil && cb.GetPks != nil {
		return cb.GetPks(name)
	}
	return 0
}

func cbSetPks(name Combatant, pks int64) {
	if cb := callbacks; cb != nil && cb.SetPks != nil {
		cb.SetPks(name, pks)
	}
}

func cbGetConstitution(name Combatant) int {
	if cb := callbacks; cb != nil && cb.GetConstitution != nil {
		return cb.GetConstitution(name)
	}
	return 0
}

func cbSetConstitution(name Combatant, val int) {
	if cb := callbacks; cb != nil && cb.SetConstitution != nil {
		cb.SetConstitution(name, val)
	}
}

func cbMakeCorpse(victim Combatant, attackType int) {
	if cb := callbacks; cb != nil && cb.MakeCorpse != nil {
		cb.MakeCorpse(victim, attackType)
	}
}

func cbMakeDust(victim Combatant, attackType int) {
	if cb := callbacks; cb != nil && cb.MakeDust != nil {
		cb.MakeDust(victim, attackType)
	}
}

func cbExtractChar(name Combatant) {
	if cb := callbacks; cb != nil && cb.ExtractChar != nil {
		cb.ExtractChar(name)
	}
}

func cbRunDeathScript(killer, victim Combatant, roomVNum int) {
	if cb := callbacks; cb != nil && cb.RunDeathScript != nil {
		cb.RunDeathScript(killer, victim, roomVNum)
	}
}

// ---------------------------------------------------------------------------
// Wimpy helper.
// ---------------------------------------------------------------------------

func cbGetWimpyLev(name Combatant) int {
	if cb := callbacks; cb != nil && cb.GetWimpyLev != nil {
		return cb.GetWimpyLev(name)
	}
	return 0
}

// ---------------------------------------------------------------------------
// Group/Party helpers.
// ---------------------------------------------------------------------------

func cbGetFollowersInRoom(name Combatant, roomVNum int) int {
	if cb := callbacks; cb != nil && cb.GetFollowersInRoom != nil {
		return cb.GetFollowersInRoom(name, roomVNum)
	}
	return 0
}

func cbGetMasterInRoom(name Combatant, roomVNum int) bool {
	if cb := callbacks; cb != nil && cb.GetMasterInRoom != nil {
		return cb.GetMasterInRoom(name, roomVNum)
	}
	return false
}

func cbGetFellowFollowersInRoom(name Combatant, roomVNum int) bool {
	if cb := callbacks; cb != nil && cb.GetFellowFollowersInRoom != nil {
		return cb.GetFellowFollowersInRoom(name, roomVNum)
	}
	return false
}

func cbCountGroupMembers(leaderName Combatant, roomVNum int) int {
	if cb := callbacks; cb != nil && cb.CountGroupMembers != nil {
		return cb.CountGroupMembers(leaderName, roomVNum)
	}
	return 1
}

func cbApplyToGroupMembers(leaderName Combatant, roomVNum int, fn func(name Combatant)) {
	if cb := callbacks; cb != nil && cb.ApplyToGroupMembers != nil {
		cb.ApplyToGroupMembers(leaderName, roomVNum, fn)
	}
}

// ---------------------------------------------------------------------------
// Gold helpers.
// ---------------------------------------------------------------------------

func cbGetGold(name Combatant) int {
	if cb := callbacks; cb != nil && cb.GetGold != nil {
		return cb.GetGold(name)
	}
	return 0
}

func cbSetGold(name Combatant, gold int) {
	if cb := callbacks; cb != nil && cb.SetGold != nil {
		cb.SetGold(name, gold)
	}
}

// ---------------------------------------------------------------------------
// Item helpers.
// ---------------------------------------------------------------------------

func cbJunkInventoryItems(chName Combatant) {
	if cb := callbacks; cb != nil && cb.JunkInventoryItems != nil {
		cb.JunkInventoryItems(chName)
	}
}

// ---------------------------------------------------------------------------
// Command helpers.
// ---------------------------------------------------------------------------

func cbPerformCommand(chName Combatant, cmd string) {
	if cb := callbacks; cb != nil && cb.PerformCommand != nil {
		cb.PerformCommand(chName, cmd)
	}
}

func cbStopFollowerOfMaster(victimName, masterName Combatant) {
	if cb := callbacks; cb != nil && cb.StopFollowerOfMaster != nil {
		cb.StopFollowerOfMaster(victimName, masterName)
	}
}

// ---------------------------------------------------------------------------
// Flee/Retreat helpers.
// ---------------------------------------------------------------------------

func cbDoFlee(name Combatant) {
	if cb := callbacks; cb != nil && cb.DoFlee != nil {
		cb.DoFlee(name)
	}
}

func cbDoRetreat(name Combatant) {
	if cb := callbacks; cb != nil && cb.DoRetreat != nil {
		cb.DoRetreat(name)
	}
}

// ---------------------------------------------------------------------------
// World helpers.
// ---------------------------------------------------------------------------

func cbIncreaseMaxStat(name Combatant, stat string) {
	if cb := callbacks; cb != nil && cb.IncreaseMaxStat != nil {
		cb.IncreaseMaxStat(name, stat)
	}
}

func cbHealAllPlayers() {
	if cb := callbacks; cb != nil && cb.HealAllPlayers != nil {
		cb.HealAllPlayers()
	}
}

// sendCombatMessage retains body identity and the existing text-event framing.
func sendCombatMessage(body Combatant, message string) {
	if callbacks != nil && callbacks.SendText != nil {
		callbacks.SendText(body, message)
		return
	}
	body.SendMessage(message)
}
