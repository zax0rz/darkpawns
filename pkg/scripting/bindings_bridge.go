package scripting

import (
	"strconv"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

// The bindings below are the cmdlib functions (scripts.c:1609-1664) as C
// defines them, run through the game's Bridge. They replace the port's
// earlier approximations whenever a script runs with a bridge (every run in
// the server); the older implementations remain only for engine tests that
// construct a world without one.

// bridged returns a binding that uses the bridge implementation while a
// bridged script is running, and the legacy one otherwise.
func (e *Engine) bridged(withBridge func(*lua.LState, Bridge) int, legacy lua.LGFunction) lua.LGFunction {
	return func(L *lua.LState) int {
		if b := e.activeBridge; b != nil {
			return withBridge(L, b)
		}
		return legacy(L)
	}
}

// meRef is lua_getglobal(L, "me") + "struct": the bindings that act "as me".
func meRef(L *lua.LState) (CharRef, bool) {
	return charRefOf(L.GetGlobal("me"))
}

func argString(L *lua.LState, n int) (string, bool) {
	switch v := L.Get(n).(type) {
	case lua.LString:
		return string(v), true
	case lua.LNumber:
		return v.String(), true // lua_isstring accepts numbers
	}
	return "", false
}

func argNumber(L *lua.LState, n int) (int, bool) {
	switch v := L.Get(n).(type) {
	case lua.LNumber:
		return int(v), true
	case lua.LString:
		if f, err := parseLuaNumber(string(v)); err == nil {
			return int(f), true // lua_isnumber accepts numeric strings
		}
	}
	return 0, false
}

// lua_act (scripts.c:80-120): act(txt, hide_invisible, me, obj, vict, where).
func (e *Engine) bridgeAct(L *lua.LState, b Bridge) int {
	txt, _ := argString(L, 1)
	invis, _ := argNumber(L, 2)
	var me, vict *CharRef
	var obj *ObjRef
	if ref, ok := charRefOf(L.Get(3)); ok {
		me = &ref
	}
	if ref, ok := objRefOf(L.Get(4)); ok {
		obj = &ref
	}
	if ref, ok := charRefOf(L.Get(5)); ok {
		vict = &ref
	}
	where, _ := argNumber(L, 6)
	b.Act(txt, invis != 0, me, obj, vict, where)
	return 0
}

// lua_say (scripts.c:1271-1290): do_say as me.
func (e *Engine) bridgeSay(L *lua.LState, b Bridge) int {
	text, ok := argString(L, 1)
	me, meOK := meRef(L)
	if !ok || !meOK {
		b.Log("[Lua] Invalid argument passed to lua_say.")
		return 0
	}
	b.Say(me, text)
	return 0
}

// lua_tell (scripts.c:1564-1589): do_tell(me, " name message").
func (e *Engine) bridgeTell(L *lua.LState, b Bridge) int {
	name, ok1 := argString(L, 1)
	text, ok2 := argString(L, 2)
	me, meOK := meRef(L)
	if !ok1 || !ok2 || !meOK {
		b.Log("[Lua] Invalid argument passed to lua_tell.")
		return 0
	}
	b.Tell(me, " "+name+" "+text)
	return 0
}

// lua_emote (scripts.c:385-401): do_echo(me, text, SCMD_EMOTE).
func (e *Engine) bridgeEmote(L *lua.LState, b Bridge) int {
	me, ok := meRef(L)
	if !ok {
		return 0
	}
	text, _ := argString(L, 1)
	b.Emote(me, text)
	return 0
}

// lua_action (scripts.c:122-140): command_interpreter(vict, argument).
func (e *Engine) bridgeAction(L *lua.LState, b Bridge) int {
	ref, ok := charRefOf(L.Get(1))
	line, lineOK := argString(L, 2)
	if !ok || !lineOK {
		b.Log("[Lua] Invalid argument passed to lua_action.")
		return 0
	}
	b.Command(ref, line)
	return 0
}

// lua_ishunt (scripts.c:676-695): TRUE when HUNTING(ch), else nil.
func (e *Engine) bridgeIsHunt(L *lua.LState, b Bridge) int {
	ref, ok := charRefOf(L.Get(1))
	if !ok {
		b.Log("[Lua] Invalid argument passed to lua_ishunt.")
		return 0
	}
	if b.CharIsHunting(ref) {
		L.Push(lua.LNumber(1))
	} else {
		L.Push(lua.LNil)
	}
	return 1
}

// lua_log (scripts.c:780-793): mudlog the text.
func (e *Engine) bridgeLog(L *lua.LState, b Bridge) int {
	text, ok := argString(L, 1)
	if !ok {
		b.Log("[Lua] Invalid argument passed to lua_log.")
		return 1
	}
	b.Log(text)
	return 0
}

// lua_extobj (scripts.c:496-510): extract_obj.
func (e *Engine) bridgeExtObj(L *lua.LState, b Bridge) int {
	ref, ok := objRefOf(L.Get(1))
	if !ok {
		b.Log("[Lua] Invalid object passed to lua_extobj")
		return 0
	}
	b.ExtractObj(ref)
	return 0
}

// lua_oload (scripts.c:974-1011): read_object to "room" (ch's room) or to
// "char"; returns the new object's table.
func (e *Engine) bridgeOLoad(L *lua.LState, b Bridge) int {
	ref, ok := charRefOf(L.Get(1))
	vnum, vnumOK := argNumber(L, 2)
	location, locOK := argString(L, 3)
	if !ok || !vnumOK || !locOK {
		b.Log("[Lua] Invalid argument passed to lua_oload.")
		return 0
	}
	var obj ObjRef
	var loaded bool
	switch {
	case equalFoldASCII(location, "room"):
		room, inRoom := b.CharRoom(ref)
		if !inRoom {
			return 0
		}
		obj, loaded = b.LoadObjToRoom(vnum, room)
	case equalFoldASCII(location, "char"):
		obj, loaded = b.LoadObjToChar(vnum, ref)
	default:
		b.Log("[Lua] Invalid location specified in lua_oload.")
		return 0
	}
	if !loaded {
		b.Log("[Lua] lua_oload returned an unknown object.")
		return 0
	}
	L.Push(e.cObjToTable(b, obj))
	return 1
}

// lua_save_char (scripts.c:1292-1307): table_to_char, then affect_total.
func (e *Engine) bridgeSaveChar(L *lua.LState, b Bridge) int {
	if _, ok := charRefOf(L.Get(1)); !ok {
		b.Log("[Lua] Invalid argument passed to lua_save_char.")
		return 0
	}
	tableToChar(b, L.Get(1))
	// C then runs affect_total, which removes and re-applies every equipment
	// and spell modifier and clamps the abilities. The port applies modifiers
	// incrementally, and table_to_char touches no ability, so it would change
	// nothing.
	return 0
}

// lua_save_obj (scripts.c:1309-1323): table_to_obj.
func (e *Engine) bridgeSaveObj(L *lua.LState, b Bridge) int {
	ref, ok := objRefOf(L.Get(1))
	if !ok {
		b.Log("[Lua] Invalid argument passed to lua_save_obj.")
		return 0
	}
	tbl := L.Get(1).(*lua.LTable)
	num := func(v lua.LValue) int { return int(lua.LVAsNumber(v)) }
	w := ObjWrite{
		Name:   lua.LVAsString(tbl.RawGetString("name")),
		Cost:   maxInt(0, num(tbl.RawGetString("cost"))),
		Weight: maxInt(1, num(tbl.RawGetString("weight"))),
		Timer:  maxInt(0, num(tbl.RawGetString("timer"))),
	}
	if val, ok := tbl.RawGetString("val").(*lua.LTable); ok {
		for i := 0; i < 4; i++ {
			w.Val[i] = num(val.RawGetInt(i + 1))
		}
	}
	b.ApplyObj(ref, w)
	return 0
}

// lua_save_room (scripts.c:1325-1339) calls table_to_room, which reads the
// room from the table's "struct"; room_to_table never sets one, so in C this
// dereferences NULL. That is undefined behaviour, not game behaviour (R1a):
// the port logs and changes nothing.
func (e *Engine) bridgeSaveRoom(L *lua.LState, b Bridge) int {
	b.Log("[Lua] save_room: room tables carry no struct (C dereferences NULL here); ignored.")
	return 0
}

// lua_set_skill (scripts.c:1365-1383): SET_SKILL + affect_total.
func (e *Engine) bridgeSetSkill(L *lua.LState, b Bridge) int {
	ref, ok := charRefOf(L.Get(1))
	skill, ok2 := argNumber(L, 2)
	level, ok3 := argNumber(L, 3)
	if ok && ok2 && ok3 {
		b.SetSkill(ref, skill, level)
	}
	return 1 // C returns 1 without pushing: the script sees its own last argument
}

// lua_tport (scripts.c:1518-1562): move vict to the room and look, then
// refresh the me or ch global if vict is one of them.
func (e *Engine) bridgeTport(L *lua.LState, b Bridge) int {
	ref, ok := charRefOf(L.Get(1))
	room, roomOK := argNumber(L, 2)
	if !ok || !roomOK {
		b.Log("[Lua] Invalid argument passed to lua_tport.")
		return 0
	}
	if !b.Teleport(ref, room) {
		b.Log("[Lua] Invalid room passed to lua_tport.")
		return 0
	}
	if me, meOK := meRef(L); meOK && me == ref {
		L.SetGlobal("me", e.charToTable(b, ref))
	} else if ch, chOK := charRefOf(L.GetGlobal("ch")); chOK && ch == ref {
		L.SetGlobal("ch", e.charToTable(b, ref))
	}
	return 0
}

// lua_raw_kill (scripts.c:1225-1258): raw_kill(vict, killer or nil, type).
func (e *Engine) bridgeRawKill(L *lua.LState, b Bridge) int {
	vict, ok := charRefOf(L.Get(1))
	attackType, typeOK := argNumber(L, 3)
	killerValue := L.Get(2)
	var killer *CharRef
	if ref, kOK := charRefOf(killerValue); kOK {
		killer = &ref
	} else if killerValue != lua.LNil {
		ok = false
	}
	if !ok || !typeOK {
		b.Log("[Lua] Invalid arguments passed to lua_raw_kill.")
		return 0
	}
	b.RawKill(vict, killer, attackType)
	return 0
}

func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

// parseLuaNumber is Lua 4's string-to-number coercion for lua_isnumber.
func parseLuaNumber(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}
