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
	VNum        int
	Name        string
	Description string
	Zone        int
	Flags       []string
	Sector      int
	Exits       map[string]Exit
}

// Exit represents a room exit.
type Exit struct {
	Direction   string
	ToRoom      int
	DoorState   int // 0=open, 1=closed, 2=locked
	Key         int // vnum of key, or -1
	Keywords    string
	Description string
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

// parseRoom parses a single room from the scanner.
func parseRoom(scanner *bufio.Scanner, vnum int) (Room, error) {
	room := Room{
		VNum:  vnum,
		Exits: make(map[string]Exit),
	}

	// Parse name (ends with ~)
	if !scanner.Scan() {
		return room, fmt.Errorf("expected room name")
	}
	room.Name = strings.TrimSuffix(scanner.Text(), "~")

	// Parse description (ends with ~)
	var descLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasSuffix(line, "~") {
			descLines = append(descLines, strings.TrimSuffix(line, "~"))
			break
		}
		descLines = append(descLines, line)
	}
	room.Description = strings.Join(descLines, "\n")

	// Parse numeric line: zone flags sector 0 0 0
	if !scanner.Scan() {
		return room, fmt.Errorf("expected numeric line")
	}
	nums := strings.Fields(scanner.Text())
	if len(nums) < 6 {
		return room, fmt.Errorf("numeric line has %d fields, expected 6", len(nums))
	}

	room.Zone, _ = strconv.Atoi(nums[0])
	// nums[1] = flags (bitmask)
	flags, _ := strconv.Atoi(nums[1])
	room.Flags = parseRoomFlags(flags)
	// nums[2] = sector type
	room.Sector, _ = strconv.Atoi(nums[2])

	// Parse exits and other sections until 'S'
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "S" {
			break // End of room
		}

		if strings.HasPrefix(line, "D") && len(line) == 2 {
			// Exit: D0-D5 (N,E,S,W,U,D)
			dirNum, _ := strconv.Atoi(line[1:])
			directions := []string{"north", "east", "south", "west", "up", "down"}
			if dirNum >= 0 && dirNum < 6 {
				exit, err := parseExit(scanner, directions[dirNum])
				if err != nil {
					return room, fmt.Errorf("parse exit %s: %w", directions[dirNum], err)
				}
				room.Exits[directions[dirNum]] = exit
			}
		}
	}

	return room, nil
}

// parseRoomFlags converts a bitmask integer to flag names.
// Source: structs.h - ROOM_DEATH = 1 (bit 1), ROOM_NOMOB = 2 (bit 2)
func parseRoomFlags(bitmask int) []string {
	var flags []string
	// Map bit positions to flag names (from structs.h)
	flagMap := map[int]string{
		0:  "dark",        // ROOM_DARK
		1:  "death",       // ROOM_DEATH
		2:  "nomob",       // ROOM_NOMOB
		3:  "indoors",     // ROOM_INDOORS
		4:  "peaceful",    // ROOM_PEACEFUL
		5:  "soundproof",  // ROOM_SOUNDPROOF
		6:  "notrack",     // ROOM_NOTRACK
		7:  "nomagic",     // ROOM_NOMAGIC
		8:  "tunnel",      // ROOM_TUNNEL
		9:  "private",     // ROOM_PRIVATE
		10: "godroom",     // ROOM_GODROOM
		11: "house",       // ROOM_HOUSE
		12: "house_crash", // ROOM_HOUSE_CRASH
		13: "atrium",      // ROOM_ATRIUM
		14: "olc",         // ROOM_OLC
		15: "bspace",      // ROOM_BSPACE
	}

	for bit, name := range flagMap {
		if bitmask&(1<<bit) != 0 {
			flags = append(flags, name)
		}
	}
	return flags
}

// parseExit parses a single exit section.
func parseExit(scanner *bufio.Scanner, direction string) (Exit, error) {
	exit := Exit{Direction: direction}

	// Description (ends with ~)
	if !scanner.Scan() {
		return exit, fmt.Errorf("expected exit description")
	}
	exit.Description = strings.TrimSuffix(scanner.Text(), "~")

	// Keywords (ends with ~)
	if !scanner.Scan() {
		return exit, fmt.Errorf("expected exit keywords")
	}
	exit.Keywords = strings.TrimSuffix(scanner.Text(), "~")

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
