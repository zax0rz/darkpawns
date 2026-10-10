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
