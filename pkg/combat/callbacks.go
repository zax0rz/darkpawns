package combat

// GameCallbacks holds the bridge functions between the combat package and the
// game layer. The combat package only sees the Combatant interface; it uses
// these callbacks to query other characters' state and to perform game-layer
// side effects (sending messages, creating corpses, awarding XP, etc.).
//
// This struct is owned by CombatEngine and validated at construction time.
// During the multi-PR migration to GameCallbacks, un-migrated hooks remain as
// package-level vars in fight_core.go and are read through the temporary
// package-level `callbacks` variable set via SetCallbacks.
type GameCallbacks struct {
	// Messaging (Group 1 — migrated in PR 1)
	Broadcast    func(roomVNum int, msg string, exclude string)
	SendToChar   func(name string, msg string)
	SkillMessage func(dam int, ch, vict string, attackType int, roomVNum int) bool
	BroadChat    func(chName string, msg string)
	Log          func(msg string, level string, minLevel int, toLog bool)

	// Character identity (Group 2 — migrated in PR 2)
	GetRace      func(name string) int
	GetRaceHate  func(name string, index int) int
	GetAlignment func(name string) int
	SetAlignment func(name string, val int)
	GetSex       func(name string) int
	GetSkill     func(name string, skillNum int) int

	// Affects (Group 2)
	HasAffect        func(name string, aff int) bool
	HasAffectStr     func(name string, aff string) bool
	RemoveAffect     func(name string, skillNum int)
	RemoveAllAffects func(name string)

	// Player/Mob/Room flags (Group 2)
	HasPlrFlag    func(name string, flag string) bool
	SetPlrFlag    func(name string) bool
	HasPrfFlag    func(name string, flag string) bool
	HasMobFlag    func(name string, flag string) bool
	HasMobVNum    func(name string, vnum int) bool
	HasRoomFlag   func(roomVNum int, flag string) bool
	HasScriptFlag func(name string, flag string) bool
	IsShopkeeper  func(name string) bool

	// Equipment & mounts (Group 2)
	IsMounted     func(name string) bool
	Dismount      func(name string)
	Unmount       func(name string)
	GetWeaponInfo func(chName string) (wType, damDice, damSize int, isBlessed bool)

	// Room navigation (Group 2)
	GetAdjacentRoom func(roomVNum, door int) int

	// Kill/Death/Stats (Group 2)
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

	// Corpse & extraction (Group 2)
	MakeCorpse     func(victim string, attackType int)
	MakeDust       func(victim string, attackType int)
	ExtractChar    func(name string)
	RunDeathScript func(killer, victim string, roomVNum int)
}

// callbacks is the temporary package-level accessor used while fight_core
// functions are being migrated to receive *GameCallbacks directly. It is set
// during CombatEngine initialization and removed in PR 3.
var callbacks *GameCallbacks

// SetCallbacks sets the package-level callback accessor. This is temporary
// scaffolding for the migration and will be removed once all fight_core
// functions accept *GameCallbacks as a parameter.
func SetCallbacks(cb *GameCallbacks) {
	callbacks = cb
}

// GetCallbacks returns the current package-level callback accessor. It is
// exposed so that cmd/server/main.go can validate wiring after init.
func GetCallbacks() *GameCallbacks {
	return callbacks
}

// ---------------------------------------------------------------------------
// Temporary dual-read helpers for PR 1 messaging hooks.
// These prefer callbacks.X when available and fall back to the legacy
// package-level vars so existing tests keep working until PR 3.
// ---------------------------------------------------------------------------

func cbBroadcast(roomVNum int, msg string, exclude string) {
	if cb := callbacks; cb != nil && cb.Broadcast != nil {
		cb.Broadcast(roomVNum, msg, exclude)
	} else if BroadcastMessage != nil {
		BroadcastMessage(roomVNum, msg, exclude)
	}
}

func cbSendToChar(name string, msg string) {
	if cb := callbacks; cb != nil && cb.SendToChar != nil {
		cb.SendToChar(name, msg)
	} else if SendToCharFunc != nil {
		SendToCharFunc(name, msg)
	}
}

func cbSkillMessage(dam int, ch, vict string, attackType int, roomVNum int) bool {
	if cb := callbacks; cb != nil && cb.SkillMessage != nil {
		return cb.SkillMessage(dam, ch, vict, attackType, roomVNum)
	}
	if SkillMessageFunc != nil {
		return SkillMessageFunc(dam, ch, vict, attackType, roomVNum)
	}
	return false
}

func cbBroadChat(chName string, msg string) {
	if cb := callbacks; cb != nil && cb.BroadChat != nil {
		cb.BroadChat(chName, msg)
	} else if BroadChatFunc != nil {
		BroadChatFunc(chName, msg)
	}
}

func cbLog(msg string, level string, minLevel int, toLog bool) {
	if cb := callbacks; cb != nil && cb.Log != nil {
		cb.Log(msg, level, minLevel, toLog)
	} else if LogMessage != nil {
		LogMessage(msg, level, minLevel, toLog)
	}
}

// ---------------------------------------------------------------------------
// Temporary dual-read helpers for PR 2 character-state hooks.
// These prefer callbacks.X when available and fall back to the legacy
// package-level vars so existing tests keep working until PR 3.
// ---------------------------------------------------------------------------

func cbGetRace(name string) int {
	if cb := callbacks; cb != nil && cb.GetRace != nil {
		return cb.GetRace(name)
	}
	if GetRace != nil {
		return GetRace(name)
	}
	return 0
}

func cbGetRaceHate(name string, index int) int {
	if cb := callbacks; cb != nil && cb.GetRaceHate != nil {
		return cb.GetRaceHate(name, index)
	}
	if GetRaceHate != nil {
		return GetRaceHate(name, index)
	}
	return 0
}

func cbGetAlignment(name string) int {
	if cb := callbacks; cb != nil && cb.GetAlignment != nil {
		return cb.GetAlignment(name)
	}
	if GetAlignment != nil {
		return GetAlignment(name)
	}
	return 0
}

func cbSetAlignment(name string, val int) {
	if cb := callbacks; cb != nil && cb.SetAlignment != nil {
		cb.SetAlignment(name, val)
	} else if SetAlignment != nil {
		SetAlignment(name, val)
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
	if GetSkill != nil {
		return GetSkill(name, skillNum)
	}
	return 0
}

func cbHasAffect(name string, aff int) bool {
	if cb := callbacks; cb != nil && cb.HasAffect != nil {
		return cb.HasAffect(name, aff)
	}
	return HasAffect != nil && HasAffect(name, aff)
}

func cbHasAffectStr(name string, aff string) bool {
	if cb := callbacks; cb != nil && cb.HasAffectStr != nil {
		return cb.HasAffectStr(name, aff)
	}
	return HasAffectStr != nil && HasAffectStr(name, aff)
}

func cbRemoveAffect(name string, skillNum int) {
	if cb := callbacks; cb != nil && cb.RemoveAffect != nil {
		cb.RemoveAffect(name, skillNum)
	} else if RemoveAffect != nil {
		RemoveAffect(name, skillNum)
	}
}

func cbRemoveAllAffects(name string) {
	if cb := callbacks; cb != nil && cb.RemoveAllAffects != nil {
		cb.RemoveAllAffects(name)
	} else if RemoveAllAffects != nil {
		RemoveAllAffects(name)
	}
}

func cbHasPlrFlag(name string, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasPlrFlag != nil {
		return cb.HasPlrFlag(name, flag)
	}
	return HasPlrFlag != nil && HasPlrFlag(name, flag)
}

func cbSetPlrFlag(name string) {
	if cb := callbacks; cb != nil && cb.SetPlrFlag != nil {
		cb.SetPlrFlag(name)
	} else if SetPlrFlag != nil {
		SetPlrFlag(name)
	}
}

func cbHasPrfFlag(name string, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasPrfFlag != nil {
		return cb.HasPrfFlag(name, flag)
	}
	return HasPrfFlag != nil && HasPrfFlag(name, flag)
}

func cbHasMobFlag(name string, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasMobFlag != nil {
		return cb.HasMobFlag(name, flag)
	}
	return HasMobFlag != nil && HasMobFlag(name, flag)
}

func cbHasMobVNum(name string, vnum int) bool {
	if cb := callbacks; cb != nil && cb.HasMobVNum != nil {
		return cb.HasMobVNum(name, vnum)
	}
	return HasMobVNum != nil && HasMobVNum(name, vnum)
}

func cbHasRoomFlag(roomVNum int, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasRoomFlag != nil {
		return cb.HasRoomFlag(roomVNum, flag)
	}
	return HasRoomFlag != nil && HasRoomFlag(roomVNum, flag)
}

func cbHasScriptFlag(name string, flag string) bool {
	if cb := callbacks; cb != nil && cb.HasScriptFlag != nil {
		return cb.HasScriptFlag(name, flag)
	}
	return HasScriptFlag != nil && HasScriptFlag(name, flag)
}

func cbIsShopkeeper(name string) bool {
	if cb := callbacks; cb != nil && cb.IsShopkeeper != nil {
		return cb.IsShopkeeper(name)
	}
	return IsShopkeeper != nil && IsShopkeeper(name)
}

func cbIsMounted(name string) bool {
	if cb := callbacks; cb != nil && cb.IsMounted != nil {
		return cb.IsMounted(name)
	}
	return IsMounted != nil && IsMounted(name)
}

func cbDismount(name string) {
	if cb := callbacks; cb != nil && cb.Dismount != nil {
		cb.Dismount(name)
	} else if Dismount != nil {
		Dismount(name)
	}
}

func cbUnmount(name string) {
	if cb := callbacks; cb != nil && cb.Unmount != nil {
		cb.Unmount(name)
	} else if Unmount != nil {
		Unmount(name)
	}
}

func cbGetWeaponInfo(chName string) (wType, damDice, damSize int, isBlessed bool) {
	if cb := callbacks; cb != nil && cb.GetWeaponInfo != nil {
		return cb.GetWeaponInfo(chName)
	}
	if GetWeaponInfo != nil {
		return GetWeaponInfo(chName)
	}
	return 0, 0, 0, false
}

func cbGetAdjacentRoom(roomVNum, door int) int {
	if cb := callbacks; cb != nil && cb.GetAdjacentRoom != nil {
		return cb.GetAdjacentRoom(roomVNum, door)
	}
	if GetAdjacentRoom != nil {
		return GetAdjacentRoom(roomVNum, door)
	}
	return -1
}

func cbGainExp(name string, amount int) {
	if cb := callbacks; cb != nil && cb.GainExp != nil {
		cb.GainExp(name, amount)
	} else if GainExp != nil {
		GainExp(name, amount)
	}
}

func cbGetExp(name string) int {
	if cb := callbacks; cb != nil && cb.GetExp != nil {
		return cb.GetExp(name)
	}
	if GetExp != nil {
		return GetExp(name)
	}
	return 0
}

func cbGetKills(name string) int64 {
	if cb := callbacks; cb != nil && cb.GetKills != nil {
		return cb.GetKills(name)
	}
	if GetKills != nil {
		return GetKills(name)
	}
	return 0
}

func cbSetKills(name string, kills int64) {
	if cb := callbacks; cb != nil && cb.SetKills != nil {
		cb.SetKills(name, kills)
	} else if SetKills != nil {
		SetKills(name, kills)
	}
}

func cbGetDeaths(name string) int64 {
	if cb := callbacks; cb != nil && cb.GetDeaths != nil {
		return cb.GetDeaths(name)
	}
	if GetDeaths != nil {
		return GetDeaths(name)
	}
	return 0
}

func cbSetDeaths(name string, deaths int64) {
	if cb := callbacks; cb != nil && cb.SetDeaths != nil {
		cb.SetDeaths(name, deaths)
	} else if SetDeaths != nil {
		SetDeaths(name, deaths)
	}
}

func cbSetLastDeath(name string, t int64) {
	if cb := callbacks; cb != nil && cb.SetLastDeath != nil {
		cb.SetLastDeath(name, t)
	} else if SetLastDeath != nil {
		SetLastDeath(name, t)
	}
}

func cbGetPks(name string) int64 {
	if cb := callbacks; cb != nil && cb.GetPks != nil {
		return cb.GetPks(name)
	}
	if GetPks != nil {
		return GetPks(name)
	}
	return 0
}

func cbSetPks(name string, pks int64) {
	if cb := callbacks; cb != nil && cb.SetPks != nil {
		cb.SetPks(name, pks)
	} else if SetPks != nil {
		SetPks(name, pks)
	}
}

func cbGetConstitution(name string) int {
	if cb := callbacks; cb != nil && cb.GetConstitution != nil {
		return cb.GetConstitution(name)
	}
	if GetConstitution != nil {
		return GetConstitution(name)
	}
	return 0
}

func cbSetConstitution(name string, val int) {
	if cb := callbacks; cb != nil && cb.SetConstitution != nil {
		cb.SetConstitution(name, val)
	} else if SetConstitution != nil {
		SetConstitution(name, val)
	}
}

func cbMakeCorpse(victim string, attackType int) {
	if cb := callbacks; cb != nil && cb.MakeCorpse != nil {
		cb.MakeCorpse(victim, attackType)
	} else if MakeCorpseFunc != nil {
		MakeCorpseFunc(victim, attackType)
	}
}

func cbMakeDust(victim string, attackType int) {
	if cb := callbacks; cb != nil && cb.MakeDust != nil {
		cb.MakeDust(victim, attackType)
	} else if MakeDustFunc != nil {
		MakeDustFunc(victim, attackType)
	}
}

func cbExtractChar(name string) {
	if cb := callbacks; cb != nil && cb.ExtractChar != nil {
		cb.ExtractChar(name)
	} else if ExtractChar != nil {
		ExtractChar(name)
	}
}

func cbRunDeathScript(killer, victim string, roomVNum int) {
	if cb := callbacks; cb != nil && cb.RunDeathScript != nil {
		cb.RunDeathScript(killer, victim, roomVNum)
	} else if RunDeathScript != nil {
		RunDeathScript(killer, victim, roomVNum)
	}
}
