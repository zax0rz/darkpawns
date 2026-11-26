package session

import (
	"testing"
)

func TestParseLinesSizeMirrorsCAtoi(t *testing.T) {
	tests := map[string]int{
		"foo":   0,
		"25abc": 25,
		"-8":    -8,
		"+8":    8,
		"007":   7,
	}
	for input, want := range tests {
		if got := parseLinesSize(input); got != want {
			t.Errorf("parseLinesSize(%q) = %d, want %d", input, got, want)
		}
	}
}
