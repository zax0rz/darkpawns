package scripting

import (
	lua "github.com/yuin/gopher-lua"
)

// This file ports the C<->Lua bridge of scripts.c: the tables a script sees
// for a character, object and room (char_to_table, obj_to_table,
// room_to_table, scripts.c:1823-1970), the write-back after a script runs
// (table_to_char, scripts.c:1975-2052), and the "struct" field that ties a
// table back to the live game thing it describes.
//
// C stores the object's pointer in "struct" as userdata. The port stores a
// CharRef or ObjRef in an LUserData: unique per live character or object,
// opaque to the script, and resolved by the game through Bridge. (The port
// used to store a player's ID or a mobile's vnum there, so two copies of the
// same mobile were indistinguishable and an ID could collide with a vnum.)

// CharRef names one live character: a player by ID, or a mobile instance by
// its runtime ID.
type CharRef struct {
	NPC bool
	ID  int
}

// ObjRef names one live object instance by its runtime ID.
type ObjRef struct {
	ID int
}

// CharFields is what char_to_table reads from a character.
type CharFields struct {
	Name   string // GET_NAME
	Alias  string // ch->player.name (keyword list)
	Align  int
	Gold   int
	Level  int
	HP     int
	MaxHP  int
	Pos    int
	Carry  []ObjRef // ch->carrying, in list order
	Worn   []ObjRef // GET_EQ(ch, 0..NUM_WEARS-1), present slots only
	Evil   bool     // IS_EVIL
	Master *CharRef // ch->master, nil when none or self
	NPC    bool
	// Players only.
	IDNum     int
	Exp       int
	Bank      int
	Mana      int
	Move      int
	Cha       int
	Jail      int
	Tattoo    int
	Followers int
	// Mobiles only.
	VNum  int
	Timer int // GET_MOB_WAIT
}

// CharWrite is what table_to_char writes back. Fields a script cannot change
// for the character's kind are ignored by the game (C writes timer only for
// NPCs, and bank/exp/mana/move/jail/tattoo only for players).
type CharWrite struct {
	Align, Gold, Level, HP, Pos int
	Timer                       int
	Bank, Exp, Mana, Move       int
	Jail, Tattoo                int
}

// ObjFields is what obj_to_table reads from an object.
type ObjFields struct {
	Name     string // short_description
	Alias    string // obj->name (keyword list)
	VNum     int
	Cost     int
	Type     int
	Weight   int
	PercLoad int
	Timer    int
	Val      [4]int
	Contents []ObjRef
}

// ObjWrite is what table_to_obj writes back (scripts.c:2054-2094).
type ObjWrite struct {
	Name   string
	Cost   int
	Weight int
	Val    [4]int
	Timer  int
}

// RoomFields is what room_to_table reads from a room.
type RoomFields struct {
	VNum   int
	Sect   int
	Exits  [6]int // destination room vnum per direction, or -1 when absent
	People []CharRef
	Objs   []ObjRef
}

// Bridge is the game's side of the C<->Lua bridge. Every method answers or
// performs exactly what the C binding or table function does, through the
// port's own game functions (act(), do_say, command_interpreter, ...).
type Bridge interface {
	CharFields(ref CharRef) (CharFields, bool)
	ObjFields(ref ObjRef) (ObjFields, bool)
	RoomFields(vnum int) (RoomFields, bool)
	// ApplyChar is table_to_char. It does nothing when the character was
	// extracted during the script (MOB_EXTRACT/PLR_EXTRACT).
	ApplyChar(ref CharRef, w CharWrite)
	// ApplyObj is table_to_obj.
	ApplyObj(ref ObjRef, w ObjWrite)
	// CharRoom reports the character's room vnum (ch->in_room).
	CharRoom(ref CharRef) (int, bool)
	// InRoomAboveZero is run_script's "ch->in_room > 0": the character is in
	// a room whose real number is not 0, the condition for the write-back.
	InRoomAboveZero(ref CharRef) bool

	// Act is act(txt, hide_invisible, me, obj, vict, where) with the given
	// actors; any of them may be nil.
	Act(txt string, hideInvisible bool, me *CharRef, obj *ObjRef, vict *CharRef, where int)
	// Say, Emote and Command run do_say, do_emote and command_interpreter
	// as the character; Tell runs do_tell(me, "name message").
	Say(me CharRef, text string)
	Emote(me CharRef, text string)
	Tell(me CharRef, argument string)
	Command(me CharRef, line string)
	// CharIsHunting is HUNTING(ch) != NULL.
	CharIsHunting(ref CharRef) bool
	// ExtractObj is extract_obj.
	ExtractObj(ref ObjRef)
	// LoadObjToChar is read_object + obj_to_char; LoadObjToRoom is
	// read_object + obj_to_room. ok is false when the vnum does not exist.
	LoadObjToChar(vnum int, to CharRef) (ObjRef, bool)
	LoadObjToRoom(vnum int, roomVNum int) (ObjRef, bool)
	// SetSkill is SET_SKILL + affect_total (lua_set_skill).
	SetSkill(ref CharRef, skill, level int)
	// Teleport is lua_tport's char_from_room, char_to_room and
	// look_at_room. ok is false when the room does not exist.
	Teleport(ref CharRef, roomVNum int) bool
	// RawKill is raw_kill(vict, killer, type), with lua_raw_kill's mudlog for
	// a player victim.
	RawKill(vict CharRef, killer *CharRef, attackType int)
	// Log writes to the server log (mudlog / log).
	Log(msg string)
}

type (
	charHandle struct{ ref CharRef }
	objHandle  struct{ ref ObjRef }
)

func (e *Engine) newCharHandle(ref CharRef) *lua.LUserData {
	ud := e.l.NewUserData()
	ud.Value = charHandle{ref: ref}
	return ud
}

func (e *Engine) newObjHandle(ref ObjRef) *lua.LUserData {
	ud := e.l.NewUserData()
	ud.Value = objHandle{ref: ref}
	return ud
}

// charRefOf returns the character a table stands for (its "struct").
func charRefOf(v lua.LValue) (CharRef, bool) {
	tbl, ok := v.(*lua.LTable)
	if !ok {
		return CharRef{}, false
	}
	ud, ok := tbl.RawGetString("struct").(*lua.LUserData)
	if !ok {
		return CharRef{}, false
	}
	h, ok := ud.Value.(charHandle)
	return h.ref, ok
}

// objRefOf returns the object a table stands for (its "struct").
func objRefOf(v lua.LValue) (ObjRef, bool) {
	tbl, ok := v.(*lua.LTable)
	if !ok {
		return ObjRef{}, false
	}
	ud, ok := tbl.RawGetString("struct").(*lua.LUserData)
	if !ok {
		return ObjRef{}, false
	}
	h, ok := ud.Value.(objHandle)
	return h.ref, ok
}

// charToTable is char_to_table (scripts.c:1823-1891).
func (e *Engine) charToTable(b Bridge, ref CharRef) lua.LValue {
	return e.charToTableDepth(b, ref, 0)
}

// charToTableDepth bounds the leader recursion: C follows ch->master without
// a limit and relies on the follower graph having no cycles.
func (e *Engine) charToTableDepth(b Bridge, ref CharRef, depth int) lua.LValue {
	f, ok := b.CharFields(ref)
	if !ok {
		return lua.LNil
	}
	L := e.l
	t := L.NewTable()
	t.RawSetString("name", lua.LString(f.Name))
	t.RawSetString("align", lua.LNumber(f.Align))
	t.RawSetString("gold", lua.LNumber(f.Gold))
	t.RawSetString("level", lua.LNumber(f.Level))
	t.RawSetString("hp", lua.LNumber(f.HP))
	t.RawSetString("maxhp", lua.LNumber(f.MaxHP))
	t.RawSetString("pos", lua.LNumber(f.Pos))
	t.RawSetString("alias", lua.LString(f.Alias))
	if len(f.Carry) > 0 {
		objs := L.NewTable()
		for i, o := range f.Carry {
			objs.RawSetInt(i+1, e.cObjToTable(b, o))
		}
		t.RawSetString("objs", objs)
	}
	if len(f.Worn) > 0 {
		wear := L.NewTable()
		for i, o := range f.Worn {
			wear.RawSetInt(i+1, e.cObjToTable(b, o))
		}
		t.RawSetString("wear", wear)
	}
	t.RawSetString("evil", luaBool(f.Evil))
	if f.Master != nil && depth < 32 {
		t.RawSetString("leader", e.charToTableDepth(b, *f.Master, depth+1))
	}
	if !f.NPC {
		t.RawSetString("id", lua.LNumber(f.IDNum))
		t.RawSetString("exp", lua.LNumber(f.Exp))
		t.RawSetString("bank", lua.LNumber(f.Bank))
		t.RawSetString("mana", lua.LNumber(f.Mana))
		t.RawSetString("move", lua.LNumber(f.Move))
		t.RawSetString("cha", lua.LNumber(f.Cha))
		t.RawSetString("jail", lua.LNumber(f.Jail))
		t.RawSetString("tattoo", lua.LNumber(f.Tattoo))
		t.RawSetString("followers", lua.LNumber(f.Followers))
	} else {
		t.RawSetString("vnum", lua.LNumber(f.VNum))
		t.RawSetString("timer", lua.LNumber(f.Timer))
	}
	t.RawSetString("struct", e.newCharHandle(ref))
	return t
}

// cObjToTable is obj_to_table (scripts.c:1893-1926).
func (e *Engine) cObjToTable(b Bridge, ref ObjRef) lua.LValue {
	f, ok := b.ObjFields(ref)
	if !ok {
		return lua.LNil
	}
	L := e.l
	t := L.NewTable()
	t.RawSetString("name", lua.LString(f.Name))
	t.RawSetString("alias", lua.LString(f.Alias))
	t.RawSetString("vnum", lua.LNumber(f.VNum))
	t.RawSetString("cost", lua.LNumber(f.Cost))
	t.RawSetString("type", lua.LNumber(f.Type))
	t.RawSetString("weight", lua.LNumber(f.Weight))
	t.RawSetString("perc_load", lua.LNumber(f.PercLoad))
	t.RawSetString("timer", lua.LNumber(f.Timer))
	val := L.NewTable()
	for i := 0; i < 4; i++ {
		val.RawSetInt(i+1, lua.LNumber(f.Val[i]))
	}
	t.RawSetString("val", val)
	if len(f.Contents) > 0 {
		contents := L.NewTable()
		for i, o := range f.Contents {
			contents.RawSetInt(i+1, e.cObjToTable(b, o))
		}
		t.RawSetString("contents", contents)
	}
	t.RawSetString("struct", e.newObjHandle(ref))
	return t
}

// roomToTable is room_to_table (scripts.c:1928-1970): "char" lists the room's
// people except me, "objs" its contents.
func (e *Engine) roomToTable(b Bridge, vnum int, me *CharRef) lua.LValue {
	f, ok := b.RoomFields(vnum)
	if !ok {
		return lua.LNil
	}
	L := e.l
	t := L.NewTable()
	t.RawSetString("vnum", lua.LNumber(f.VNum))
	t.RawSetString("sect", lua.LNumber(f.Sect))
	exits := L.NewTable()
	for dir, to := range f.Exits {
		if to >= 0 {
			exits.RawSetInt(dir, lua.LNumber(to))
		}
	}
	t.RawSetString("exit", exits)
	people := L.NewTable()
	n := 0
	for _, p := range f.People {
		if me != nil && p == *me {
			continue
		}
		n++
		people.RawSetInt(n, e.charToTable(b, p))
	}
	t.RawSetString("char", people)
	if len(f.Objs) > 0 {
		objs := L.NewTable()
		for i, o := range f.Objs {
			objs.RawSetInt(i+1, e.cObjToTable(b, o))
		}
		t.RawSetString("objs", objs)
	}
	return t
}

// tableToChar is table_to_char (scripts.c:1975-2052): the fields a script
// may change are clamped exactly as C clamps them and written back.
func tableToChar(b Bridge, v lua.LValue) {
	ref, ok := charRefOf(v)
	if !ok {
		return
	}
	tbl := v.(*lua.LTable)
	num := func(key string) int { return int(lua.LVAsNumber(tbl.RawGetString(key))) }
	w := CharWrite{
		Align:  clampInt(num("align"), -1000, 1000),
		Gold:   maxInt(0, num("gold")),
		Level:  maxInt(0, num("level")),
		HP:     num("hp"),
		Pos:    maxInt(0, num("pos")),
		Timer:  maxInt(0, num("timer")),
		Bank:   maxInt(0, num("bank")),
		Exp:    maxInt(0, num("exp")),
		Mana:   num("mana"),
		Move:   num("move"),
		Jail:   maxInt(0, num("jail")),
		Tattoo: maxInt(0, num("tattoo")),
	}
	b.ApplyChar(ref, w)
}

func luaBool(v bool) lua.LValue {
	if v {
		return lua.LNumber(1) // Lua 4: TRUE is 1, FALSE is 0 (globals.lua)
	}
	return lua.LNumber(0)
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// bridgeWriteBack is the end of run_script (scripts.c:1805-1815). The code
// reads as "table_to_char for ch, then for me unless me is ch", but
// table_to_char reads its table from stack slot 1 (lua_pushvalue(L, 1)) and
// pops nothing. A top-level run_script starts on an empty stack (the last
// one ended with clear_stack, and lua_setglobal popped retval), so slot 1 is
// the ch table both times: ch is written back twice and me never is. A
// script's changes to me are lost, even when me is the same character as ch
// (a pulse trigger's ch and me are separate tables). A run_script nested
// inside a binding (a script's action() reaching another script) finds that
// binding's arguments in slot 1 instead; the port does not model that case.
func (e *Engine) bridgeWriteBack(b Bridge, ctx *ScriptContext) {
	if ctx.ChRef == nil || !b.InRoomAboveZero(*ctx.ChRef) {
		return
	}
	tableToChar(b, e.l.GetGlobal("ch"))
}
