// Package game — ported from src/graph.c (BFS pathfinding).
//
// BFS shortest-path for mob tracking (hunt_victim) and the
// player 'track' skill (do_track).
//
// Source: CircleMUD / Dark Pawns, copyrights as in original header.

package game

import (
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// =========================================================================
// Constants from graph.c
// =========================================================================

const (
	// Symbolic error codes returned by findFirstStep
	BFS_ERROR         = -1
	BFS_ALREADY_THERE = -2
	BFS_NO_PATH       = -3
)

// TrackSkill reference (from C SKILL_TRACK)
const skillTrack = "SKILL_TRACK" // placeholder — actual skill name TBD

// =========================================================================
// BFS queue (slice-based, replaces linked list from graph.c)
// =========================================================================

type bfsEntry struct {
	room int // real index into worldRooms
	dir  int // direction from src (first step direction)
}

// findFirstStep returns the first direction to take on the shortest path
// from src to target within the worldRooms slice.
//
// C: find_first_step()
// Returns BFS_ERROR, BFS_ALREADY_THERE, BFS_NO_PATH, or a direction (0–5).
func findFirstStep(src int, target int, topOfWorld int, worldRooms []*parser.Room) int {
	if src < 0 || src > topOfWorld || target < 0 || target > topOfWorld {
		return BFS_ERROR
	}
	if src == target {
		return BFS_ALREADY_THERE
	}

	// roomMarks tracks visited rooms (C: room flags with ROOM_BFS_MARK)
	roomMarks := make([]bool, topOfWorld+1)

	// Mark source
	roomMarks[src] = true

	// Queue for BFS
	queue := make([]bfsEntry, 0)

	// Enqueue initial edges
	for dir := 0; dir < numOfDirs; dir++ {
		toRoom, ok := validEdge(src, dir, worldRooms, topOfWorld, roomMarks)
		if ok {
			roomMarks[toRoom] = true
			queue = append(queue, bfsEntry{room: toRoom, dir: dir})
		}
	}

	// BFS loop
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]

		if cur.room == target {
			return cur.dir
		}

		for dir := 0; dir < numOfDirs; dir++ {
			toRoom, ok := validEdge(cur.room, dir, worldRooms, topOfWorld, roomMarks)
			if ok {
				roomMarks[toRoom] = true
				queue = append(queue, bfsEntry{room: toRoom, dir: cur.dir})
			}
		}
	}

	return BFS_NO_PATH
}

// =========================================================================
// validEdge checks if an edge from a room in a given direction is valid
// for BFS traversal. C: VALID_EDGE macro (TRACK_THROUGH_DOORS version).
// =========================================================================

func validEdge(roomIdx int, dir int, worldRooms []*parser.Room, topOfWorld int, marks []bool) (int, bool) {
	if roomIdx < 0 || roomIdx > topOfWorld {
		return -1, false
	}
	r := worldRooms[roomIdx]
	if r == nil {
		return -1, false
	}

	dirNames := []string{"north", "east", "south", "west", "up", "down"}
	if dir < 0 || dir >= len(dirNames) {
		return -1, false
	}

	exit, hasExit := r.Exits[dirNames[dir]]
	if !hasExit {
		return -1, false
	}
	toRoom := exit.ToRoom
	if toRoom <= 0 {
		return -1, false
	}

	toRoomIdx := roomVNumToIndex(toRoom, worldRooms, topOfWorld)
	if toRoomIdx < 0 {
		return -1, false
	}

	if marks[toRoomIdx] {
		return -1, false
	}

	// Check ROOM_NOTRACK flag (if available)
	if HasRoomFlag != nil && HasRoomFlag(toRoomIdx, "ROOM_NOTRACK") {
		return -1, false
	}

	// Check sector: no water swim/noswim
	r2 := worldRooms[toRoomIdx]
	if r2 != nil {
		if r2.Sector == sectWaterSwim || r2.Sector == sectWaterNoSwim {
			return -1, false
		}
	}

	return toRoomIdx, true
}

// roomVNumToIndex finds the real-room index for a vnum.
// This should use the World's mapping, but as a fallback we do a linear scan.
var roomVNumMap func(vnum int, worldRooms []*parser.Room, topOfWorld int) int

func roomVNumToIndex(vnum int, worldRooms []*parser.Room, topOfWorld int) int {
	if roomVNumMap != nil {
		return roomVNumMap(vnum, worldRooms, topOfWorld)
	}
	// Fallback: linear scan (slow, avoid in production)
	for i := 0; i <= topOfWorld; i++ {
		if worldRooms[i] != nil && worldRooms[i].VNum == vnum {
			return i
		}
	}
	return -1
}

// =========================================================================
// Constants for sector types
// =========================================================================

const (
	sectWaterSwim   = 11
	sectWaterNoSwim = 12
	numOfDirs       = 6
)

// =========================================================================
// External hooks set by the game package
// =========================================================================

// HasRoomFlag is set by the game package to check room flags.
var HasRoomFlag func(roomIdx int, flag string) bool

// CanGo checks if a player can go in a given direction.
var CanGo func(ch *Player, dir int) bool

// ImproveSkill improves a character's skill.
var ImproveSkill func(ch *Player, skill string)

// IsAffected checks if a character has an affect.
var IsAffected func(ch *Player, aff string) bool

// MobFlagged checks if a mob has a flag.
var MobFlagged func(ch *Player, flag string) bool

// Outside checks if a character is outdoors.
var Outside func(ch *Player) bool

// SkyCondition returns the current sky condition.
var SkyCondition func() int

// IsWarrior / IsPaladin / IsRanger are class checks.
var IsWarrior func(ch *Player) bool
var IsPaladin func(ch *Player) bool
var IsRanger func(ch *Player) bool

// GetSkill returns a character's skill level.
var GetSkill func(ch *Player, skill string) int
