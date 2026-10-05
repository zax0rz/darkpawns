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
