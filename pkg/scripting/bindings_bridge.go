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

// lua_save_room (scripts.c:1325-1339): table_to_room (scripts.c:2096-2109)
// writes the room table's sect back through its struct.
func (e *Engine) bridgeSaveRoom(L *lua.LState, b Bridge) int {
	tbl, ok := L.Get(1).(*lua.LTable)
	if !ok {
		b.Log("[Lua] Invalid argument passed to lua_save_room.")
		return 0
	}
	if ref, ok := roomRefOf(tbl); ok {
		b.SetRoomSector(ref, int(lua.LVAsNumber(tbl.RawGetString("sect"))))
	}
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

// pushTrueOrNil is C's lua_pushnumber(L, TRUE) / lua_pushnil(L).
func pushTrueOrNil(L *lua.LState, v bool) int {
	if v {
		L.Push(lua.LNumber(1))
	} else {
		L.Push(lua.LNil)
	}
	return 1
}

// lua_isnpc (scripts.c:697-715).
func (e *Engine) bridgeIsNPC(L *lua.LState, b Bridge) int {
	if _, ok := L.Get(1).(*lua.LTable); !ok {
		b.Log("[Lua] Invalid argument to lua_isnpc.")
		return 0
	}
	ref, _ := charRefOf(L.Get(1))
	return pushTrueOrNil(L, ref.NPC)
}

// lua_round (scripts.c:1260-1269): (int)lua_tonumber, which truncates toward
// zero; a non-numeric argument is 0.
func luaRound4(L *lua.LState) int {
	n, _ := argNumber(L, 1)
	L.Push(lua.LNumber(n))
	return 1
}

// lua_inworld (scripts.c:589-634): inworld("mob", vnum) or
// inworld("char", name), the character's table or nil.
func (e *Engine) bridgeInWorld(L *lua.LState, b Bridge) int {
	kind, ok := argString(L, 1)
	if !ok {
		b.Log("[Lua] Invalid argument passed to lua_inworld.")
		return 0
	}
	var ref CharRef
	var found bool
	switch strings.ToLower(kind) { // str_cmp
	case "mob":
		vnum, isNum := argNumber(L, 2)
		if !isNum {
			b.Log("[Lua] Invalid argument passed to lua_inworld.")
			return 0
		}
		ref, found = b.InWorldMob(vnum)
	case "char":
		name, isStr := argString(L, 2)
		if !isStr {
			b.Log("[Lua] Invalid argument passed to lua_inworld.")
			return 0
		}
		ref, found = b.InWorldChar(name)
	default:
		b.Log("[Lua] Invalid argument passed to lua_inworld.")
		return 0
	}
	if !found {
		L.Push(lua.LNil)
		return 1
	}
	L.Push(e.charToTable(b, ref))
	return 1
}

// lua_cansee (scripts.c:221-244): CAN_SEE(me, vict).
func (e *Engine) bridgeCanSee(L *lua.LState, b Bridge) int {
	me, _ := meRef(L)
	vict, ok := charRefOf(L.Get(1))
	if !ok {
		b.Log("[Lua] Invalid argument to lua_cansee.")
		return 0
	}
	return pushTrueOrNil(L, b.CanSee(me, vict))
}

// flagArgs reads the (table, number) pair the *_flagged bindings take.
func flagArgs(L *lua.LState) (lua.LValue, int, bool) {
	if _, ok := L.Get(1).(*lua.LTable); !ok {
		return nil, 0, false
	}
	bit, ok := argNumber(L, 2)
	return L.Get(1), bit, ok
}

// setFlagArgs reads the (table, "set"|"remove", number) triple of the flag
// setters. valid is false for bad arguments; op is "" for an unknown verb.
func setFlagArgs(L *lua.LState) (tbl lua.LValue, op string, bit int, valid bool) {
	if _, ok := L.Get(1).(*lua.LTable); !ok {
		return nil, "", 0, false
	}
	verb, okVerb := argString(L, 2)
	bit, okBit := argNumber(L, 3)
	if !okVerb || !okBit {
		return nil, "", 0, false
	}
	if verb != "set" && verb != "remove" { // strcmp
		verb = ""
	}
	return L.Get(1), verb, bit, true
}

// lua_aff_flagged (scripts.c:142-163).
func (e *Engine) bridgeAffFlagged(L *lua.LState, b Bridge) int {
	tbl, bit, ok := flagArgs(L)
	if !ok {
		b.Log("[Lua] Invalid argument passed to lua_aff_flagged.")
		return 0
	}
	ref, _ := charRefOf(tbl)
	return pushTrueOrNil(L, b.AffFlagged(ref, bit))
}

// lua_aff_flags (scripts.c:165-191).
func (e *Engine) bridgeAffFlags(L *lua.LState, b Bridge) int {
	tbl, op, bit, ok := setFlagArgs(L)
	switch {
	case !ok:
		b.Log("[Lua] Invalid argument passed to lua_aff_flags.")
	case op == "":
		b.Log("[Lua] Invalid set/remove to lua_aff_flags.")
	default:
		ref, _ := charRefOf(tbl)
		b.SetAffFlag(ref, bit, op == "set")
	}
	return 0
}

// lua_plr_flagged (scripts.c:1174-1195): PLR_FLAGGED is false for a mobile.
func (e *Engine) bridgePlrFlagged(L *lua.LState, b Bridge) int {
	tbl, bit, ok := flagArgs(L)
	if !ok {
		b.Log("[Lua] Invalid argument passed to lua_plr_flagged.")
		return 0
	}
	ref, _ := charRefOf(tbl)
	return pushTrueOrNil(L, !ref.NPC && b.ActFlagged(ref, bit))
}

// lua_mob_flagged (scripts.c:820-841): MOB_FLAGGED is false for a player.
func (e *Engine) bridgeMobFlagged(L *lua.LState, b Bridge) int {
	tbl, bit, ok := flagArgs(L)
	if !ok {
		b.Log("[Lua] Invalid argument passed to lua_mob_flagged.")
		return 0
	}
	ref, _ := charRefOf(tbl)
	return pushTrueOrNil(L, ref.NPC && b.ActFlagged(ref, bit))
}

// actFlagsBinding is lua_plr_flags (scripts.c:1197-1223) and lua_mob_flags
// (scripts.c:843-869): both SET_BIT_AR the shared act field, whatever kind
// of character the table is.
func (e *Engine) actFlagsBinding(name string) func(*lua.LState, Bridge) int {
	return func(L *lua.LState, b Bridge) int {
		tbl, op, bit, ok := setFlagArgs(L)
		switch {
		case !ok:
			b.Log("[Lua] Invalid argument passed to lua_" + name + ".")
		case op == "":
			b.Log("[Lua] Invalid set/remove to lua_" + name + ".")
		default:
			ref, _ := charRefOf(tbl)
			b.SetActFlag(ref, bit, op == "set")
		}
		return 0
	}
}

// lua_obj_flagged (scripts.c:951-972).
func (e *Engine) bridgeObjFlagged(L *lua.LState, b Bridge) int {
	tbl, bit, ok := flagArgs(L)
	if !ok {
		b.Log("[Lua] Invalid argument passed to lua_obj_flagged.")
		return 0
	}
	ref, _ := objRefOf(tbl)
	return pushTrueOrNil(L, b.ObjFlagged(ref, bit))
}

// lua_obj_extra (scripts.c:923-949).
func (e *Engine) bridgeObjExtra(L *lua.LState, b Bridge) int {
	tbl, op, bit, ok := setFlagArgs(L)
	switch {
	case !ok:
		b.Log("[Lua] Invalid argument passed to lua_obj_extra.")
	case op == "":
		b.Log("[Lua] Invalid set/remove to lua_obj_extra.")
	default:
		ref, _ := objRefOf(tbl)
		b.SetObjExtra(ref, bit, op == "set")
	}
	return 0
}

// lua_exit_flagged (scripts.c:456-478): EXIT_FLAGGED(room->dir_option[door],
// flag), where flag is an EX_* mask. C dereferences a missing exit; the port
// answers nil and logs (R1a).
func (e *Engine) bridgeExitFlagged(L *lua.LState, b Bridge) int {
	room, okRoom := roomRefOf(L.Get(1))
	door, okDoor := argNumber(L, 2)
	mask, okMask := argNumber(L, 3)
	if _, isTable := L.Get(1).(*lua.LTable); !isTable || !okDoor || !okMask {
		b.Log("[Lua] Invalid argument passed to lua_exit_flagged.")
		return 0
	}
	info, ok := b.RoomExitInfo(room, door)
	if !okRoom || !ok {
		b.Log("[Lua] exit_flagged: no such exit (C dereferences NULL here).")
		L.Push(lua.LNil)
		return 1
	}
	return pushTrueOrNil(L, info&mask != 0)
}

// lua_exit_flags (scripts.c:427-454): SET_BIT/REMOVE_BIT of an EX_* mask.
func (e *Engine) bridgeExitFlags(L *lua.LState, b Bridge) int {
	room, okRoom := roomRefOf(L.Get(1))
	door, okDoor := argNumber(L, 2)
	verb, okVerb := argString(L, 3)
	mask, okMask := argNumber(L, 4)
	if _, isTable := L.Get(1).(*lua.LTable); !isTable || !okDoor || !okVerb || !okMask {
		b.Log("[Lua] Invalid argument passed to lua_exit_flags.")
		return 0
	}
	if verb != "set" && verb != "remove" {
		b.Log("[Lua] Invalid set/remove to lua_exit_flags.")
		return 0
	}
	info, ok := b.RoomExitInfo(room, door)
	if !okRoom || !ok {
		b.Log("[Lua] exit_flags: no such exit (C dereferences NULL here).")
		return 0
	}
	if verb == "set" {
		info |= mask
	} else {
		info &^= mask
	}
	b.SetRoomExitInfo(room, door, info)
	return 0
}

// lua_load_room (scripts.c:758-778): room_to_table for the vnum, with me
// left out of its people. C indexes world[] with real_room's -1 for a
// missing vnum; the port answers nil and logs (R1a).
func (e *Engine) bridgeLoadRoom(L *lua.LState, b Bridge) int {
	vnum, ok := argNumber(L, 1)
	if !ok {
		b.Log("[Lua] Invalid argument passed to lua_load_room.")
		return 0
	}
	me, hasMe := meRef(L)
	var mePtr *CharRef
	if hasMe {
		mePtr = &me
	}
	t := e.roomToTable(b, vnum, mePtr)
	if t == lua.LNil {
		b.Log("[Lua] load_room: no such room (C indexes world[-1] here).")
	}
	L.Push(t)
	return 1
}

// lua_mload (scripts.c:795-818).
func (e *Engine) bridgeMLoad(L *lua.LState, b Bridge) int {
	vnum, okVnum := argNumber(L, 1)
	room, okRoom := argNumber(L, 2)
	if !okVnum || !okRoom {
		b.Log("[Lua] Invalid arguments passed to lua_mload.")
		return 0
	}
	ref, ok := b.LoadMob(vnum, room)
	if !ok {
		b.Log("[Lua] Invalid mobile vnum passed to lua_mload.")
		return 0
	}
	L.Push(e.charToTable(b, ref))
	return 1
}

// lua_extchar (scripts.c:480-494).
func (e *Engine) bridgeExtChar(L *lua.LState, b Bridge) int {
	ref, ok := charRefOf(L.Get(1))
	if _, isTable := L.Get(1).(*lua.LTable); !isTable {
		b.Log("[Lua] Invalid char passed to lua_extchar")
		return 0
	}
	if ok {
		b.ExtractChar(ref)
	}
	return 0
}

// lua_obj_list (scripts.c:1040-1126): the object found becomes the obj
// global and, when a character held it, that character the ch global.
func (e *Engine) bridgeObjList(L *lua.LState, b Bridge) int {
	arg, okArg := argString(L, 1)
	where, okWhere := argString(L, 2)
	if !okArg || !okWhere {
		b.Log("[Lua] Invalid argument passed to lua_obj_list.")
		return 0
	}
	switch where {
	case "room", "char", "vict", "cont", "corpse":
	default:
		b.Log("[Lua] Invalid location to search in lua_obj_list.")
		return 0
	}
	me, _ := meRef(L)
	obj, vict, found := b.ObjList(me, arg, where)
	if !found {
		L.Push(lua.LNil)
		return 1
	}
	L.SetGlobal("obj", e.cObjToTable(b, obj))
	if vict != nil {
		L.SetGlobal("ch", e.charToTable(b, *vict))
	}
	L.Push(lua.LNumber(1))
	return 1
}

// lua_objfrom (scripts.c:1013-1038): returns the object as userdata.
func (e *Engine) bridgeObjFrom(L *lua.LState, b Bridge) int {
	ref, okObj := objRefOf(L.Get(1))
	from, okFrom := argString(L, 2)
	if _, isTable := L.Get(1).(*lua.LTable); !isTable || !okFrom {
		b.Log("[Lua] Invalid argument passed to lua_objfrom.")
		return 0
	}
	if okObj {
		b.ObjFrom(ref, from)
	}
	L.Push(e.newObjHandle(ref))
	return 1
}

// lua_objto (scripts.c:1128-1172): returns the object as userdata. For a
// missing room C returns -1 results, which Lua 4 does not define (R1a):
// the port returns none and logs.
func (e *Engine) bridgeObjTo(L *lua.LState, b Bridge) int {
	ref, okObj := objRefOf(L.Get(1))
	to, okTo := argString(L, 2)
	if _, isTable := L.Get(1).(*lua.LTable); !isTable || !okTo {
		b.Log("[Lua] Invalid argument passed to lua_objto.")
		return 0
	}
	switch to {
	case "room":
		vnum := int(lua.LVAsNumber(L.Get(3)))
		if !b.ObjToRoom(ref, vnum) {
			b.Log("[Lua] objto: no such room (C returns -1 here).")
			return 0
		}
	case "char":
		if ch, ok := charRefOf(L.Get(3)); ok && okObj {
			b.ObjToChar(ref, ch)
		}
	case "obj":
		if into, ok := objRefOf(L.Get(3)); ok && okObj {
			b.ObjToObj(ref, into)
		}
	}
	L.Push(e.newObjHandle(ref))
	return 1
}

// lua_steal (scripts.c:1491-1516).
func (e *Engine) bridgeSteal(L *lua.LState, b Bridge) int {
	_, isVict := L.Get(1).(*lua.LTable)
	obj, okObj := objRefOf(L.Get(2))
	if _, isTable := L.Get(2).(*lua.LTable); !isVict || !isTable {
		b.Log("[Lua] Invalid argument passed to lua_steal.")
		return 0
	}
	if me, ok := meRef(L); ok && okObj {
		b.Steal(me, obj)
	}
	return 0
}

// lua_equip_char (scripts.c:403-425).
func (e *Engine) bridgeEquipChar(L *lua.LState, b Bridge) int {
	ch, okCh := charRefOf(L.Get(1))
	obj, okObj := objRefOf(L.Get(2))
	_, isCh := L.Get(1).(*lua.LTable)
	_, isObj := L.Get(2).(*lua.LTable)
	if !isCh || !isObj {
		b.Log("[Lua] Invalid arguments passed to lua_equip_char.")
		return 0
	}
	if okCh && okObj {
		b.EquipCharObj(ch, obj)
	}
	return 0
}

// lua_extra (scripts.c:512-540). C returns 1 without pushing, so the script
// gets the value on top of the stack: the object's struct userdata.
func (e *Engine) bridgeExtra(L *lua.LState, b Bridge) int {
	ref, okObj := objRefOf(L.Get(1))
	text, okText := argString(L, 2)
	if _, isTable := L.Get(1).(*lua.LTable); !isTable || !okText {
		b.Log("[Lua] Invalid argument passed to lua_extra.")
		return 0
	}
	if okObj {
		b.AppendExtraDescs(ref, text)
	}
	L.Push(e.newObjHandle(ref))
	return 1
}

// lua_echo (scripts.c:342-383).
func (e *Engine) bridgeEcho(L *lua.LState, b Bridge) int {
	kind, okKind := argString(L, 2)
	text, okText := argString(L, 3)
	if _, isTable := L.Get(1).(*lua.LTable); !isTable || !okKind || !okText {
		b.Log("[Lua] Invalid arguments passed to lua_echo.")
		return 0
	}
	switch kind {
	case "room":
		if room, ok := roomRefOf(L.Get(1)); ok {
			b.Echo(kind, &room, nil, text)
		} else {
			// C reads a room pointer out of whatever table it was given (R1a).
			b.Log("[Lua] echo 'room' needs a room table; ignored.")
		}
	case "outdoor":
		b.Echo(kind, nil, nil, text)
	default: // "zone", "local", "global"; any other kind does nothing
		if ch, ok := charRefOf(L.Get(1)); ok && (kind == "zone" || kind == "local" || kind == "global") {
			b.Echo(kind, nil, &ch, text)
		}
	}
	return 0
}

// lua_gossip (scripts.c:571-587).
func (e *Engine) bridgeGossip(L *lua.LState, b Bridge) int {
	me, ok := meRef(L)
	if !ok {
		return 0
	}
	text, _ := argString(L, 1)
	b.Gossip(me, text)
	return 0
}

// lua_social (scripts.c:1399-1426).
func (e *Engine) bridgeSocial(L *lua.LState, b Bridge) int {
	vict, okVict := charRefOf(L.Get(1))
	social, okSocial := argString(L, 2)
	if _, isTable := L.Get(1).(*lua.LTable); !isTable || !okSocial {
		b.Log("[Lua] Invalid argument passed to lua_social.")
		return 0
	}
	me, okMe := meRef(L)
	if okMe && okVict && !b.Social(me, vict, social) {
		b.Log("[Lua] Unknown command passed to lua_social.")
	}
	return 0
}

// lua_follow (scripts.c:542-569). C returns 1 without pushing: the script
// gets the leader table's struct, the last value lua_follow pushed.
func (e *Engine) bridgeFollow(L *lua.LState, b Bridge) int {
	leader, okLeader := charRefOf(L.Get(1))
	charm, okCharm := argNumber(L, 2)
	if _, isTable := L.Get(1).(*lua.LTable); !isTable || !okCharm {
		b.Log("[Lua] Invalid argument passed to lua_follow.")
		return 0
	}
	if me, ok := meRef(L); ok && okLeader {
		b.Follow(me, leader, charm != 0)
	}
	L.Push(L.Get(1).(*lua.LTable).RawGetString("struct"))
	return 1
}

// lua_set_hunt (scripts.c:1341-1363).
func (e *Engine) bridgeSetHunt(L *lua.LState, b Bridge) int {
	hunter, okHunter := charRefOf(L.Get(1))
	_, isTable := L.Get(1).(*lua.LTable)
	if !isTable || !tableOrNil(L, 2) {
		b.Log("[Lua] Invalid arguments passed to lua_set_hunt.")
		return 0
	}
	var vict *CharRef
	if ref, ok := charRefOf(L.Get(2)); ok {
		vict = &ref
	}
	if okHunter {
		b.SetHunt(hunter, vict)
	}
	return 0
}

// lua_spell (scripts.c:1428-1489). C returns 1 without a final push, so the
// script gets the top of lua_spell's stack: the victim's new table when it
// is neither me nor ch, else the object's struct, else the victim's struct,
// else ch's struct.
func (e *Engine) bridgeSpell(L *lua.LState, b Bridge) int {
	_, victTable := L.Get(1).(*lua.LTable)
	_, objTable := L.Get(2).(*lua.LTable)
	spell, okSpell := argNumber(L, 3)
	vocal, okVocal := argNumber(L, 4)
	if !tableOrNil(L, 1) || !tableOrNil(L, 2) || !okSpell || !okVocal {
		b.Log("[Lua] Invalid argument passed to lua_spell.")
		return 0
	}
	if spell == 0 {
		b.Log("[Lua] No spell passed to lua_spell.")
		return 0
	}
	me, _ := meRef(L)
	var vict *CharRef
	if ref, ok := charRefOf(L.Get(1)); ok {
		vict = &ref
	}
	var obj *ObjRef
	if ref, ok := objRefOf(L.Get(2)); ok {
		obj = &ref
	}
	b.Spell(me, vict, obj, spell, vocal != 0)
	top := lua.LValue(lua.LNil)
	if chTable, ok := L.GetGlobal("ch").(*lua.LTable); ok {
		top = chTable.RawGetString("struct")
	}
	if victTable {
		top = L.Get(1).(*lua.LTable).RawGetString("struct")
	}
	if objTable {
		top = L.Get(2).(*lua.LTable).RawGetString("struct")
	}
	if vict != nil {
		// "Now save the char in case damage/healing was done."
		table := e.charToTable(b, *vict)
		chRef, chOK := charRefOf(L.GetGlobal("ch"))
		switch {
		case *vict == me:
			L.SetGlobal("me", table)
		case chOK && *vict == chRef:
			L.SetGlobal("ch", table)
		default:
			top = table
		}
	}
	L.Push(top)
	return 1
}

// lua_unaffect (scripts.c:1591-1607).
func (e *Engine) bridgeUnaffect(L *lua.LState, b Bridge) int {
	ref, ok := charRefOf(L.Get(1))
	if _, isTable := L.Get(1).(*lua.LTable); !isTable {
		b.Log("[Lua] Invalid argument passed to lua_unaffect.")
		return 0
	}
	if ok {
		b.Unaffect(ref)
	}
	return 0
}

// lua_mount (scripts.c:871-906).
func (e *Engine) bridgeMount(L *lua.LState, b Bridge) int {
	rider, okRider := charRefOf(L.Get(1))
	_, riderTable := L.Get(1).(*lua.LTable)
	how, okHow := argString(L, 3)
	if !riderTable || !tableOrNil(L, 2) || !okHow {
		b.Log("[Lua] Invalid argument passed to lua_mount.")
		return 0
	}
	if how != "ride" && how != "dismount" && how != "unmount" {
		b.Log("[Lua] Invalid position to lua_mount.")
		return 0
	}
	var mount *CharRef
	if ref, ok := charRefOf(L.Get(2)); ok {
		mount = &ref
	}
	if okRider {
		b.Mount(rider, mount, how)
	}
	return 0
}

// tableOrNil is C's (lua_istable(L, n) || lua_isnil(L, n)).
func tableOrNil(L *lua.LState, n int) bool {
	_, isTable := L.Get(n).(*lua.LTable)
	return isTable || L.Get(n) == lua.LNil
}
