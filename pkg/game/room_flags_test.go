package game

import (
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
