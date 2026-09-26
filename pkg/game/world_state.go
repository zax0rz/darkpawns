// Package game — legacy world-snapshot handling.
//
// The C server keeps no transient world state between runs. src/comm.c
// init_game() boots the world, runs the game loop, and saves only the clan
// table, the in-game date, and the player file on the way out (comm.c:289-291);
// src/db.c boot_db() then reconstructs everything else from the static area
// files plus reset_zone() for every zone:
//
//   - boot_world() -> boot_world_files() index_boot()s the .zon/.wld/.mob/.obj/
//     .shp tables and calls init_review_strings(), which blanks the 25-slot
//     review buffer (db.c:250-261, 264-299).
//   - boot_db() then calls reset_zone(i) for every zone (db.c:385-392), which
//     re-reads mobiles ('M') and objects ('O'/'P'/'G'/'E'), removes stale ones
//     ('R'), and forces door states ('D') from the reset table; reset_zone()
//     finishes with zone_table[zone].age = 0 (db.c:2285).
//   - Ground objects, corpses, loose money, ash, mob positions and HP, door
//     state, and recent gossip therefore all disappear on restart.
//
// An earlier Go build invented the opposite contract: SerializeWorld /
// SaveWorld wrote a data/world_state.json snapshot on shutdown (and on demand
// from the admin API) and DeserializeWorld / LoadWorld replayed it at startup.
// That reproduced none of C's reset (RULEBOOK R4), and the replay emitted one
// "unknown obj vnum" warning per synthetic object because a corpse, coin pile,
// or ash object carries VNum == -1 and has no prototype to restore from. The
// snapshot code is gone.
//
// What remains is the operator-facing half: a leftover snapshot from an older
// build is recognised, summarised in one structured line, and otherwise
// ignored. It is never read into the world, never rewritten, and never fatal.
package game

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
)

// worldStateFile is the path an older Go build used for its invented world
// snapshot. Nothing writes it any more; the constant exists so boot can report
// a leftover file and so tests can pin the location.
const worldStateFile = "./data/world_state.json"

// legacyWorldStateSummary counts the transient sections of a leftover
// snapshot. The counts give the operator context for the boot line; none of
// them is state to restore.
type legacyWorldStateSummary struct {
	SaveVersion    int
	MobEntries     int
	RoomItemRooms  int
	RoomItems      int
	DoorStateRooms int
	DoorStates     int
	GossipEntries  int
}

// IgnoreLegacyWorldState reports a leftover world snapshot from an older Go
// build and otherwise does nothing: C reconstructs the world from the area
// files and zone reset tables, so replaying a snapshot would be a fidelity
// regression. Safe to call on every boot — a missing snapshot logs nothing,
// and an unreadable or malformed one is a single warning rather than a boot
// failure (an obsolete transient file must not keep the server down).
func IgnoreLegacyWorldState() {
	summary, found, err := inspectLegacyWorldState(worldStateFile)
	switch {
	case err != nil:
		slog.Warn("ignoring unreadable legacy world snapshot; transient world state resets on boot",
			"path", worldStateFile, "error", err)
	case found:
		slog.Info("ignoring legacy world snapshot; transient world state resets on boot",
			"path", worldStateFile,
			"save_version", summary.SaveVersion,
			"mobs", summary.MobEntries,
			"rooms_with_items", summary.RoomItemRooms,
			"room_items", summary.RoomItems,
			"rooms_with_door_state", summary.DoorStateRooms,
			"door_states", summary.DoorStates,
			"gossip_entries", summary.GossipEntries)
	}
}

// inspectLegacyWorldState reads path and counts the transient sections an
// older Go build persisted. found is false when the file does not exist. err is
// non-nil only when the file exists but cannot be read or parsed.
//
// One summary is produced for the whole file; the per-entry loop that produced
// ~3,200 startup warnings before is deliberately absent (the counts are taken
// structurally, without touching individual entries).
func inspectLegacyWorldState(path string) (legacyWorldStateSummary, bool, error) {
	raw, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return legacyWorldStateSummary{}, false, nil
		}
		return legacyWorldStateSummary{}, false, fmt.Errorf("read legacy world snapshot: %w", err)
	}

	// SaveWorld wrote the snapshot with json.Encoder.Encode(string), so the
	// file holds a JSON string whose contents are the snapshot object. Accept a
	// bare object too, for hand-edited or partially written files.
	payload := raw
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err == nil {
		payload = []byte(encoded)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(bytes.TrimSpace(payload), &fields); err != nil {
		return legacyWorldStateSummary{}, false, fmt.Errorf("parse legacy world snapshot: %w", err)
	}

	var summary legacyWorldStateSummary
	_ = json.Unmarshal(fields["save_version"], &summary.SaveVersion)
	summary.MobEntries = countJSONArray(fields["mobs"])
	summary.GossipEntries = countJSONArray(fields["gossip"])
	summary.RoomItemRooms, summary.RoomItems = countJSONMapOfArrays(fields["room_items"])
	summary.DoorStateRooms, summary.DoorStates = countJSONMapOfMaps(fields["door_states"])
	return summary, true, nil
}

// countJSONArray returns the number of elements in a JSON array, or 0 when the
// field is absent or not an array.
func countJSONArray(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return 0
	}
	return len(items)
}

// countJSONMapOfArrays returns the number of keys and the total number of
// elements across a JSON object whose values are arrays.
func countJSONMapOfArrays(raw json.RawMessage) (keys, elements int) {
	if len(raw) == 0 {
		return 0, 0
	}
	var byKey map[string][]json.RawMessage
	if err := json.Unmarshal(raw, &byKey); err != nil {
		return 0, 0
	}
	for _, values := range byKey {
		elements += len(values)
	}
	return len(byKey), elements
}

// countJSONMapOfMaps returns the number of keys and the total number of inner
// entries across a JSON object whose values are objects.
func countJSONMapOfMaps(raw json.RawMessage) (keys, entries int) {
	if len(raw) == 0 {
		return 0, 0
	}
	var byKey map[string]map[string]json.RawMessage
	if err := json.Unmarshal(raw, &byKey); err != nil {
		return 0, 0
	}
	for _, inner := range byKey {
		entries += len(inner)
	}
	return len(byKey), entries
}
