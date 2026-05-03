// Package scripting provides Lua scripting support for Dark Pawns MUD.
package scripting

import "github.com/zax0rz/darkpawns/pkg/parser"

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
	// SetHunting sets the mob's hunting target by name.
	SetHunting(targetName string)
	// IsHunting returns true if the mob has an active hunting target.
	IsHunting() bool
	// SetFollowing sets the mob's follow target by name.
	SetFollowing(leaderName string)
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
	// GetRoomInWorld returns a room by VNum, or nil if not found.
	GetRoomInWorld(vnum int) *parser.Room
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
	// IsRoomDark returns true if the given room VNum is dark.
	// Based on utils.h IS_DARK() macro — checks ROOM_DARK flag.
	IsRoomDark(roomVNum int) bool
	// GetRoomZone returns the zone number for a given room VNum.
	GetRoomZone(roomVNum int) int
	// SendToZone sends a message to all players in the same zone as the given room.
	// Source: comm.c send_to_zone().
	SendToZone(roomVNum int, msg string)
	// SendToAll sends a message to all online players.
	// Source: comm.c send_to_all().
	SendToAll(msg string)
	// ExecuteMobCommand makes a mob execute a game command.
	// Source: scripts.c lua_action() → command_interpreter().
	ExecuteMobCommand(mobVNum int, cmdStr string)
	// FindFirstStep returns direction (0-5) from src room to target room.
	// Returns -1 on error, -2 if already there, -3 if no path.
	FindFirstStep(src, target int) int
	// CreateEvent schedules a timed event on the world's event queue.
	// delay is in game pulses (1 pulse = 1/10 second in original C).
	// trigger is the Lua function name to call when the event fires.
	// eventType is LT_MOB (1), LT_OBJ (2), or LT_ROOM (3) from structs.h.
	// Returns the event ID, or 0 if the event could not be scheduled.
	// Source: scripts.c lua_create_event() — create_event(source, target, obj, argument, trigger, delay, type)
	CreateEvent(delay int, source, target, obj, argument int, trigger string, eventType int) uint64

	// --- STATE MUTATIONS & LOOKUPS (lua_batch2_mutations.go) ---

	// EquipChar equips an object (by vnum) on a character.
	// Returns true on success.
	// Source: scripts.c lua_equip_char()
	EquipChar(charName string, isMob bool, objVNum int) bool

	// SetFollower makes followerName follow leaderName.
	// followerIsMob indicates whether the follower is a mob (true) or player (false).
	// Source: scripts.c lua_follow()
	SetFollower(followerName, leaderName string, followerIsMob bool) error

	// MountPlayer sets a player's mount name.
	// Source: scripts.c lua_mount()
	MountPlayer(playerName, mountName string) error

	// DismountPlayer clears a player's mount.
	// Source: scripts.c lua_mount()
	DismountPlayer(playerName string) error

	// ClearAffects removes all affects from a character.
	// Source: scripts.c lua_unaffect()
	ClearAffects(charName string, isMob bool)
	// EquipMob equips an object on a mob by vnums.
	// Source: scripts.c lua_equip_char() lines 403-425.
	EquipMob(mobVNum, roomVNum, objVNum int)

	// CanCarryObject returns true if the named character can carry the object.
	// Source: scripts.c lua_canget()
	CanCarryObject(charName string, objVNum int) bool

	// IsCorpseObj returns true if the object prototype is a corpse.
	// Source: scripts.c lua_iscorpse()
	IsCorpseObj(objVNum int) bool

	// SetHunting sets a character's hunting target.
	// Source: scripts.c lua_set_hunt()
	SetHunting(hunterName, preyName string, hunterIsMob bool)

	// IsHunting returns true if the character is hunting.
	// Source: scripts.c lua_ishunt()
	IsHunting(charName string, isMob bool) bool

	// GetPlayerByID returns a player by instance ID, or nil if not found.
	GetPlayerByID(id int) ScriptablePlayer
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
