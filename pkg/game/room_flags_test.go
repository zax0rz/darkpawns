package game

import "testing"

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
