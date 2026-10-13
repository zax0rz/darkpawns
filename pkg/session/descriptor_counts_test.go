package session

import "testing"

func TestRecordUsageCountsDescriptorsAndConPlayingOnly(t *testing.T) {
	f := newConPlayingFixture(t)
	connected, playing := f.manager.CountSessions()
	if connected != 8 || playing != 4 {
		t.Fatalf("record_usage counts = connected %d, playing %d; want 8 descriptors and 4 CON_PLAYING", connected, playing)
	}
}

func TestZoneOccupancyExcludesNonPlayingDescriptors(t *testing.T) {
	f := newConPlayingFixture(t)
	got := f.manager.occupiedZoneRooms()
	if len(got) != 4 {
		t.Fatalf("occupied rooms = %v, want the four CON_PLAYING bodies", got)
	}
	for _, room := range got {
		if room != 1001 {
			t.Fatalf("occupied rooms = %v, want four playing bodies in room 1001", got)
		}
	}
}
