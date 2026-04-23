// Package scripting provides Lua scripting support for Dark Pawns MUD.
// Based on original C code from scripts.c.
package scripting

import (
	"fmt"
	"log"
	"math/rand"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	lua "github.com/yuin/gopher-lua"
)

// Engine manages the Lua VM.
// Based on boot_lua() in scripts.c lines 1703-1716.
type Engine struct {
	scriptsDir   string
	L            *lua.LState
	world        ScriptableWorld
	transitItems map[int]ScriptableObject // in-flight items moved by objfrom/objto
}

// NewEngine creates a new Lua scripting engine.
func NewEngine(scriptsDir string, world ScriptableWorld) *Engine {
	L := lua.NewState()
	engine := &Engine{
		scriptsDir:   scriptsDir,
		L:            L,
		world:        world,
		transitItems: make(map[int]ScriptableObject),
	}

	// Open standard libraries
	L.OpenLibs()

	// Remove dangerous functions for security
	// Remove file system access
	L.SetGlobal("dofile", lua.LNil)
	L.SetGlobal("loadfile", lua.LNil)
	L.SetGlobal("load", lua.LNil)

	// Remove OS access
	osTable := L.GetGlobal("os").(*lua.LTable)
	osTable.RawSetString("execute", lua.LNil)
	osTable.RawSetString("exit", lua.LNil)
	osTable.RawSetString("remove", lua.LNil)
	osTable.RawSetString("rename", lua.LNil)
	osTable.RawSetString("setlocale", lua.LNil)
	osTable.RawSetString("tmpname", lua.LNil)

	// Remove package library (can load arbitrary code)
	L.SetGlobal("package", lua.LNil)

	// Remove debug library
	L.SetGlobal("debug", lua.LNil)

	// Remove io library
	L.SetGlobal("io", lua.LNil)

	// Set memory limit
	L.SetMx(1000) // Limit memory allocation

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

		// Populate room.char with all players + mobs in the room
		charTbl := e.L.NewTable()
		idx := 1
		if e.world != nil {
			for _, p := range e.world.GetPlayersInRoom(ctx.RoomVNum) {
				pt := e.L.NewTable()
				pt.RawSetString("name", lua.LString(p.GetName()))
				pt.RawSetString("level", lua.LNumber(p.GetLevel()))
				pt.RawSetString("hp", lua.LNumber(p.GetHealth()))
				pt.RawSetString("maxhp", lua.LNumber(p.GetMaxHealth()))
				pt.RawSetString("evil", lua.LBool(p.GetAlignment() < -350))
				pt.RawSetString("pos", lua.LNumber(combat.PosStanding)) // POS_STANDING
				charTbl.RawSetInt(idx, pt)
				idx++
			}
			for _, m := range e.world.GetMobsInRoom(ctx.RoomVNum) {
				mt := e.L.NewTable()
				mt.RawSetString("name", lua.LString(m.GetName()))
				mt.RawSetString("level", lua.LNumber(m.GetLevel()))
				mt.RawSetString("hp", lua.LNumber(m.GetHealth()))
				mt.RawSetString("maxhp", lua.LNumber(m.GetMaxHealth()))
				// Check mob alignment (negative = evil)
				alignment := m.GetPrototype().GetAlignment()
				mt.RawSetString("evil", lua.LBool(alignment < 0))
				mt.RawSetString("pos", lua.LNumber(combat.PosStanding)) // POS_STANDING
				mt.RawSetString("vnum", lua.LNumber(m.GetVNum()))
				mt.RawSetString("room", lua.LNumber(ctx.RoomVNum))
				charTbl.RawSetInt(idx, mt)
				idx++
			}
		}
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
	e.L.SetGlobal("gossip", e.L.NewFunction(e.luaGossip))
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
	// dofile/call: shared-script delegation pattern used by cityguard, breed_killer, etc.
	e.L.SetGlobal("dofile", e.L.NewFunction(e.luaDofile))
	e.L.SetGlobal("call", e.L.NewFunction(e.luaCall))
	e.L.SetGlobal("save_room", e.L.NewFunction(e.luaSaveRoom))
	e.L.SetGlobal("set_skill", e.L.NewFunction(e.luaSetSkill))
	e.L.SetGlobal("spell", e.L.NewFunction(e.luaSpell))
	e.L.SetGlobal("tport", e.L.NewFunction(e.luaTport))

	// Additional functions needed for combat AI scripts
	e.L.SetGlobal("isfighting", e.L.NewFunction(e.luaIsFighting))
	e.L.SetGlobal("round", e.L.NewFunction(e.luaRound))

	// Functions needed for RESTORE scripts
	e.L.SetGlobal("has_item", e.L.NewFunction(e.luaHasItem))
	e.L.SetGlobal("obj_in_room", e.L.NewFunction(e.luaObjInRoom))
	e.L.SetGlobal("objfrom", e.L.NewFunction(e.luaObjFrom))
	e.L.SetGlobal("objto", e.L.NewFunction(e.luaObjTo))
	e.L.SetGlobal("obj_extra", e.L.NewFunction(e.luaObjExtra))
	e.L.SetGlobal("create_event", e.L.NewFunction(e.luaCreateEvent))
	e.L.SetGlobal("tell", e.L.NewFunction(e.luaTell))
	e.L.SetGlobal("plr_flagged", e.L.NewFunction(e.luaPlrFlagged))
	e.L.SetGlobal("cansee", e.L.NewFunction(e.luaCanSee))
	e.L.SetGlobal("isnpc", e.L.NewFunction(e.luaIsNPC))
	e.L.SetGlobal("aff_flagged", e.L.NewFunction(e.luaAffFlagged))
	e.L.SetGlobal("plr_flags", e.L.NewFunction(e.luaPlrFlags))
	e.L.SetGlobal("obj_list", e.L.NewFunction(e.luaObjList))

	// Stubs needed by Tier 3 Economy scripts
	e.L.SetGlobal("item_check", e.L.NewFunction(e.luaItemCheck))
	e.L.SetGlobal("load_room", e.L.NewFunction(e.luaLoadRoom))
	e.L.SetGlobal("inworld", e.L.NewFunction(e.luaInworld))
	e.L.SetGlobal("mob_flagged", e.L.NewFunction(e.luaMobFlagged))
	e.L.SetGlobal("aff_flags", e.L.NewFunction(e.luaAffFlags))
	e.L.SetGlobal("follow", e.L.NewFunction(e.luaFollow))
	e.L.SetGlobal("mount", e.L.NewFunction(e.luaMount))
	e.L.SetGlobal("direction", e.L.NewFunction(e.luaDirection))
	e.L.SetGlobal("set_hunt", e.L.NewFunction(e.luaSetHunt))
	e.L.SetGlobal("mxp", e.L.NewFunction(e.luaMxp))
	e.L.SetGlobal("skip_spaces", e.L.NewFunction(e.luaSkipSpaces))
	e.L.SetGlobal("social", e.L.NewFunction(e.luaSocial))
	e.L.SetGlobal("obj_flagged", e.L.NewFunction(e.luaObjFlagged))
	e.L.SetGlobal("get_group_lvl", e.L.NewFunction(e.luaGetGroupLvl))
	e.L.SetGlobal("get_group_pts", e.L.NewFunction(e.luaGetGroupPts))
	e.L.SetGlobal("skill_group", e.L.NewFunction(e.luaSkillGroup))
	e.L.SetGlobal("unaffect", e.L.NewFunction(e.luaUnaffect))
	e.L.SetGlobal("equip_char", e.L.NewFunction(e.luaEquipChar))
	// echo(ch, type, msg) — zone-wide sound broadcast. Used by werewolf.lua.
	// TODO: requires zone broadcast implementation
	e.L.SetGlobal("echo", e.L.NewFunction(e.luaEcho))

	// Stubs needed by Batch C Quest/Mechanic NPC scripts
	e.L.SetGlobal("extra", e.L.NewFunction(e.luaExtra))
	e.L.SetGlobal("strlen", e.L.NewFunction(e.luaStrlen))
	e.L.SetGlobal("iscorpse", e.L.NewFunction(e.luaIsCorpse))
	e.L.SetGlobal("canget", e.L.NewFunction(e.luaCanGet))
	e.L.SetGlobal("steal", e.L.NewFunction(e.luaSteal))
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
	e.L.SetGlobal("POS_DEAD", lua.LNumber(combat.PosDead))
	e.L.SetGlobal("POS_MORTALLYW", lua.LNumber(1))
	e.L.SetGlobal("POS_INCAP", lua.LNumber(combat.PosIncap))
	e.L.SetGlobal("POS_STUNNED", lua.LNumber(combat.PosStunned))
	e.L.SetGlobal("POS_SLEEPING", lua.LNumber(combat.PosSleeping))
	e.L.SetGlobal("POS_RESTING", lua.LNumber(combat.PosResting))
	e.L.SetGlobal("POS_SITTING", lua.LNumber(combat.PosSitting))
	e.L.SetGlobal("POS_STANDING", lua.LNumber(combat.PosStanding))

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
	// SPELL_PARALYSE: not in original globals.lua; assigned 105 as next available value.
	// Used by paralyse.lua and head_shrinker.lua. TODO: verify against original spells.h.
	e.L.SetGlobal("SPELL_PARALYSE", lua.LNumber(105))

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

	// is_npc: false for players. Source: utils.h IS_NPC() macro.
	tbl.RawSetString("is_npc", lua.LBool(false))

	// Expose raw PLR flags bitmask so plr_flagged() can check individual bits.
	// Source: structs.h PLR_FLAGS, utils.h PLR_FLAGGED() macro.
	tbl.RawSetString("plr_flags_raw", lua.LNumber(float64(player.GetFlags())))

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

	// is_npc: true for mobs. Source: utils.h IS_NPC() macro — MOB_ISNPC flag always set.
	tbl.RawSetString("is_npc", lua.LBool(true))

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

	// Object prototype fields (stubbed)
	tbl.RawSetString("perc_load", lua.LNumber(0)) // Default 0% load chance

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

// luaGossip broadcasts a mob's message to the room (gossip channel stub).
// gossip(msg) — in the original, this goes to all players world-wide.
// Since we have no global channel yet, we send to the mob's current room.
// Based on gossip channel in comm.c; used by quanlo.lua.
func (e *Engine) luaGossip(L *lua.LState) int {
	msg := L.ToString(1)

	// Resolve room from the me global
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

	// TODO: send to all online players when global channel is implemented
	if e.world == nil || roomVNum == 0 {
		log.Printf("[GOSSIP] %s", msg)
		return 0
	}

	formatted := "[gossip] " + msg + "\r\n"
	for _, player := range e.world.GetPlayersInRoom(roomVNum) {
		player.SendMessage(formatted)
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

	// Get caster level
	casterLevel := 1
	if casterTbl.Type() == lua.LTTable {
		L.GetField(casterTbl, "level")
		if L.Get(-1).Type() == lua.LTNumber {
			casterLevel = int(L.ToNumber(-1))
		}
		L.Pop(1)
	}

	// Helper function for dice rolls
	dice := func(num, sides int) int {
		total := 0
		for i := 0; i < num; i++ {
			total += rand.Intn(sides) + 1
		}
		return total
	}

	// Handle different spell types
	switch spellNum {
	case 2: // SPELL_TELEPORT
		// Move target to a random room
		if targetTbl.Type() == lua.LTTable && targetTbl != lua.LNil {
			// Get a random room from world (for now, just room 8004 - death room)
			newRoom := 8004
			L.SetField(targetTbl, "room", lua.LNumber(newRoom))
			// Send message to target if it's a player
			L.GetField(targetTbl, "name")
			targetName := L.ToString(-1)
			L.Pop(1)
			log.Printf("[SPELL] Teleported %s to room %d", targetName, newRoom)
		}
		return 0

	case 16: // SPELL_CURE_LIGHT
		if targetTbl.Type() == lua.LTTable && targetTbl != lua.LNil {
			// Healing spell: dice(2,8) + 1 + (level >> 2) - magic.c mag_points() line ~1765
			healAmount := dice(2, 8) + 1 + (casterLevel >> 2)
			L.GetField(targetTbl, "hp")
			currentHP := int(L.ToNumber(-1))
			L.Pop(1)
			L.GetField(targetTbl, "maxhp")
			maxHP := int(L.ToNumber(-1))
			L.Pop(1)
			newHP := currentHP + healAmount
			if newHP > maxHP {
				newHP = maxHP
			}
			L.SetField(targetTbl, "hp", lua.LNumber(newHP))
			log.Printf("[SPELL] Cure Light: healed %d HP (new HP: %d)", healAmount, newHP)
		}
		return 0

	case 28: // SPELL_HEAL
		if targetTbl.Type() == lua.LTTable && targetTbl != lua.LNil {
			// Healing spell: 100 + dice(3,8) - magic.c mag_points() line ~1783
			healAmount := 100 + dice(3, 8)
			L.GetField(targetTbl, "hp")
			currentHP := int(L.ToNumber(-1))
			L.Pop(1)
			L.GetField(targetTbl, "maxhp")
			maxHP := int(L.ToNumber(-1))
			L.Pop(1)
			newHP := currentHP + healAmount
			if newHP > maxHP {
				newHP = maxHP
			}
			L.SetField(targetTbl, "hp", lua.LNumber(newHP))
			log.Printf("[SPELL] Heal: restored %d HP (new HP: %d)", healAmount, newHP)
		}
		return 0

	case 67: // SPELL_VITALITY
		if targetTbl.Type() == lua.LTTable && targetTbl != lua.LNil {
			// Healing spell: dice(5,10) HP + dice(10,10) move - magic.c mag_points() line ~1800
			healAmount := dice(5, 10)
			L.GetField(targetTbl, "hp")
			currentHP := int(L.ToNumber(-1))
			L.Pop(1)
			L.GetField(targetTbl, "maxhp")
			maxHP := int(L.ToNumber(-1))
			L.Pop(1)
			newHP := currentHP + healAmount
			if newHP > maxHP {
				newHP = maxHP
			}
			L.SetField(targetTbl, "hp", lua.LNumber(newHP))
			// TODO: Restore move points when move tracking is implemented
			log.Printf("[SPELL] Vitality: restored %d HP (new HP: %d)", healAmount, newHP)
		}
		return 0
	}

	// For offensive spells (aggressive=true)
	if aggressive && targetTbl.Type() == lua.LTTable && targetTbl != lua.LNil {
		// Calculate damage based on spell type and caster level
		// Formulas from Dark Pawns magic.c
		damage := 0
		switch spellNum {
		case 32: // SPELL_MAGIC_MISSILE
			damage = dice(4, 3) + casterLevel
		case 5: // SPELL_BURNING_HANDS
			damage = dice(4, 5) + casterLevel
		case 37: // SPELL_SHOCKING_GRASP
			damage = dice(4, 7) + casterLevel
		case 30: // SPELL_LIGHTNING_BOLT
			damage = dice(9, 4) + casterLevel
		case 10: // SPELL_COLOR_SPRAY
			damage = dice(9, 7) + casterLevel
		case 26: // SPELL_FIREBALL
			// magic.c mag_damage() line ~690 - non-mage formula (we don't track reagents)
			damage = dice(12, 8) + casterLevel*2
		case 58: // SPELL_HELLFIRE
			// SPELL_HELLFIRE: disabled in original source (magic.c spell_hellfire - dummy) line ~1583
			log.Printf("[SPELL] SPELL_HELLFIRE: disabled in original source")
			return 0
		case 92: // SPELL_DISRUPT
			// magic.c mag_damage() line ~720 - non-mage formula
			damage = dice(20, 7) + casterLevel
		case 93: // SPELL_DISINTEGRATE
			// magic.c mag_damage() line ~700 - non-mage formula
			damage = dice(18, 8) + casterLevel
		case 96: // SPELL_FLAMESTRIKE
			// SPELL_FLAMESTRIKE: NOT a direct damage spell in mag_damage() - it's an outdoor AFF_FLAMING affect
			log.Printf("[SPELL] SPELL_FLAMESTRIKE: outdoor affect spell, not direct damage")
			return 0
		case 75: // SPELL_ACID_BLAST
			// magic.c mag_damage() line ~790
			damage = dice(4, 3) + casterLevel
		case 22: // SPELL_DISPEL_EVIL
			// magic.c mag_damage() line ~730
			damage = dice(9, 5) + casterLevel + 5 + casterLevel/2
		case 46: // SPELL_DISPEL_GOOD
			// magic.c mag_damage() line ~740
			damage = dice(9, 5) + casterLevel + 5
		case 27: // SPELL_HARM
			// magic.c mag_damage() line ~750
			damage = dice(12, 8) + casterLevel*2
		case 4: // SPELL_BLINDNESS
			// SPELL_BLINDNESS: affect only, not damage
			log.Printf("[SPELL] SPELL_BLINDNESS: affect only, not damage")
			return 0
		case 17: // SPELL_CURSE
			// SPELL_CURSE: affect only, not damage
			log.Printf("[SPELL] SPELL_CURSE: affect only, not damage")
			return 0
		case 33: // SPELL_POISON
			// SPELL_POISON: affect only, not damage
			log.Printf("[SPELL] SPELL_POISON: affect only, not damage")
			return 0
		case 23: // SPELL_EARTHQUAKE
			// magic.c mag_damage() line ~785
			damage = dice(7, 7) + casterLevel
		case 81: // SPELL_DIVINE_INT
			// SPELL_DIVINE_INT: NOT a damage spell — it summons an angel
			log.Printf("[SPELL] SPELL_DIVINE_INT: summon spell, not damage")
			return 0
		case 82: // SPELL_MIND_BAR
			// SPELL_MIND_BAR: NOT a damage spell — it's an INT debuff affect
			log.Printf("[SPELL] SPELL_MIND_BAR: INT debuff affect, not damage")
			return 0
		case 41: // SPELL_METEOR_SWARM
			// SPELL_METEOR_SWARM: Area spell that calls damage() per person in room — not single-target damage
			log.Printf("[SPELL] SPELL_METEOR_SWARM: area spell handled separately")
			return 0
		case 100: // SPELL_PSIBLAST
			// magic.c mag_damage() line ~805
			damage = dice(15, 13) + 3*casterLevel
		case 104: // SPELL_PETRIFY
			// SPELL_PETRIFY: raw_kill mechanic, not mag_damage
			log.Printf("[SPELL] SPELL_PETRIFY: raw_kill mechanic, not mag_damage")
			return 0
		case 8: // SPELL_CHILL_TOUCH
			// magic.c mag_damage() line ~636
			damage = dice(5, 3) + casterLevel
		case 6: // SPELL_CALL_LIGHTNING
			// magic.c mag_damage() line ~748
			damage = dice(10, 8) + casterLevel + 5
		case 25: // SPELL_ENERGY_DRAIN
		case 83: // SPELL_SOUL_LEECH
			// magic.c mag_damage() line ~778
			damage = dice(10, 6) + casterLevel
			// TODO: soul leech also heals caster by dam/3
		case 62: // SPELL_MINDBLAST
			// magic.c mag_damage() line ~800
			damage = dice(9, 7) + casterLevel + casterLevel/2
		default:
			// Default formula for unknown offensive spells
			minDamage := casterLevel
			maxDamage := casterLevel * 3
			damage = casterLevel*2 + rand.Intn(maxDamage-minDamage+1) + minDamage
		}

		// Get current HP
		L.GetField(targetTbl, "hp")
		currentHP := int(L.ToNumber(-1))
		L.Pop(1)

		// Apply damage
		newHP := currentHP - damage
		if newHP < 0 {
			newHP = 0
		}
		L.SetField(targetTbl, "hp", lua.LNumber(newHP))

		// Check for death
		if newHP == 0 && e.world != nil {
			// Get target name and room
			L.GetField(targetTbl, "name")
			targetName := L.ToString(-1)
			L.Pop(1)
			L.GetField(targetTbl, "room")
			roomVNum := int(L.ToNumber(-1))
			L.Pop(1)

			// Notify world of spell death
			e.world.HandleSpellDeath(targetName, spellNum, roomVNum)
		}

		// Log
		L.GetField(casterTbl, "name")
		casterName := L.ToString(-1)
		L.Pop(1)
		L.GetField(targetTbl, "name")
		targetName := L.ToString(-1)
		L.Pop(1)

		// Convert spell number to name for logging
		spellName := fmt.Sprintf("SPELL_%d", spellNum)
		switch spellNum {
		case 32:
			spellName = "MAGIC_MISSILE"
		case 5:
			spellName = "BURNING_HANDS"
		case 30:
			spellName = "LIGHTNING_BOLT"
		case 26:
			spellName = "FIREBALL"
		case 58:
			spellName = "HELLFIRE"
		case 10:
			spellName = "COLOR_SPRAY"
		case 92:
			spellName = "DISRUPT"
		case 93:
			spellName = "DISINTEGRATE"
		case 96:
			spellName = "FLAMESTRIKE"
		case 75:
			spellName = "ACID_BLAST"
		case 22:
			spellName = "DISPEL_EVIL"
		case 46:
			spellName = "DISPEL_GOOD"
		case 27:
			spellName = "HARM"
		case 4:
			spellName = "BLINDNESS"
		case 17:
			spellName = "CURSE"
		case 33:
			spellName = "POISON"
		case 23:
			spellName = "EARTHQUAKE"
		case 81:
			spellName = "DIVINE_INT"
		case 82:
			spellName = "MIND_BAR"
		case 8:
			spellName = "CHILL_TOUCH"
		case 6:
			spellName = "CALL_LIGHTNING"
		case 25:
			spellName = "ENERGY_DRAIN"
		case 83:
			spellName = "SOUL_LEECH"
		case 62:
			spellName = "MINDBLAST"
		case 100:
			spellName = "PSIBLAST"
		case 37:
			spellName = "SHOCKING_GRASP"
		}

		log.Printf("[SPELL] %s casts %s on %s for %d damage (HP: %d -> %d)",
			casterName, spellName, targetName, damage, currentHP, newHP)

		// TODO: Handle death if newHP == 0
	} else {
		// Non-aggressive or no target
		L.GetField(casterTbl, "name")
		casterName := L.ToString(-1)
		L.Pop(1)

		spellName := fmt.Sprintf("SPELL_%d", spellNum)
		log.Printf("[SPELL] %s casts %s (non-aggressive or no target)", casterName, spellName)
	}

	return 0
}

func (e *Engine) luaTport(L *lua.LState) int {
	// tport(ch, room)
	// Based on lua_tport() (not shown in snippets)
	return 0
}

func (e *Engine) luaIsFighting(L *lua.LState) int {
	// isfighting(mob) - returns the mob's current combat target as a table, or nil
	// Based on lua_isfighting() in scripts.c — checks mob's fighting pointer
	mobTbl := L.Get(1)
	if mobTbl.Type() != lua.LTTable {
		L.Push(lua.LNil)
		return 1
	}
	// Get the mob's vnum from the table to look it up in world
	L.GetField(mobTbl, "vnum")
	vnum := int(L.ToNumber(-1))
	L.Pop(1)

	if e.world != nil && vnum > 0 {
		// Look up the mob's fighting target via ScriptableWorld
		// GetMobByVNumAndRoom is approximate — use vnum to find the mob
		L.GetField(mobTbl, "room")
		roomVNum := int(L.ToNumber(-1))
		L.Pop(1)

		mob := e.world.GetMobByVNumAndRoom(vnum, roomVNum)
		if mob != nil {
			targetName := mob.GetFighting()
			if targetName != "" {
				// Return a minimal table with .name so scripts can do fighting.name
				tgt := L.NewTable()
				tgt.RawSetString("name", lua.LString(targetName))
				tgt.RawSetString("level", lua.LNumber(1)) // Unknown — fighting target level
				L.Push(tgt)
				return 1
			}
		}
	}
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaDofile(L *lua.LState) int {
	// dofile(path) - load and execute a Lua file, used for shared script delegation
	// Source: cityguard.lua, breed_killer.lua — dofile+call pattern for shared AI
	path := L.ToString(1)
	if path == "" {
		return 0
	}
	fullPath := e.scriptsDir + "/" + path
	if err := e.L.DoFile(fullPath); err != nil {
		log.Printf("[SCRIPT] dofile(%s): %v", path, err)
	}
	return 0
}

func (e *Engine) luaCall(L *lua.LState) int {
	// call(fn, arg1, arg2) - call a function loaded via dofile.
	// fn may be a function reference (most common: call(fight, ch, "x"))
	// or a string global name.
	// Source: dracula.lua, pyros.lua, breed_killer.lua — dofile+call delegation pattern.
	nArgs := L.GetTop() - 1
	arg1 := L.Get(1)
	var fn lua.LValue
	if arg1.Type() == lua.LTFunction {
		// Direct function reference — the common case after dofile redefines the global
		fn = arg1
	} else {
		fnName := L.ToString(1)
		if fnName == "" {
			return 0
		}
		fn = L.GetGlobal(fnName)
		if fn.Type() == lua.LTNil {
			log.Printf("[SCRIPT] call: function %s not found", fnName)
			return 0
		}
	}
	// Push function then remaining arguments
	L.Push(fn)
	for i := 2; i <= nArgs+1; i++ {
		L.Push(L.Get(i))
	}
	if err := L.PCall(nArgs, 0, nil); err != nil {
		log.Printf("[SCRIPT] call: %v", err)
	}
	return 0
}

func (e *Engine) luaRound(L *lua.LState) int {
	// round(n) - rounds a number to nearest integer
	// Lua 4 compat function used in combat AI scripts
	n := L.ToNumber(1)
	rounded := int(n + 0.5)
	L.Push(lua.LNumber(rounded))
	return 1
}

func (e *Engine) luaHasItem(L *lua.LState) int {
	// has_item(ch, vnum) - returns true if ch has an item with vnum in inventory.
	// Source: scripts.c lua_has_item() — searches char inventory for matching vnum.
	chTbl := L.Get(1)
	vnum := L.ToInt(2)

	if e.world == nil || chTbl.Type() != lua.LTTable {
		L.Push(lua.LBool(false))
		return 1
	}

	nameVal := L.GetField(chTbl, "name")
	if nameVal.Type() != lua.LTString {
		L.Push(lua.LBool(false))
		return 1
	}
	charName := string(nameVal.(lua.LString))

	L.Push(lua.LBool(e.world.HasItemByVNum(charName, vnum)))
	return 1
}

func (e *Engine) luaObjInRoom(L *lua.LState) int {
	// obj_in_room(room_vnum, obj_vnum) - returns item table if obj_vnum is in room, else nil.
	// Source: scripts.c lua_obj_in_room().
	roomVNum := L.ToInt(1)
	objVNum := L.ToInt(2)

	if e.world == nil {
		L.Push(lua.LNil)
		return 1
	}

	for _, item := range e.world.GetItemsInRoom(roomVNum) {
		if item.GetVNum() == objVNum {
			tbl := L.NewTable()
			tbl.RawSetString("vnum", lua.LNumber(item.GetVNum()))
			tbl.RawSetString("name", lua.LString(item.GetShortDesc()))
			tbl.RawSetString("alias", lua.LString(item.GetKeywords()))
			tbl.RawSetString("cost", lua.LNumber(item.GetCost()))
			tbl.RawSetString("timer", lua.LNumber(item.GetTimer()))
			// _src_room lets objfrom know which room to remove from
			tbl.RawSetString("_src_room", lua.LNumber(roomVNum))
			L.Push(tbl)
			return 1
		}
	}

	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaObjFrom(L *lua.LState) int {
	// objfrom(item, location) - remove item from location ('char' or 'room').
	// Removed item is held in e.transitItems until objto places it.
	// Source: scripts.c lua_objfrom() — calls obj_from_char() or obj_from_room().
	itemTbl := L.Get(1)
	location := L.ToString(2)

	if e.world == nil || itemTbl.Type() != lua.LTTable {
		return 0
	}

	vnumVal := L.GetField(itemTbl, "vnum")
	if vnumVal.Type() != lua.LTNumber {
		return 0
	}
	vnum := int(vnumVal.(lua.LNumber))

	var removed ScriptableObject

	switch location {
	case "room":
		// Determine source room: prefer _src_room field, fall back to me.room
		roomVNum := 0
		srcRoomVal := L.GetField(itemTbl, "_src_room")
		if srcRoomVal.Type() == lua.LTNumber {
			roomVNum = int(srcRoomVal.(lua.LNumber))
		} else {
			meVal := L.GetGlobal("me")
			if meVal.Type() == lua.LTTable {
				roomField := L.GetField(meVal, "room")
				if roomField.Type() == lua.LTNumber {
					roomVNum = int(roomField.(lua.LNumber))
				}
			}
		}
		if roomVNum > 0 {
			removed = e.world.RemoveItemFromRoom(vnum, roomVNum)
		}

	case "char":
		// Remove from mob's own inventory (me)
		meVal := L.GetGlobal("me")
		if meVal.Type() == lua.LTTable {
			nameField := L.GetField(meVal, "name")
			if nameField.Type() == lua.LTString {
				removed = e.world.RemoveItemFromChar(string(nameField.(lua.LString)), vnum)
			}
		}
	}

	if removed != nil {
		e.transitItems[vnum] = removed
	} else {
		log.Printf("[OBJFROM] item vnum %d not found in location %q", vnum, location)
	}
	return 0
}

func (e *Engine) luaObjTo(L *lua.LState) int {
	// objto(item, location, target) - place item at location ('char' or 'room').
	// Retrieves the in-transit item placed by objfrom.
	// Source: scripts.c lua_objto() — calls obj_to_char() or obj_to_room().
	itemTbl := L.Get(1)
	location := L.ToString(2)
	targetVal := L.Get(3)

	if e.world == nil || itemTbl.Type() != lua.LTTable {
		return 0
	}

	vnumVal := L.GetField(itemTbl, "vnum")
	if vnumVal.Type() != lua.LTNumber {
		return 0
	}
	vnum := int(vnumVal.(lua.LNumber))

	item, ok := e.transitItems[vnum]
	if !ok {
		log.Printf("[OBJTO] no in-transit item for vnum %d", vnum)
		return 0
	}

	switch location {
	case "char":
		charName := ""
		if targetVal.Type() == lua.LTTable {
			nameField := L.GetField(targetVal, "name")
			if nameField.Type() == lua.LTString {
				charName = string(nameField.(lua.LString))
			}
		}
		if charName == "" {
			log.Printf("[OBJTO] 'char' target has no name")
			return 0
		}
		if err := e.world.GiveItemToChar(charName, item); err != nil {
			log.Printf("[OBJTO] GiveItemToChar(%s, vnum %d): %v", charName, vnum, err)
		} else {
			delete(e.transitItems, vnum)
		}

	case "room":
		roomVNum := 0
		if targetVal.Type() == lua.LTNumber {
			roomVNum = int(targetVal.(lua.LNumber))
		} else {
			// Default: me's room
			meVal := L.GetGlobal("me")
			if meVal.Type() == lua.LTTable {
				roomField := L.GetField(meVal, "room")
				if roomField.Type() == lua.LTNumber {
					roomVNum = int(roomField.(lua.LNumber))
				}
			}
		}
		if roomVNum == 0 {
			log.Printf("[OBJTO] 'room' target: room vnum is 0")
			return 0
		}
		if err := e.world.AddItemToRoom(item, roomVNum); err != nil {
			log.Printf("[OBJTO] AddItemToRoom(room %d, vnum %d): %v", roomVNum, vnum, err)
		} else {
			delete(e.transitItems, vnum)
		}
	}

	return 0
}

func (e *Engine) luaObjExtra(L *lua.LState) int {
	// obj_extra(item, operation, flag) - set/clear object extra flags
	log.Printf("[STUB] obj_extra(item, operation, flag)")
	return 0
}

func (e *Engine) luaCreateEvent(L *lua.LState) int {
	// create_event(source, target, obj, argument, trigger, delay, type)
	log.Printf("[STUB] create_event(source, target, obj, argument, trigger, delay, type)")
	return 0
}

// luaTell sends a private message to a named player.
// tell(player_name, message)
// Source: act.comm.c do_tell().
func (e *Engine) luaTell(L *lua.LState) int {
	targetName := L.CheckString(1)
	message := L.CheckString(2)
	if e.world != nil && targetName != "" {
		e.world.SendTell(targetName, message)
	}
	return 0
}

// luaPlrFlagged checks whether a player character has a given PLR_* flag set.
// plr_flagged(ch, flag) → bool
// Returns false for NPCs (they use mob flags, not PLR flags).
// Source: utils.h PLR_FLAGGED(ch, flag) macro.
func (e *Engine) luaPlrFlagged(L *lua.LState) int {
	chTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		L.Push(lua.LBool(false))
		return 1
	}
	flagNum, ok := L.Get(2).(lua.LNumber)
	if !ok {
		L.Push(lua.LBool(false))
		return 1
	}

	// NPCs never have PLR flags.
	if isNPC, ok := chTbl.RawGetString("is_npc").(lua.LBool); ok && bool(isNPC) {
		L.Push(lua.LBool(false))
		return 1
	}

	// Read raw flags bitmask serialised into the ch table by charToTable().
	rawFlags, ok := chTbl.RawGetString("plr_flags_raw").(lua.LNumber)
	if !ok {
		L.Push(lua.LBool(false))
		return 1
	}

	bit := int(flagNum)
	if bit < 0 || bit >= 64 {
		L.Push(lua.LBool(false))
		return 1
	}
	result := (uint64(rawFlags)>>uint(bit))&1 == 1
	L.Push(lua.LBool(result))
	return 1
}

// luaCanSee checks whether the mob (me) can see character ch.
// cansee(ch) → bool
// Simplified: returns true if ch exists (level > 0).
// TODO: full impl — check PLR_INVISIBLE, room DARK flag, AFF_BLIND — Phase 6.
// Source: utils.h CAN_SEE() macro.
func (e *Engine) luaCanSee(L *lua.LState) int {
	chTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		L.Push(lua.LBool(false))
		return 1
	}
	lvl, ok := chTbl.RawGetString("level").(lua.LNumber)
	if !ok || lvl <= 0 {
		L.Push(lua.LBool(false))
		return 1
	}
	L.Push(lua.LBool(true))
	return 1
}

// luaIsNPC returns true if ch is an NPC/mob, false if a player.
// isnpc(ch) → bool
// Reads the 'is_npc' field set by charToTable() (false) or mobToTable() (true).
// Source: utils.h IS_NPC(ch) macro — checks MOB_ISNPC in MOB_FLAGS(ch).
func (e *Engine) luaIsNPC(L *lua.LState) int {
	chTbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		// Unknown type — treat as non-NPC to be safe.
		L.Push(lua.LBool(false))
		return 1
	}
	isNPC, ok := chTbl.RawGetString("is_npc").(lua.LBool)
	if !ok {
		L.Push(lua.LBool(false))
		return 1
	}
	L.Push(isNPC)
	return 1
}

func (e *Engine) luaAffFlagged(L *lua.LState) int {
	// aff_flagged(ch, flag) - check if character has affect flag set (e.g. AFF_VAMPIRE)
	// Based on AFF_FLAGGED() macro in utils.h
	log.Printf("[STUB] aff_flagged(ch, flag)")
	L.Push(lua.LBool(false))
	return 1
}

func (e *Engine) luaPlrFlags(L *lua.LState) int {
	// plr_flags(ch, operation, flag) - set or remove a player flag
	// operation is "set" or "remove"; flag is PLR_* constant
	// Based on PLR_FLAGS() macro in utils.h
	log.Printf("[STUB] plr_flags(ch, operation, flag)")
	return 0
}

func (e *Engine) luaObjList(L *lua.LState) int {
	// obj_list(keyword, location) - search mob's inventory for item matching keyword
	// location: "char" = mob's inventory, "room" = room floor, "vict" = player inventory
	// Returns the object table if found, NIL otherwise
	// Based on lua_obj_list() pattern in scripts.c
	log.Printf("[STUB] obj_list(keyword, location)")
	L.Push(lua.LNil)
	return 1
}

// --- Tier 3 Economy stubs ---

func (e *Engine) luaItemCheck(L *lua.LState) int {
	// item_check(obj) - validates whether object is a production item for the shop.
	// Source: shop_give.lua — checks if given item is valid shop production.
	// Engine gap: always returns false — shop production tables not yet implemented.
	log.Printf("[STUB] item_check(obj) — always returns false")
	L.Push(lua.LBool(false))
	return 1
}

func (e *Engine) luaLoadRoom(L *lua.LState) int {
	// load_room(vnum) - returns a room table with vnum, char, exit, objs fields.
	// Source: pet_store.lua, merchant_inn.lua — loads adjacent room for pet listing.
	log.Printf("[STUB] load_room(vnum)")
	vnum := L.ToInt(1)
	tbl := L.NewTable()
	tbl.RawSetString("vnum", lua.LNumber(vnum))
	tbl.RawSetString("char", L.NewTable())
	tbl.RawSetString("exit", L.NewTable())
	tbl.RawSetString("objs", L.NewTable())
	L.Push(tbl)
	return 1
}

func (e *Engine) luaInworld(L *lua.LState) int {
	// inworld(type, vnum) - check if a mob/obj with given vnum exists in the world.
	// Source: merchant_inn.lua — checks if travelling merchant (6805) already exists.
	log.Printf("[STUB] inworld(type, vnum)")
	L.Push(lua.LNil)
	return 1
}

func (e *Engine) luaMobFlagged(L *lua.LState) int {
	// mob_flagged(mob, flag) - check if mob has given MOB_* flag set.
	// Source: stable.lua find_mount() — checks MOB_MOUNTABLE.
	log.Printf("[STUB] mob_flagged(mob, flag)")
	L.Push(lua.LBool(false))
	return 1
}

func (e *Engine) luaAffFlags(L *lua.LState) int {
	// aff_flags(ch, operation, flag) - set or remove an affect flag on a character.
	// Source: stable.lua — removes AFF_CHARM from mount when stabling.
	log.Printf("[STUB] aff_flags(ch, operation, flag)")
	return 0
}

func (e *Engine) luaFollow(L *lua.LState) int {
	// follow(ch, charm) - makes mob follow ch, optionally charmed.
	// Source: pet_store.lua — makes purchased pet follow the buyer.
	log.Printf("[STUB] follow(ch, charm)")
	return 0
}

func (e *Engine) luaMount(L *lua.LState) int {
	// mount(ch, nil, "unmount") - dismount a player from their mount.
	// Source: stable.lua — dismounts player before stabling.
	log.Printf("[STUB] mount(ch, nil, operation)")
	return 0
}

func (e *Engine) luaDirection(L *lua.LState) int {
	// direction(from_vnum, to_vnum) - returns direction (0-5) from one room to another.
	// Source: merchant_walk.lua — pathfinding from current room toward target room 4860.
	log.Printf("[STUB] direction(from, to)")
	L.Push(lua.LNumber(-1))
	return 1
}

func (e *Engine) luaSetHunt(L *lua.LState) int {
	// set_hunt(hunter, prey) - set mob to hunt a target.
	// Source: merchant_walk.lua attack_time() — bandits hunt merchant and escort.
	log.Printf("[STUB] set_hunt(hunter, prey)")
	return 0
}

func (e *Engine) luaMxp(L *lua.LState) int {
	// mxp(text, command) - returns MXP-enabled link text. Falls back to plain text.
	// Source: merchant_inn.lua — creates clickable "interested?" link.
	if L.GetTop() >= 1 {
		L.Push(L.Get(1)) // Return the display text as-is
	} else {
		L.Push(lua.LString(""))
	}
	return 1
}

func (e *Engine) luaSkipSpaces(L *lua.LState) int {
	// skip_spaces(s) - trim leading spaces from a string.
	// Source: merchant_inn.lua — strips leading space from say argument.
	s := L.ToString(1)
	L.Push(lua.LString(strings.TrimLeft(s, " ")))
	return 1
}

func (e *Engine) luaSocial(L *lua.LState) int {
	// social(mob, social_name) - perform a social command.
	// Source: remove_curse.lua — performs "cough" social.
	log.Printf("[STUB] social(mob, social_name)")
	return 0
}

func (e *Engine) luaObjFlagged(L *lua.LState) int {
	// obj_flagged(obj, flag) - check if object has given ITEM_* flag set.
	// Source: identifier.lua — checks ITEM_MAGIC; remove_curse.lua — checks ITEM_NODROP.
	log.Printf("[STUB] obj_flagged(obj, flag)")
	L.Push(lua.LBool(false))
	return 1
}

func (e *Engine) luaGetGroupLvl(L *lua.LState) int {
	// get_group_lvl(ch, group[, newval]) - get or set character's skill group level.
	// Source: teacher.lua — reads and writes group level for skill training.
	log.Printf("[STUB] get_group_lvl(ch, group[, newval])")
	if L.GetTop() >= 3 {
		// Set mode: update ch.group_lvl (stub — no-op)
		return 0
	}
	L.Push(lua.LNumber(0))
	return 1
}

func (e *Engine) luaGetGroupPts(L *lua.LState) int {
	// get_group_pts(ch[, newval]) - get or set character's available group points.
	// Source: teacher.lua — reads and writes group points for skill training.
	log.Printf("[STUB] get_group_pts(ch[, newval])")
	if L.GetTop() >= 2 {
		return 0
	}
	L.Push(lua.LNumber(0))
	return 1
}

func (e *Engine) luaSkillGroup(L *lua.LState) int {
	// skill_group(name) - converts skill group name to numeric ID.
	// Source: teacher.lua — maps group names like "Rejuvenation" to IDs.
	log.Printf("[STUB] skill_group(name)")
	L.Push(lua.LNumber(0))
	return 1
}

func (e *Engine) luaUnaffect(L *lua.LState) int {
	// unaffect(ch) - remove all spell affections from character.
	// Source: memory_moss.lua line 33 — removes all spell affects from victim.
	log.Printf("[STUB] unaffect(ch)")
	return 0
}

func (e *Engine) luaEquipChar(L *lua.LState) int {
	// equip_char(mob, obj) - equip a mob with an object.
	// Source: phoenix.lua line 14 — equips rider with trident.
	log.Printf("[STUB] equip_char(mob, obj)")
	return 0
}

// --- Batch C Quest/Mechanic NPC stubs ---

func (e *Engine) luaExtra(L *lua.LState) int {
	// extra(obj, text) - set extra description on an object.
	// Source: head_shrinker.lua — writes head names into necklace extra desc.
	// Engine gap: extra description table not yet implemented.
	log.Printf("[STUB] extra(obj, text)")
	return 0
}

func (e *Engine) luaStrlen(L *lua.LState) int {
	// strlen(s) - Lua 4 compat string length function.
	// Source: head_shrinker.lua make_necklace() — pads owner name to 29 chars.
	s := L.ToString(1)
	L.Push(lua.LNumber(len(s)))
	return 1
}

func (e *Engine) luaIsCorpse(L *lua.LState) int {
	// iscorpse(obj) - returns true if the object is a player or mob corpse.
	// Source: janitor.lua — skips corpses when picking up trash.
	// Engine gap: corpse type not yet exposed on object tables.
	log.Printf("[STUB] iscorpse(obj)")
	L.Push(lua.LBool(false))
	return 1
}

func (e *Engine) luaCanGet(L *lua.LState) int {
	// canget(obj) - returns true if the mob is permitted to pick up the object.
	// Source: janitor.lua — checks ITEM_WEAR_TAKE and weight before picking up.
	// Engine gap: carry-weight and item permission checks not yet implemented.
	log.Printf("[STUB] canget(obj)")
	L.Push(lua.LBool(true))
	return 1
}

func (e *Engine) luaSteal(L *lua.LState) int {
	// steal(ch, obj) - steal an item from a character's inventory.
	// Source: mymic.lua — steals food items; eq_thief.lua — steals equipment.
	// Engine gap: theft mechanic not yet implemented.
	log.Printf("[STUB] steal(ch, obj)")
	return 0
}

func (e *Engine) luaEcho(L *lua.LState) int {
	// echo(ch, type, msg) — broadcast a message to a zone or room.
	// type="zone" sends to all players in the mob's zone.
	// type="room" sends to all players in the current room.
	// Based on lua_echo() in scripts.c lines 308-345.
	// Source: werewolf.lua — echo(me, "zone", "You hear a loud howling...")
	// TODO: requires zone broadcast implementation; currently logs only.
	msg := L.ToString(3)
	echoType := L.ToString(2)
	log.Printf("[STUB] echo(ch, %q, %q)", echoType, msg)
	return 0
}
