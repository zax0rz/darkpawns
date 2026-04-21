// Package scripting provides Lua scripting support for Dark Pawns MUD.
// Based on original C code from scripts.c.
package scripting

import (
	"fmt"
	"log"
	"math/rand"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// Engine manages the Lua VM.
// Based on boot_lua() in scripts.c lines 1703-1716.
type Engine struct {
	scriptsDir string
	L          *lua.LState
	world      ScriptableWorld
}

// NewEngine creates a new Lua scripting engine.
func NewEngine(scriptsDir string, world ScriptableWorld) *Engine {
	L := lua.NewState()
	engine := &Engine{
		scriptsDir: scriptsDir,
		L:          L,
		world:      world,
	}

	// Open standard libraries
	L.OpenLibs()

	// Register our custom functions
	engine.registerFunctions()

	// Load globals.lua
	engine.loadGlobals()

	return engine
}

// RunScript loads and executes a named trigger function in a script file.
// fname is relative to scriptsDir (e.g. "mob/144/hisc.lua").
// triggerName is the function to call (e.g. "oncmd", "sound", "fight").
// Returns true if the script handled the event (returned TRUE), false otherwise.
// Based on run_script() in scripts.c lines 1718-1810.
func (e *Engine) RunScript(ctx *ScriptContext, fname string, triggerName string) (bool, error) {
	// Set globals based on context
	// Based on run_script() lines 1732-1761
	if ctx.Ch != nil {
		e.charToTable(ctx.Ch, "ch")
		log.Printf("[SCRIPT] Set ch global for player %s", ctx.Ch.GetName())
	} else {
		log.Printf("[SCRIPT] ctx.Ch is nil")
	}
	if ctx.Me != nil {
		e.mobToTable(ctx.Me, "me")
	}
	if ctx.Obj != nil {
		e.objToTable(ctx.Obj, "obj")
	}
	if ctx.Argument != "" {
		e.L.SetGlobal("argument", lua.LString(ctx.Argument))
	}

	// Set room global if we have room vnum
	if ctx.RoomVNum > 0 {
		// Create a room table with vnum and char array
		roomTbl := e.L.NewTable()
		roomTbl.RawSetString("vnum", lua.LNumber(ctx.RoomVNum))
		
		// Create empty char table (for now)
		// In real implementation, this would contain characters in the room
		charTbl := e.L.NewTable()
		roomTbl.RawSetString("char", charTbl)
		
		e.L.SetGlobal("room", roomTbl)
	}

	// Load and execute the script file
	// Based on open_lua_file() in scripts.c lines 1641-1701
	scriptPath := e.scriptsDir + "/" + fname
	if err := e.L.DoFile(scriptPath); err != nil {
		log.Printf("[Lua] Error loading script %s: %v", fname, err)
		return false, err
	}

	// Call the trigger function
	// Based on run_script() lines 1780-1795
	fn := e.L.GetGlobal(triggerName)
	log.Printf("[SCRIPT] Calling function %s, type: %v", triggerName, fn.Type())
	e.L.Push(fn)
	if fn.Type() == lua.LTNil {
		// Function doesn't exist
		e.L.Pop(1)
		log.Printf("[SCRIPT] Function %s not found in script %s", triggerName, fname)
		return false, nil
	}

	if err := e.L.PCall(0, 1, nil); err != nil {
		log.Printf("[Lua] Error calling function %s in script %s: %v", triggerName, fname, err)
		// Pop the error from stack if it exists
		if e.L.GetTop() > 0 {
			e.L.Pop(1)
		}
		return false, err
	}

	// Get return value (0 or 1 results)
	// Lua functions return 0 values by default, we treat that as false (not handled)
	stackTop := e.L.GetTop()
	log.Printf("[SCRIPT] Stack top after PCall: %d", stackTop)
	
	var ret lua.LValue = lua.LFalse
	if stackTop > 0 {
		ret = e.L.Get(-1)
		log.Printf("[SCRIPT] Function returned: type=%v, value=%v", ret.Type(), ret)
		e.L.Pop(1)
	}

	// Read back changes from tables
	// Based on run_script() lines 1797-1808
	if ctx.Ch != nil {
		log.Printf("[SCRIPT] Reading back ch changes, stack top: %d", e.L.GetTop())
		// Get the global value
		chVal := e.L.GetGlobal("ch")
		log.Printf("[SCRIPT] ch global type: %v", chVal.Type())
		if chVal.Type() != lua.LTNil {
			// Push onto stack for tableToChar
			e.L.Push(chVal)
			e.tableToChar(ctx.Ch)
			log.Printf("[SCRIPT] After tableToChar, stack top: %d", e.L.GetTop())
			// tableToChar should leave the table on stack, pop it
			if e.L.GetTop() > 0 {
				e.L.Pop(1)
			}
		}
	}

	if ctx.Me != nil {
		// Get the global value
		meVal := e.L.GetGlobal("me")
		log.Printf("[SCRIPT] me global type: %v", meVal.Type())
		if meVal.Type() != lua.LTNil {
			// Push onto stack for tableToMob
			e.L.Push(meVal)
			e.tableToMob(ctx.Me)
			// tableToMob should leave the table on stack, pop it
			if e.L.GetTop() > 0 {
				e.L.Pop(1)
			}
		}
	}

	// Check return value
	if ret.Type() == lua.LTNumber {
		return lua.LVAsNumber(ret) == 1, nil
	}
	return false, nil
}

// registerFunctions registers all Lua API functions.
// Based on cmdlib array in scripts.c lines 1609-1668.
func (e *Engine) registerFunctions() {
	// Core functions mentioned in the task
	e.L.SetGlobal("act", e.L.NewFunction(e.luaAct))
	e.L.SetGlobal("do_damage", e.L.NewFunction(e.luaDoDamage))
	e.L.SetGlobal("say", e.L.NewFunction(e.luaSay))
	e.L.SetGlobal("emote", e.L.NewFunction(e.luaEmote))
	e.L.SetGlobal("action", e.L.NewFunction(e.luaAction))
	e.L.SetGlobal("oload", e.L.NewFunction(e.luaOload))
	e.L.SetGlobal("mload", e.L.NewFunction(e.luaMload))
	e.L.SetGlobal("extobj", e.L.NewFunction(e.luaExtobj))
	e.L.SetGlobal("extchar", e.L.NewFunction(e.luaExtchar))
	e.L.SetGlobal("number", e.L.NewFunction(e.luaNumber))
	e.L.SetGlobal("send_to_room", e.L.NewFunction(e.luaSendToRoom))
	e.L.SetGlobal("strlower", e.L.NewFunction(e.luaStrlower))
	e.L.SetGlobal("strfind", e.L.NewFunction(e.luaStrfind))
	e.L.SetGlobal("strsub", e.L.NewFunction(e.luaStrsub))
	e.L.SetGlobal("gsub", e.L.NewFunction(e.luaGsub))
	e.L.SetGlobal("getn", e.L.NewFunction(e.luaGetn))
	e.L.SetGlobal("tonumber", e.L.NewFunction(e.luaTonumber))
	// Don't override tostring - it's a Lua built-in
	// e.L.SetGlobal("tostring", e.L.NewFunction(e.luaTostring))

	// Additional functions from cmdlib that might be needed
	e.L.SetGlobal("log", e.L.NewFunction(e.luaLog))
	e.L.SetGlobal("raw_kill", e.L.NewFunction(e.luaRawKill))
	e.L.SetGlobal("save_char", e.L.NewFunction(e.luaSaveChar))
	e.L.SetGlobal("save_obj", e.L.NewFunction(e.luaSaveObj))
	e.L.SetGlobal("save_room", e.L.NewFunction(e.luaSaveRoom))
	e.L.SetGlobal("set_skill", e.L.NewFunction(e.luaSetSkill))
	e.L.SetGlobal("spell", e.L.NewFunction(e.luaSpell))
	e.L.SetGlobal("tport", e.L.NewFunction(e.luaTport))
	
	// Additional functions needed for combat AI scripts
	e.L.SetGlobal("isfighting", e.L.NewFunction(e.luaIsFighting))
	e.L.SetGlobal("round", e.L.NewFunction(e.luaRound))
}

// loadGlobals loads the globals.lua file.
// Based on boot_lua() lines 1711-1714.
func (e *Engine) loadGlobals() {
	globalsPath := e.scriptsDir + "/globals.lua"
	log.Printf("[SCRIPT] Loading globals from %s", globalsPath)
	if err := e.L.DoFile(globalsPath); err != nil {
		log.Printf("[Lua] Warning: Could not load globals.lua: %v", err)
	} else {
		log.Printf("[SCRIPT] globals.lua loaded successfully")
	}
	// Always set up basic constants
	e.setupBasicConstants()
}

// setupBasicConstants sets up essential constants when globals.lua is missing.
func (e *Engine) setupBasicConstants() {
	// Direction constants
	e.L.SetGlobal("NORTH", lua.LNumber(0))
	e.L.SetGlobal("EAST", lua.LNumber(1))
	e.L.SetGlobal("SOUTH", lua.LNumber(2))
	e.L.SetGlobal("WEST", lua.LNumber(3))
	e.L.SetGlobal("UP", lua.LNumber(4))
	e.L.SetGlobal("DOWN", lua.LNumber(5))
	
	// Message types for act()
	e.L.SetGlobal("TO_ROOM", lua.LNumber(1))
	e.L.SetGlobal("TO_VICT", lua.LNumber(2))
	e.L.SetGlobal("TO_NOTVICT", lua.LNumber(3))
	e.L.SetGlobal("TO_CHAR", lua.LNumber(4))
	
	// Boolean constants
	e.L.SetGlobal("TRUE", lua.LNumber(1))
	e.L.SetGlobal("FALSE", lua.LNumber(0))
	e.L.SetGlobal("NIL", lua.LNil)
	
	// Level constants
	e.L.SetGlobal("LVL_IMMORT", lua.LNumber(31))
	e.L.SetGlobal("LVL_IMPL", lua.LNumber(40))
	
	// Player flags
	e.L.SetGlobal("PLR_OUTLAW", lua.LNumber(0))
	e.L.SetGlobal("PLR_WEREWOLF", lua.LNumber(16))
	e.L.SetGlobal("PLR_VAMPIRE", lua.LNumber(17))
	
	// Mob flags
	e.L.SetGlobal("MOB_SENTINEL", lua.LNumber(1))
	e.L.SetGlobal("MOB_HUNTER", lua.LNumber(18))
	e.L.SetGlobal("MOB_MOUNTABLE", lua.LNumber(21))
	
	// Affect flags
	e.L.SetGlobal("AFF_DETECT_MAGIC", lua.LNumber(4))
	e.L.SetGlobal("AFF_GROUP", lua.LNumber(8))
	e.L.SetGlobal("AFF_POISON", lua.LNumber(11))
	e.L.SetGlobal("AFF_CHARM", lua.LNumber(21))
	e.L.SetGlobal("AFF_FLY", lua.LNumber(26))
	e.L.SetGlobal("AFF_WEREWOLF", lua.LNumber(27))
	e.L.SetGlobal("AFF_VAMPIRE", lua.LNumber(28))
	e.L.SetGlobal("AFF_MOUNT", lua.LNumber(29))
	
	// Position constants
	e.L.SetGlobal("POS_DEAD", lua.LNumber(0))
	e.L.SetGlobal("POS_MORTALLYW", lua.LNumber(1))
	e.L.SetGlobal("POS_INCAP", lua.LNumber(2))
	e.L.SetGlobal("POS_STUNNED", lua.LNumber(3))
	e.L.SetGlobal("POS_SLEEPING", lua.LNumber(4))
	e.L.SetGlobal("POS_RESTING", lua.LNumber(5))
	e.L.SetGlobal("POS_SITTING", lua.LNumber(6))
	e.L.SetGlobal("POS_STANDING", lua.LNumber(8))
	
	// Item type constants
	e.L.SetGlobal("ITEM_STAFF", lua.LNumber(4))
	e.L.SetGlobal("ITEM_WEAPON", lua.LNumber(5))
	e.L.SetGlobal("ITEM_ARMOR", lua.LNumber(9))
	e.L.SetGlobal("ITEM_WORN", lua.LNumber(11))
	e.L.SetGlobal("ITEM_TRASH", lua.LNumber(13))
	e.L.SetGlobal("ITEM_NOTE", lua.LNumber(16))
	e.L.SetGlobal("ITEM_DRINKCON", lua.LNumber(17))
	e.L.SetGlobal("ITEM_KEY", lua.LNumber(18))
	e.L.SetGlobal("ITEM_FOOD", lua.LNumber(19))
	e.L.SetGlobal("ITEM_PEN", lua.LNumber(21))
	
	// Object extra flags
	e.L.SetGlobal("ITEM_GLOW", lua.LNumber(0))
	e.L.SetGlobal("ITEM_MAGIC", lua.LNumber(6))
	e.L.SetGlobal("ITEM_NODROP", lua.LNumber(7))
	e.L.SetGlobal("ITEM_NOSELL", lua.LNumber(16))
	
	// Item wear positions
	e.L.SetGlobal("ITEM_WEAR_TAKE", lua.LNumber(0))
	
	// Spell constants (from spells.h and globals.lua)
	e.L.SetGlobal("SPELL_TELEPORT", lua.LNumber(2))
	e.L.SetGlobal("SPELL_BLINDNESS", lua.LNumber(4))
	e.L.SetGlobal("SPELL_BURNING_HANDS", lua.LNumber(5))
	e.L.SetGlobal("SPELL_CHARM", lua.LNumber(7))
	e.L.SetGlobal("SPELL_COLOR_SPRAY", lua.LNumber(10))
	e.L.SetGlobal("SPELL_CURE_LIGHT", lua.LNumber(16))
	e.L.SetGlobal("SPELL_CURSE", lua.LNumber(17))
	e.L.SetGlobal("SPELL_DISPEL_EVIL", lua.LNumber(22))
	e.L.SetGlobal("SPELL_EARTHQUAKE", lua.LNumber(23))
	e.L.SetGlobal("SPELL_ENCHANT_WEAPON", lua.LNumber(24))
	e.L.SetGlobal("SPELL_FIREBALL", lua.LNumber(26))
	e.L.SetGlobal("SPELL_HARM", lua.LNumber(27))
	e.L.SetGlobal("SPELL_HEAL", lua.LNumber(28))
	e.L.SetGlobal("SPELL_LIGHTNING_BOLT", lua.LNumber(30))
	e.L.SetGlobal("SPELL_MAGIC_MISSILE", lua.LNumber(32))
	e.L.SetGlobal("SPELL_POISON", lua.LNumber(33))
	e.L.SetGlobal("SPELL_SANCTUARY", lua.LNumber(36))
	e.L.SetGlobal("SPELL_SHOCKING_GRASP", lua.LNumber(37))
	e.L.SetGlobal("SPELL_SLEEP", lua.LNumber(38))
	e.L.SetGlobal("SPELL_METEOR_SWARM", lua.LNumber(41))
	e.L.SetGlobal("SPELL_WORD_OF_RECALL", lua.LNumber(42))
	e.L.SetGlobal("SPELL_REMOVE_POISON", lua.LNumber(43))
	e.L.SetGlobal("SPELL_DISPEL_GOOD", lua.LNumber(46))
	e.L.SetGlobal("SPELL_HELLFIRE", lua.LNumber(58))
	e.L.SetGlobal("SPELL_ENCHANT_ARMOR", lua.LNumber(59))
	e.L.SetGlobal("SPELL_IDENTIFY", lua.LNumber(60))
	e.L.SetGlobal("SPELL_MINDBLAST", lua.LNumber(62))
	e.L.SetGlobal("SPELL_INVULNERABILITY", lua.LNumber(66))
	e.L.SetGlobal("SPELL_VITALITY", lua.LNumber(67))
	e.L.SetGlobal("SPELL_ACID_BLAST", lua.LNumber(75))
	e.L.SetGlobal("SPELL_DIVINE_INT", lua.LNumber(81))
	e.L.SetGlobal("SPELL_MIND_BAR", lua.LNumber(82))
	e.L.SetGlobal("SPELL_SOUL_LEECH", lua.LNumber(83))
	e.L.SetGlobal("SPELL_DISRUPT", lua.LNumber(92))
	e.L.SetGlobal("SPELL_DISINTEGRATE", lua.LNumber(93))
	e.L.SetGlobal("SPELL_FLAMESTRIKE", lua.LNumber(96))
	e.L.SetGlobal("SPELL_PSIBLAST", lua.LNumber(100))
	e.L.SetGlobal("SPELL_PETRIFY", lua.LNumber(104))
	
	// Dragon Breath spells
	e.L.SetGlobal("SPELL_FIRE_BREATH", lua.LNumber(202))
	e.L.SetGlobal("SPELL_GAS_BREATH", lua.LNumber(203))
	e.L.SetGlobal("SPELL_FROST_BREATH", lua.LNumber(204))
	e.L.SetGlobal("SPELL_ACID_BREATH", lua.LNumber(205))
	e.L.SetGlobal("SPELL_LIGHTNING_BREATH", lua.LNumber(206))
	
	// Skill constants
	e.L.SetGlobal("SKILL_BASH", lua.LNumber(132))
	e.L.SetGlobal("SKILL_HEADBUTT", lua.LNumber(141))
	e.L.SetGlobal("SKILL_BERSERK", lua.LNumber(171))
	e.L.SetGlobal("SKILL_PARRY", lua.LNumber(172))
	e.L.SetGlobal("SKILL_KICK", lua.LNumber(134))
	e.L.SetGlobal("SKILL_TRIP", lua.LNumber(144))
	
	// Raw kill types
	e.L.SetGlobal("TYPE_UNDEFINED", lua.LNumber(-1))
	
	// Sector types
	e.L.SetGlobal("SECT_FOREST", lua.LNumber(3))
	e.L.SetGlobal("SECT_UNDERWATER", lua.LNumber(8))
	e.L.SetGlobal("SECT_FIRE", lua.LNumber(11))
	e.L.SetGlobal("SECT_EARTH", lua.LNumber(12))
	e.L.SetGlobal("SECT_WIND", lua.LNumber(13))
	e.L.SetGlobal("SECT_WATER", lua.LNumber(14))
	
	// Exit flags
	e.L.SetGlobal("EX_ISDOOR", lua.LNumber(0))
	e.L.SetGlobal("EX_CLOSED", lua.LNumber(1))
	e.L.SetGlobal("EX_LOCKED", lua.LNumber(2))
	e.L.SetGlobal("EX_PICKPROOF", lua.LNumber(3))
	
	// Lua script flags
	e.L.SetGlobal("LT_MOB", lua.LString("mob"))
	e.L.SetGlobal("LT_OBJ", lua.LString("obj"))
	e.L.SetGlobal("LT_ROOM", lua.LString("room"))
}

// charToTable converts a ScriptablePlayer to a Lua table.
// Based on char_to_table() in scripts.c lines 1812-1916.
func (e *Engine) charToTable(player ScriptablePlayer, globalName string) {
	L := e.L
	tbl := L.NewTable()

	// Basic fields
	tbl.RawSetString("name", lua.LString(player.GetName()))
	tbl.RawSetString("level", lua.LNumber(player.GetLevel()))
	tbl.RawSetString("hp", lua.LNumber(player.GetHealth()))
	tbl.RawSetString("maxhp", lua.LNumber(player.GetMaxHealth()))
	tbl.RawSetString("gold", lua.LNumber(player.GetGold()))
	tbl.RawSetString("race", lua.LNumber(player.GetRace()))
	tbl.RawSetString("class", lua.LNumber(player.GetClass()))
	tbl.RawSetString("alignment", lua.LNumber(player.GetAlignment()))
	tbl.RawSetString("room", lua.LNumber(player.GetRoomVNum()))
	
	// Evil property (based on alignment - negative = evil)
	alignment := player.GetAlignment()
	evil := 0 // FALSE
	if alignment < 0 {
		evil = 1 // TRUE
	}
	tbl.RawSetString("evil", lua.LNumber(evil))

	// Skills table (stub for now)
	skillsTbl := L.NewTable()
	tbl.RawSetString("skills", skillsTbl)

	// Store pointer to struct for write-back
	tbl.RawSetString("struct", lua.LNumber(player.GetID()))

	L.SetGlobal(globalName, tbl)
	log.Printf("[SCRIPT] Set global %s with level=%d", globalName, player.GetLevel())
}

// mobToTable converts a ScriptableMob to a Lua table.
// Based on char_to_table() for NPCs in scripts.c lines 1904-1910.
func (e *Engine) mobToTable(mob ScriptableMob, globalName string) {
	L := e.L
	tbl := L.NewTable()

	proto := mob.GetPrototype()
	
	// Basic fields
	tbl.RawSetString("name", lua.LString(proto.GetShortDesc()))
	tbl.RawSetString("level", lua.LNumber(proto.GetLevel()))
	tbl.RawSetString("hp", lua.LNumber(mob.GetHealth()))
	tbl.RawSetString("maxhp", lua.LNumber(mob.GetMaxHealth()))
	tbl.RawSetString("vnum", lua.LNumber(mob.GetVNum()))
	tbl.RawSetString("gold", lua.LNumber(proto.GetGold()))
	tbl.RawSetString("room", lua.LNumber(mob.GetRoomVNum()))
	
	// Evil property (based on alignment - negative = evil)
	// In Dark Pawns, alignment ranges from -1000 to +1000
	// Negative alignment = evil (TRUE), positive = good (FALSE)
	alignment := proto.GetAlignment()
	evil := 0 // FALSE
	if alignment < 0 {
		evil = 1 // TRUE
	}
	tbl.RawSetString("evil", lua.LNumber(evil))
	
	// Wear property (array of worn items) - placeholder empty table
	wearTbl := L.NewTable()
	tbl.RawSetString("wear", wearTbl)

	// Store pointer to struct for write-back
	tbl.RawSetString("struct", lua.LNumber(mob.GetVNum()))

	L.SetGlobal(globalName, tbl)
}

// objToTable converts a ScriptableObject to a Lua table.
// Based on obj_to_table() in scripts.c lines 1918-2016.
func (e *Engine) objToTable(obj ScriptableObject, globalName string) {
	L := e.L
	tbl := L.NewTable()

	// Basic fields
	tbl.RawSetString("vnum", lua.LNumber(obj.GetVNum()))
	tbl.RawSetString("alias", lua.LString(obj.GetKeywords()))
	tbl.RawSetString("name", lua.LString(obj.GetShortDesc()))
	tbl.RawSetString("cost", lua.LNumber(obj.GetCost()))
	tbl.RawSetString("timer", lua.LNumber(obj.GetTimer()))

	// Store pointer to struct for write-back
	tbl.RawSetString("struct", lua.LNumber(obj.GetVNum()))

	L.SetGlobal(globalName, tbl)
}

// tableToChar reads back changes from the ch table to the ScriptablePlayer.
// Based on table_to_char() in scripts.c lines 2018-2116.
func (e *Engine) tableToChar(player ScriptablePlayer) {
	L := e.L
	log.Printf("[tableToChar] Stack top: %d", L.GetTop())
	tbl := L.Get(-1)
	log.Printf("[tableToChar] tbl type: %v", tbl.Type())

	if tbl.Type() != lua.LTTable {
		log.Printf("[tableToChar] Not a table, returning")
		return
	}

	// Read hp changes
	hpVal := L.GetField(tbl, "hp")
	log.Printf("[tableToChar] hp field: %v (type: %v)", hpVal, hpVal.Type())
	if hpVal.Type() == lua.LTNumber {
		player.SetHealth(int(hpVal.(lua.LNumber)))
	}

	// Read gold changes
	goldVal := L.GetField(tbl, "gold")
	log.Printf("[tableToChar] gold field: %v (type: %v)", goldVal, goldVal.Type())
	if goldVal.Type() == lua.LTNumber {
		player.SetGold(int(goldVal.(lua.LNumber)))
	}
}

// tableToMob reads back changes from the me table to the ScriptableMob.
// Based on table_to_char() for NPCs in scripts.c lines 2040-2045.
func (e *Engine) tableToMob(mob ScriptableMob) {
	L := e.L
	tbl := L.Get(-1)

	if tbl.Type() != lua.LTTable {
		return
	}

	// Read hp changes
	L.GetField(tbl, "hp")
	if val := L.Get(-1); val.Type() == lua.LTNumber {
		mob.SetHealth(int(lua.LVAsNumber(val)))
	}
	L.Pop(1)
}

// Lua function implementations
// Based on corresponding functions in scripts.c

func (e *Engine) luaAct(L *lua.LState) int {
	// act(msg, visible, ch, obj, vict, type)
	// Based on lua_act() in scripts.c lines 79-124
	msg := L.ToString(1)
	_ = L.ToBool(2) // visible - unused for now
	where := L.ToInt(6)

	// Get room vnum from context
	var roomVNum int
	roomVal := L.GetGlobal("room")
	if roomVal.Type() == lua.LTNumber {
		roomVNum = int(roomVal.(lua.LNumber))
	}

	// Get me from global for room context
	L.GetGlobal("me")
	if L.Get(-1).Type() == lua.LTTable {
		L.GetField(L.Get(-1), "room")
		if L.Get(-1).Type() == lua.LTNumber {
			roomVNum = int(L.ToNumber(-1))
		}
		L.Pop(1)
	}
	L.Pop(1)

	// Get ch from global for TO_VICT/TO_CHAR
	L.GetGlobal("ch")
	var ch ScriptablePlayer = nil
	if L.Get(-1).Type() == lua.LTTable {
		// In real implementation, we'd get the actual player pointer
		// For now, we'll use the world to find players
	}
	L.Pop(1)

	if e.world == nil || roomVNum == 0 {
		log.Printf("[ACT] No world or room context: %s (type: %d)", msg, where)
		return 0
	}

	players := e.world.GetPlayersInRoom(roomVNum)
	
	switch where {
	case 1: // TO_ROOM
		for _, player := range players {
			player.SendMessage(msg + "\r\n")
		}
	case 2: // TO_VICT
		if ch != nil {
			ch.SendMessage(msg + "\r\n")
		}
	case 3: // TO_NOTVICT
		for _, player := range players {
			if ch == nil || player.GetID() != ch.GetID() {
				player.SendMessage(msg + "\r\n")
			}
		}
	case 4: // TO_CHAR
		if ch != nil {
			ch.SendMessage(msg + "\r\n")
		}
	}

	return 0
}

func (e *Engine) luaDoDamage(L *lua.LState) int {
	// do_damage(amount)
	// Based on pattern_dmg.lua example
	amount := L.ToInt(1)

	// Get ch from global
	L.GetGlobal("ch")
	if L.Get(-1).Type() == lua.LTTable {
		// Apply damage to ch
		L.GetField(L.Get(-1), "hp")
		if L.Get(-1).Type() == lua.LTNumber {
			currentHP := int(L.ToNumber(-1))
			newHP := currentHP - amount
			if newHP < 0 {
				newHP = 0
			}
			L.Pop(1) // pop hp value
			
			// Update hp in table
			L.Push(lua.LNumber(newHP))
			L.SetField(L.Get(-2), "hp", lua.LNumber(newHP))
			
			// Check for death
			if newHP <= 0 && e.world != nil {
				// Get the actual player object from the world
				// For now, just log
				log.Printf("[DO_DAMAGE] Player would die from %d damage", amount)
			}
		} else {
			L.Pop(1)
		}
	}
	L.Pop(1)

	return 0
}

func (e *Engine) luaSay(L *lua.LState) int {
	// say(msg)
	// Based on lua_say() (not shown in snippets but referenced)
	msg := L.ToString(1)
	
	// Get me from global for room context
	var roomVNum int
	L.GetGlobal("me")
	if L.Get(-1).Type() == lua.LTTable {
		L.GetField(L.Get(-1), "room")
		if L.Get(-1).Type() == lua.LTNumber {
			roomVNum = int(L.ToNumber(-1))
		}
		L.Pop(1)
	}
	L.Pop(1)
	
	if e.world == nil || roomVNum == 0 {
		log.Printf("[SAY] No world or room context: %s", msg)
		return 0
	}
	
	// Format message: "mob says 'message'"
	L.GetGlobal("me")
	var mobName string = "someone"
	if L.Get(-1).Type() == lua.LTTable {
		L.GetField(L.Get(-1), "name")
		if L.Get(-1).Type() == lua.LTString {
			mobName = L.ToString(-1)
		}
		L.Pop(1)
	}
	L.Pop(1)
	
	formattedMsg := mobName + " says '" + msg + "'\r\n"
	players := e.world.GetPlayersInRoom(roomVNum)
	for _, player := range players {
		player.SendMessage(formattedMsg)
	}
	
	return 0
}

func (e *Engine) luaEmote(L *lua.LState) int {
	// emote(msg)
	// Based on lua_emote() in scripts.c lines 291-306
	msg := L.ToString(1)
	
	// Get me from global for room context
	var roomVNum int
	L.GetGlobal("me")
	if L.Get(-1).Type() == lua.LTTable {
		L.GetField(L.Get(-1), "room")
		if L.Get(-1).Type() == lua.LTNumber {
			roomVNum = int(L.ToNumber(-1))
		}
		L.Pop(1)
	}
	L.Pop(1)
	
	if e.world == nil || roomVNum == 0 {
		log.Printf("[EMOTE] No world or room context: %s", msg)
		return 0
	}
	
	// Format message: "mob message"
	L.GetGlobal("me")
	var mobName string = "someone"
	if L.Get(-1).Type() == lua.LTTable {
		L.GetField(L.Get(-1), "name")
		if L.Get(-1).Type() == lua.LTString {
			mobName = L.ToString(-1)
		}
		L.Pop(1)
	}
	L.Pop(1)
	
	formattedMsg := mobName + " " + msg + "\r\n"
	players := e.world.GetPlayersInRoom(roomVNum)
	for _, player := range players {
		player.SendMessage(formattedMsg)
	}
	
	return 0
}

func (e *Engine) luaAction(L *lua.LState) int {
	// action(mob, cmdstr)
	// Based on lua_action() in scripts.c lines 126-144
	// mob would be table, cmdstr is string
	// In real implementation: mob executes command
	
	// Get parameters
	mobTbl := L.Get(1)
	cmdStr := L.ToString(2)
	
	// Log the action for now
	// TODO: Implement actual command execution for mobs
	mobName := "unknown"
	
	if mobTbl.Type() == lua.LTTable {
		L.GetField(mobTbl, "name")
		if L.Get(-1).Type() == lua.LTString {
			mobName = L.ToString(-1)
		}
		L.Pop(1)
	}
	
	log.Printf("[ACTION] %s executes command: %s", mobName, cmdStr)
	
	return 0
}

func (e *Engine) luaOload(L *lua.LState) int {
	// oload(target, vnum, location)
	// Based on lua_oload() in scripts.c lines 1047-1090
	// target is ch table, vnum is number, location is string
	vnum := L.ToInt(2)
	location := L.ToString(3)
	
	if e.world == nil {
		log.Printf("[OLOAD] No world context: vnum %d, location %s", vnum, location)
		L.Push(lua.LNil)
		return 1
	}
	
	// Get object prototype
	objProto := e.world.GetObjPrototype(vnum)
	if objProto == nil {
		log.Printf("[OLOAD] Object prototype not found: vnum %d", vnum)
		L.Push(lua.LNil)
		return 1
	}
	
	// Create object instance from prototype
	// For now, just return the prototype as a table
	tbl := L.NewTable()
	tbl.RawSetString("vnum", lua.LNumber(objProto.GetVNum()))
	tbl.RawSetString("alias", lua.LString(objProto.GetKeywords()))
	tbl.RawSetString("name", lua.LString(objProto.GetShortDesc()))
	tbl.RawSetString("cost", lua.LNumber(objProto.GetCost()))
	tbl.RawSetString("timer", lua.LNumber(objProto.GetTimer()))
	
	// TODO: Actually add to room or character inventory based on location
	// For now, just log
	log.Printf("[OLOAD] Created object vnum %d at location %s", vnum, location)
	
	L.Push(tbl)
	return 1
}

func (e *Engine) luaMload(L *lua.LState) int {
	// mload(room_vnum, mob_vnum)
	// Based on lua_mload() in scripts.c lines 661-688
	return 0
}

func (e *Engine) luaExtobj(L *lua.LState) int {
	// extobj(obj)
	// Based on lua_extobj() in scripts.c lines 443-456
	return 0
}

func (e *Engine) luaExtchar(L *lua.LState) int {
	// extchar(ch)
	// Based on lua_extchar() in scripts.c lines 430-441
	return 0
}

func (e *Engine) luaNumber(L *lua.LState) int {
	// number(low, high)
	// Based on lua_number() in scripts.c lines 817-830
	low := L.ToInt(1)
	high := L.ToInt(2)
	
	if low > high {
		low, high = high, low
	}
	
	result := low + rand.Intn(high-low+1)
	L.Push(lua.LNumber(result))
	return 1
}

func (e *Engine) luaSendToRoom(L *lua.LState) int {
	// send_to_room(msg, room_vnum)
	// Based on lua_echo() with type="room" in scripts.c lines 308-345
	msg := L.ToString(1)
	roomVNum := L.ToInt(2)
	
	if e.world == nil {
		log.Printf("[SEND_TO_ROOM] No world context: Room %d: %s", roomVNum, msg)
		return 0
	}
	
	players := e.world.GetPlayersInRoom(roomVNum)
	for _, player := range players {
		player.SendMessage(msg + "\r\n")
	}
	
	return 0
}

func (e *Engine) luaStrlower(L *lua.LState) int {
	// strlower(s)
	// Lua 4 compat function
	s := L.ToString(1)
	L.Push(lua.LString(strings.ToLower(s)))
	return 1
}

func (e *Engine) luaStrfind(L *lua.LState) int {
	// strfind(s, pattern)
	// Already in Lua 5.1 as string.find, expose as global
	// Just call the built-in
	L.GetGlobal("string")
	L.GetField(L.Get(-1), "find")
	L.Push(L.Get(1))
	L.Push(L.Get(2))
	L.Call(2, 1)
	return 1
}

func (e *Engine) luaStrsub(L *lua.LState) int {
	// strsub(s, i, j)
	// Already in Lua 5.1 as string.sub, expose as global
	L.GetGlobal("string")
	L.GetField(L.Get(-1), "sub")
	L.Push(L.Get(1))
	L.Push(L.Get(2))
	L.Push(L.Get(3))
	L.Call(3, 1)
	return 1
}

func (e *Engine) luaGsub(L *lua.LState) int {
	// gsub(s, pattern, repl)
	// Already in Lua 5.1 as string.gsub, expose as global
	L.GetGlobal("string")
	L.GetField(L.Get(-1), "gsub")
	L.Push(L.Get(1))
	L.Push(L.Get(2))
	L.Push(L.Get(3))
	L.Call(3, 1)
	return 1
}

func (e *Engine) luaGetn(L *lua.LState) int {
	// getn(t)
	// Lua 4 compat for table length
	tbl := L.Get(1)
	if tbl.Type() != lua.LTTable {
		L.Push(lua.LNumber(0))
		return 1
	}
	
	L.Push(lua.LNumber(L.ObjLen(tbl)))
	return 1
}

func (e *Engine) luaTonumber(L *lua.LState) int {
	// tonumber(s)
	// Already in Lua 5.1, expose as global
	L.GetGlobal("tonumber")
	L.Push(L.Get(1))
	L.Call(1, 1)
	return 1
}

func (e *Engine) luaTostring(L *lua.LState) int {
	// tostring(v)
	// Already in Lua 5.1, expose as global
	L.GetGlobal("tostring")
	L.Push(L.Get(1))
	L.Call(1, 1)
	return 1
}

func (e *Engine) luaLog(L *lua.LState) int {
	// log(txt)
	// Based on lua_log() in scripts.c lines 690-703
	txt := L.ToString(1)
	log.Printf("[LUA LOG] %s", txt)
	return 0
}

func (e *Engine) luaRawKill(L *lua.LState) int {
	// raw_kill(vict, killer, type)
	// Based on lua_raw_kill() (not shown in snippets)
	return 0
}

func (e *Engine) luaSaveChar(L *lua.LState) int {
	// save_char() - calls table_to_char
	// Based on lua_save_char() (not shown in snippets)
	return 0
}

func (e *Engine) luaSaveObj(L *lua.LState) int {
	// save_obj() - calls table_to_obj
	// Based on lua_save_obj() (not shown in snippets)
	return 0
}

func (e *Engine) luaSaveRoom(L *lua.LState) int {
	// save_room() - calls table_to_room
	// Based on lua_save_room() (not shown in snippets)
	return 0
}

func (e *Engine) luaSetSkill(L *lua.LState) int {
	// set_skill(ch, skill, value)
	// Based on lua_set_skill() (not shown in snippets)
	return 0
}

func (e *Engine) luaSpell(L *lua.LState) int {
	// spell(caster, target, spellnum, aggressive)
	// Based on lua_spell() function called from Lua scripts
	// caster: table representing mob/player casting spell
	// target: table representing target (can be NIL)
	// spellnum: spell number constant
	// aggressive: boolean indicating if spell is offensive
	
	// Get parameters
	casterTbl := L.Get(1)
	targetTbl := L.Get(2)
	spellNum := L.ToInt(3)
	aggressive := L.ToBool(4)
	
	// Log the spell cast for now
	// TODO: Implement actual spell casting logic
	casterName := "unknown"
	targetName := "nil"
	
	if casterTbl.Type() == lua.LTTable {
		L.GetField(casterTbl, "name")
		if L.Get(-1).Type() == lua.LTString {
			casterName = L.ToString(-1)
		}
		L.Pop(1)
	}
	
	if targetTbl.Type() == lua.LTTable && targetTbl != lua.LNil {
		L.GetField(targetTbl, "name")
		if L.Get(-1).Type() == lua.LTString {
			targetName = L.ToString(-1)
		}
		L.Pop(1)
	}
	
	// Convert spell number to name for logging
	spellName := fmt.Sprintf("SPELL_%d", spellNum)
	// Map common spell numbers to names
	switch spellNum {
	case 32: spellName = "MAGIC_MISSILE"
	case 5: spellName = "BURNING_HANDS"
	case 30: spellName = "LIGHTNING_BOLT"
	case 26: spellName = "FIREBALL"
	case 58: spellName = "HELLFIRE"
	case 10: spellName = "COLOR_SPRAY"
	case 92: spellName = "DISRUPT"
	case 93: spellName = "DISINTEGRATE"
	case 96: spellName = "FLAMESTRIKE"
	case 75: spellName = "ACID_BLAST"
	case 2: spellName = "TELEPORT"
	case 22: spellName = "DISPEL_EVIL"
	case 46: spellName = "DISPEL_GOOD"
	case 28: spellName = "HEAL"
	case 67: spellName = "VITALITY"
	case 16: spellName = "CURE_LIGHT"
	case 27: spellName = "HARM"
	case 4: spellName = "BLINDNESS"
	case 17: spellName = "CURSE"
	case 33: spellName = "POISON"
	case 23: spellName = "EARTHQUAKE"
	case 81: spellName = "DIVINE_INT"
	case 82: spellName = "MIND_BAR"
	}
	
	log.Printf("[SPELL] %s casts %s on %s (aggressive: %v)", casterName, spellName, targetName, aggressive)
	
	return 0
}

func (e *Engine) luaTport(L *lua.LState) int {
	// tport(ch, room)
	// Based on lua_tport() (not shown in snippets)
	return 0
}

func (e *Engine) luaIsFighting(L *lua.LState) int {
	// isfighting(mob) - returns mob's current combat target or nil
	// Based on original C implementation that would check mob's fighting pointer
	// For now, return nil as we don't have combat tracking yet
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaRound(L *lua.LState) int {
	// round(n) - rounds a number to nearest integer
	// Lua 4 compat function used in combat AI scripts
	n := L.ToNumber(1)
	rounded := int(n + 0.5)
	L.Push(lua.LNumber(rounded))
	return 1
}