package game

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/scripting"
)

// lua_char_table_depth_test.go — DP-1433. C's char_to_table (src/scripts.c:1823)
// has exactly one shape for both character kinds. These tests run a real Lua
// script through the production bridge (WorldScriptableAdapter) and read the
// table back out of the script's own say() line, so a field the adapter stops
// supplying, an invented key, or a wrong wear order fails on exact bytes.

// luaCharTableObject is the prototype of one table-fixture object.
func luaCharTableObject(vnum int, shortDesc string) parser.Obj {
	return parser.Obj{
		VNum:      vnum,
		Keywords:  strings.ToLower(shortDesc),
		ShortDesc: shortDesc,
		TypeFlag:  12, // ITEM_OTHER
		Weight:    1,
	}
}

// luaCharTableFixture is a room with a PC (carrying and wearing one object,
// following a leader), an NPC (same), and an observer that receives the
// script's say() line.
type luaCharTableFixture struct {
	world    *World
	adapter  *WorldScriptableAdapter
	pc       *Player
	npc      *MobInstance
	leader   *Player
	observer *Player
	output   map[string]*strings.Builder
}

func newLuaCharTableFixture(t *testing.T) *luaCharTableFixture {
	t.Helper()
	w, err := NewWorld(&parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Table Room", Zone: 1}},
		Mobs: []parser.Mob{{
			VNum: 9001, Keywords: "healer probe", ShortDesc: "the healer",
			Level: 20, Alignment: -400, Gold: 7,
		}},
		Objs: []parser.Obj{
			luaCharTableObject(3001, "a pc trinket"),
			luaCharTableObject(3002, "a pc cuirass"),
			luaCharTableObject(3003, "an npc bauble"),
			luaCharTableObject(3004, "an npc helm"),
		},
	})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	spawn := func(vnum int) *ObjectInstance {
		t.Helper()
		obj, err := w.SpawnObject(vnum, 1001)
		if err != nil {
			t.Fatalf("SpawnObject %d: %v", vnum, err)
		}
		return obj
	}

	leader := NewPlayer(10, "Chainleader", 1001)
	leader.SetPosition(8)
	if err := w.AddPlayer(leader); err != nil {
		t.Fatalf("AddPlayer leader: %v", err)
	}
	observer := NewPlayer(11, "Watcher", 1001)
	observer.SetPosition(8)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatalf("AddPlayer observer: %v", err)
	}
	retainer := NewPlayer(13, "Retainer", 1001)
	retainer.SetPosition(8)
	if err := w.AddPlayer(retainer); err != nil {
		t.Fatalf("AddPlayer retainer: %v", err)
	}

	pc := NewPlayer(12, "Tabletester", 1001)
	pc.SetPosition(8)
	pc.Level = 17
	pc.Alignment = -400
	pc.Gold = 33
	pc.BankGold = 44
	pc.Exp = 555
	pc.Mana = 66
	pc.MaxMana = 70
	pc.Move = 77
	pc.MaxMove = 80
	pc.Health = 88
	pc.MaxHealth = 90
	pc.JailTimer = 3
	pc.Tattoo = 9
	pc.Stats.Cha = 15
	pc.CopyBaseAttributes()
	if err := w.AddPlayer(pc); err != nil {
		t.Fatalf("AddPlayer pc: %v", err)
	}
	if err := pc.Inventory.AddItem(spawn(3001)); err != nil {
		t.Fatalf("pc inventory: %v", err)
	}
	if err := pc.Equipment.SetSlot(SlotBody, spawn(3002)); err != nil {
		t.Fatalf("pc equip: %v", err)
	}
	// C's ch->master is the follow master.
	if CircleFollow(w, pc, leader) {
		t.Fatal("pc -> leader would create a follow cycle")
	}
	AddFollowerQuiet(pc, leader)
	// C's num_followers(ch) counts ch's own followers (src/act.other.c), not
	// the chain above it: a retainer follows the PC so followers is 1.
	AddFollowerQuiet(retainer, pc)

	npc, err := w.SpawnMob(9001, 1001)
	if err != nil {
		t.Fatalf("SpawnMob: %v", err)
	}
	// NewMob rolls C's +/-20% gold variance (db.c:1766-1774), so pin the value
	// the script reads back.
	npc.SetGold(7)
	npc.Inventory = append(npc.Inventory, spawn(3003))
	npc.Equipment[6] = spawn(3004) // C WEAR_HEAD
	AddFollowerQuietMob(npc, leader)

	return &luaCharTableFixture{
		world: w, adapter: NewWorldScriptableAdapter(w), pc: pc, npc: npc, leader: leader,
		observer: observer, output: captureMovementOutput(w),
	}
}

// runScript writes one trigger body into a temp scripts dir and runs it as a
// real mobile script through the bridge.
func (f *luaCharTableFixture) runScript(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "globals.lua"), nil, 0o600); err != nil {
		t.Fatalf("globals.lua: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "table.lua"), []byte(body), 0o600); err != nil {
		t.Fatalf("table.lua: %v", err)
	}
	engine := scripting.NewEngine(dir, f.adapter)
	t.Cleanup(engine.Close)

	chRef := scripting.CharRef{ID: f.pc.ID}
	meRef := scripting.CharRef{NPC: true, ID: f.npc.GetID()}
	ctx := &scripting.ScriptContext{
		World: f.adapter, ChRef: &chRef, MeRef: &meRef, RoomVNum: 1001, OwnerType: "mob",
	}
	handled, err := engine.RunScript(ctx, "table.lua", "oncmd")
	if err != nil || !handled {
		t.Fatalf("RunScript: handled=%t err=%v", handled, err)
	}
}

// said returns the observer's line with the say() framing removed.
func (f *luaCharTableFixture) said(t *testing.T) string {
	t.Helper()
	line := f.output[f.observer.Name].String()
	const prefix = "The healer says, '"
	const suffix = "'\r\n"
	if !strings.HasPrefix(line, prefix) || !strings.HasSuffix(line, suffix) {
		t.Fatalf("observer line = %q, want the say() framing %q...%q", line, prefix, suffix)
	}
	return strings.TrimSuffix(strings.TrimPrefix(line, prefix), suffix)
}

var _ = strconv.Itoa

// TestLuaCharTablePlayerFields pins C's player arm field for field, including
// nil for the mobile-only keys and the absence of the invented skills table.
// Source: src/scripts.c:1832-1890.
func TestLuaCharTablePlayerFields(t *testing.T) {
	f := newLuaCharTableFixture(t)
	f.runScript(t, `function oncmd()
  say("pc|" .. ch.name .. "|" .. ch.align .. "|" .. ch.gold .. "|" .. ch.level ..
      "|" .. ch.hp .. "|" .. ch.maxhp .. "|" .. ch.pos .. "|" .. ch.alias ..
      "|" .. ch.objs[1].name .. "|" .. ch.wear[1].name .. "|" .. ch.evil ..
      "|" .. ch.leader.name .. "|" .. ch.id .. "|" .. ch.exp .. "|" .. ch.bank ..
      "|" .. ch.mana .. "|" .. ch.move .. "|" .. ch.cha .. "|" .. ch.jail ..
      "|" .. ch.tattoo .. "|" .. ch.followers ..
      "|" .. tostring(ch.vnum) .. "|" .. tostring(ch.timer) ..
      "|" .. tostring(ch.skills))
  return TRUE
end
`)
	want := "pc|Tabletester|-400|33|17|88|90|8|Tabletester|a pc trinket|a pc cuirass|1|" +
		"Chainleader|12|555|44|66|77|15|3|9|1|nil|nil|nil"
	if got := f.said(t); got != want {
		t.Fatalf("player table = %q, want %q", got, want)
	}
}

// TestLuaCharTableMobileFields pins C's mobile arm, including the carried and
// worn object lists the legacy builder used to leave empty.
// Source: src/scripts.c:1885-1888.
func TestLuaCharTableMobileFields(t *testing.T) {
	f := newLuaCharTableFixture(t)
	f.runScript(t, `function oncmd()
  say("npc|" .. me.name .. "|" .. me.align .. "|" .. me.gold .. "|" .. me.level ..
      "|" .. me.hp .. "|" .. me.maxhp .. "|" .. me.pos .. "|" .. me.alias ..
      "|" .. me.objs[1].name .. "|" .. me.wear[1].name .. "|" .. me.evil ..
      "|" .. me.leader.name .. "|" .. me.vnum .. "|" .. me.timer ..
      "|" .. tostring(me.id) .. "|" .. tostring(me.exp) .. "|" .. tostring(me.bank) ..
      "|" .. tostring(me.followers) .. "|" .. tostring(me.skills))
  return TRUE
end
`)
	want := "npc|the healer|-400|7|20|" + strconv.Itoa(f.npc.GetHP()) + "|" +
		strconv.Itoa(f.npc.GetMaxHP()) +
		"|8|healer probe|an npc bauble|an npc helm|1|Chainleader|9001|0|nil|nil|nil|nil|nil"
	if got := f.said(t); got != want {
		t.Fatalf("mobile table = %q, want %q", got, want)
	}
}
