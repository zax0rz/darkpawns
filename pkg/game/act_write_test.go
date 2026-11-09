package game

import (
	"strings"
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
	if got := lastMsg(); !strings.Contains(got, "Write?  With what?  ON what?  What are you trying to do?!?") {
		t.Fatalf("empty write = %q", got)
	}

	w.DoWrite(ch, "missing")
	if got := lastMsg(); !strings.Contains(got, "There is no missing in your inventory.") {
		t.Fatalf("write missing = %q, want the inventory-miss line", got)
	}

	w.DoWrite(ch, "nothing author")
	got := lastMsg()
	if !strings.Contains(got, "You have no nothing.") {
		t.Fatalf("write nothing author = %q, want the missing paper name", got)
	}
	if strings.Contains(got, "author") {
		t.Fatalf("write nothing author = %q reported the pen name, not the paper name", got)
	}
}
