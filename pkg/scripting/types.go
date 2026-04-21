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