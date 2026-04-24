// Package parser provides utilities for parsing Dark Pawns world files.
package parser

import (
	"fmt"
)

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
				fmt.Printf("[WARN] Room %d exit %q points to non-existent room vnum %d\n", r.VNum, dir, exit.ToRoom)
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
					fmt.Printf("[WARN] Zone %d cmd[%d] ('M'): mob vnum %d not found\n", z.Number, i, cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !roomVnums[cmd.Arg3] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('M'): room vnum %d not found\n", z.Number, i, cmd.Arg3)
				}
			case "O":
				// 'O' <obj_vnum> <max_in_world> <room_vnum>
				if !objVnums[cmd.Arg1] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('O'): obj vnum %d not found\n", z.Number, i, cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !roomVnums[cmd.Arg3] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('O'): room vnum %d not found\n", z.Number, i, cmd.Arg3)
				}
			case "G":
				// 'G' <obj_vnum> <max_in_world> (give to object)
				if !objVnums[cmd.Arg1] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('G'): obj vnum %d not found\n", z.Number, i, cmd.Arg1)
				}
			case "E":
				// 'E' <obj_vnum> <equip_position> <room_vnum>
				if !objVnums[cmd.Arg1] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('E'): obj vnum %d not found\n", z.Number, i, cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !roomVnums[cmd.Arg3] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('E'): room vnum %d not found\n", z.Number, i, cmd.Arg3)
				}
			case "P":
				// 'P' <obj_vnum> <max_in_world> <container_vnum>
				if !objVnums[cmd.Arg1] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('P'): obj vnum %d not found\n", z.Number, i, cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !objVnums[cmd.Arg3] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('P'): container obj vnum %d not found\n", z.Number, i, cmd.Arg3)
				}
			case "D":
				// 'D' <room_vnum> <door_state> <key_vnum>
				if !roomVnums[cmd.Arg1] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('D'): room vnum %d not found\n", z.Number, i, cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !objVnums[cmd.Arg3] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('D'): key obj vnum %d not found\n", z.Number, i, cmd.Arg3)
				}
			case "L":
				// 'L' <room_vnum> <door_state> <key_vnum> (like 'D')
				if !roomVnums[cmd.Arg1] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('L'): room vnum %d not found\n", z.Number, i, cmd.Arg1)
				}
				if cmd.Arg3 > 0 && !objVnums[cmd.Arg3] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('L'): key obj vnum %d not found\n", z.Number, i, cmd.Arg3)
				}
			case "R":
				// 'R' <room_vnum> <last_room> (remove rooms... unclear)
				if cmd.Arg1 > 0 && !roomVnums[cmd.Arg1] {
					fmt.Printf("[WARN] Zone %d cmd[%d] ('R'): room vnum %d not found\n", z.Number, i, cmd.Arg1)
				}
			}
		}
	}
}
