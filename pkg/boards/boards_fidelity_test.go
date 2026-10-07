package boards

import (
	"strings"
	"testing"
)

func writeStart(t *testing.T, bs *BoardSystem, name string, boardType int) (int, *mockBoardPlayer) {
	t.Helper()
	ch := newMockBoardPlayer(name, boardImplLevel+1, 1001)
	magic := bs.WriteMessage(boardType, ch, name+" headline")
	if magic <= 0 {
		t.Fatalf("WriteMessage(%s) = %d, want > 0", name, magic)
	}
	return magic, ch
}

// C's editor caps a post at MAX_MESSAGE_LENGTH (boards.h:34, modify.c:131):
// a first line over the cap is truncated with C's exact reply; a later line
// that would cross it is skipped with C's exact reply. The port grew the
// buffer without bound while the editor intercept runs ahead of the command
// rate limiter (VULN-011).
func TestBoardMessageCappedAtMaxLength(t *testing.T) {
	bs := InitBoards(t.TempDir())
	magic, ch := writeStart(t, bs, "Cap", 0)

	bs.AppendBoardLine(magic, ch, strings.Repeat("a", MaxMessageLength+50))
	if got := len(bs.msgStorage[bs.writerSlots["Cap"]]); got > MaxMessageLength {
		t.Fatalf("first line stored at %d bytes, want <= %d", got, MaxMessageLength)
	}
	if !strings.Contains(ch.lastMessage(), "String too long - Truncated.") {
		t.Fatalf("truncation reply = %q, want C's \"String too long - Truncated.\"", ch.lastMessage())
	}

	bs.AppendBoardLine(magic, ch, strings.Repeat("b", 200))
	if !strings.Contains(ch.lastMessage(), "String too long.  Last line skipped.") {
		t.Fatalf("overflow reply = %q, want C's \"String too long.  Last line skipped.\"", ch.lastMessage())
	}
	if strings.Contains(bs.msgStorage[bs.writerSlots["Cap"]], "bbbb") {
		t.Fatal("overflow line was appended despite the cap")
	}
}

// C binds the editor to the writer's own slot via desc->str (boards.c:268).
// The port resolved "newest message on the board", so a second writer's post
// re-targeted the first writer's lines — cross-writer content injection
// under the victim's attribution (VULN-012).
func TestBoardEditorBoundToWriterSlot(t *testing.T) {
	bs := InitBoards(t.TempDir())
	magicA, a := writeStart(t, bs, "Alice", 0)
	magicB, b := writeStart(t, bs, "Bob", 0)

	bs.AppendBoardLine(magicA, a, "alice line")
	bs.AppendBoardLine(magicB, b, "bob line")
	bs.AppendBoardLine(magicA, a, "alice second")

	slotA, slotB := bs.writerSlots["Alice"], bs.writerSlots["Bob"]
	if slotA == slotB {
		t.Fatalf("writers share slot %d", slotA)
	}
	if !strings.Contains(bs.msgStorage[slotA], "alice line") || !strings.Contains(bs.msgStorage[slotA], "alice second") {
		t.Fatalf("Alice's post = %q, want both of Alice's lines", bs.msgStorage[slotA])
	}
	if strings.Contains(bs.msgStorage[slotA], "bob line") {
		t.Fatal("Bob's line landed in Alice's post — editor still bound to newest message")
	}
	if !strings.Contains(bs.msgStorage[slotB], "bob line") {
		t.Fatalf("Bob's post = %q, want Bob's line", bs.msgStorage[slotB])
	}
}

// The edge of the cap, exactly (#1832 review): a line that fits only if the
// deferred "\r\n" separator is ignored must be skipped, because C's stored
// string already carries its line break at check time. One byte smaller is
// accepted. No scenario writes 4,096 bytes, so only this test can catch it.
func TestBoardCapEdgeExactlyTwoBytes(t *testing.T) {
	bs := InitBoards(t.TempDir())
	magic, ch := writeStart(t, bs, "Edge", 0)

	first := strings.Repeat("a", 100)
	bs.AppendBoardLine(magic, ch, first)

	// Sizes the second line so len(first)+len(second)+3 == Max exactly — the
	// old check accepted it; C (stored text already ends in a break) skips it.
	edgeLen := MaxMessageLength - 3 - len(first)
	bs.AppendBoardLine(magic, ch, strings.Repeat("b", edgeLen))
	if !strings.Contains(ch.lastMessage(), "String too long.  Last line skipped.") {
		t.Fatalf("edge line (old-check-exact) appended: last reply = %q, want C's skip reply", ch.lastMessage())
	}
	if strings.Contains(bs.msgStorage[bs.writerSlots["Edge"]], "bbb") {
		t.Fatal("2-byte-window line was stored")
	}

	// Two bytes smaller (the separator allowance) must still be accepted —
	// and an accepted line sends no reply, so assert the message count is
	// unchanged rather than re-reading the stale skip reply.
	ch.mu.Lock()
	before := len(ch.messages)
	ch.mu.Unlock()
	bs.AppendBoardLine(magic, ch, strings.Repeat("c", edgeLen-2))
	ch.mu.Lock()
	after := len(ch.messages)
	ch.mu.Unlock()
	if after != before {
		t.Fatalf("under-edge line produced a reply (%d -> %d): %q", before, after, ch.lastMessage())
	}
	if got := len(bs.msgStorage[bs.writerSlots["Edge"]]); got > MaxMessageLength {
		t.Fatalf("stored %d bytes, want <= %d", got, MaxMessageLength)
	}
	if !strings.Contains(bs.msgStorage[bs.writerSlots["Edge"]], "ccc") {
		t.Fatal("under-edge line was not stored")
	}
}

// Every live speech path strips terminal control characters; the board
// editor was the one stored-text path without the rule, giving every board
// reader persistent terminal-escape injection (VULN-013).
func TestBoardTextStripsTerminalControls(t *testing.T) {
	bs := InitBoards(t.TempDir())
	magic, ch := writeStart(t, bs, "Esc", 0)

	bs.AppendBoardLine(magic, ch, "clean\x1b[2Jline")
	if strings.Contains(bs.msgStorage[bs.writerSlots["Esc"]], "\x1b") {
		t.Fatal("ESC byte survived into board storage")
	}
	if !strings.Contains(bs.msgStorage[bs.writerSlots["Esc"]], "clean") {
		t.Fatal("printable text lost by the strip")
	}
}
