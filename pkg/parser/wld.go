// Package parser handles loading Dark Pawns world files.
package parser

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Room represents a parsed room from a .wld file.
type Room struct {
	VNum            int
	Name            string
	Description     string
	Zone            int
	Flags           []string // 4-element array of flag bitmask hex strings
	Sector          int
	Exits           map[string]Exit
	ExtraDescs      []ExtraDesc
	ScriptName      string
	ScriptFunctions int
}

// Exit represents a room exit.
type Exit struct {
	Direction    string
	ToRoom       int
	DoorState    int // 0=open, 1=EX_ISDOOR, 2=EX_ISDOOR|EX_PICKPROOF
	Key          int // vnum of key, or -1
	Keywords     string
	Description  string
}



// ParseWldFile parses a single .wld file and returns all rooms.
func ParseWldFile(path string) ([]Room, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	var rooms []Room
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}

		// Room starts with #<vnum>
		if strings.HasPrefix(line, "#") {
			vnumStr := line[1:]

			// Special case: #99999 is end-of-world marker
			if vnumStr == "99999" {
				break
			}

			vnum, err := strconv.Atoi(vnumStr)
			if err != nil {
				return nil, fmt.Errorf("invalid room vnum: %s", line)
			}

			room, err := parseRoom(scanner, vnum)
			if err != nil {
				return nil, fmt.Errorf("parse room %d: %w", vnum, err)
			}
			rooms = append(rooms, room)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return rooms, nil
}

// readTildeString reads a ~-terminated string that may span multiple lines.
func readTildeString(scanner *bufio.Scanner) (string, error) {
	var parts []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "~") {
			parts = append(parts, strings.TrimSuffix(line, "~"))
			return strings.Join(parts, "\n"), nil
		}
		parts = append(parts, line)
	}
	return "", fmt.Errorf("unexpected EOF while reading ~-terminated string")
}

// parseRoom parses a single room from the scanner.
func parseRoom(scanner *bufio.Scanner, vnum int) (Room, error) {
	room := Room{
		VNum:  vnum,
		Exits: make(map[string]Exit),
	}

	// Parse name (~-terminated, may span multiple lines)
	name, err := readTildeString(scanner)
	if err != nil {
		return room, fmt.Errorf("expected room name: %w", err)
	}
	room.Name = name

	// Parse description (~-terminated, may span multiple lines)
	desc, err := readTildeString(scanner)
	if err != nil {
		return room, fmt.Errorf("expected room description: %w", err)
	}
	room.Description = desc

	// Parse numeric line: zone flags[0] flags[1] flags[2] flags[3] sector
	if !scanner.Scan() {
		return room, fmt.Errorf("expected numeric line")
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 6 {
		return room, fmt.Errorf("numeric line has %d fields, expected 6", len(fields))
	}

	room.Zone, _ = strconv.Atoi(fields[0])
	room.Flags = []string{fields[1], fields[2], fields[3], fields[4]}
	room.Sector, _ = strconv.Atoi(fields[5])

	// Parse sections until 'S' (end of room)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "S" {
			break
		}

		if strings.HasPrefix(line, "D") {
			dirNum, err := strconv.Atoi(line[1:])
			if err != nil || dirNum < 0 || dirNum >= 6 {
				continue
			}
			directions := []string{"north", "east", "south", "west", "up", "down"}
			exit, err := parseExit(scanner, directions[dirNum])
			if err != nil {
				return room, fmt.Errorf("parse exit %s in room %d: %w", directions[dirNum], vnum, err)
			}
			room.Exits[directions[dirNum]] = exit
		} else if strings.HasPrefix(line, "E") {
			// Extra description block
			extra, err := parseExtraDesc(scanner)
			if err != nil {
				return room, fmt.Errorf("parse extra desc in room %d: %w", vnum, err)
			}
			room.ExtraDescs = append(room.ExtraDescs, extra)
		} else if strings.HasPrefix(line, "R") {
			// Room script line: R <script_name> <functions>
			rest := strings.TrimSpace(line[1:])
			scriptParts := strings.Fields(rest)
			if len(scriptParts) >= 2 {
				room.ScriptName = scriptParts[0]
				fnCount, _ := strconv.Atoi(scriptParts[1])
				room.ScriptFunctions = fnCount
			} else if len(scriptParts) == 1 {
				room.ScriptName = scriptParts[0]
			}
		}
	}

	return room, nil
}

// parseExtraDesc parses an 'E' extra description block: keyword line + ~-terminated description.
func parseExtraDesc(scanner *bufio.Scanner) (ExtraDesc, error) {
	keyword, err := readTildeString(scanner)
	if err != nil {
		return ExtraDesc{}, fmt.Errorf("expected extra desc keyword: %w", err)
	}

	desc, err := readTildeString(scanner)
	if err != nil {
		return ExtraDesc{}, fmt.Errorf("expected extra desc text: %w", err)
	}

	return ExtraDesc{Keywords: keyword, Description: desc}, nil
}

// parseExit parses a single exit section.
func parseExit(scanner *bufio.Scanner, direction string) (Exit, error) {
	exit := Exit{Direction: direction}

	// Description (~-terminated, may span multiple lines)
	desc, err := readTildeString(scanner)
	if err != nil {
		return exit, fmt.Errorf("expected exit description: %w", err)
	}
	exit.Description = desc

	// Keywords (~-terminated, may span multiple lines)
	kw, err := readTildeString(scanner)
	if err != nil {
		return exit, fmt.Errorf("expected exit keywords: %w", err)
	}
	exit.Keywords = kw

	// Numeric line: door_state key to_room
	if !scanner.Scan() {
		return exit, fmt.Errorf("expected exit numeric line")
	}
	nums := strings.Fields(scanner.Text())
	if len(nums) >= 3 {
		exit.DoorState, _ = strconv.Atoi(nums[0])
		exit.Key, _ = strconv.Atoi(nums[1])
		exit.ToRoom, _ = strconv.Atoi(nums[2])
	}

	return exit, nil
}

// ParseAllWldFiles parses all .wld files in a directory.
func ParseAllWldFiles(dir string) ([]Room, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}

	var allRooms []Room
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".wld") {
			continue
		}

		path := dir + "/" + entry.Name()
		rooms, err := ParseWldFile(path)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", entry.Name(), err)
		}

		allRooms = append(allRooms, rooms...)
	}

	return allRooms, nil
}
