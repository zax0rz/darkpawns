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
	Warmup      []string
	Probe       []string
	ProbeActor  string
	// PeerDrop names one passive peer whose TCP connection is closed after
	// setup/warmup and before the compared probe. The character remains in the
	// live-world lifecycle, which exposes C's linkless descriptor branches.
	PeerDrop         string
	Peers            map[string]*PeerSetup
	Fixtures         []ObjectFixture
	ObjectSpawns     []ObjectSpawnFixture
	MobFixtures      []MobFixture
	MobAffFixtures   []MobAffFixture
	ObjIndexFixtures []ObjIndexFixture
	QuietZones       []int
	QuietAllMobs     bool
	EmptyPlayers     bool
	ScriptlessMobIDs []int
	ForceLoadVNums   []int
	RoomExitFixtures []RoomExitFixture
	RoomFlagFixtures []RoomFlagFixture
	RoomSectors      []RoomSectorFixture
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

// MobAffFixture patches a mob prototype's innate affected-by bitmask (the
// flag line's second field) in each server's disposable world copy. C
// read_mobile copies those bits onto every instance, and mag_affects'
// mob-affection gate (magic.c:1387-1394) refuses spells whose bitvector the
// mob carries innately — this fixture is the live vehicle for that gate.
type MobAffFixture struct {
	MobVNum int
	AffMask int
}

// ObjIndexFixture adds one filename to the disposable obj index so the boot
// loader reads an otherwise-unindexed .obj file. Real world vehicles live in
// files the shipped index omits entirely (131.obj's plaid potion casts
// blindness, 58.obj's return scroll casts curse), which made every scenario
// step touching them vacuously fail on BOTH servers.
type ObjIndexFixture struct {
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
//	add-obj-index 131.obj    # load an otherwise-unindexed obj file's prototypes
//	spawn-obj 8010 1 8004 80  # object, max existing, room, zone file
//	quiet-zone 80             # suppress mobile resets in a disposable zone
//	quiet-mobs                # suppress mobile resets in every disposable zone
//	strip-mob-script 18306    # force native special dispatch in both copies
//	force-load 4903           # rewrite the prototype load percent to 500% in both copies
//	replace-room-exits 8162 none
//	replace-room-exits 8162 all 8161 0
//	replace-room-exits 8162 north 8161 1 gate
//	set-room-flag 8161 1 on  # ROOM_DEATH
//	set-room-sector 8161 7   # SECT_WATER_NOSWIM
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
			peerDropSection = false
			lower := strings.ToLower(line)
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
			case "[probe]":
				section = &sc.Probe
				sc.ProbeActor = ""
			case "[warmup]":
				section = &sc.Warmup
			case "[fixture]", "[fixtures]":
				section = nil
				fixtureSection = true
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
			if len(fields) == 3 && strings.EqualFold(fields[0], "set-mob-aff") {
				mobVNum, mobErr := strconv.Atoi(fields[1])
				affMask, affErr := strconv.Atoi(fields[2])
				if mobErr == nil && affErr == nil && mobVNum > 0 && affMask > 0 {
					sc.MobAffFixtures = append(sc.MobAffFixtures, MobAffFixture{MobVNum: mobVNum, AffMask: affMask})
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
// resulting output separately for the actor and every passive peer.
func RunAudienceProbe(primary Conn, peers map[string]Conn, probe []string, quiescence time.Duration) ([]AudienceProbeBlock, error) {
	peerNames := make([]string, 0, len(peers))
	for name := range peers {
		peerNames = append(peerNames, name)
	}
	sort.Strings(peerNames)

	blocks := make([]AudienceProbeBlock, 0, len(probe)*(len(peers)+1))
	for i, step := range probe {
		if err := primary.Send(step); err != nil {
			return blocks, fmt.Errorf("probe step %d send %q: %w", i+1, step, err)
		}
		output, err := primary.ReadUntilQuiescent(quiescence)
		if err != nil && (i != len(probe)-1 || !errors.Is(err, io.EOF)) {
			return blocks, fmt.Errorf("probe step %d read actor after %q: %w\noutput so far:\n%s", i+1, step, err, output)
		}
		blocks = append(blocks, AudienceProbeBlock{Command: step, Audience: "actor", Output: output})

		for _, name := range peerNames {
			peerOutput, peerErr := peers[name].ReadUntilQuiescent(quiescence)
			if peerErr != nil {
				return blocks, fmt.Errorf("probe step %d read %s after %q: %w\noutput so far:\n%s", i+1, name, step, peerErr, peerOutput)
			}
			blocks = append(blocks, AudienceProbeBlock{Command: step, Audience: name, Output: peerOutput})
		}
	}
	return blocks, nil
}

// RunSetup plays one server's setup lines and returns the captured transcript.
// It reads and discards the initial greeting before the first scripted line.
func RunSetup(conn Conn, setup []string, quiescence time.Duration) (string, error) {
	return runSetup(conn, setup, -1, 0, quiescence)
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
