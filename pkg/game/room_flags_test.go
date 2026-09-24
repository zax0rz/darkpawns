package game

import (
	"fmt"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestRoomHasFlagBit_DecimalNotHex(t *testing.T) {
	// 32768 decimal = bit 15 only. Should NOT have bit 8 (ROOM_TUNNEL).
	if roomHasFlagBit([]string{"32768"}, 8) {
		t.Fatal("32768 decimal should not set bit 8 — base-16 parse bug")
	}
	// 256 decimal = bit 8 (ROOM_TUNNEL).
	if !roomHasFlagBit([]string{"256"}, 8) {
		t.Fatal("256 decimal should set bit 8")
	}
	// 0 decimal = no bits.
	if roomHasFlagBit([]string{"0"}, 8) {
		t.Fatal("0 should set no bits")
	}
}

func TestRoomHasFlagBit_MultipleFlags(t *testing.T) {
	// Flags "260" = bit 2 + bit 8
	if !roomHasFlagBit([]string{"260"}, 2) {
		t.Fatal("260 should have bit 2")
	}
	if !roomHasFlagBit([]string{"260"}, 8) {
		t.Fatal("260 should have bit 8")
	}
	if roomHasFlagBit([]string{"260"}, 3) {
		t.Fatal("260 should not have bit 3")
	}
}

func TestRoomFlagReadersAgreeOnRealHighFlagRooms(t *testing.T) {
	libWorldDir, err := filepath.Abs("../../lib/world")
	if err != nil {
		t.Fatal(err)
	}
	world, err := parser.ParseWorld(libWorldDir)
	if err != nil {
		t.Fatalf("ParseWorld(real world): %v", err)
	}
	rooms := make(map[int]*parser.Room, len(world.Rooms))
	for i := range world.Rooms {
		room := &world.Rooms[i]
		rooms[room.VNum] = room
	}
	tests := []struct {
		room int
		bit  int
	}{
		{8152, 16},  // ROOM_NEUTRAL
		{14473, 17}, // ROOM_BFR
		{1372, 18},  // ROOM_REGENROOM
		{14193, 19}, // ROOM_NO_WHO_ROOM
		{14317, 20}, // ROOM_SECRET_MARK
		{1294, 21},  // ROOM_FLOW_NORTH
		{1351, 22},  // ROOM_FLOW_SOUTH
		{1349, 23},  // ROOM_FLOW_EAST
		{1356, 24},  // ROOM_FLOW_WEST
		{1295, 26},  // ROOM_FLOW_DOWN
		{8152, 27},  // ROOM_ARENA
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("room-%d-bit-%d", tt.room, tt.bit), func(t *testing.T) {
			room := rooms[tt.room]
			if room == nil {
				t.Fatalf("real world room %d missing", tt.room)
			}
			parserResult := room.HasFlag(tt.bit)
			gameResult := roomHasFlagBit(room.Flags, tt.bit)
			if !parserResult || !gameResult {
				t.Fatalf("room %d bit %d: parser=%v game=%v, want both set", tt.room, tt.bit, parserResult, gameResult)
			}
		})
	}
}

func TestRoomHasNamedFlagResolvesCanonicalBits(t *testing.T) {
	for bit, name := range RoomBitNames {
		t.Run(name, func(t *testing.T) {
			room := &parser.Room{Flags: []string{strconv.FormatUint(1<<uint(bit), 10), "0", "0", "0"}}
			if !roomHasNamedFlag(room, name) {
				t.Fatalf("%s did not resolve to room bit %d", name, bit)
			}
		})
	}
}

func TestRoomHasNamedFlagAcceptsCallSiteAliases(t *testing.T) {
	tests := []struct {
		flag string
		bit  int
	}{
		{"ROOM_PEACEFUL", 4},
		{"!MOB", 2},
		{"nomob", 2},
		{"!TRACK", 6},
		{"nomagic", 7},
		{"housecrash", 12},
		{"regen_room", 18},
		{"no_who_room", 19},
		{"flow_north", 21},
	}
	for _, tt := range tests {
		t.Run(tt.flag, func(t *testing.T) {
			room := &parser.Room{Flags: []string{strconv.FormatUint(1<<uint(tt.bit), 10), "0", "0", "0"}}
			if !roomHasNamedFlag(room, tt.flag) {
				t.Fatalf("%q did not resolve to room bit %d", tt.flag, tt.bit)
			}
		})
	}
}

func TestRoomHasNamedFlagPreservesDynamicNames(t *testing.T) {
	room := &parser.Room{Flags: []string{"0", "0", "0", "0", "house", "atrium", "custom_flag"}}
	for _, flag := range []string{"HOUSE", "Atrium", "custom_flag"} {
		if !roomHasNamedFlag(room, flag) {
			t.Errorf("dynamic flag %q was not preserved", flag)
		}
	}
	if roomHasNamedFlag(room, "peaceful") {
		t.Error("dynamic flags falsely matched peaceful")
	}
}

func TestWorldRoomHasFlagReadsStaticWorldBitvectors(t *testing.T) {
	w := &World{rooms: map[int]*parser.Room{
		8162: {VNum: 8162, Flags: []string{"28", "0", "0", "0"}},
		8163: {VNum: 8163, Flags: []string{"12", "0", "0", "0"}},
	}}

	if !w.RoomHasFlag(8162, "peaceful") {
		t.Error("room 8162 bitvector 28 should contain ROOM_PEACEFUL")
	}
	if w.RoomHasFlag(8163, "peaceful") {
		t.Error("room 8163 bitvector 12 should not contain ROOM_PEACEFUL")
	}
	if w.RoomHasFlag(9999, "peaceful") {
		t.Error("missing room should not contain any flag")
	}
}
