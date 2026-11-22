package game

import (
	"testing"
)

// TestDoWriteMissingObjectRefusals pins do_write's parsing and its two
// missing-object refusals (src/act.comm.c:1046-1062): two_arguments makes the
// FIRST token the paper and the SECOND the pen, so "write nothing author"
// reports the missing paper name, and a one-argument "write missing" reports
// the inventory miss rather than the empty-argument message (DP-1516 item 7).
func TestDoWriteMissingObjectRefusals(t *testing.T) {
	w, ch, lastMsg := newDonateTestWorld(t)

	w.DoWrite(ch, "")
	if got, want := lastMsg(), "Write?  With what?  ON what?  What are you trying to do?!?\r\n"; got != want {
		t.Fatalf("empty write = %q, want %q", got, want)
	}

	w.DoWrite(ch, "missing")
	if got, want := lastMsg(), "There is no missing in your inventory.\r\n"; got != want {
		t.Fatalf("write missing = %q, want %q", got, want)
	}

	w.DoWrite(ch, "nothing author")
	if got, want := lastMsg(), "You have no nothing.\r\n"; got != want {
		t.Fatalf("write nothing author = %q, want %q", got, want)
	}
}
