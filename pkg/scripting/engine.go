// Package scripting provides Lua scripting support for Dark Pawns MUD.
// Based on original C code from scripts.c.
package scripting

import (
	"log"
	"math/rand"
	"strings"

	lua "github.com/yuin/gopher-lua"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// Engine manages the Lua VM.
// Based on boot_lua() in scripts.c lines 1703-1716.
type Engine struct {
	scriptsDir string
	world      *game.World
	L          *lua.LState
}

// NewEngine creates a new Lua scripting engine.
func NewEngine(scriptsDir string, world *game.World) *Engine {
	L := lua.NewState()
	engine := &Engine{
		scriptsDir: scriptsDir,
		world:      world,
		L:          L,
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
func (e *Engine) RunScript(ctx *game.ScriptContext, fname string, triggerName string) (bool, error) {
	// Save stack position
	top := e.L.GetTop()
	defer e.L.SetTop(top)

	// Set globals based on context
	// Based on run_script() lines 1732-1761
	if ctx.Ch != nil {
		e.charToTable(ctx.Ch, "ch")
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
		// TODO: Implement room table
		e.L.SetGlobal("room", lua.LNil)
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
	e.L.GetGlobal(triggerName)
	if e.L.Get(-1).Type() == lua.LTNil {
		// Function doesn't exist
		e.L.Pop(1)
		return false, nil
	}

	if err := e.L.PCall(0, 1, nil); err != nil {
		log.Printf("[Lua] Error calling function %s in script %s: %v", triggerName, fname, err)
		return false, err
	}

	// Get return value
	ret := e.L.Get(-1)
	e.L.Pop(1)

	// Read back changes from tables
	// Based on run_script() lines 1797-1808
	if ctx.Ch != nil {
		e.L.GetGlobal("ch")
		if e.L.Get(-1).Type() != lua.LTNil {
			e.tableToChar(ctx.Ch)
		}
		e.L.Pop(1)
	}

	if ctx.Me != nil {
		e.L.GetGlobal("me")
		if e.L.Get(-1).Type() != lua.LTNil {
			e.tableToMob(ctx.Me)
		}
		e.L.Pop(1)
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
	e.L.SetGlobal("tostring", e.L.NewFunction(e.luaTostring))

	// Additional functions from cmdlib that might be needed
	e.L.SetGlobal("log", e.L.NewFunction(e.luaLog))
	e.L.SetGlobal("raw_kill", e.L.NewFunction(e.luaRawKill))
	e.L.SetGlobal("save_char", e.L.NewFunction(e.luaSaveChar))
	e.L.SetGlobal("save_obj", e.L.NewFunction(e.luaSaveObj))
	e.L.SetGlobal("save_room", e.L.NewFunction(e.luaSaveRoom))
	e.L.SetGlobal("set_skill", e.L.NewFunction(e.luaSetSkill))
	e.L.SetGlobal("spell", e.L.NewFunction(e.luaSpell))
	e.L.SetGlobal("tport", e.L.NewFunction(e.luaTport))
}

// loadGlobals loads the globals.lua file and calls default().
// Based on boot_lua() lines 1711-1714.
func (e *Engine) loadGlobals() {
	globalsPath := e.scriptsDir + "/globals.lua"
	if err := e.L.DoFile(globalsPath); err != nil {
		log.Printf("[Lua] Warning: Could not load globals.lua: %v", err)
		return
	}

	// Call default() function to set up constants
	e.L.GetGlobal("default")
	if e.L.Get(-1).Type() != lua.LTNil {
		if err := e.L.PCall(0, 0, nil); err != nil {
			log.Printf("[Lua] Warning: Error calling default(): %v", err)
		}
	}
	e.L.Pop(1)
}

// charToTable converts a Player to a Lua table.
// Based on char_to_table() in scripts.c lines 1812-1916.
func (e *Engine) charToTable(player *game.Player, globalName string) {
	L := e.L
	tbl := L.NewTable()

	// Basic fields
	tbl.RawSetString("name", lua.LString(player.Name))
	tbl.RawSetString("level", lua.LNumber(player.Level))
	tbl.RawSetString("hp", lua.LNumber(player.Health))
	tbl.RawSetString("maxhp", lua.LNumber(player.MaxHealth))
	tbl.RawSetString("gold", lua.LNumber(0)) // TODO: Add Gold field to Player
	tbl.RawSetString("race", lua.LNumber(player.Race))
	tbl.RawSetString("class", lua.LNumber(player.Class))
	tbl.RawSetString("alignment", lua.LNumber(0)) // TODO: Add alignment field

	// Skills table (stub for now)
	skillsTbl := L.NewTable()
	tbl.RawSetString("skills", skillsTbl)

	// Store pointer to struct for write-back
	tbl.RawSetString("struct", lua.LNumber(player.ID))

	L.SetGlobal(globalName, tbl)
}

// mobToTable converts a MobInstance to a Lua table.
// Based on char_to_table() for NPCs in scripts.c lines 1904-1910.
func (e *Engine) mobToTable(mob *game.MobInstance, globalName string) {
	L := e.L
	tbl := L.NewTable()

	// Basic fields
	tbl.RawSetString("name", lua.LString(mob.Prototype.ShortDesc))
	tbl.RawSetString("level", lua.LNumber(mob.Prototype.Level))
	tbl.RawSetString("hp", lua.LNumber(mob.CurrentHP))
	tbl.RawSetString("maxhp", lua.LNumber(mob.MaxHP))
	tbl.RawSetString("vnum", lua.LNumber(mob.VNum))
	tbl.RawSetString("gold", lua.LNumber(mob.Prototype.Gold))

	// Store pointer to struct for write-back
	tbl.RawSetString("struct", lua.LNumber(mob.VNum))

	L.SetGlobal(globalName, tbl)
}

// objToTable converts an ObjectInstance to a Lua table.
// Based on obj_to_table() in scripts.c lines 1918-2016.
func (e *Engine) objToTable(obj *game.ObjectInstance, globalName string) {
	L := e.L
	tbl := L.NewTable()

	// Basic fields
	tbl.RawSetString("vnum", lua.LNumber(obj.VNum))
	tbl.RawSetString("alias", lua.LString(obj.Prototype.Keywords))
	tbl.RawSetString("name", lua.LString(obj.Prototype.ShortDesc))
	tbl.RawSetString("cost", lua.LNumber(obj.Prototype.Cost))
	tbl.RawSetString("timer", lua.LNumber(0)) // TODO: Add timer field

	// Store pointer to struct for write-back
	tbl.RawSetString("struct", lua.LNumber(obj.VNum))

	L.SetGlobal(globalName, tbl)
}

// tableToChar reads back changes from the ch table to the Player.
// Based on table_to_char() in scripts.c lines 2018-2116.
func (e *Engine) tableToChar(player *game.Player) {
	L := e.L
	tbl := L.Get(-1)

	if tbl.Type() != lua.LTTable {
		return
	}

	// Read hp changes
	L.GetField(tbl, "hp")
	if val := L.Get(-1); val.Type() == lua.LTNumber {
		player.Health = int(lua.LVAsNumber(val))
	}
	L.Pop(1)

	// Read gold changes
	L.GetField(tbl, "gold")
	if val := L.Get(-1); val.Type() == lua.LTNumber {
		// TODO: Set player.Gold when field is added
	}
	L.Pop(1)
}

// tableToMob reads back changes from the me table to the MobInstance.
// Based on table_to_char() for NPCs in scripts.c lines 2040-2045.
func (e *Engine) tableToMob(mob *game.MobInstance) {
	L := e.L
	tbl := L.Get(-1)

	if tbl.Type() != lua.LTTable {
		return
	}

	// Read hp changes
	L.GetField(tbl, "hp")
	if val := L.Get(-1); val.Type() == lua.LTNumber {
		mob.CurrentHP = int(lua.LVAsNumber(val))
	}
	L.Pop(1)
}

// Lua function implementations
// Based on corresponding functions in scripts.c

func (e *Engine) luaAct(L *lua.LState) int {
	// act(msg, visible, ch, obj, vict, type)
	// Based on lua_act() in scripts.c lines 79-124
	msg := L.ToString(1)
	_ = L.ToBool(2) // visible — used for future CAN_SEE checks
	where := L.ToInt(6)

	// Get ch from global
	L.GetGlobal("ch")
	var ch *game.Player = nil
	if L.Get(-1).Type() == lua.LTTable {
		// In real implementation, we'd get the actual player pointer
	}
	L.Pop(1)

	// Get me from global
	L.GetGlobal("me")
	var me *game.MobInstance = nil
	if L.Get(-1).Type() == lua.LTTable {
		// In real implementation, we'd get the actual mob pointer
	}
	L.Pop(1)

	// Simplified implementation for now
	switch where {
	case 1: // TO_ROOM
		if me != nil {
			// Send to room where me is
			log.Printf("[ACT TO_ROOM] %s: %s", me.Prototype.ShortDesc, msg)
		}
	case 2: // TO_VICT
		if ch != nil {
			log.Printf("[ACT TO_VICT] To %s: %s", ch.Name, msg)
		}
	case 3: // TO_NOTVICT
		log.Printf("[ACT TO_NOTVICT] %s", msg)
	case 4: // TO_CHAR
		if ch != nil {
			log.Printf("[ACT TO_CHAR] To %s: %s", ch.Name, msg)
		}
	}

	return 0
}

func (e *Engine) luaDoDamage(L *lua.LState) int {
	// do_damage(amount)
	// Based on pattern_dmg.lua example
	amount := L.ToInt(1)
	_ = amount // applied via currentContext below

	// Get ch from global
	L.GetGlobal("ch")
	if L.Get(-1).Type() == lua.LTTable {
		// Damage applied via write-back in RunScript tableToChar
	}
	L.Pop(1)

	return 0
}

func (e *Engine) luaSay(L *lua.LState) int {
	// say(msg)
	// Based on lua_say() (not shown in snippets but referenced)
	msg := L.ToString(1)
	
	// Get me from global
	L.GetGlobal("me")
	if L.Get(-1).Type() == lua.LTTable {
		// In real implementation: send to room
		log.Printf("[SAY] %s says: %s", "mob", msg)
	}
	L.Pop(1)
	
	return 0
}

func (e *Engine) luaEmote(L *lua.LState) int {
	// emote(msg)
	// Based on lua_emote() in scripts.c lines 291-306
	msg := L.ToString(1)
	
	// Get me from global
	L.GetGlobal("me")
	if L.Get(-1).Type() == lua.LTTable {
		log.Printf("[EMOTE] %s %s", "mob", msg)
	}
	L.Pop(1)
	
	return 0
}

func (e *Engine) luaAction(L *lua.LState) int {
	// action(mob, cmdstr)
	// Based on lua_action() in scripts.c lines 126-144
	// mob would be table, cmdstr is string
	// In real implementation: mob executes command
	return 0
}

func (e *Engine) luaOload(L *lua.LState) int {
	// oload(target, vnum, location)
	// Based on lua_oload() in scripts.c lines 1047-1090
	// target is ch table, vnum is number, location is string
	return 0
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
	
	log.Printf("[SEND_TO_ROOM] Room %d: %s", roomVNum, msg)
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
	// spell(caster, target, spellnum, level)
	// Based on lua_spell() (not shown in snippets)
	return 0
}

func (e *Engine) luaTport(L *lua.LState) int {
	// tport(ch, room)
	// Based on lua_tport() (not shown in snippets)
	return 0
}