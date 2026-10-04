package boards

import (
	"strings"
	"testing"
)

// C board_info[] (src/boards.c:93-98) with the real ladder from
// src/structs.h:610-623: LVL_IMMORT=31, LVL_GRGOD=38, LVL_IMPL=40. The port
// carried a foreign 50/60/61 scale that no character on this MUD can reach, so
// every immortal-board gate was dead. TestBoardInfoMatchesCLevelLadder fails if
// any of those values comes back (R5h).
func TestBoardInfoMatchesCLevelLadder(t *testing.T) {
	bs := InitBoards(t.TempDir())
	want := []struct {
		vnum, read, write, remove int
	}{
		{8099, 0, 0, 31},   // LVL_IMMORT remove
		{8064, 0, 0, 31},   // LVL_IMMORT remove
		{8065, 0, 0, 31},   // LVL_IMMORT remove
		{8098, 31, 31, 38}, // LVL_IMMORT read/write, LVL_GRGOD remove
		{8096, 31, 31, 38}, // LVL_IMMORT read/write, LVL_GRGOD remove
		{8097, 31, 31, 40}, // LVL_IMMORT read/write, LVL_IMPL remove
	}
	for i, w := range want {
		got := bs.BoardInfo(i)
		if got.VNum != w.vnum || got.ReadLvl != w.read || got.WriteLvl != w.write || got.RemoveLvl != w.remove {
			t.Errorf("board %d = %+v, want vnum %d read/write/remove %d/%d/%d",
				i, got, w.vnum, w.read, w.write, w.remove)
		}
	}
}

// A level-40 Implementor must be able to read the immortal board (board 3,
// vnum 8098, reset into room 1204). C's READ_LVL gate admits LVL_IMMORT and
// above (src/boards.c:288-290); the foreign 50 answered the Implementor with
// "You try but fail to understand the holy words.".
// TestImmortalBoardReadableByImplementor fails on the old scale.
func TestImmortalBoardReadableByImplementor(t *testing.T) {
	bs := InitBoards(t.TempDir())
	impl := newMockBoardPlayer("Implementor", 40, 1204)
	if !bs.ShowBoard(3, impl) {
		t.Fatal("ShowBoard = false, want true")
	}
	if strings.Contains(impl.allMessages(), "holy words") {
		t.Fatalf("level-40 Implementor refused the immortal board: %q", impl.allMessages())
	}
	if !strings.Contains(impl.allMessages(), "The board is empty.") {
		t.Fatalf("Implementor did not reach the board listing: %q", impl.allMessages())
	}
}

// A mortal is still refused the immortal board, so the gate was corrected, not
// merely opened.
func TestImmortalBoardRefusedToMortal(t *testing.T) {
	bs := InitBoards(t.TempDir())
	mortal := newMockBoardPlayer("Mortal", 1, 1204)
	if !bs.ShowBoard(3, mortal) {
		t.Fatal("ShowBoard = false, want true")
	}
	if !strings.Contains(mortal.allMessages(), "holy words") {
		t.Fatalf("mortal was not refused the immortal board: %q", mortal.allMessages())
	}
}
