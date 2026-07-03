package game

import "testing"

func TestRoomBitNamesMatchCSourceOrder(t *testing.T) {
	expected := []string{
		"DARK", "DEATH", "NO_MOB", "INDOORS", "PEACEFUL", "SOUNDPROOF",
		"NOTRACK", "NO_MAGIC", "TUNNEL", "PRIVATE", "GODROOM",
		"HOUSE", "HOUSE_CRASH", "ATRIUM", "OLC", "BFS_MARK",
		"NEUTRAL", "BFR", "REGENROOM", "NO_WHO", "SECRET_MARK",
		"FLOW_N", "FLOW_S", "FLOW_E", "FLOW_W", "FLOW_U", "FLOW_D",
		"ARENA",
	}
	if len(RoomBitNames) != len(expected) {
		t.Fatalf("RoomBitNames has %d entries, expected %d", len(RoomBitNames), len(expected))
	}
	for i, name := range RoomBitNames {
		if name != expected[i] {
			t.Errorf("RoomBitNames[%d] = %q, expected %q", i, name, expected[i])
		}
	}
}

func TestRoomBitConstantsMatchCStructs(t *testing.T) {
	// Verify key constants match C src/structs.h values
	tests := []struct{ name string; got, want uint32 }{
		{"ROOM_DARK", RoomDark, 1 << 0},
		{"ROOM_DEATH", RoomDeath, 1 << 1},
		{"ROOM_TUNNEL", RoomTunnel, 1 << 8},
		{"ROOM_NEUTRAL", RoomNeutral, 1 << 16},
		{"ROOM_BFR", RoomBfr, 1 << 17},
		{"ROOM_ARENA", RoomArena, 1 << 27},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}
