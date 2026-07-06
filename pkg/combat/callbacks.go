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
