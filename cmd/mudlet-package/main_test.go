package main

import (
	"bytes"
	"encoding/xml"
	"os"
	"strings"
	"testing"

	lua "github.com/yuin/gopher-lua"

	"github.com/zax0rz/darkpawns/mudlet"
)

const root = "../../mudlet"

// TestPackageIsCurrent: the committed darkpawns.xml is exactly what the
// sources build, so the file players import cannot drift from src/.
func TestPackageIsCurrent(t *testing.T) {
	built, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile(root + "/darkpawns.xml")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(built, committed) {
		t.Fatal("mudlet/darkpawns.xml is out of date: run go run ./cmd/mudlet-package")
	}
}

// TestPackageXMLRoundTrips parses the package the way Mudlet's importer
// does and checks every script survives escaping byte for byte.
func TestPackageXMLRoundTrips(t *testing.T) {
	built, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	var pkg struct {
		XMLName xml.Name `xml:"MudletPackage"`
		Scripts []struct {
			Name   string `xml:"name"`
			Script string `xml:"script"`
		} `xml:"ScriptPackage>ScriptGroup>Script"`
		Aliases []struct {
			Regex string `xml:"regex"`
		} `xml:"AliasPackage>Alias"`
	}
	if err := xml.Unmarshal(built, &pkg); err != nil {
		t.Fatalf("package XML does not parse: %v", err)
	}
	_, scripts, err := Scripts(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(pkg.Scripts) != len(scripts) {
		t.Fatalf("package has %d scripts, sources have %d", len(pkg.Scripts), len(scripts))
	}
	for i, script := range scripts {
		if pkg.Scripts[i].Name != script.Name || pkg.Scripts[i].Script != script.Source {
			t.Fatalf("script %q did not round-trip through XML", script.Name)
		}
	}
	if len(pkg.Aliases) != 1 || pkg.Aliases[0].Regex != `^dp(?:\s+(\w+))?$` {
		t.Fatalf("aliases = %+v", pkg.Aliases)
	}
}

// TestVersionIsSingleSourced: the Go constant the server announces in
// Client.GUI, the version baked into the scripts, and the newest changelog
// entry all agree.
func TestVersionIsSingleSourced(t *testing.T) {
	version, scripts, err := Scripts(root)
	if err != nil {
		t.Fatal(err)
	}
	if version != mudlet.Version || version == "" {
		t.Fatalf("VERSION %q, mudlet.Version %q", version, mudlet.Version)
	}
	if !strings.Contains(scripts[0].Source, `DarkPawns.version = "`+version+`"`) {
		t.Fatal("core script does not carry the package version")
	}
	changelog, err := os.ReadFile(root + "/CHANGELOG.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(changelog), "\n") {
		if strings.HasPrefix(line, "## ") {
			if !strings.HasPrefix(line, "## "+version+" ") && line != "## "+version {
				t.Fatalf("newest CHANGELOG.md entry is %q; add one for %s", line, version)
			}
			return
		}
	}
	t.Fatal("CHANGELOG.md has no version entries")
}

// mudletStub is the slice of Mudlet's Lua API the package touches, recorded
// so a test can load the package and drive it with GMCP events. Anything the
// package calls that is missing here fails the load, the way a nil global
// would put an error in Mudlet's console.
const mudletStub = `
gmcp = {}
calls = {}
local function record(name, ...) table.insert(calls, {name = name, args = {...}}) end

handlers, nextHandler = {}, 0
function registerAnonymousEventHandler(event, fn)
  nextHandler = nextHandler + 1
  handlers[nextHandler] = {event = event, fn = fn}
  return nextHandler
end
function killAnonymousEventHandler(id) handlers[id] = nil end
function fire(event, ...)
  for _, h in pairs(handlers) do
    if h.event == event then h.fn(event, ...) end
  end
end
function handlerCount(event)
  local n = 0
  for _, h in pairs(handlers) do if h.event == event then n = n + 1 end end
  return n
end

yajl = {to_string = function(list)
  local parts = {}
  for i, v in ipairs(list) do parts[i] = '"' .. v .. '"' end
  return "[" .. table.concat(parts, ",") .. "]"
end}
connected = true
function getConnectionInfo() return "darkpawns.org", 7777, connected end
function sendGMCP(msg) record("sendGMCP", msg) end
function send(cmd, echo) record("send", cmd) end
function getMainWindowSize() return 1000, 700 end
function setBorderRight(px) record("setBorderRight", px) end
function getTime() return "12:00" end
function cecho(text) record("cecho", text) end
function ansi2decho(text) return text end

local function widget(kind, cons)
  local w = {kind = kind, cons = cons, hidden = false, log = {}}
  function w:setStyleSheet(css) self.css = css end
  function w:show() self.hidden = false end
  function w:hide() self.hidden = true end
  function w:echo(text) self.text = text end
  function w:decho(text) table.insert(self.log, text) end
  function w:clear() self.log = {} end
  function w:setValue(current, maximum, text) self.value = {current, maximum, text} end
  return w
end
Geyser = {Fixed = "fixed", Dynamic = "dynamic"}
for _, kind in ipairs({"VBox", "Label", "Gauge", "Mapper", "MiniConsole"}) do
  Geyser[kind] = {new = function(self, cons, parent)
    local w = widget(kind, cons)
    if kind == "Gauge" then
      w.front, w.back, w.text = widget("Label"), widget("Label"), widget("Label")
    end
    return w
  end}
end

rooms, areas, nextArea = {}, {}, 0
function addAreaName(name) nextArea = nextArea + 1; areas[name] = nextArea; return nextArea end
function getAreaTable() return areas end
function addRoom(id) rooms[id] = {exits = {}, stubs = {}}; return true end
function roomExists(id) return rooms[id] ~= nil end
function setRoomArea(id, area) rooms[id].area = area end
function getRoomArea(id) return rooms[id].area end
function setRoomCoordinates(id, x, y, z) rooms[id].xyz = {x, y, z} end
function getRoomCoordinates(id) local c = rooms[id].xyz; return c[1], c[2], c[3] end
function setRoomName(id, name) rooms[id].name = name end
function setExit(from, to, dir) rooms[from].exits[dir] = to; return true end
function setExitStub(id, dir, on) rooms[id].stubs[dir] = on or nil end
function setCustomEnvColor(...) end
function setRoomEnv(id, env) rooms[id].env = env end
function centerview(id) centred = id end
speedWalkDir = {}
function getPath(from, to) speedWalkDir = {"n", "up"}; return true end
`

// scenario drives the loaded package the way a session on the four-room keep
// in pkg/telnet's GMCP harness would, and asserts what Mudlet would show.
const scenario = `
local function check(ok, msg) if not ok then error(msg, 2) end end
local function sent(name, value)
  for _, c in ipairs(calls) do
    if c.name == name and c.args[1] == value then return true end
  end
  return false
end

-- Loading while connected with GMCP already up negotiates immediately.
check(not sent("sendGMCP", 'Core.Supports.Add ["Char 1","Room 1","Comm.Channel 1"]'), "negotiated with an empty gmcp table")
fire("sysProtocolEnabled", "GMCP")
check(sent("sendGMCP", 'Core.Supports.Add ["Char 1","Room 1","Comm.Channel 1"]'), "no Core.Supports.Add on GMCP enable")
check(DarkPawns.ui.dock and not DarkPawns.ui.dock.hidden, "dock not built")

gmcp.Char = {Vitals = {hp = 80, maxhp = 100, mp = 20, maxmp = 20, mv = 98, maxmv = 100},
             Status = {name = "Walker", level = 7, race = "Elven", class = "Magic User", gold = 12}}
fire("gmcp.Char.Vitals")
fire("gmcp.Char.Status")
local hp = DarkPawns.ui.gauges.hp.value
check(hp[1] == 80 and hp[2] == 100 and hp[3] == "HP 80 / 100", "hp gauge")
check(DarkPawns.ui.gauges.mv.value[3] == "Move 98 / 100", "move gauge")
check(DarkPawns.ui.status.text:find("Walker", 1, true) and DarkPawns.ui.status.text:find("Magic User", 1, true), "status line")

local function enter(num, name, area, exits)
  gmcp.Room = {Info = {num = num, name = name, area = area, environment = "Inside", exits = exits}}
  fire("gmcp.Room.Info")
end
enter(8004, "The Gate Hall", "The Test Keep", {n = 8005, w = 8006})
check(rooms[8004].xyz[1] == 0 and rooms[8004].xyz[2] == 0, "first room not at origin")
check(rooms[8004].stubs.n and rooms[8004].stubs.w, "unexplored exits not stubbed")
enter(8005, "A Narrow Stair", "The Test Keep", {s = 8004})
check(rooms[8005].xyz[2] == 1 and rooms[8005].xyz[1] == 0, "stair not placed one step north")
check(rooms[8004].exits.n == 8005 and rooms[8005].exits.s == 8004, "exits not linked both ways")
check(not rooms[8004].stubs.n, "stub not cleared once its room was mapped")
enter(8004, "The Gate Hall", "The Test Keep", {n = 8005, w = 8006})
check(centred == 8004, "map did not recentre on a known room")
enter(9001, "Elsewhere", "Another Zone", {})
check(rooms[9001].area ~= rooms[8004].area and rooms[9001].xyz[1] == 0, "new area not started at its own origin")

gmcp.Comm = {Channel = {Text = {channel = "gossip", talker = "Someone", text = "Someone gossips, 'hi'"}}}
fire("gmcp.Comm.Channel.Text")
local log = DarkPawns.ui.chat.log
check(#log == 2 and log[2] == "Someone gossips, 'hi'\n", "chat line not captured")

speedWalkFrom, speedWalkTo = 8004, 8005
doSpeedWalk()
check(sent("send", "n") and sent("send", "up"), "speedwalk did not send the path")

-- Reloading the package replaces its handlers instead of stacking them.
local before = handlerCount("gmcp.Room.Info")
reload()
check(handlerCount("gmcp.Room.Info") == before, "reload stacked a second Room.Info handler")

fire("sysUninstallPackage", "darkpawns")
check(DarkPawns.ui.dock.hidden and handlerCount("gmcp.Room.Info") == 0, "uninstall left the dock or handlers behind")
`

// TestPackageLoadsAndDrives loads every script into a Lua 5.1 state with the
// Mudlet stub, then replays a session: this is the closest thing to "no
// console errors on load" that runs without Mudlet itself.
func TestPackageLoadsAndDrives(t *testing.T) {
	_, scripts, err := Scripts(root)
	if err != nil {
		t.Fatal(err)
	}
	L := lua.NewState()
	defer L.Close()
	if err := L.DoString(mudletStub); err != nil {
		t.Fatalf("stub: %v", err)
	}
	load := func() {
		for _, script := range scripts {
			fn, err := L.LoadString(script.Source)
			if err != nil {
				t.Fatalf("%s does not compile: %v", script.Name, err)
			}
			L.Push(fn)
			if err := L.PCall(0, lua.MultRet, nil); err != nil {
				t.Fatalf("%s fails on load: %v", script.Name, err)
			}
		}
	}
	load()
	L.SetGlobal("reload", L.NewFunction(func(*lua.LState) int { load(); return 0 }))
	if err := L.DoString(scenario); err != nil {
		t.Fatal(err)
	}
}
