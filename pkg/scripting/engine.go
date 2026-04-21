// Package scripting provides Lua scripting support for Dark Pawns MUD.
// Based on original C code from scripts.c.
package scripting

import (
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
		e.L.SetGlobal("room", lua.LNumber(ctx.RoomVNum))
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
	e.L.Push(fn)
	if fn.Type() == lua.LTNil {
		// Function doesn't exist
		e.L.Pop(1)
		return false, nil
	}

	if err := e.L.PCall(0, 1, nil); err != nil {
		log.Printf("[Lua] Error calling function %s in script %s: %v", triggerName, fname, err)
		// Pop the error from stack
		e.L.Pop(1)
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

// loadGlobals loads the globals.lua file.
// Based on boot_lua() lines 1711-1714.
func (e *Engine) loadGlobals() {
	globalsPath := e.scriptsDir + "/globals.lua"
	if err := e.L.DoFile(globalsPath); err != nil {
		log.Printf("[Lua] Warning: Could not load globals.lua: %v", err)
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

	// Skills table (stub for now)
	skillsTbl := L.NewTable()
	tbl.RawSetString("skills", skillsTbl)

	// Store pointer to struct for write-back
	tbl.RawSetString("struct", lua.LNumber(player.GetID()))

	L.SetGlobal(globalName, tbl)
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
	tbl := L.Get(-1)

	if tbl.Type() != lua.LTTable {
		return
	}

	// Read hp changes
	L.GetField(tbl, "hp")
	if val := L.Get(-1); val.Type() == lua.LTNumber {
		player.SetHealth(int(lua.LVAsNumber(val)))
	}
	L.Pop(1)

	// Read gold changes
	L.GetField(tbl, "gold")
	if val := L.Get(-1); val.Type() == lua.LTNumber {
		player.SetGold(int(lua.LVAsNumber(val)))
	}
	L.Pop(1)
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
	L.GetGlobal("room")
	if L.Get(-1).Type() == lua.LTNumber {
		roomVNum = int(L.ToNumber(-1))
	}
	L.Pop(1)

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
	// spell(caster, target, spellnum, level)
	// Based on lua_spell() (not shown in snippets)
	return 0
}

func (e *Engine) luaTport(L *lua.LState) int {
	// tport(ch, room)
	// Based on lua_tport() (not shown in snippets)
	return 0
}