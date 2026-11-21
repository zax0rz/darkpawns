package common

import (
	"testing"
	"time"
)

// C asctime pads the day with a space (%3d), so days 1-9 carry two spaces
// after the month. Go's "Jan 2" layout drops that pad.
func TestAsctimePadsSingleDigitDay(t *testing.T) {
	tests := []struct {
		day  int
		want string
	}{
		{5, "Thu Nov  5 07:08:09 2026"},
		{15, "Sun Nov 15 07:08:09 2026"},
	}
	for _, tt := range tests {
		got := Asctime(time.Date(2026, time.November, tt.day, 7, 8, 9, 0, time.UTC))
		if got != tt.want {
			t.Errorf("Asctime(day %d) = %q, want %q", tt.day, got, tt.want)
		}
	}
}
