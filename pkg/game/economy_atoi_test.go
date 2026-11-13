package game

import (
	"strings"
	"testing"
)

// TestSplitOverflowRefusesLikeC pins do_split's amount parse
// (src/act.other.c:823-839): the is_number gate passes a 20-digit token and
// atoi wraps it negative, taking "Sorry, you can't do that." — strconv.Atoi's
// error arm instead answered "How many coins do you wish to split with your
// group?". Fails if the site reverts (R5h; DP-1523 economy batch).
func TestSplitOverflowRefusesLikeC(t *testing.T) {
	w, p := newFlagWorld(t)
	var sent []string
	p.entryTextSink = func(msg string) { sent = append(sent, msg) }

	w.doSplit(p, nil, "split", "99999999999999999999")

	got := strings.Join(sent, "")
	if !strings.Contains(got, "Sorry, you can't do that.") {
		t.Fatalf("split(20-digit) output = %q, want C's <= 0 refusal", got)
	}
}

// TestJunkCoinsOverflowMatchesC pins do_drop's coin-form gate
// (src/act.item.c:587-589) and perform_drop_gold's negative arm
// (src/act.item.c:433-434): an overflowing coin count answers "Heh heh heh..
// we are jolly funny today, eh?" — strconv.Atoi instead dropped the token
// into the single-item path.
func TestJunkCoinsOverflowMatchesC(t *testing.T) {
	w, p := newFlagWorld(t)
	var sent []string
	p.entryTextSink = func(msg string) { sent = append(sent, msg) }

	w.doDispose(p, "99999999999999999999 coins", scmdJunk, "junk", 0)

	got := strings.Join(sent, "")
	if !strings.Contains(got, "Heh heh heh.. we are jolly funny today, eh?") {
		t.Fatalf("junk(20-digit coins) output = %q, want C's jolly refusal", got)
	}
}
