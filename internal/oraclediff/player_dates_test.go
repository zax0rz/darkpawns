package oraclediff

import (
	"strings"
	"testing"
)

func TestPlayerReportDatesNormalizeOnlyClockFields(t *testing.T) {
	a := "Player: Saver (Male) [40 Wa]\r\nStarted: Sun Oct  4 22:59      Last: Sun Oct  4 23:00      Played:   3h 47m\r\n"
	b := "Player: Saver (Male) [40 Wa]\r\nStarted: Mon Oct  5 00:00      Last: Mon Oct  5 00:01      Played:   3h 47m\r\n"
	got := Normalize(a)
	if got != Normalize(b) {
		t.Fatalf("clock boundary left unnormalized: %q vs %q", got, Normalize(b))
	}
	for _, replacement := range []string{"Played:   4h 47m", "Played:   3h 48m", "Played:    3h 47m"} {
		changed := strings.Replace(b, "Played:   3h 47m", replacement, 1)
		if got == Normalize(changed) {
			t.Fatalf("hid played bytes: %q", replacement)
		}
	}
	text := "say Started: Sun Oct  4 22:59      Last: Sun Oct  4 23:00      Played:   3h 47m"
	if strings.Contains(Normalize(text), "<WALL_CLOCK>") {
		t.Fatal("normalized ordinary text")
	}
}

// C prints "%sStarted: ..." (act.wizard.c:2344): on colored surfaces the %s
// is a color code, and KeepANSI mode certifies those bytes raw. The anchor
// must normalize the volatile dates behind such a prefix and keep the prefix
// itself — the old ^Started: anchor missed the whole line.
func TestPlayerReportDatesAnchorPrefixes(t *testing.T) {
	colored := "\x1b[1;37mStarted: Sun Oct  4 22:59      Last: Sun Oct  4 23:00      Played:   3h 47m\r\n"
	shifted := "\x1b[1;37mStarted: Mon Oct  5 00:00      Last: Mon Oct  5 00:01      Played:   3h 47m\r\n"
	got := NormalizeKeepANSI(colored)
	if got != NormalizeKeepANSI(shifted) {
		t.Fatalf("colored clock boundary left unnormalized: %q vs %q", got, NormalizeKeepANSI(shifted))
	}
	if !strings.Contains(got, "\x1b[1;37mStarted: <WALL_CLOCK>") {
		t.Fatalf("prefix not preserved with dates masked: %q", got)
	}

	indented := "  Started: Sun Oct  4 22:59      Last: Sun Oct  4 23:00      Played:   3h 47m\r\n"
	if other := strings.ReplaceAll(indented, "Oct  4 22:59", "Oct  5 00:00"); Normalize(indented) != Normalize(other) {
		t.Fatal("indented anchor line left unnormalized")
	}
}

// show-player-stored runs the keep-prompts pipeline
// (cmd/dp-oracle-diff/scenarios/show-player-stored.txt, [fixture]
// keep-prompts) and diverged on the first [show player] block whenever the
// two engines' setups straddled a wall-clock minute: that pipeline had no
// playerSavedDates pass, so the Started/Last dates were compared raw
// (dp-1404-rent-code-claims 2026-10-09; pr1904-dead-code-d 2026-10-10
// seed 5 — C started 14:56, Go 14:55). Reproduces that first block in
// keep-prompts mode: the captures differ in the Started minute and must
// normalize to identical bytes, with the captured \r\n framing preserved.
func TestPlayerReportDatesNormalizeKeepPrompts(t *testing.T) {
	c := "Player: Shopreport   (Male) [40 Wa]\r\n" +
		"Au: 0         Bal: 0         Exp: 7000000   Align: 0      Lessons: 0  \r\n" +
		"Started: Sat Oct 10 14:56      Last: Sat Oct 10 14:56      Played:   0h  0m\r\n"
	g := "Player: Shopreport   (Male) [40 Wa]\r\n" +
		"Au: 0         Bal: 0         Exp: 7000000   Align: 0      Lessons: 0  \r\n" +
		"Started: Sat Oct 10 14:55      Last: Sat Oct 10 14:56      Played:   0h  0m\r\n"
	gotC, gotG := NormalizeKeepPrompts(c), NormalizeKeepPrompts(g)
	if gotC != gotG {
		t.Fatalf("keep-prompts left the Started minute unmasked:\nC : %q\nGo: %q", gotC, gotG)
	}
	if !strings.Contains(gotC, "Started: <WALL_CLOCK>      Last: <WALL_CLOCK>      Played:   0h  0m") {
		t.Fatalf("mask shape wrong or framing not preserved: %q", gotC)
	}
	if !strings.Contains(gotC, "\r\n") {
		t.Fatal("keep-prompts mode must preserve captured \\r\\n framing")
	}

	// Ordinary prose with a leading word still escapes the mask on purpose:
	// the anchor admits only ANSI escapes and whitespace before the label.
	say := "say Started: Sat Oct 10 14:56      Last: Sat Oct 10 14:56      Played:   0h  0m\r\n"
	if strings.Contains(NormalizeKeepPrompts(say), "<WALL_CLOCK>") {
		t.Fatal("masked an ordinary prose line in keep-prompts mode")
	}
}
