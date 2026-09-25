package scripting

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

// fakeBridge is a Bridge over a handful of characters, objects and rooms,
// recording what the bindings ask the game to do.
type fakeBridge struct {
	ScriptableWorld // nil; the bridge path does not use it

	chars map[CharRef]CharFields
	objs  map[ObjRef]ObjFields
	rooms map[int]RoomFields
	room  map[CharRef]int

	applied  []appliedChar
	acts     []string
	commands []string
	says     []string
	logs     []string

	// onCommand, when set, runs for each Command (used to nest a script).
	onCommand func(me CharRef, line string)
}

type appliedChar struct {
	Ref   CharRef
	Write CharWrite
}

func (f *fakeBridge) CharFields(ref CharRef) (CharFields, bool) {
	c, ok := f.chars[ref]
	return c, ok
}

func (f *fakeBridge) ObjFields(ref ObjRef) (ObjFields, bool) {
	o, ok := f.objs[ref]
	return o, ok
}

func (f *fakeBridge) RoomFields(vnum int) (RoomFields, bool) {
	r, ok := f.rooms[vnum]
	return r, ok
}

func (f *fakeBridge) ApplyChar(ref CharRef, w CharWrite) {
	f.applied = append(f.applied, appliedChar{ref, w})
}
func (f *fakeBridge) ApplyObj(ObjRef, ObjWrite) {}
func (f *fakeBridge) CharRoom(ref CharRef) (int, bool) {
	r, ok := f.room[ref]
	return r, ok
}
func (f *fakeBridge) InRoomAboveZero(ref CharRef) bool { return f.room[ref] > 0 }
func (f *fakeBridge) Act(txt string, _ bool, _ *CharRef, _ *ObjRef, _ *CharRef, _ int) {
	f.acts = append(f.acts, txt)
}
func (f *fakeBridge) Say(_ CharRef, text string) { f.says = append(f.says, text) }
func (f *fakeBridge) Emote(CharRef, string)      {}
func (f *fakeBridge) Tell(CharRef, string)       {}
func (f *fakeBridge) Command(me CharRef, line string) {
	f.commands = append(f.commands, line)
	if f.onCommand != nil {
		f.onCommand(me, line)
	}
}
func (f *fakeBridge) CharIsHunting(CharRef) bool                { return false }
func (f *fakeBridge) ExtractObj(ObjRef)                         {}
func (f *fakeBridge) LoadObjToChar(int, CharRef) (ObjRef, bool) { return ObjRef{}, false }
func (f *fakeBridge) LoadObjToRoom(int, int) (ObjRef, bool)     { return ObjRef{}, false }
func (f *fakeBridge) SetSkill(CharRef, int, int)                {}
func (f *fakeBridge) Teleport(CharRef, int) bool                { return false }
func (f *fakeBridge) RawKill(CharRef, *CharRef, int)            {}
func (f *fakeBridge) Log(msg string)                            { f.logs = append(f.logs, msg) }

func (f *fakeBridge) CanSee(CharRef, CharRef) bool           { return true }
func (f *fakeBridge) InWorldMob(int) (CharRef, bool)         { return CharRef{}, false }
func (f *fakeBridge) InWorldChar(string) (CharRef, bool)     { return CharRef{}, false }
func (f *fakeBridge) AffFlagged(CharRef, int) bool           { return false }
func (f *fakeBridge) SetAffFlag(CharRef, int, bool)          {}
func (f *fakeBridge) ActFlagged(CharRef, int) bool           { return false }
func (f *fakeBridge) SetActFlag(CharRef, int, bool)          {}
func (f *fakeBridge) ObjFlagged(ObjRef, int) bool            { return false }
func (f *fakeBridge) SetObjExtra(ObjRef, int, bool)          {}
func (f *fakeBridge) RoomExitInfo(RoomRef, int) (int, bool)  { return 0, false }
func (f *fakeBridge) SetRoomExitInfo(RoomRef, int, int) bool { return false }
func (f *fakeBridge) SetRoomSector(RoomRef, int)             {}
func (f *fakeBridge) LoadMob(int, int) (CharRef, bool)       { return CharRef{}, false }
func (f *fakeBridge) ExtractChar(CharRef)                    {}
func (f *fakeBridge) ObjList(CharRef, string, string) (ObjRef, *CharRef, bool) {
	return ObjRef{}, nil, false
}
func (f *fakeBridge) ObjFrom(ObjRef, string)          {}
func (f *fakeBridge) ObjToRoom(ObjRef, int) bool      { return false }
func (f *fakeBridge) ObjToChar(ObjRef, CharRef)       {}
func (f *fakeBridge) ObjToObj(ObjRef, ObjRef)         {}
func (f *fakeBridge) Steal(CharRef, ObjRef)           {}
func (f *fakeBridge) EquipCharObj(CharRef, ObjRef)    {}
func (f *fakeBridge) AppendExtraDescs(ObjRef, string) {}

// The engine finds the bridge by a runtime type assertion; this keeps the
// fake a Bridge at compile time, so a new method cannot silently turn these
// tests into legacy-path tests.
var _ Bridge = (*fakeBridge)(nil)

var (
	player = CharRef{ID: 7}
	healer = CharRef{NPC: true, ID: 1001}
	guard  = CharRef{NPC: true, ID: 1002}
)

func newFakeBridge() *fakeBridge {
	return &fakeBridge{
		chars: map[CharRef]CharFields{
			player: {
				Name: "Zach", Alias: "zach", Align: -400, Gold: 50, Level: 10, HP: 90, MaxHP: 100,
				Pos: 8, Evil: true, IDNum: 7, Exp: 1234, Carry: []ObjRef{{ID: 501}},
			},
			healer: {
				Name: "the healer", Alias: "healer", Gold: 3, Level: 20, HP: 200, MaxHP: 200,
				Pos: 8, NPC: true, VNum: 12220, Timer: 0,
			},
			guard: {
				Name: "a guard", Alias: "guard", Level: 15, HP: 150, MaxHP: 150,
				Pos: 8, NPC: true, VNum: 12221, Master: &healer,
			},
		},
		objs: map[ObjRef]ObjFields{
			{ID: 501}: {
				Name: "a loaf of bread", Alias: "bread loaf", VNum: 3010, Cost: 5, Type: 19, Weight: 1,
				Val: [4]int{24, 0, 0, 0},
			},
		},
		rooms: map[int]RoomFields{
			12200: {
				VNum: 12200, Sect: 1, Exits: [6]int{12201, -1, -1, -1, -1, -1},
				People: []CharRef{guard, healer, player}, Objs: nil,
			},
		},
		room: map[CharRef]int{player: 12200, healer: 12200, guard: 12200},
	}
}

func writeScript(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func bridgeContext(b *fakeBridge, ch, me CharRef, argument string) *ScriptContext {
	return &ScriptContext{World: b, ChRef: &ch, MeRef: &me, RoomVNum: 12200, Argument: argument}
}

// The tables a script sees are C's char_to_table and room_to_table: player
// and mobile fields by kind, IS_EVIL as 1/0, the leader chain, and a
// room.char list without me.
func TestBridgeTables(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "t.lua", `function ongive()
  seen = ch.name .. "|" .. ch.evil .. "|" .. ch.exp .. "|" .. ch.objs[1].name .. "|" .. ch.objs[1].val[1]
  seen = seen .. "|" .. me.vnum .. "|" .. tostring(me.exp) .. "|" .. room.vnum .. "|" .. room.exit[0]
  seen = seen .. "|" .. getn(room.char) .. "|" .. room.char[1].name .. "|" .. room.char[1].leader.name
  seen = seen .. "|" .. argument
  log(seen)
end
`)
	b := newFakeBridge()
	e := NewEngine(dir, nil)
	defer e.Close()
	if _, err := e.RunScript(bridgeContext(b, player, healer, "hello"), "t.lua", "ongive"); err != nil {
		t.Fatalf("RunScript: %v", err)
	}
	want := "Zach|1|1234|a loaf of bread|24|12220|nil|12200|12201|2|a guard|the healer|hello"
	if len(b.logs) != 1 || b.logs[0] != want {
		t.Fatalf("script saw %q, want %q", b.logs, want)
	}
}

// run_script's write-back (scripts.c:1805-1815) reaches only the ch table:
// table_to_char reads stack slot 1 both times. C's clamps apply; a change
// to me is lost.
func TestBridgeWriteBackIsChOnly(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "t.lua", `function ongive()
  ch.gold = -5
  ch.align = 5000
  ch.hp = -3
  me.gold = 999
  return TRUE
end
`)
	b := newFakeBridge()
	e := NewEngine(dir, nil)
	defer e.Close()
	handled, err := e.RunScript(bridgeContext(b, player, healer, ""), "t.lua", "ongive")
	if err != nil || !handled {
		t.Fatalf("RunScript = %v, %v; want true, nil", handled, err)
	}
	if len(b.applied) != 1 || b.applied[0].Ref != player {
		t.Fatalf("applied %+v, want one write-back to the player", b.applied)
	}
	w := b.applied[0].Write
	if w.Gold != 0 || w.Align != 1000 || w.HP != -3 || w.Exp != 1234 {
		t.Fatalf("write-back %+v, want gold 0, align 1000, hp -3 (unclamped), exp unchanged", w)
	}
}

// No write-back at all when ch is not in a room above 0.
func TestBridgeWriteBackNeedsRoom(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "t.lua", `function ongive() ch.gold = 1 end`)
	b := newFakeBridge()
	b.room[player] = 0
	e := NewEngine(dir, nil)
	defer e.Close()
	if _, err := e.RunScript(bridgeContext(b, player, healer, ""), "t.lua", "ongive"); err != nil {
		t.Fatalf("RunScript: %v", err)
	}
	if len(b.applied) != 0 {
		t.Fatalf("applied %+v, want none", b.applied)
	}
}

// retval = (int)lua_tonumber(L, -1): any nonzero number handles the event.
func TestBridgeReturnValue(t *testing.T) {
	cases := []struct {
		ret  string
		want bool
	}{
		{"TRUE", true},
		{"FALSE", false},
		{"2", true},
		{"-1", true},
		{"0.5", false},
		{`"1"`, true},
		{"nil", false},
		{"", false},
	}
	for _, c := range cases {
		dir := t.TempDir()
		writeScript(t, dir, "t.lua", "function oncmd()\n  return "+c.ret+"\nend\n")
		b := newFakeBridge()
		e := NewEngine(dir, nil)
		handled, err := e.RunScript(bridgeContext(b, player, healer, "south"), "t.lua", "oncmd")
		e.Close()
		if err != nil || handled != c.want {
			t.Errorf("return %s: handled = %v, %v; want %v", c.ret, handled, err, c.want)
		}
	}
}

// The say/act/action bindings reach the game through the bridge, as me.
func TestBridgeBindings(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "t.lua", `function oncmd()
  act("$n blocks $N's way.", TRUE, me, NIL, ch, TO_NOTVICT)
  say("I cannot let you pass.")
  action(me, "bow zach")
  return TRUE
end
`)
	b := newFakeBridge()
	e := NewEngine(dir, nil)
	defer e.Close()
	if _, err := e.RunScript(bridgeContext(b, player, healer, "south"), "t.lua", "oncmd"); err != nil {
		t.Fatalf("RunScript: %v", err)
	}
	if !reflect.DeepEqual(b.acts, []string{"$n blocks $N's way."}) ||
		!reflect.DeepEqual(b.says, []string{"I cannot let you pass."}) ||
		!reflect.DeepEqual(b.commands, []string{"bow zach"}) {
		t.Fatalf("acts %q says %q commands %q", b.acts, b.says, b.commands)
	}
}

// A script whose binding reaches another script runs it nested, as C's
// run_script does: no deadlock, the outer script keeps its bridge, and it
// continues with the globals the inner run left behind.
func TestBridgeNestedRun(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "outer.lua", `function oncmd()
  action(me, "give bread guard")
  say("after " .. me.name)
  return TRUE
end
`)
	writeScript(t, dir, "inner.lua", `function ongive()
  say("inner " .. me.name)
end
`)
	b := newFakeBridge()
	e := NewEngine(dir, nil)
	defer e.Close()
	b.onCommand = func(me CharRef, _ string) {
		if _, err := e.RunScript(bridgeContext(b, me, guard, ""), "inner.lua", "ongive"); err != nil {
			t.Errorf("nested RunScript: %v", err)
		}
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		handled, err := e.RunScript(bridgeContext(b, player, healer, "south"), "outer.lua", "oncmd")
		if err != nil || !handled {
			t.Errorf("outer RunScript = %v, %v", handled, err)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("nested RunScript deadlocked")
	}
	want := []string{"inner a guard", "after a guard"}
	if !reflect.DeepEqual(b.says, want) {
		t.Fatalf("says %q, want %q", b.says, want)
	}
	if e.activeBridge != nil || e.owner.Load() != 0 {
		t.Fatalf("engine left running state behind: bridge %v owner %d", e.activeBridge, e.owner.Load())
	}
}

// A bridged run does not clear the globals it was not given: C's
// run_script sets ch, me, room, obj and argument only when it has them.
func TestBridgeGlobalsPersistAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "t.lua", `function oncmd() log("argument=" .. tostring(argument)) end`)
	b := newFakeBridge()
	e := NewEngine(dir, nil)
	defer e.Close()
	if _, err := e.RunScript(bridgeContext(b, player, healer, "south"), "t.lua", "oncmd"); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RunScript(bridgeContext(b, player, healer, ""), "t.lua", "oncmd"); err != nil {
		t.Fatal(err)
	}
	want := []string{"argument=south", "argument=south"}
	if !reflect.DeepEqual(b.logs, want) {
		t.Fatalf("logs %q, want %q", b.logs, want)
	}
}

// Lua 4's string functions as C's scripts call them: strfind's several
// results feed strsub (no_move.lua) and a capture (assembler.lua's
// return_obj), format prints %u, and tonumber is the builtin.
func TestLua4StringFunctions(t *testing.T) {
	dir := t.TempDir()
	writeScript(t, dir, "t.lua", `function oncmd()
  local command = strsub(argument, 1, strfind(argument, "%a%s"))
  local found, e, alias = strfind("bread loaf", "^(%a+)")
  log(command .. "|" .. found .. "|" .. e .. "|" .. alias)
  log(format("That will cost %u coins", 2000) .. "|" .. format("%u", -1) .. "|" .. format("%d%%", 5))
  log(gsub("a-b-c", "-", "+") .. "|" .. tonumber("42") + 1)
end
`)
	b := newFakeBridge()
	e := NewEngine(dir, nil)
	defer e.Close()
	if _, err := e.RunScript(bridgeContext(b, player, healer, "south please"), "t.lua", "oncmd"); err != nil {
		t.Fatalf("RunScript: %v", err)
	}
	want := []string{"south|1|5|bread", "That will cost 2000 coins|4294967295|5%", "a+b+c|43"}
	if !reflect.DeepEqual(b.logs, want) {
		t.Fatalf("logs %q, want %q", b.logs, want)
	}
}
