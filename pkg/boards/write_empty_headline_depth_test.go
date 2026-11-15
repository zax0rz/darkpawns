package boards

import "testing"

// TestWriteMessageEmptyHeadlineKeepsCAllocationOrder pins C's allocation order
// in Board_write_message (src/boards.c:213-240): the level and full checks run
// first, then find_slot() marks the slot taken (src/boards.c:115-125), and only
// afterwards does the empty-headline check return "We must have a headline!".
// C therefore consumes one storage slot for an empty headline, exactly as the
// port's WriteMessage does; DP-1388 reports that leak, but it is C parity, not
// a Go regression (R1/R4). This test is the control for any future approved
// divergence that releases the slot: it must change deliberately.
func TestWriteMessageEmptyHeadlineKeepsCAllocationOrder(t *testing.T) {
	bs := InitBoards(t.TempDir())
	player := newMockBoardPlayer("Headlineprobe", 0, 8008)

	if got := bs.WriteMessage(0, player, ""); got != -1 {
		t.Fatalf("WriteMessage(empty headline) = %d, want -1", got)
	}
	if got := player.lastMessage(); got != "We must have a headline!\r\n" {
		t.Fatalf("empty-headline message = %q", got)
	}
	if !bs.msgStorageTaken[0] {
		t.Fatal("slot 0 was not marked taken; C find_slot marks before the headline check")
	}

	if got := bs.WriteMessage(0, player, "headline"); got != BoardMagic {
		t.Fatalf("WriteMessage(headline) = %d, want the board magic", got)
	}
	if slot := bs.writerSlots[player.GetName()]; slot != 1 {
		t.Fatalf("second post took slot %d, want 1 (slot 0 stays consumed, as in C)", slot)
	}
}
