// Package parser provides utilities for parsing Dark Pawns world files.
package parser

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

// validateWorldPath ensures a path does not contain directory traversal
// components before opening world data files.
func validateWorldPath(path string) error {
	clean := filepath.Clean(path)
	if strings.Contains(clean, "..") {
		return fmt.Errorf("invalid path contains '..': %s", path)
	}
	return nil
}

// World represents the entire parsed game world.
type World struct {
	Rooms []Room
	Mobs  []Mob
	Objs  []Obj
	Zones []Zone
}

// Stats returns statistics about the parsed world.
func (w *World) Stats() string {
	return fmt.Sprintf(
		"World Stats:\n"+
			"  Rooms: %d\n"+
			"  Mobs:  %d\n"+
			"  Objs:  %d\n"+
			"  Zones: %d",
		len(w.Rooms), len(w.Mobs), len(w.Objs), len(w.Zones),
	)
}

// ParseWorld parses all world files from the given lib directory.
func ParseWorld(libDir string) (*World, error) {
	world := &World{}

	// Parse rooms
	rooms, err := ParseAllWldFiles(libDir + "/wld")
	if err != nil {
		return nil, fmt.Errorf("parse rooms: %w", err)
	}
	world.Rooms = rooms

	// Parse mobs
	mobs, err := ParseAllMobFiles(libDir + "/mob")
	if err != nil {
		return nil, fmt.Errorf("parse mobs: %w", err)
	}
	world.Mobs = mobs

	// Parse objects
	objs, err := ParseAllObjFiles(libDir + "/obj")
	if err != nil {
		return nil, fmt.Errorf("parse objects: %w", err)
	}
	world.Objs = objs

	// Parse zones
	zones, err := ParseAllZonFiles(libDir + "/zon")
	if err != nil {
		return nil, fmt.Errorf("parse zones: %w", err)
	}
	world.Zones = zones

	world.ValidateCrossReferences()
	return world, nil
}

// ValidateCrossReferences checks all room exits and zone commands for broken references.
// Based on the original C check_exits / renum_world logic in src/db.c.
func (w *World) ValidateCrossReferences() {
	// Build set of all valid room vnums
	roomVnums := make(map[int]bool)
	for _, r := range w.Rooms {
		roomVnums[r.VNum] = true
	}

	// Build set of all valid mob vnums
	mobVnums := make(map[int]bool)
	for _, m := range w.Mobs {
		mobVnums[m.VNum] = true
	}

	// Build set of all valid object vnums
	objVnums := make(map[int]bool)
	for _, o := range w.Objs {
		objVnums[o.VNum] = true
	}

	nowhere := -1

	// Check room exits
	for _, r := range w.Rooms {
		for dir, exit := range r.Exits {
			if exit.ToRoom <= nowhere {
				continue // NOWHERE is valid
			}
			if !roomVnums[exit.ToRoom] {
				slog.Warn("exit points to non-existent room",
					"room_vnum", r.VNum, "direction", dir, "target_vnum", exit.ToRoom)
			}
		}
	}

	// Check zone commands
	for _, z := range w.Zones {
		for i, cmd := range z.Commands {
			if cmd.Command == "S" {
				continue // stop command
			}
			switch cmd.Command {
			case "M":
				// 'M' <mob_vnum> <max_in_world> <room_vnum>
				if !mobVnums[cmd.Arg1] {
					slog.Warn("zone command references non-existent mob",
						"zone", z.Number, "cmd_index", i, "command", "M", "mob_vnum", cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !roomVnums[cmd.Arg3] {
					slog.Warn("zone command references non-existent room",
						"zone", z.Number, "cmd_index", i, "command", "M", "room_vnum", cmd.Arg3)
				}
			case "O":
				// 'O' <obj_vnum> <max_in_world> <room_vnum>
				if !objVnums[cmd.Arg1] {
					slog.Warn("zone command references non-existent object",
						"zone", z.Number, "cmd_index", i, "command", "O", "obj_vnum", cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !roomVnums[cmd.Arg3] {
					slog.Warn("zone command references non-existent room",
						"zone", z.Number, "cmd_index", i, "command", "O", "room_vnum", cmd.Arg3)
				}
			case "G":
				// 'G' <obj_vnum> <max_in_world> (give to object)
				if !objVnums[cmd.Arg1] {
					slog.Warn("zone command references non-existent object",
						"zone", z.Number, "cmd_index", i, "command", "G", "obj_vnum", cmd.Arg1)
				}
			case "E":
				// 'E' <obj_vnum> <equip_position> <room_vnum>
				if !objVnums[cmd.Arg1] {
					slog.Warn("zone command references non-existent object",
						"zone", z.Number, "cmd_index", i, "command", "E", "obj_vnum", cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !roomVnums[cmd.Arg3] {
					slog.Warn("zone command references non-existent room",
						"zone", z.Number, "cmd_index", i, "command", "E", "room_vnum", cmd.Arg3)
				}
			case "P":
				// 'P' <obj_vnum> <max_in_world> <container_vnum>
				if !objVnums[cmd.Arg1] {
					slog.Warn("zone command references non-existent object",
						"zone", z.Number, "cmd_index", i, "command", "P", "obj_vnum", cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !objVnums[cmd.Arg3] {
					slog.Warn("zone command references non-existent container",
						"zone", z.Number, "cmd_index", i, "command", "P", "container_vnum", cmd.Arg3)
				}
			case "D":
				// 'D' <room_vnum> <door_state> <key_vnum>
				if !roomVnums[cmd.Arg1] {
					slog.Warn("zone command references non-existent room",
						"zone", z.Number, "cmd_index", i, "command", "D", "room_vnum", cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !objVnums[cmd.Arg3] {
					slog.Warn("zone command references non-existent key object",
						"zone", z.Number, "cmd_index", i, "command", "D", "key_vnum", cmd.Arg3)
				}
			case "L":
				// 'L' <room_vnum> <door_state> <key_vnum> (like 'D')
				if !roomVnums[cmd.Arg1] {
					slog.Warn("zone command references non-existent room",
						"zone", z.Number, "cmd_index", i, "command", "L", "room_vnum", cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !objVnums[cmd.Arg3] {
					slog.Warn("zone command references non-existent key object",
						"zone", z.Number, "cmd_index", i, "command", "L", "key_vnum", cmd.Arg3)
				}
			case "R":
				// 'R' <room_vnum> <last_room> (remove rooms... unclear)
				if cmd.Arg1 > 0 && !roomVnums[cmd.Arg1] {
					slog.Warn("zone command references non-existent room",
						"zone", z.Number, "cmd_index", i, "command", "R", "room_vnum", cmd.Arg1)
				}
			}
		}
	}
}
