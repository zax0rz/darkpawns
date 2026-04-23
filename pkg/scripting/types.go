// Package scripting provides Lua scripting support for Dark Pawns MUD.
package scripting

// ScriptablePlayer represents a player that can be exposed to Lua scripts.
type ScriptablePlayer interface {
	GetID() int
	GetName() string
	GetLevel() int
	GetHealth() int
	SetHealth(int)
	GetMaxHealth() int
	GetGold() int
	SetGold(int)
	GetRace() int
	GetClass() int
	GetAlignment() int
	GetRoomVNum() int
	SendMessage(string)
	// GetFlags returns the raw PLR flags bitmask.
	// Source: structs.h PLR_FLAGS, utils.h PLR_FLAGGED() macro.
	GetFlags() uint64
}

// ScriptableMob represents a mob that can be exposed to Lua scripts.
type ScriptableMob interface {
	GetVNum() int
	GetName() string
	GetLevel() int
	GetHealth() int
	SetHealth(int)
	GetMaxHealth() int
	GetGold() int
	GetRoomVNum() int
	GetPrototype() ScriptableMobPrototype
	GetFighting() string // Returns the name of the mob's combat target, or "" if not fighting
}

// ScriptableMobPrototype represents mob prototype data.
type ScriptableMobPrototype interface {
	GetShortDesc() string
	GetGold() int
	GetLevel() int
	GetAlignment() int
	GetScriptName() string
	GetLuaFunctions() int
}

// ScriptableObject represents an object that can be exposed to Lua scripts.
type ScriptableObject interface {
	GetVNum() int
	GetKeywords() string
	GetShortDesc() string
	GetCost() int
	GetTimer() int
	SetTimer(int)
}

// ScriptableWorld represents the game world for script context.
type ScriptableWorld interface {
	// GetPlayersInRoom returns all players in a given room.
	GetPlayersInRoom(roomVNum int) []ScriptablePlayer
	// GetMobsInRoom returns all mobs in a given room.
	GetMobsInRoom(roomVNum int) []ScriptableMob
	// GetMobByVNumAndRoom returns a mob by its vnum and room.
	GetMobByVNumAndRoom(vnum int, roomVNum int) ScriptableMob
	// GetObjPrototype returns an object prototype by vnum.
	GetObjPrototype(vnum int) ScriptableObject
	// AddItemToRoom adds an item to a room.
	AddItemToRoom(obj ScriptableObject, roomVNum int) error
	// HandleNonCombatDeath handles player death from non-combat damage.
	HandleNonCombatDeath(player ScriptablePlayer)
	// HandleSpellDeath handles death caused by a spell.
	HandleSpellDeath(victimName string, spellNum int, roomVNum int)
	// SendTell delivers a private tell message to a named player.
	// Source: act.comm.c do_tell().
	SendTell(targetName, message string)
	// GetItemsInRoom returns all items in a given room as ScriptableObject.
	GetItemsInRoom(roomVNum int) []ScriptableObject
	// HasItemByVNum returns true if the named character has an item with the given vnum.
	HasItemByVNum(charName string, vnum int) bool
	// RemoveItemFromRoom removes the first item with the given vnum from the room and returns it.
	RemoveItemFromRoom(vnum int, roomVNum int) ScriptableObject
	// RemoveItemFromChar removes the first item with the given vnum from the character's inventory.
	RemoveItemFromChar(charName string, vnum int) ScriptableObject
	// GiveItemToChar adds an item to the named character's inventory.
	GiveItemToChar(charName string, obj ScriptableObject) error
	// CreateEvent schedules a timed event on the world's event queue.
	// delay is in game pulses (1 pulse = 1/10 second in original C).
	// trigger is the Lua function name to call when the event fires.
	// eventType is LT_MOB (1), LT_OBJ (2), or LT_ROOM (3) from structs.h.
	// Returns the event ID, or 0 if the event could not be scheduled.
	// Source: scripts.c lua_create_event() — create_event(source, target, obj, argument, trigger, delay, type)
	CreateEvent(delay int, source, target, obj, argument int, trigger string, eventType int) uint64
}

// ScriptContext holds the game objects exposed to Lua as globals.
type ScriptContext struct {
	Ch       ScriptablePlayer
	Me       ScriptableMob
	Obj      ScriptableObject
	RoomVNum int
	Argument string
	World    ScriptableWorld
}
