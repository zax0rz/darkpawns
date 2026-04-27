// Package parser handles loading Dark Pawns world files.
package parser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Zone represents a parsed zone from a .zon file.
type Zone struct {
	Number    int
	Name      string
	TopRoom   int
	Lifespan  int // minutes between resets
	ResetMode int // 0=never, 1=if empty, 2=always
	Commands  []ZoneCommand
}

// ZoneCommand represents a single reset command in a zone.
type ZoneCommand struct {
	Command string // 'M', 'O', 'G', 'E', 'P', 'D', 'L', 'R'
	IfFlag  int    // 0=always, 1=only if previous command succeeded
	Arg1    int    // vnum (mob/obj/room)
	Arg2    int    // max in world / equip position / door state
	Arg3    int    // room vnum / container vnum / probability
}

// ParseZonFile parses a single .zon file and returns the zone.
func ParseZonFile(path string) (*Zone, error) {
// #nosec G304
	file, err := os.Open(path) // #nosec G703 — world data, trusted internal path
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// First line: #<zone_number>
	if !scanner.Scan() {
		return nil, fmt.Errorf("expected zone number line")
	}
	line := strings.TrimSpace(scanner.Text())
	if !strings.HasPrefix(line, "#") {
		return nil, fmt.Errorf("expected zone number, got: %s", line)
	}

	zoneNum, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, fmt.Errorf("invalid zone number: %s", line)
	}

	zone := &Zone{Number: zoneNum}

	// Second line: zone name (ends with ~)
	if !scanner.Scan() {
		return nil, fmt.Errorf("expected zone name")
	}
	zone.Name = strings.TrimSuffix(scanner.Text(), "~")

	// Third line: top_room lifespan reset_mode
	if !scanner.Scan() {
		return nil, fmt.Errorf("expected zone constants line")
	}
	consts := strings.Fields(scanner.Text())
	if len(consts) >= 3 {
		zone.TopRoom, _ = strconv.Atoi(consts[0])
		zone.Lifespan, _ = strconv.Atoi(consts[1])
		zone.ResetMode, _ = strconv.Atoi(consts[2])
	}

	// Parse commands until 'S'
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}

		if line == "S" {
			break
		}

		cmd, err := parseZoneCommand(line)
		if err != nil {
			return nil, fmt.Errorf("parse command: %w", err)
		}
		zone.Commands = append(zone.Commands, cmd)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return zone, nil
}

func parseZoneCommand(line string) (ZoneCommand, error) {
	var cmd ZoneCommand

	fields := strings.Fields(line)
	if len(fields) < 1 {
		return cmd, fmt.Errorf("empty command")
	}

	cmd.Command = fields[0]

	// Command format varies by type:
	// M <if_flag> <mob_vnum> <max_in_world> <room_vnum>
	// O <if_flag> <obj_vnum> <max_in_world> <room_vnum>
	// G <if_flag> <obj_vnum> <max_in_world>
	// E <if_flag> <obj_vnum> <max_in_world> <equip_position>
	// P <if_flag> <obj_vnum> <max_in_world> <container_vnum>
	// D <if_flag> <room_vnum> <direction> <door_state>
	// L <if_flag> <room_vnum> <direction> <lock_state>
	// R <if_flag> <room_vnum> <obj_or_mob_vnum> <is_obj>

	switch cmd.Command {
	case "M": // Load mobile
		if len(fields) >= 5 {
			cmd.IfFlag, _ = strconv.Atoi(fields[1])
			cmd.Arg1, _ = strconv.Atoi(fields[2]) // mob vnum
			cmd.Arg2, _ = strconv.Atoi(fields[3]) // max in world
			cmd.Arg3, _ = strconv.Atoi(fields[4]) // room vnum
		}
	case "O": // Load object to room
		if len(fields) >= 5 {
			cmd.IfFlag, _ = strconv.Atoi(fields[1])
			cmd.Arg1, _ = strconv.Atoi(fields[2]) // obj vnum
			cmd.Arg2, _ = strconv.Atoi(fields[3]) // max in world
			cmd.Arg3, _ = strconv.Atoi(fields[4]) // room vnum
		}
	case "G": // Give object to last loaded mob
		if len(fields) >= 4 {
			cmd.IfFlag, _ = strconv.Atoi(fields[1])
			cmd.Arg1, _ = strconv.Atoi(fields[2]) // obj vnum
			cmd.Arg2, _ = strconv.Atoi(fields[3]) // max in world
		}
	case "E": // Equip object on last loaded mob
		if len(fields) >= 5 {
			cmd.IfFlag, _ = strconv.Atoi(fields[1])
			cmd.Arg1, _ = strconv.Atoi(fields[2]) // obj vnum
			cmd.Arg2, _ = strconv.Atoi(fields[3]) // max in world
			cmd.Arg3, _ = strconv.Atoi(fields[4]) // equip position
		}
	case "P": // Put object in container
		if len(fields) >= 5 {
			cmd.IfFlag, _ = strconv.Atoi(fields[1])
			cmd.Arg1, _ = strconv.Atoi(fields[2]) // obj vnum
			cmd.Arg2, _ = strconv.Atoi(fields[3]) // max in world
			cmd.Arg3, _ = strconv.Atoi(fields[4]) // container vnum
		}
	case "D": // Door state
		if len(fields) >= 5 {
			cmd.IfFlag, _ = strconv.Atoi(fields[1])
			cmd.Arg1, _ = strconv.Atoi(fields[2]) // room vnum
			cmd.Arg2, _ = strconv.Atoi(fields[3]) // direction
			cmd.Arg3, _ = strconv.Atoi(fields[4]) // door state (0=open, 1=closed, 2=locked)
		}
	case "L": // Lock door
		if len(fields) >= 5 {
			cmd.IfFlag, _ = strconv.Atoi(fields[1])
			cmd.Arg1, _ = strconv.Atoi(fields[2]) // room vnum
			cmd.Arg2, _ = strconv.Atoi(fields[3]) // direction
			cmd.Arg3, _ = strconv.Atoi(fields[4]) // lock state
		}
	case "R": // Remove obj/mob from room
		if len(fields) >= 5 {
			cmd.IfFlag, _ = strconv.Atoi(fields[1])
			cmd.Arg1, _ = strconv.Atoi(fields[2]) // room vnum
			cmd.Arg2, _ = strconv.Atoi(fields[3]) // obj or mob vnum
			cmd.Arg3, _ = strconv.Atoi(fields[4]) // 1=obj, 0=mob
		}
	}

	return cmd, nil
}

// ParseAllZonFiles parses all .zon files in a directory.
func ParseAllZonFiles(dir string) ([]Zone, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	var zones []Zone
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".zon") {
			continue
		}

		path := dir + "/" + entry.Name()
		zone, err := ParseZonFile(path)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		zones = append(zones, *zone)
	}

	return zones, nil
}
