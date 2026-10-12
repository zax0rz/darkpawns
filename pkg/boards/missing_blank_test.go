package boards

// missing_blank_test.go — PF-063 cluster 5 proof for the board read: C's
// editor stores each accepted line WITH its break (modify.c string_add), so
// its stored body ends "\r\n" and the read format's trailing "%s\r\n"
// (boards.c:361-364) prints one more break — the blank line before the
// pager prompt. This port stores between-line breaks, so the read restores
// the body's final break. Exact bytes; reverting fails.

import (
	"strings"
	"testing"
)

func TestDisplayMsgRestoresBodyTrailingBreak(t *testing.T) {
	bs := InitBoards(t.TempDir())
	ch := newMockBoardPlayer("Grace", 1, 6001)

	magic := bs.WriteMessage(0, ch, "depth headline")
	if magic == 0 {
		t.Fatal("WriteMessage returned no magic")
	}
	bs.AppendBoardLine(magic, ch, "board body")
	bs.FinalizeBoardWrite(magic, ch)

	if !bs.DisplayMsg(0, ch, "1") {
		t.Fatal("DisplayMsg(1) = false, want true")
	}
	// The heading carries the wall-clock date, so pin the byte-exact shape
	// around it: header line, blank, body, the body's restored break, then
	// the format's — C's exact wire shape (boards.c:361-364).
	got := ch.lastMessage()
	if !strings.HasPrefix(got, "Message 1 : ") || !strings.HasSuffix(got, "\r\n\r\nboard body\r\n\r\n") {
		t.Fatalf("read bytes = %q, want Message 1 : <heading>\\r\\n\\r\\nboard body\\r\\n\\r\\n (stored break restored, then the format's)", got)
	}
}

// A body that already carries its trailing break (C-format storage, or a
// single-line body stored with one) must not gain a second.
func TestDisplayMsgDoesNotDoubleAnExistingBreak(t *testing.T) {
	bs := InitBoards(t.TempDir())
	ch := newMockBoardPlayer("Grace", 1, 6001)

	magic := bs.WriteMessage(0, ch, "already broken")
	if magic == 0 {
		t.Fatal("WriteMessage returned no magic")
	}
	bs.AppendBoardLine(magic, ch, "board body")
	bs.AppendBoardLine(magic, ch, "")
	bs.FinalizeBoardWrite(magic, ch)

	if !bs.DisplayMsg(0, ch, "1") {
		t.Fatal("DisplayMsg(1) = false, want true")
	}
	got := ch.lastMessage()
	if !strings.HasPrefix(got, "Message 1 : ") || !strings.HasSuffix(got, "\r\n\r\nboard body\r\n\r\n") {
		t.Fatalf("read bytes = %q, want Message 1 : <heading>\\r\\n\\r\\nboard body\\r\\n\\r\\n (existing break not doubled)", got)
	}
}
