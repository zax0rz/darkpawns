package boards

import (
	"strings"
	"testing"
)

// TestDisplayRemoveMsgParseLikeCAtoi pins C's Board_display_msg /
// Board_remove_msg number parse (src/boards.c:330-332, 378-381): a
// digit-leading token with junk parses ("1junk" is message 1) and a huge
// token wraps negative through (int)strtol into "That message exists only in
// your imagination."; only a non-digit first byte or an atoi of 0 falls
// through (returns false). Fails if the sites go back to strconv.Atoi (R5h).
func TestDisplayRemoveMsgParseLikeCAtoi(t *testing.T) {
	bs := InitBoards(t.TempDir())
	magic, ch := writeStart(t, bs, "Poster", 0)
	bs.AppendBoardLine(magic, ch, "board body")
	bs.FinalizeBoardWrite(magic, ch)

	// Digit-leading junk reads message 1, like C's atoi("1junk").
	if !bs.DisplayMsg(0, ch, "1junk") {
		t.Fatal(`DisplayMsg("1junk") = false, want message 1 read`)
	}
	if !strings.Contains(ch.lastMessage(), "Message 1 :") {
		t.Fatalf("DisplayMsg(1junk) reply = %q, want the message body", ch.lastMessage())
	}

	// A huge token wraps negative: C answers "That message exists only in
	// your imagination." rather than falling through.
	if !bs.DisplayMsg(0, ch, "99999999999999999999") {
		t.Fatal(`DisplayMsg(20-digit) = false, want the imagination arm`)
	}
	if !strings.Contains(ch.lastMessage(), "That message exists only in your imagination.") {
		t.Fatalf("DisplayMsg(20-digit) reply = %q, want the imagination arm", ch.lastMessage())
	}
	if !bs.RemoveMsg(0, ch, "99999999999999999999") {
		t.Fatal(`RemoveMsg(20-digit) = false, want the imagination arm`)
	}
	if !strings.Contains(ch.lastMessage(), "That message exists only in your imagination.") {
		t.Fatalf("RemoveMsg(20-digit) reply = %q, want the imagination arm", ch.lastMessage())
	}

	// Digit-leading junk removes message 1, like C's atoi("1junk").
	if !bs.RemoveMsg(0, ch, "1junk") {
		t.Fatal(`RemoveMsg("1junk") = false, want message 1 removed`)
	}
	if !strings.Contains(ch.lastMessage(), "Message removed.") {
		t.Fatalf("RemoveMsg(1junk) reply = %q, want C's removal ack", ch.lastMessage())
	}

	// Non-digit first byte falls through to the caller on both verbs.
	if bs.DisplayMsg(0, ch, "abc") {
		t.Fatal(`DisplayMsg("abc") = true, want false fallthrough`)
	}
	if bs.RemoveMsg(0, ch, "abc") {
		t.Fatal(`RemoveMsg("abc") = true, want false fallthrough`)
	}
}
