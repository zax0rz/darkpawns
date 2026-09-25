package oraclediff

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

const enterStep = "<ENTER>"

const enterGameStep = "1"

const pulseControl = "~dpclock pulse "

// Scenario is a split differential script: per-server setup (not diffed) plus
// a shared probe (diffed block-by-block).
type Scenario struct {
	Name        string
	SetupOracle []string
	SetupPort   []string
	// ReloginOracle and ReloginPort are the login lines each server needs for
	// a returning character, played by ReloginStep.
	ReloginOracle []string
	ReloginPort   []string
	Warmup        []string
	Probe         []string
	ProbeActor    string
	// PeerDrop names one passive peer whose TCP connection is closed after
	// setup/warmup and before the compared probe. The character remains in the
	// live-world lifecycle, which exposes C's linkless descriptor branches.
	PeerDrop          string
	Peers             map[string]*PeerSetup
	Fixtures          []ObjectFixture
	ObjectSpawns      []ObjectSpawnFixture
	MobFixtures       []MobFixture
	MobObjectFixtures []MobObjectFixture
	MobAffFixtures    []MobAffFixture
	MobFlagFixtures   []MobFlagFixture
	ObjIndexFixtures  []ObjIndexFixture
	WldIndexFixtures  []WldIndexFixture
	QuietZones        []int
	QuietAllMobs      bool
	EmptyPlayers      bool
	ScriptlessMobIDs  []int
	ForceLoadVNums    []int
	RoomExitFixtures  []RoomExitFixture
	RoomFlagFixtures  []RoomFlagFixture
	RoomSectors       []RoomSectorFixture
	HouseControls     []HouseControlFixture
	ScriptTwin        *ScriptTwinFixture
	// LuaScripts are files written into both servers' script trees, and
	// MobScripts attach a script to a mobile prototype in both worlds. With
	// them a scenario certifies a Lua binding that no script C ships calls.
	LuaScripts []LuaScriptFixture
	MobScripts []MobScriptFixture
	// SkipSetupSettle leaves the frozen clock untouched after character
	// creation. Focused vehicles use this when a spawned autonomous mob must
	// survive until a later warmup command places the actor beside it.
	SkipSetupSettle bool
	// KeepANSI compares probe blocks with ANSI escapes intact (NormalizeKeepANSI)
	// instead of stripping them. It is the raw-byte proof mode for C surfaces
	// whose embedded colors are player-facing bytes, such as OLC menus colored
	// through get_char_cols. Scenarios using it must keep every probe block
	// inside that surface: colored prompts and vitals outside it are masked by
	// rules that expect ANSI already stripped.
	KeepANSI bool
	// KeepPrompts compares the captured telnet text bytes without Tier-1
	// normalization. Use only focused prompt scenarios: the ordinary normalizer
	// discards prompt lines, trailing spaces, and line-ending distinctions.
	KeepPrompts bool
	// EntryPromptOnly compares the final newline run and simple command prompt
	// of an un-settled creation transcript. Earlier entry text has separate
	// coverage and can contain unrelated raw-byte gaps.
	EntryPromptOnly bool
	// DiffSetup diffs the primary client's whole setup transcript (the
	// character-creation dialogue) as one normalized block, instead of
	// draining it. Set by the [creation:oracle]/[creation:port] sections,
	// whose keystrokes still feed the ordinary setup machinery.
	DiffSetup bool
}

// PeerSetup describes a passive client that remains connected while the
// primary actor runs the probe. This lets a scenario compare per-recipient
// room messages, including TO_VICT and TO_NOTVICT output.
type PeerSetup struct {
	SetupOracle []string
	SetupPort   []string
}

// ScriptTwinFixture describes a direct on-disk twin for a live editor save.
// The oracle vehicle writes Append plus LF through the editor, then the
// harness compares both disposable server files with the directly-written
// expected twin.
type ScriptTwinFixture struct {
	Path   string
	Append string
}

// LuaScriptFixture is a "[lua-script <path>]" section: the section's lines
// (trimmed; blank and #-comment lines dropped) written to <scripts>/<path>
// in both disposable trees. Path is relative to the scripts root, e.g.
// mob/oracle/probe.lua.
type LuaScriptFixture struct {
	Path  string
	Lines []string
}

// MobScriptFixture is "mob-script <vnum> <path> <flags>": the mobile's
// "Script: <path> <flags>" line, which both servers read relative to
// scripts/mob/ with C's MS_* flag mask.
type MobScriptFixture struct {
	MobVNum int
	Path    string
	Flags   int
}

// ObjectFixture identifies an object prototype to turn into an inert scroll in
// each server's disposable world copy. The source world trees are never modified.
type ObjectFixture struct {
	ObjectVNum int
}

// ObjectSpawnFixture adds one object reset command (O 0 vnum max room) to a
// disposable zone file. Used to place deterministic objects in a room for an
// oracle scenario without modifying either source world tree.
type ObjectSpawnFixture struct {
	ObjectVNum  int
	MaxExisting int
	RoomVNum    int
	ZoneNumber  int
}

// MobFixture adds one reset command to a disposable zone file. It is used to
// place deterministic special-procedure actors in an oracle scenario without
// modifying either source world trees.
type MobFixture struct {
	MobVNum     int
	MaxExisting int
	RoomVNum    int
	ZoneNumber  int
}

// MobObjectFixture appends a G reset for the last mob reset in a disposable
// zone. It is deliberately paired with spawn-mob in focused vehicles so a
// scenario can populate a known keeper without editing authoritative zones.
type MobObjectFixture struct {
	MobVNum     int
	ObjectVNum  int
	MaxExisting int
	ZoneNumber  int
}

// MobAffFixture patches a mob prototype's innate affected-by bitmask (the
// flag line's second field) in each server's disposable world copy. C
// read_mobile copies those bits onto every instance, and mag_affects'
// mob-affection gate (magic.c:1387-1394) refuses spells whose bitvector the
// mob carries innately — this fixture is the live vehicle for that gate.
type MobAffFixture struct {
	MobVNum int
	AffMask int
}

// MobFlagFixture sets or clears one action flag on a mob prototype in
// disposable worlds. This is used when an authoritative procedure's authored
// placement flags need a focused two-engine vehicle (for example, enabling a
// registered but dormant MOB_SPEC procedure).
type MobFlagFixture struct {
	MobVNum int
	Flag    string
	Enabled bool
}

// ObjIndexFixture adds one filename to the disposable obj index so the boot
// loader reads an otherwise-unindexed .obj file. Real world vehicles live in
// files the shipped index omits entirely (131.obj's plaid potion casts
// blindness, 58.obj's return scroll casts curse), which made every scenario
// step touching them vacuously fail on BOTH servers.
type ObjIndexFixture struct {
	FileName string
}

// WldIndexFixture adds a world-file filename to the disposable wld/index so
// an otherwise-unindexed authoritative room file can be loaded by both
// engines for a focused oracle vehicle.
type WldIndexFixture struct {
	FileName string
}

// RoomExitFixture replaces every exit on a disposable room with either no
// exits, one explicitly described exit, or all six directions to one room. Keeping this deliberately small
// makes RNG-sensitive movement scenarios deterministic without creating a
// second world-file language inside scenario files.
type RoomExitFixture struct {
	RoomVNum  int
	Direction string
	ToRoom    int
	DoorState int
	Keyword   string
}

// RoomFlagFixture enables or disables one C ROOM_* bit on a disposable room.
type RoomFlagFixture struct {
	RoomVNum int
	Bit      int
	Enabled  bool
}

// RoomSectorFixture replaces the sector type on one disposable room.
type RoomSectorFixture struct {
	RoomVNum int
	Sector   int
}

// HouseControlFixture adds one valid player-house record to both disposable
// engines. The C oracle consumes its native binary house-control record while
// the Go port consumes the equivalent JSON record.
type HouseControlFixture struct {
	VNum      int
	Atrium    int
	ExitNum   int
	Owner     int64
	PortOwner int64
	Key       int
}

// ProbeBlock is one probe command and the raw output it produced.
type ProbeBlock struct {
	Command string
	Output  string
}

// AudienceProbeBlock is one command's output as seen by one connected client.
type AudienceProbeBlock struct {
	Command  string
	Audience string
	Output   string
}

// ParseScenario reads a sectioned scenario file:
//
//	[setup:oracle]      # sent only to the C oracle; not diffed
//	<creation keystrokes…>
//	[setup:port]        # sent only to the Go port; not diffed
//	<creation keystrokes…>
//	[setup:oracle:victim] / [setup:port:victim]
//	<optional passive-client creation keystrokes…>
//	[fixture]
//	inert-scroll 8038         # patch this prototype in disposable worlds only
//	spawn-mob 18306 1 8162 80 # mob, max existing, room, zone file
//	set-mob-aff 18306 128     # mob, innate affected-by bitmask (AFF_* positions)
//	clear-mob-flag 4 RANDZON # clear one action flag in the disposable copy
//	set-mob-flag 14411 SPEC # set one action flag in the disposable copy
//	add-obj-index 131.obj    # load an otherwise-unindexed obj file's prototypes
//	add-wld-index 181.wld    # load an otherwise-unindexed room file
//	spawn-obj 8010 1 8004 80  # object, max existing, room, zone file
//	give-object 12100 12132 1 121 # mob, object, max existing, zone file
//	quiet-zone 80             # suppress mobile resets in a disposable zone
//	quiet-mobs                # suppress mobile resets in every disposable zone
//	strip-mob-script 18306    # force native special dispatch in both copies
//	force-load 4903           # rewrite the prototype load percent to 500% in both copies
//	replace-room-exits 8162 none
//	replace-room-exits 8162 all 8161 0
//	replace-room-exits 8162 north 8161 1 gate
//	set-room-flag 8161 1 on  # ROOM_DEATH
//	set-room-sector 8161 7   # SECT_WATER_NOSWIM
//	house-control 19676 19674 south 1 19604
//	house-control 19676 19674 south 1 0 19604 # optional Go-port owner
//	[warmup]            # shared commands sent and discarded after peer setup
//	get scroll
//	[peer-drop]         # close one named passive peer before [probe]
//	peer
//	[probe]             # sent to BOTH; this is the only diffed section
//	look
//	look sign
//	quit
//	[probe:victim]      # alternatively, send and diff from a named peer
//
// Blank lines and lines beginning with # are comments; <ENTER> represents an
// intentional empty command.
func ParseScenario(name string, r io.Reader) (Scenario, error) {
	sc := Scenario{Name: name, Peers: make(map[string]*PeerSetup)}
	scanner := bufio.NewScanner(r)
	var section *[]string
	fixtureSection := false
	scriptTwinSection := false
	var luaScript *LuaScriptFixture
	peerDropSection := false
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			fixtureSection = false
			scriptTwinSection = false
			peerDropSection = false
			luaScript = nil
			lower := strings.ToLower(line)
			if strings.HasPrefix(lower, "[lua-script ") {
				path := strings.TrimSpace(line[len("[lua-script ") : len(line)-1])
				if !safeScriptPath(path) {
					return Scenario{}, fmt.Errorf("scenario %q line %d: unsafe lua-script path %q", name, lineNo, path)
				}
				sc.LuaScripts = append(sc.LuaScripts, LuaScriptFixture{Path: path})
				luaScript = &sc.LuaScripts[len(sc.LuaScripts)-1]
				section = nil
				continue
			}
			switch lower {
			case "[setup:oracle]":
				section = &sc.SetupOracle
			case "[setup:port]", "[setup:go]":
				section = &sc.SetupPort
			case "[creation:oracle]":
				// Same keystroke stream as [setup:oracle], but the resulting
				// transcript is diffed rather than drained.
				section = &sc.SetupOracle
				sc.DiffSetup = true
			case "[creation:port]", "[creation:go]":
				section = &sc.SetupPort
				sc.DiffSetup = true
			case "[relogin:oracle]":
				section = &sc.ReloginOracle
			case "[relogin:port]", "[relogin:go]":
				section = &sc.ReloginPort
			case "[probe]":
				section = &sc.Probe
				sc.ProbeActor = ""
			case "[warmup]":
				section = &sc.Warmup
			case "[fixture]", "[fixtures]":
				section = nil
				fixtureSection = true
			case "[script-twin]":
				section = nil
				scriptTwinSection = true
			case "[peer-drop]":
				section = nil
				peerDropSection = true
			default:
				parts := strings.Split(strings.Trim(lower, "[]"), ":")
				if len(parts) == 2 && parts[0] == "probe" && parts[1] != "" {
					section = &sc.Probe
					sc.ProbeActor = parts[1]
					continue
				}
				if len(parts) != 3 || parts[0] != "setup" || parts[2] == "" {
					return Scenario{}, fmt.Errorf("scenario %q line %d: unknown section %q", name, lineNo, line)
				}
				peer := sc.Peers[parts[2]]
				if peer == nil {
					peer = &PeerSetup{}
					sc.Peers[parts[2]] = peer
				}
				switch parts[1] {
				case "oracle":
					section = &peer.SetupOracle
				case "port", "go":
					section = &peer.SetupPort
				default:
					return Scenario{}, fmt.Errorf("scenario %q line %d: unknown section %q", name, lineNo, line)
				}
			}
			continue
		}
		if luaScript != nil {
			luaScript.Lines = append(luaScript.Lines, line)
			continue
		}
		if scriptTwinSection {
			fields := strings.Fields(line)
			if len(fields) < 2 || sc.ScriptTwin != nil {
				return Scenario{}, fmt.Errorf("scenario %q line %d: invalid script twin %q", name, lineNo, line)
			}
			path := fields[0]
			if strings.HasPrefix(path, "/") || strings.Contains(path, "..") || !strings.HasSuffix(path, ".lua") {
				return Scenario{}, fmt.Errorf("scenario %q line %d: unsafe script twin path %q", name, lineNo, path)
			}
			appendText := strings.TrimSpace(strings.TrimPrefix(line, path))
			if appendText == "" {
				return Scenario{}, fmt.Errorf("scenario %q line %d: empty script twin append", name, lineNo)
			}
			sc.ScriptTwin = &ScriptTwinFixture{Path: path, Append: appendText}
			continue
		}
		if fixtureSection {
			fields := strings.Fields(line)
			if len(fields) == 3 && strings.EqualFold(fields[0], "replace-room-exits") && strings.EqualFold(fields[2], "none") {
				roomVNum, roomErr := strconv.Atoi(fields[1])
				if roomErr == nil && roomVNum > 0 {
					sc.RoomExitFixtures = append(sc.RoomExitFixtures, RoomExitFixture{RoomVNum: roomVNum})
					continue
				}
			}
			if (len(fields) == 5 || len(fields) == 6) && strings.EqualFold(fields[0], "replace-room-exits") {
				roomVNum, roomErr := strconv.Atoi(fields[1])
				toRoom, toErr := strconv.Atoi(fields[3])
				doorState, doorErr := strconv.Atoi(fields[4])
				direction := strings.ToLower(fields[2])
				if roomErr == nil && toErr == nil && doorErr == nil && roomVNum > 0 && toRoom > 0 && doorState >= 0 && doorState <= 2 && validFixtureDirection(direction) {
					fixture := RoomExitFixture{RoomVNum: roomVNum, Direction: direction, ToRoom: toRoom, DoorState: doorState}
					if len(fields) == 6 {
						fixture.Keyword = fields[5]
					}
					sc.RoomExitFixtures = append(sc.RoomExitFixtures, fixture)
					continue
				}
			}
			if len(fields) == 4 && strings.EqualFold(fields[0], "set-room-flag") {
				roomVNum, roomErr := strconv.Atoi(fields[1])
				bit, bitErr := strconv.Atoi(fields[2])
				enabled, enabledOK := parseFixtureToggle(fields[3])
				if roomErr == nil && bitErr == nil && enabledOK && roomVNum > 0 && bit >= 0 && bit < 64 {
					sc.RoomFlagFixtures = append(sc.RoomFlagFixtures, RoomFlagFixture{RoomVNum: roomVNum, Bit: bit, Enabled: enabled})
					continue
				}
			}
			if len(fields) == 3 && strings.EqualFold(fields[0], "set-room-sector") {
				roomVNum, roomErr := strconv.Atoi(fields[1])
				sector, sectorErr := strconv.Atoi(fields[2])
				if roomErr == nil && sectorErr == nil && roomVNum > 0 && sector >= 0 && sector <= 15 {
					sc.RoomSectors = append(sc.RoomSectors, RoomSectorFixture{RoomVNum: roomVNum, Sector: sector})
					continue
				}
			}
			if (len(fields) == 6 || len(fields) == 7) && strings.EqualFold(fields[0], "house-control") {
				vnum, vnumErr := strconv.Atoi(fields[1])
				atrium, atriumErr := strconv.Atoi(fields[2])
				direction := strings.ToLower(fields[3])
				owner, ownerErr := strconv.ParseInt(fields[4], 10, 64)
				portOwner := owner
				var portOwnerErr error
				keyField := 5
				if len(fields) == 7 {
					portOwner, portOwnerErr = strconv.ParseInt(fields[5], 10, 64)
					keyField = 6
				}
				key, keyErr := strconv.Atoi(fields[keyField])
				if vnumErr == nil && atriumErr == nil && ownerErr == nil && portOwnerErr == nil && keyErr == nil && vnum > 0 && atrium > 0 && owner > 0 && portOwner >= 0 && key > 0 && validFixtureDirection(direction) && fixtureDirectionIndex(direction) >= 0 {
					sc.HouseControls = append(sc.HouseControls, HouseControlFixture{
						VNum: vnum, Atrium: atrium, ExitNum: fixtureDirectionIndex(direction), Owner: owner, PortOwner: portOwner, Key: key,
					})
					continue
				}
			}
			if len(fields) == 2 && strings.EqualFold(fields[0], "inert-scroll") {
				objVNum, objErr := strconv.Atoi(fields[1])
				if objErr != nil || objVNum <= 0 {
					return Scenario{}, fmt.Errorf("scenario %q line %d: invalid fixture %q", name, lineNo, line)
				}
				sc.Fixtures = append(sc.Fixtures, ObjectFixture{ObjectVNum: objVNum})
				continue
			}
			if len(fields) == 2 && strings.EqualFold(fields[0], "add-obj-index") {
				name := fields[1]
				if strings.HasSuffix(name, ".obj") && !strings.ContainsRune(name, '/') {
					sc.ObjIndexFixtures = append(sc.ObjIndexFixtures, ObjIndexFixture{FileName: name})
					continue
				}
			}
			if len(fields) == 2 && strings.EqualFold(fields[0], "add-wld-index") {
				name := fields[1]
				if strings.HasSuffix(name, ".wld") && !strings.ContainsRune(name, '/') {
					sc.WldIndexFixtures = append(sc.WldIndexFixtures, WldIndexFixture{FileName: name})
					continue
				}
			}
			if len(fields) == 3 && strings.EqualFold(fields[0], "set-mob-aff") {
				mobVNum, mobErr := strconv.Atoi(fields[1])
				affMask, affErr := strconv.Atoi(fields[2])
				if mobErr == nil && affErr == nil && mobVNum > 0 && affMask > 0 {
					sc.MobAffFixtures = append(sc.MobAffFixtures, MobAffFixture{MobVNum: mobVNum, AffMask: affMask})
					continue
				}
			}
			if len(fields) == 3 && (strings.EqualFold(fields[0], "clear-mob-flag") || strings.EqualFold(fields[0], "set-mob-flag")) {
				mobVNum, mobErr := strconv.Atoi(fields[1])
				flag := strings.ToUpper(fields[2])
				if mobErr == nil && mobVNum > 0 && (flag == "AGGRESSIVE" || flag == "RANDZON" || flag == "SPEC") {
					sc.MobFlagFixtures = append(sc.MobFlagFixtures, MobFlagFixture{MobVNum: mobVNum, Flag: flag, Enabled: strings.EqualFold(fields[0], "set-mob-flag")})
					continue
				}
			}
			if len(fields) == 5 && strings.EqualFold(fields[0], "spawn-mob") {
				values := make([]int, 4)
				valid := true
				for i := range values {
					parsed, parseErr := strconv.Atoi(fields[i+1])
					values[i] = parsed
					if parseErr != nil || values[i] <= 0 {
						valid = false
						break
					}
				}
				if valid {
					sc.MobFixtures = append(sc.MobFixtures, MobFixture{
						MobVNum: values[0], MaxExisting: values[1], RoomVNum: values[2], ZoneNumber: values[3],
					})
					continue
				}
			}
			if len(fields) == 5 && strings.EqualFold(fields[0], "give-object") {
				values := make([]int, 4)
				valid := true
				for i := range values {
					parsed, parseErr := strconv.Atoi(fields[i+1])
					values[i] = parsed
					if parseErr != nil || values[i] <= 0 {
						valid = false
						break
					}
				}
				if valid {
					sc.MobObjectFixtures = append(sc.MobObjectFixtures, MobObjectFixture{
						MobVNum: values[0], ObjectVNum: values[1], MaxExisting: values[2], ZoneNumber: values[3],
					})
					continue
				}
			}
			if len(fields) == 5 && strings.EqualFold(fields[0], "spawn-obj") {
				values := make([]int, 4)
				valid := true
				for i := range values {
					parsed, parseErr := strconv.Atoi(fields[i+1])
					values[i] = parsed
					if parseErr != nil || values[i] < 0 {
						valid = false
						break
					}
				}
				if valid && values[0] > 0 && values[1] > 0 && values[2] > 0 && values[3] >= 0 {
					sc.ObjectSpawns = append(sc.ObjectSpawns, ObjectSpawnFixture{
						ObjectVNum: values[0], MaxExisting: values[1], RoomVNum: values[2], ZoneNumber: values[3],
					})
					continue
				}
			}
			if len(fields) == 2 && strings.EqualFold(fields[0], "quiet-zone") {
				zoneNumber, zoneErr := strconv.Atoi(fields[1])
				if zoneErr == nil && zoneNumber > 0 {
					sc.QuietZones = append(sc.QuietZones, zoneNumber)
					continue
				}
			}
			if len(fields) == 1 && strings.EqualFold(fields[0], "quiet-mobs") {
				sc.QuietAllMobs = true
				continue
			}
			if len(fields) == 1 && strings.EqualFold(fields[0], "empty-players") {
				sc.EmptyPlayers = true
				continue
			}
			if len(fields) == 1 && strings.EqualFold(fields[0], "no-settle") {
				sc.SkipSetupSettle = true
				continue
			}
			if len(fields) == 1 && strings.EqualFold(fields[0], "keep-ansi") {
				sc.KeepANSI = true
				continue
			}
			if len(fields) == 1 && strings.EqualFold(fields[0], "keep-prompts") {
				sc.KeepPrompts = true
				continue
			}
			if len(fields) == 1 && strings.EqualFold(fields[0], "entry-prompt") {
				sc.EntryPromptOnly = true
				continue
			}
			if len(fields) == 4 && strings.EqualFold(fields[0], "mob-script") {
				mobVNum, vnumErr := strconv.Atoi(fields[1])
				flags, flagErr := strconv.Atoi(fields[3])
				if vnumErr != nil || flagErr != nil || mobVNum <= 0 || flags <= 0 || !safeScriptPath(fields[2]) {
					return Scenario{}, fmt.Errorf("scenario %q line %d: invalid mob-script %q", name, lineNo, line)
				}
				sc.MobScripts = append(sc.MobScripts, MobScriptFixture{MobVNum: mobVNum, Path: fields[2], Flags: flags})
				continue
			}
			if len(fields) == 2 && strings.EqualFold(fields[0], "strip-mob-script") {
				mobVNum, mobErr := strconv.Atoi(fields[1])
				if mobErr == nil && mobVNum > 0 {
					sc.ScriptlessMobIDs = append(sc.ScriptlessMobIDs, mobVNum)
					continue
				}
			}
			if len(fields) == 2 && strings.EqualFold(fields[0], "force-load") {
				objVNum, objErr := strconv.Atoi(fields[1])
				if objErr == nil && objVNum > 0 {
					sc.ForceLoadVNums = append(sc.ForceLoadVNums, objVNum)
					continue
				}
			}
			return Scenario{}, fmt.Errorf("scenario %q line %d: invalid fixture %q", name, lineNo, line)
		}
		if peerDropSection {
			if sc.PeerDrop != "" {
				return Scenario{}, fmt.Errorf("scenario %q line %d: duplicate peer-drop", name, lineNo)
			}
			if len(strings.Fields(line)) != 1 {
				return Scenario{}, fmt.Errorf("scenario %q line %d: invalid peer-drop %q", name, lineNo, line)
			}
			sc.PeerDrop = line
			continue
		}
		if section == nil {
			return Scenario{}, fmt.Errorf("scenario %q line %d: command %q before any [section]", name, lineNo, line)
		}
		if line == enterStep {
			line = ""
		}
		*section = append(*section, line)
	}
	if err := scanner.Err(); err != nil {
		return Scenario{}, fmt.Errorf("read scenario: %w", err)
	}
	if len(sc.Probe) == 0 && !sc.DiffSetup {
		return Scenario{}, fmt.Errorf("scenario %q has no [probe] steps", name)
	}
	if sc.EntryPromptOnly && (!sc.DiffSetup || !sc.SkipSetupSettle || !sc.KeepPrompts) {
		return Scenario{}, fmt.Errorf("scenario %q: entry-prompt requires creation, no-settle, and keep-prompts", name)
	}
	for _, step := range sc.Probe {
		if step == ReloginStep && (len(sc.ReloginOracle) == 0 || len(sc.ReloginPort) == 0) {
			return Scenario{}, fmt.Errorf("scenario %q uses %s without both [relogin:oracle] and [relogin:port]", name, ReloginStep)
		}
		if step == ReloginStep && sc.ProbeActor != "" {
			return Scenario{}, fmt.Errorf("scenario %q: %s relogs the primary client, not probe actor %q", name, ReloginStep, sc.ProbeActor)
		}
	}
	if sc.ProbeActor != "" {
		if _, ok := sc.Peers[sc.ProbeActor]; !ok {
			return Scenario{}, fmt.Errorf("scenario %q probe actor %q is not a configured peer", name, sc.ProbeActor)
		}
	}
	if sc.PeerDrop != "" {
		if _, ok := sc.Peers[sc.PeerDrop]; !ok {
			return Scenario{}, fmt.Errorf("scenario %q peer-drop target %q is not a configured peer", name, sc.PeerDrop)
		}
	}
	return sc, nil
}

func fixtureDirectionIndex(direction string) int {
	switch direction {
	case "north":
		return 0
	case "east":
		return 1
	case "south":
		return 2
	case "west":
		return 3
	case "up":
		return 4
	case "down":
		return 5
	default:
		return -1
	}
}

func validFixtureDirection(direction string) bool {
	switch direction {
	case "north", "east", "south", "west", "up", "down", "all":
		return true
	default:
		return false
	}
}

func parseFixtureToggle(value string) (bool, bool) {
	switch strings.ToLower(value) {
	case "on", "true", "1":
		return true, true
	case "off", "false", "0":
		return false, true
	default:
		return false, false
	}
}

// RunWarmup plays shared commands after every client has completed setup and
// discards their output. It is useful for acquiring disposable fixtures before
// the first compared probe command.
func RunWarmup(primary Conn, peers map[string]Conn, steps []string, quiescence time.Duration) error {
	if len(steps) == 0 {
		return nil
	}
	_, err := RunAudienceProbe(primary, peers, steps, quiescence)
	return err
}

// RunAudienceProbe plays commands through the primary actor and captures the
// resulting output separately for the actor and every passive peer. A directed
// step of the form send:<peer> <line> sends the line to that named peer and
// captures the primary plus all other peers as its audience. This is used by
// multi-descriptor vehicles whose C behavior depends on which connection owns
// the next input line.
func RunAudienceProbe(primary Conn, peers map[string]Conn, probe []string, quiescence time.Duration) ([]AudienceProbeBlock, error) {
	blocks := make([]AudienceProbeBlock, 0, len(probe)*(len(peers)+1))
	for i, step := range probe {
		// The connection may close at the last step, or just before a relogin
		// (a quit that ends the session); either is the scenario's intent.
		mayClose := i == len(probe)-1 || probe[i+1] == ReloginStep
		var output string
		target, targetName, audience := primary, "actor", peers
		if step == ReloginStep {
			relogger, ok := primary.(Relogger)
			if !ok {
				return blocks, fmt.Errorf("probe step %d: %w", i+1, errNoRelogin)
			}
			transcript, err := relogger.Relogin(quiescence)
			if err != nil {
				return blocks, fmt.Errorf("probe step %d relogin: %w\ntranscript so far:\n%s", i+1, err, transcript)
			}
			output = transcript
		} else {
			var sendLine string
			var err error
			target, targetName, audience, sendLine, err = resolveAudienceProbeTarget(primary, peers, step)
			if err != nil {
				return blocks, fmt.Errorf("probe step %d %q: %w", i+1, step, err)
			}
			if err := target.Send(sendLine); err != nil {
				return blocks, fmt.Errorf("probe step %d send %q: %w", i+1, step, err)
			}
			output, err = target.ReadUntilQuiescent(quiescence)
			if err != nil && (!mayClose || !errors.Is(err, io.EOF)) {
				return blocks, fmt.Errorf("probe step %d read actor after %q: %w\noutput so far:\n%s", i+1, step, err, output)
			}
		}
		blocks = append(blocks, AudienceProbeBlock{Command: step, Audience: targetName, Output: output})

		peerNames := make([]string, 0, len(audience))
		for name := range audience {
			peerNames = append(peerNames, name)
		}
		sort.Strings(peerNames)
		for _, name := range peerNames {
			peerOutput, peerErr := audience[name].ReadUntilQuiescent(quiescence)
			// A final command may intentionally close an audience connection
			// (for example, the C do_dc lower-level target). Preserve the
			// output captured before EOF instead of turning that expected
			// terminal state into a harness failure.
			if peerErr != nil {
				if i != len(probe)-1 || !errors.Is(peerErr, io.EOF) {
					return blocks, fmt.Errorf("probe step %d read %s after %q: %w\noutput so far:\n%s", i+1, name, step, peerErr, peerOutput)
				}
			}
			blocks = append(blocks, AudienceProbeBlock{Command: step, Audience: name, Output: peerOutput})
		}
	}
	return blocks, nil
}

func resolveAudienceProbeTarget(primary Conn, peers map[string]Conn, step string) (target Conn, targetName string, audience map[string]Conn, sendLine string, err error) {
	if !strings.HasPrefix(step, "send:") {
		return primary, "actor", peers, step, nil
	}

	directed := strings.TrimPrefix(step, "send:")
	space := strings.IndexByte(directed, ' ')
	name := directed
	if space >= 0 {
		name = directed[:space]
		sendLine = directed[space+1:]
	}
	if name == "" {
		return nil, "", nil, "", fmt.Errorf("directed probe has no target")
	}

	audience = make(map[string]Conn, len(peers)+1)
	if name == "primary" {
		target = primary
		targetName = "primary"
		for peerName, peer := range peers {
			audience[peerName] = peer
		}
		return target, targetName, audience, sendLine, nil
	}

	var ok bool
	target, ok = peers[name]
	if !ok {
		return nil, "", nil, "", fmt.Errorf("directed probe target %q is not configured", name)
	}
	targetName = name
	audience["primary"] = primary
	for peerName, peer := range peers {
		if peerName != name {
			audience[peerName] = peer
		}
	}
	return target, targetName, audience, sendLine, nil
}

// RunSetup plays one server's setup lines and returns the captured transcript.
// It reads and discards the initial greeting before the first scripted line.
func RunSetup(conn Conn, setup []string, quiescence time.Duration) (string, error) {
	return runSetup(conn, setup, -1, 0, quiescence)
}

// HasEnterGameStep reports whether a login script chooses "1" at the menu. A
// relogin that reconnects to a linkdead body has no such step: C's
// perform_dupe_check puts the descriptor straight back in the game, with no
// MOTD or menu and no newbie start-room transition to settle.
func HasEnterGameStep(setup []string) bool {
	for _, step := range setup {
		if step == enterGameStep {
			return true
		}
	}
	return false
}

// RunSetupAndSettle plays setup and advances a frozen DP_CLOCK immediately
// after the final "1" menu choice enters the game. C's newbie start-room
// transition is pulse-driven, so post-entry setup commands must not run before
// this settle point.
func RunSetupAndSettle(conn Conn, setup []string, pulses int, quiescence time.Duration) (string, error) {
	entryIndex := -1
	entryCount := 0
	for i, step := range setup {
		if step == enterGameStep {
			entryIndex = i
			entryCount++
		}
	}
	if entryIndex < 0 {
		return "", errors.New("setup has no enter-game step")
	}
	if entryCount > 1 {
		return "", errors.New("setup has ambiguous enter-game steps")
	}
	if pulses <= 0 {
		return "", errors.New("settle pulse count must be positive")
	}
	return runSetup(conn, setup, entryIndex, pulses, quiescence)
}

// setupPasswordRefusal reports the nanny's password refusal if a setup got
// one. A setup that means to enter the game and is refused leaves its
// character at a password prompt on both servers, and every later step then
// compares two identical failures: six scenarios passed that way until
// 2026-09-24, their observers never logged in.
func setupPasswordRefusal(transcript string) string {
	for _, refusal := range []string{"Illegal password.", "Passwords don't match"} {
		if strings.Contains(transcript, refusal) {
			return refusal
		}
	}
	return ""
}

func runSetup(conn Conn, setup []string, settleAfter, pulses int, quiescence time.Duration) (string, error) {
	var transcript strings.Builder
	initial, err := conn.ReadUntilQuiescent(quiescence)
	if err != nil {
		return "", fmt.Errorf("read greeting: %w", err)
	}
	transcript.WriteString(initial)
	for i, step := range setup {
		if err := conn.Send(step); err != nil {
			return transcript.String(), fmt.Errorf("setup step %d send %q: %w\ntranscript so far:\n%s", i+1, step, err, transcript.String())
		}
		output, err := conn.ReadUntilQuiescent(quiescence)
		if err != nil {
			transcript.WriteString(output)
			return transcript.String(), fmt.Errorf("setup step %d read after %q: %w\ntranscript so far:\n%s", i+1, step, err, transcript.String())
		}
		transcript.WriteString(output)
		if i == settleAfter {
			settleOutput, settleErr := PumpPulses(conn, pulses, quiescence)
			transcript.WriteString(settleOutput)
			if settleErr != nil {
				return transcript.String(), fmt.Errorf("settle after setup step %d %q: %w\ntranscript so far:\n%s", i+1, step, settleErr, transcript.String())
			}
		}
	}
	if refusal := setupPasswordRefusal(transcript.String()); refusal != "" {
		return transcript.String(), fmt.Errorf("setup never entered the game: the server answered %q (C refuses passwords over 10 characters, MAX_PWD_LENGTH in structs.h)\ntranscript:\n%s", refusal, transcript.String())
	}
	return transcript.String(), nil
}

// PumpPulses advances a DP_CLOCK-frozen server and returns all heartbeat
// side-effect output. The control line itself is intercepted before either
// command interpreter and emits no acknowledgement.
func PumpPulses(conn Conn, pulses int, quiescence time.Duration) (string, error) {
	if pulses <= 0 {
		return "", fmt.Errorf("pulse count must be positive")
	}
	if err := conn.Send(pulseControl + strconv.Itoa(pulses)); err != nil {
		return "", fmt.Errorf("send pulse control: %w", err)
	}
	output, err := conn.ReadUntilQuiescent(quiescence)
	if err != nil {
		return output, fmt.Errorf("read pulse output: %w", err)
	}
	return output, nil
}

// RunProbe plays the shared probe commands and returns a block per command.
// Each block contains only the output produced by that command.
func RunProbe(conn Conn, probe []string, quiescence time.Duration) ([]ProbeBlock, error) {
	blocks := make([]ProbeBlock, 0, len(probe))
	for i, step := range probe {
		if err := conn.Send(step); err != nil {
			return blocks, fmt.Errorf("probe step %d send %q: %w", i+1, step, err)
		}
		output, err := conn.ReadUntilQuiescent(quiescence)
		if err != nil {
			// A final quit may close the connection without emitting a goodbye
			// block. EOF at that exact boundary is a completed scenario.
			if i == len(probe)-1 && errors.Is(err, io.EOF) {
				blocks = append(blocks, ProbeBlock{Command: step, Output: output})
				break
			}
			return blocks, fmt.Errorf("probe step %d read after %q: %w\noutput so far:\n%s", i+1, step, err, output)
		}
		blocks = append(blocks, ProbeBlock{Command: step, Output: output})
	}
	return blocks, nil
}

// safeScriptPath reports whether a fixture script path is a relative .lua
// path that stays inside the scripts tree.
func safeScriptPath(path string) bool {
	return path != "" && !strings.HasPrefix(path, "/") && !strings.Contains(path, "..") &&
		!strings.Contains(path, "\\") && strings.HasSuffix(path, ".lua")
}
