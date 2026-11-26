package boards

import (
	"testing"
)

// TestBoardWriteHeadingAllocFailureUnitProof pins sweep row 4 under the
// 2026-11-11 unit-proof exception (Zach's 2026-11-25 ruling 6): when C's
// CREATE(NEW_MSG_INDEX(board_type).heading, char, ...) fails at
// src/boards.c:248-255, Board_write_message emits
// "The board is malfunctioning - sorry.\r\n" and aborts without adding a post.
func TestBoardWriteHeadingAllocFailureUnitProof(t *testing.T) {
	bs := InitBoards(t.TempDir())
	bs.allocateHeading = func(int) []byte { return nil }

	ch := &mockBoardPlayer{name: "Boardgod", level: 40}
	if magic := bs.WriteMessage(0, ch, "test subject"); magic != -1 {
		t.Fatalf("WriteMessage = %d, want -1 on heading allocation failure", magic)
	}
	const want = "The board is malfunctioning - sorry.\r\n"
	if len(ch.messages) != 1 || ch.messages[0] != want {
		t.Fatalf("messages = %q, want [%q]", ch.messages, want)
	}
	if got := bs.numOfMsgs[0]; got != 0 {
		t.Fatalf("numOfMsgs[0] = %d, want 0", got)
	}
}
