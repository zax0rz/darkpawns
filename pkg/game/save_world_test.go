package game

// Restart fidelity — transient world state resets on boot.
//
// C's boot_db() rebuilds the world from the static area files and the zone
// reset tables and restores nothing transient:
//
//	src/comm.c:288-293   init_game() shutdown saves save_clans(),
//	                     close_whod(), write_mud_date_to_file() and closes the
//	                     player file — never rooms, ground objects, mobs or gossip.
//	src/db.c:304-392     boot_db() -> boot_world() -> reset_zone(i) for every zone.
//	src/db.c:250-261     init_review_strings() blanks all 25 review slots.
//	src/db.c:2247-2281   reset 'D' forces the door byte; reset 'O'/'M' re-place
//	                     objects and mobiles from the reset table.
//	src/db.c:2285        reset_zone() ends with zone_table[zone].age = 0.
//
// An earlier Go build persisted exactly that state to data/world_state.json and
// replayed it at startup (RULEBOOK R4). These tests pin the C contract.
// See docs/fidelity/audit-reports/world_state_persistence_audit.md.

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// restartFixtureWorldData returns the static world the restart tests boot from.
// Room 5001's up exit is an OPEN door in the .wld data (so a snapshot that says
// "locked" is observable if it were replayed); the zone reset table places mob
// 4000 in 5001 and object 6001 in 5002.
func restartFixtureWorldData() *parser.World {
	return &parser.World{
		Rooms: []parser.Room{
			{VNum: 5001, Name: "Hall", Zone: 5, Exits: map[string]parser.Exit{
				"up": {Direction: "up", ToRoom: 5002, DoorState: 1, ExitInfo: parser.ExitIsDoor},
			}},
			{VNum: 5002, Name: "Vault", Zone: 5, Exits: map[string]parser.Exit{}},
		},
		Mobs: []parser.Mob{{VNum: 4000, Keywords: "ghost", ShortDesc: "a ghost", Position: 8, DefaultPos: 8}},
		Objs: []parser.Obj{
			{VNum: 6000, Keywords: "sword", ShortDesc: "a rusty sword", LoadPercent: 100},
			{VNum: 6001, Keywords: "altar", ShortDesc: "a stone altar", LoadPercent: 100},
		},
		Zones: []parser.Zone{{Number: 5, Name: "Hall", TopRoom: 5002, Commands: []parser.ZoneCommand{
			{Command: "M", Arg1: 4000, Arg2: 1, Arg3: 5001},
			{Command: "O", Arg1: 6001, Arg2: 1, Arg3: 5002},
		}}},
	}
}

// bootRestartFixture reproduces cmd/server/main.go's boot order for the fixture
// world: parse, reset every zone, then report a leftover snapshot.
func bootRestartFixture(t *testing.T) *World {
	t.Helper()
	w, err := NewWorld(restartFixtureWorldData())
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	if err := w.StartZoneResets(); err != nil {
		t.Fatalf("StartZoneResets: %v", err)
	}
	IgnoreLegacyWorldState()
	return w
}

// encodeSnapshot renders the double-encoded JSON an older Go build wrote:
// SaveWorld ran json.Encoder.Encode over the already-marshalled string.
func encodeSnapshot(t *testing.T, snapshot string) string {
	t.Helper()
	out, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal legacy snapshot: %v", err)
	}
	return string(out)
}

// writeLegacyWorldStateFile drops content at the path boot inspects and removes
// it afterwards, so no test leaves a snapshot behind.
func writeLegacyWorldStateFile(t *testing.T, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(worldStateFile), 0o750); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}
	if err := os.WriteFile(worldStateFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write legacy snapshot: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(worldStateFile) })
}

// captureSlog redirects the default logger for the duration of the test.
func captureSlog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &buf
}

// TestRestartDoesNotRestoreLegacyWorldSnapshot is the load-bearing regression:
// a snapshot in the old Go format sits on disk when the world boots, and none
// of it reaches the running world.
func TestRestartDoesNotRestoreLegacyWorldSnapshot(t *testing.T) {
	// Byte-for-byte the shape an older build wrote and replayed: a dropped
	// prototype object (6000), a corpse and a coin pile (both synthetic,
	// VNum == -1), a mob moved off its reset room, a locked door, and gossip.
	legacy := `{"save_version":2,"next_mob_id":900,"next_obj_id":900,` +
		`"door_states":{"5001":{"up":2}},` +
		`"mobs":[{"vnum":4000,"id":7,"room_vnum":5002,"current_hp":1,"max_hp":50}],` +
		`"room_items":{"5001":[{"vnum":6000,"id":11},{"vnum":-1,"id":12},{"vnum":-1,"id":13}]},` +
		`"gossip":[{"name":"Ghost","message":"before the reboot","invis":0}]}`
	writeLegacyWorldStateFile(t, encodeSnapshot(t, legacy))

	w := bootRestartFixture(t)

	t.Run("dropped prototype object is gone", func(t *testing.T) {
		if items := w.GetItemsInRoom(5001); len(items) != 0 {
			t.Fatalf("room 5001 items = %d, want 0: a dropped object survived the restart", len(items))
		}
	})

	t.Run("synthetic vnum -1 objects are gone", func(t *testing.T) {
		w.mu.RLock()
		defer w.mu.RUnlock()
		for id, obj := range w.objectInstances {
			if obj.VNum == -1 && obj.Location.Kind == ObjInRoom {
				t.Fatalf("synthetic object id=%d (vnum -1) restored into room %d", id, obj.Location.RoomVNum)
			}
		}
	})

	t.Run("reset table still places its own objects", func(t *testing.T) {
		items := w.GetItemsInRoom(5002)
		if len(items) != 1 || items[0].GetVNum() != 6001 {
			t.Fatalf("room 5002 items = %v, want the reset-loaded object 6001", items)
		}
	})

	t.Run("door state comes from the .wld file", func(t *testing.T) {
		room := w.GetRoomInWorld(5001)
		if room == nil {
			t.Fatal("room 5001 missing")
		}
		exit := room.Exits["up"]
		if exit.ExitInfo&parser.ExitIsDoor == 0 {
			t.Fatalf("up exit is no longer a door: ExitInfo = %d", exit.ExitInfo)
		}
		if exit.ExitInfo&(parser.ExitClosed|parser.ExitLocked) != 0 {
			t.Fatalf("up exit ExitInfo = %d, want open from .wld: snapshot door state was replayed", exit.ExitInfo)
		}
	})

	t.Run("mob is at its reset room", func(t *testing.T) {
		mobs := w.GetMobsInRoom(5001)
		if len(mobs) != 1 || mobs[0].GetVNum() != 4000 {
			t.Fatalf("room 5001 mobs = %v, want the reset-spawned mob 4000", mobs)
		}
		if got := len(w.GetMobsInRoom(5002)); got != 0 {
			t.Fatalf("room 5002 mobs = %d, want 0: the snapshot's mob position was replayed", got)
		}
	})

	t.Run("review buffer is blank", func(t *testing.T) {
		w.gossipMu.RLock()
		history := len(w.gossipHistory)
		w.gossipMu.RUnlock()
		if history != 0 {
			t.Fatalf("gossip history = %d entries, want 0 (src/db.c init_review_strings)", history)
		}
	})

	t.Run("id counters are not restored", func(t *testing.T) {
		w.mu.RLock()
		nextObj := w.nextObjID
		w.mu.RUnlock()
		// One reset-loaded object was created from id 1 upward; the snapshot's
		// next_obj_id of 900 must not have been adopted.
		if nextObj > 10 {
			t.Fatalf("nextObjID = %d, want a fresh counter near 1: snapshot counters were restored", nextObj)
		}
	})
}

// TestReviewCannotShowPreRestartGossip walks the do_review() path after boot
// while a snapshot carrying gossip is present on disk.
func TestReviewCannotShowPreRestartGossip(t *testing.T) {
	writeLegacyWorldStateFile(t, encodeSnapshot(t,
		`{"save_version":2,"gossip":[{"name":"Ghost","message":"before the reboot","invis":0}]}`))

	w := bootRestartFixture(t)
	viewer := NewPlayer(1, "Viewer", 5001)
	if err := w.AddPlayer(viewer); err != nil {
		t.Fatalf("AddPlayer: %v", err)
	}

	got := DoReview(viewer, w).MessageToCh
	want := "Last Gossips:\r\n-------------\r\n"
	if got != want {
		t.Fatalf("review after restart = %q, want %q", got, want)
	}
	if strings.Contains(got, "before the reboot") {
		t.Fatal("review displayed pre-restart gossip after a restart")
	}
}

// TestCorpsesMoneyAndAshDoNotSurviveRestart creates the synthetic objects the
// death path really produces (VNum == -1, no prototype) in one world and shows
// the booted world starting without them.
func TestCorpsesMoneyAndAshDoNotSurviveRestart(t *testing.T) {
	before, err := NewWorld(restartFixtureWorldData())
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(before.StopAITicker)

	money := before.createMoneyObject(250)
	if err := before.MoveObjectToRoom(money, 5001); err != nil {
		t.Fatalf("move money: %v", err)
	}
	corpse := before.makeCorpse("Ghost", 0, nil, nil, 5001, 0, 0, false, "ghost")
	if err := before.MoveObjectToRoom(corpse, 5001); err != nil {
		t.Fatalf("move corpse: %v", err)
	}

	placed := before.GetItemsInRoom(5001)
	if len(placed) < 2 {
		t.Fatalf("fixture placed %d synthetic objects, want >= 2", len(placed))
	}
	for _, obj := range placed {
		if obj.VNum != -1 {
			t.Fatalf("fixture object %d vnum = %d, want the synthetic -1", obj.ID, obj.VNum)
		}
	}

	after := bootRestartFixture(t)
	if got := len(after.GetItemsInRoom(5001)); got != 0 {
		t.Fatalf("room 5001 items after restart = %d, want 0 (corpses, money and ash are transient)", got)
	}
}

// TestLegacySnapshotWithManySyntheticObjectsLogsOnce replaces the old per-entry
// "unknown obj vnum" warning storm (~3,200 startup lines) with one summary.
func TestLegacySnapshotWithManySyntheticObjectsLogsOnce(t *testing.T) {
	var items strings.Builder
	items.WriteString(`"room_items":{"5001":[`)
	const syntheticObjects = 3200
	for index := 0; index < syntheticObjects; index++ {
		if index > 0 {
			items.WriteByte(',')
		}
		// Every entry is synthetic (VNum == -1): the class that caused the
		// warning storm, because no prototype exists to restore from.
		items.WriteString(`{"vnum":-1,"id":`)
		items.WriteString(itoaDecimal(index + 1))
		items.WriteByte('}')
	}
	items.WriteString(`]}`)
	writeLegacyWorldStateFile(t, encodeSnapshot(t, `{"save_version":2,`+items.String()+`}`))

	buf := captureSlog(t)
	IgnoreLegacyWorldState()
	output := buf.String()

	if strings.Contains(output, "unknown obj vnum") {
		t.Fatalf("per-object warnings returned:\n%s", output)
	}
	if got := strings.Count(output, "ignoring legacy world snapshot"); got != 1 {
		t.Fatalf("summary lines = %d, want exactly 1; output:\n%s", got, output)
	}
	if !strings.Contains(output, "room_items=3200") {
		t.Fatalf("summary does not report the ignored object count:\n%s", output)
	}
}

// itoaDecimal keeps the generator above dependency-free and deterministic.
func itoaDecimal(v int) string {
	if v == 0 {
		return "0"
	}
	var digits [20]byte
	pos := len(digits)
	for v > 0 {
		pos--
		digits[pos] = byte('0' + v%10)
		v /= 10
	}
	return string(digits[pos:])
}

// TestInspectLegacyWorldStateCountsTransientSections pins the summary the
// operator sees, including double-encoded (old SaveWorld) input.
func TestInspectLegacyWorldStateCountsTransientSections(t *testing.T) {
	path := filepath.Join(t.TempDir(), "world_state.json")
	snapshot := `{"save_version":2,"next_mob_id":4,"next_obj_id":9,` +
		`"door_states":{"100":{"north":2},"101":{"south":1,"east":0}},` +
		`"mobs":[{"vnum":3000,"id":1,"room_vnum":100,"current_hp":5,"max_hp":9}],` +
		`"room_items":{"100":[{"vnum":1,"id":1}],"101":[{"vnum":-1,"id":2},{"vnum":2,"id":3}]},` +
		`"gossip":[{"name":"A","message":"one","invis":0}]}`
	if err := os.WriteFile(path, []byte(encodeSnapshot(t, snapshot)), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	summary, found, err := inspectLegacyWorldState(path)
	if err != nil || !found {
		t.Fatalf("inspectLegacyWorldState found=%v err=%v, want found with no error", found, err)
	}
	want := legacyWorldStateSummary{
		SaveVersion:    2,
		MobEntries:     1,
		RoomItemRooms:  2,
		RoomItems:      3,
		DoorStateRooms: 2,
		DoorStates:     3,
		GossipEntries:  1,
	}
	if summary != want {
		t.Fatalf("summary = %+v, want %+v", summary, want)
	}
}

// TestInspectLegacyWorldStateToleratesBadInput covers the cases that must not
// stop a boot.
func TestInspectLegacyWorldStateToleratesBadInput(t *testing.T) {
	dir := t.TempDir()

	t.Run("missing file", func(t *testing.T) {
		summary, found, err := inspectLegacyWorldState(filepath.Join(dir, "absent.json"))
		if err != nil || found || summary != (legacyWorldStateSummary{}) {
			t.Fatalf("missing file: summary=%+v found=%v err=%v, want zero/false/nil", summary, found, err)
		}
	})

	t.Run("bare object without the old double encoding", func(t *testing.T) {
		path := filepath.Join(dir, "bare.json")
		if err := os.WriteFile(path, []byte(`{"save_version":2,"gossip":[{"name":"A","message":"m"}]}`), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		summary, found, err := inspectLegacyWorldState(path)
		if err != nil || !found || summary.GossipEntries != 1 {
			t.Fatalf("bare object: summary=%+v found=%v err=%v", summary, found, err)
		}
	})

	t.Run("truncated file", func(t *testing.T) {
		path := filepath.Join(dir, "truncated.json")
		if err := os.WriteFile(path, []byte(`{"save_version":2,"room_items":`), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		if _, found, err := inspectLegacyWorldState(path); err == nil || found {
			t.Fatalf("truncated file: found=%v err=%v, want found=false with an error", found, err)
		}
	})
}

// TestIgnoreLegacyWorldStateIsNotFatalAndKeepsTheFile verifies the boot hook
// logs once, does not panic, and leaves a corrupt file alone for the operator.
func TestIgnoreLegacyWorldStateIsNotFatalAndKeepsTheFile(t *testing.T) {
	const corrupt = `{"save_version":`
	writeLegacyWorldStateFile(t, corrupt)

	buf := captureSlog(t)
	IgnoreLegacyWorldState()
	output := buf.String()

	if !strings.Contains(output, "ignoring unreadable legacy world snapshot") {
		t.Fatalf("expected one warning about the unreadable snapshot, got:\n%s", output)
	}
	got, err := os.ReadFile(filepath.Clean(worldStateFile))
	if err != nil {
		t.Fatalf("legacy snapshot was removed: %v", err)
	}
	if string(got) != corrupt {
		t.Fatalf("legacy snapshot was rewritten: %q", got)
	}
}

// TestPlayerPersistenceIsSeparateFromWorldSnapshot shows durable player data
// still round-trips while no world snapshot is produced: C keeps player files
// (src/comm.c fclose(player_fl), src/db.c save_char) and rebuilds everything
// else.
func TestPlayerPersistenceIsSeparateFromWorldSnapshot(t *testing.T) {
	_ = os.Remove(filepath.Clean(worldStateFile))

	player := NewPlayer(4242, "WorldResetAuditPlayer", 5001)
	player.SetLevel(7)
	player.BankGold = 1234

	if err := SavePlayer(player); err != nil {
		t.Fatalf("SavePlayer: %v", err)
	}
	playerPath := filepath.Join(saveDir, sanitizeName(player.Name)+".json")
	t.Cleanup(func() { _ = os.Remove(playerPath) })

	loaded, err := LoadPlayer(player.Name)
	if err != nil {
		t.Fatalf("LoadPlayer: %v", err)
	}
	if loaded.GetLevel() != 7 || loaded.BankGold != 1234 {
		t.Fatalf("loaded level/bank = %d/%d, want 7/1234", loaded.GetLevel(), loaded.BankGold)
	}
	if _, err := os.Stat(filepath.Clean(worldStateFile)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a world snapshot exists after a player save: stat err = %v", err)
	}
}

// TestBootAndShutdownWriteNoWorldSnapshot pins requirement 5/7 of the audit:
// no snapshot is produced, so no new save-file schema can be introduced.
func TestBootAndShutdownWriteNoWorldSnapshot(t *testing.T) {
	_ = os.Remove(filepath.Clean(worldStateFile))

	w := bootRestartFixture(t)
	// Make the world dirty exactly as gameplay would: a player drops an object.
	if _, err := w.SpawnObject(6000, 5001); err != nil {
		t.Fatalf("SpawnObject: %v", err)
	}
	w.StopAITicker()
	w.StopPeriodicResets()

	if _, err := os.Stat(filepath.Clean(worldStateFile)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("a world snapshot was written: stat err = %v", err)
	}
}

// TestZoneResetInitializationIsDeterministic proves boot placement still comes
// from the reset table rather than from any restored state: two independent
// boots of the same area files agree on every mob room and ground object.
func TestZoneResetInitializationIsDeterministic(t *testing.T) {
	placement := func(t *testing.T) (map[int]int, map[int][]int) {
		t.Helper()
		w, err := NewWorld(restartFixtureWorldData())
		if err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
		t.Cleanup(w.StopAITicker)
		w.StopAITicker() // freeze AI wandering before reading state
		if err := w.StartZoneResets(); err != nil {
			t.Fatalf("StartZoneResets: %v", err)
		}

		mobRooms := make(map[int]int)
		for _, mob := range w.activeMobs {
			mobRooms[mob.VNum] = mob.GetRoom()
		}
		itemVnums := make(map[int][]int)
		for roomVNum, items := range w.roomItems {
			for _, obj := range items {
				itemVnums[roomVNum] = append(itemVnums[roomVNum], obj.VNum)
			}
		}
		return mobRooms, itemVnums
	}

	firstMobs, firstItems := placement(t)
	secondMobs, secondItems := placement(t)

	if !reflect.DeepEqual(firstMobs, secondMobs) {
		t.Fatalf("mob placement differs between boots: %v vs %v", firstMobs, secondMobs)
	}
	if !reflect.DeepEqual(firstItems, secondItems) {
		t.Fatalf("ground objects differ between boots: %v vs %v", firstItems, secondItems)
	}
	if len(firstMobs) != 1 || firstMobs[4000] != 5001 {
		t.Fatalf("mob 4000 room = %d, want the reset table's room 5001", firstMobs[4000])
	}
}
