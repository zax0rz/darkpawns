package game

import "strconv"

// Room flag bit values matching C's ROOM_* constants in src/structs.h.
const (
	RoomDark       = 1 << 0  // ROOM_DARK
	RoomDeath      = 1 << 1  // ROOM_DEATH
	RoomNoMob      = 1 << 2  // ROOM_NOMOB
	RoomIndoors    = 1 << 3  // ROOM_INDOORS
	RoomPeaceful   = 1 << 4  // ROOM_PEACEFUL
	RoomSoundproof = 1 << 5  // ROOM_SOUNDPROOF
	RoomNoTrack    = 1 << 6  // ROOM_NOTRACK
	RoomNoMagic    = 1 << 7  // ROOM_NOMAGIC
	RoomTunnel     = 1 << 8  // ROOM_TUNNEL
	RoomPrivate    = 1 << 9  // ROOM_PRIVATE
	RoomGodRoom    = 1 << 10 // ROOM_GODROOM
	RoomHouse      = 1 << 11 // ROOM_HOUSE
	RoomHouseCrash = 1 << 12 // ROOM_HOUSE_CRASH
	RoomAtrium     = 1 << 13 // ROOM_ATRIUM
	RoomOLC        = 1 << 14 // ROOM_OLC
	RoomBFSMark    = 1 << 15 // ROOM_BFS_MARK
	RoomNeutral    = 1 << 16 // ROOM_NEUTRAL
	RoomBfr        = 1 << 17 // ROOM_BFR
	RoomRegen      = 1 << 18 // ROOM_REGENROOM
	RoomNoWho      = 1 << 19 // ROOM_NO_WHO_ROOM
	RoomSecretMark = 1 << 20 // ROOM_SECRET_MARK
	RoomFlowNorth  = 1 << 21 // ROOM_FLOW_NORTH
	RoomFlowSouth  = 1 << 22 // ROOM_FLOW_SOUTH
	RoomFlowEast   = 1 << 23 // ROOM_FLOW_EAST
	RoomFlowWest   = 1 << 24 // ROOM_FLOW_WEST
	RoomFlowUp     = 1 << 25 // ROOM_FLOW_UP
	RoomFlowDown   = 1 << 26 // ROOM_FLOW_DOWN
	RoomArena      = 1 << 27 // ROOM_ARENA
)

// roomHasFlagBit checks if a room's decimal flag array has a specific bit set.
func roomHasFlagBit(flags []string, flagBit int) bool {
	if len(flags) < 1 {
		return false
	}
	word := flagBit / 32
	bit := flagBit % 32
	if word >= len(flags) {
		return false
	}
	val, err := strconv.ParseUint(flags[word], 10, 32)
	if err != nil {
		return false
	}
	return val&(1<<uint(bit)) != 0
}

// hasWearFlag checks if a [4]int wear flags array has a specific bit set.
func hasWearFlag(wf [4]int, bit int) bool {
	word := bit / 32
	b := bit % 32
	if word >= 4 {
		return false
	}
	return wf[word]&(1<<uint(b)) != 0
}
