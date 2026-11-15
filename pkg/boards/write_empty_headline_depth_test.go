package boards

import "testing"

// TestWriteMessageEmptyHeadlineReleasesSlotApproved pins DP-1388's approved
// divergence. C's Board_write_message takes the slot with find_slot before its
// empty-headline check (src/boards.c:228) and returns without releasing it
// (src/boards.c:237-241). Zach approved freeing the slot on that return; the
// message stays C's exact "We must have a headline!", and the next post reuses
// slot 0.
func TestWriteMessageEmptyHeadlineReleasesSlotApproved(t *testing.T) {
	bs := InitBoards(t.TempDir())
	player := newMockBoardPlayer("Headlineprobe", 0, 8008)

	if got := bs.WriteMessage(0, player, ""); got != -1 {
		t.Fatalf("WriteMessage(empty headline) = %d, want -1", got)
	}
	if got := player.lastMessage(); got != "We must have a headline!\r\n" {
		t.Fatalf("empty-headline message = %q", got)
	}
	if bs.msgStorageTaken[0] {
		t.Fatal("slot 0 stayed taken; DP-1388 approved releasing it on the empty-headline return")
	}

	if got := bs.WriteMessage(0, player, "headline"); got != BoardMagic {
		t.Fatalf("WriteMessage(headline) = %d, want the board magic", got)
	}
	if slot := bs.writerSlots[player.GetName()]; slot != 0 {
		t.Fatalf("second post took slot %d, want 0 (the released slot)", slot)
	}
}
