package game

import (
	"testing"
	"time"
)

func TestDisplayMsgBoardArgumentDoesNotLeakReadLock(t *testing.T) {
	bs := InitBoards(t.TempDir())
	ch := NewPlayer(1, "Alice", 1001)

	if bs.DisplayMsg(0, ch, "board") {
		t.Fatal("DisplayMsg(board) = true, want false redirect")
	}

	done := make(chan int, 1)
	go func() {
		done <- bs.WriteMessage(0, ch, "test heading")
	}()

	select {
	case magic := <-done:
		if magic != BoardMagic {
			t.Fatalf("WriteMessage magic = %d, want %d", magic, BoardMagic)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("WriteMessage blocked after DisplayMsg(board), read lock likely leaked")
	}
}
