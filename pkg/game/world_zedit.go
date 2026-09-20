package game

import "github.com/zax0rz/darkpawns/pkg/parser"

// CloneZone returns an independent zone definition suitable for a descriptor
// owned ZEDIT working copy. The command slice is copied because ZEDIT edits a
// room-filtered view of the zone's reset table.
func CloneZone(zone parser.Zone) parser.Zone {
	clone := zone
	clone.Commands = append([]parser.ZoneCommand(nil), zone.Commands...)
	return clone
}

// SnapshotZone returns a deep copy of one zone definition while holding the
// world lock. Zone commands in the Go parser are VNUM-keyed; unlike C's
// boot-time tables, no rnum mutation is needed at this boundary.
func (w *World) SnapshotZone(number int) (parser.Zone, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	zone, ok := w.zones[number]
	if !ok || zone == nil {
		return parser.Zone{}, false
	}
	return CloneZone(*zone), true
}

// CommitEditedZone replaces the room-scoped reset commands and, when the
// caller changed them, the zone header. The C editor removes matching live
// commands first, then inserts the working list at the front. Its save-side
// stale-carry filter begins at -2 and compares command room rnums; the Go
// parser stores VNUMs, so each room-bearing field is resolved through the
// world's stable room index before that comparison.
//
// The setup-side filter lives in session/zedit.go and intentionally operates
// on the parser's VNUM representation with its own -1 initial carry. Keeping
// the two passes separate preserves the C asymmetry documented by the editor.
func (w *World) CommitEditedZone(zoneNumber, roomVNum int, working parser.Zone) bool {
	w.mu.Lock()
	defer w.mu.Unlock()

	zone, ok := w.zones[zoneNumber]
	if !ok || zone == nil {
		return false
	}
	roomRNum, ok := w.realRoomIndexLocked(roomVNum)
	if !ok {
		return false
	}

	remaining := make([]parser.ZoneCommand, 0, len(zone.Commands))
	cmdRoom := -2
	for _, cmd := range zone.Commands {
		if room, hasRoom := ZoneCommandRoom(cmd); hasRoom {
			cmdRoom, _ = w.realRoomIndexLocked(room)
		}
		if cmdRoom != roomRNum {
			remaining = append(remaining, cmd)
		}
	}

	commands := make([]parser.ZoneCommand, 0, len(working.Commands)+len(remaining))
	for _, cmd := range working.Commands {
		// A stale-matched '*' is removed by the save-side filter and is not
		// re-added to the live command list. The disk writer independently
		// skips any other invalid '*' entry.
		if cmd.Command != "*" {
			commands = append(commands, cmd)
		}
	}
	commands = append(commands, remaining...)

	updated := CloneZone(*zone)
	updated.Name = working.Name
	updated.TopRoom = working.TopRoom
	updated.Lifespan = working.Lifespan
	updated.ResetMode = working.ResetMode
	updated.Commands = commands

	if w.parsedData != nil {
		zones := make([]parser.Zone, len(w.parsedData.Zones))
		copy(zones, w.parsedData.Zones)
		for i := range w.parsedData.Zones {
			if w.parsedData.Zones[i].Number == zoneNumber {
				zones[i] = updated
				break
			}
		}
		w.parsedData.Zones = zones
		w.zones = make(map[int]*parser.Zone, len(zones))
		for i := range w.parsedData.Zones {
			w.zones[w.parsedData.Zones[i].Number] = &w.parsedData.Zones[i]
		}
	} else {
		zones := make(map[int]*parser.Zone, len(w.zones))
		for number, existing := range w.zones {
			zones[number] = existing
		}
		zones[zoneNumber] = &updated
		w.zones = zones
	}
	return true
}

// CreateZone inserts a new zone in the parsed world and live zone map. The
// caller owns the user-visible file templates and index-file update; this
// helper is the in-memory half of zedit_new_zone's table insertion. C grows
// zone_table with a 32000-number sentinel before shifting entries; the Go
// representation has no sentinel row, so sorted insertion into the map-backed
// zone slice is the equivalent boundary operation.
func (w *World) CreateZone(number int) (*parser.Zone, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	start := number * 100
	for _, zone := range w.zones {
		if zone != nil && zone.Number*100 <= start && zone.TopRoom >= start {
			return nil, false
		}
	}

	newZone := parser.Zone{
		Number:    number,
		Name:      "New Zone",
		TopRoom:   start + 99,
		Lifespan:  30,
		ResetMode: 2,
	}
	if w.parsedData == nil {
		w.zones[number] = &newZone
		return &newZone, true
	}

	zones := make([]parser.Zone, 0, len(w.parsedData.Zones)+1)
	inserted := false
	for _, zone := range w.parsedData.Zones {
		if !inserted && zone.Number > number {
			zones = append(zones, newZone)
			inserted = true
		}
		zones = append(zones, CloneZone(zone))
	}
	if !inserted {
		zones = append(zones, newZone)
	}
	w.parsedData.Zones = zones
	w.zones = make(map[int]*parser.Zone, len(zones))
	for i := range w.parsedData.Zones {
		w.zones[w.parsedData.Zones[i].Number] = &w.parsedData.Zones[i]
	}
	return w.zones[number], true
}

// ZoneCommandRoom returns the parser VNUM field that scopes a reset command,
// or false for commands whose arguments do not name a room.
func ZoneCommandRoom(cmd parser.ZoneCommand) (int, bool) {
	switch cmd.Command {
	case "M", "O":
		return cmd.Arg3, true
	case "D", "R", "L":
		return cmd.Arg1, true
	default:
		return 0, false
	}
}

func (w *World) realRoomIndexLocked(vnum int) (int, bool) {
	index := sortSearchInts(w.sortedRoomVNums, vnum)
	if index < len(w.sortedRoomVNums) && w.sortedRoomVNums[index] == vnum {
		return index, true
	}
	return -1, false
}

// sortSearchInts is kept local so the helper can make the lock ownership
// explicit without calling the public method (which acquires the same lock).
func sortSearchInts(values []int, target int) int {
	lo, hi := 0, len(values)
	for lo < hi {
		mid := lo + (hi-lo)/2
		if values[mid] < target {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo
}
