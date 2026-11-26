package session

import (
	"testing"
)

func TestMlistAtoiMatchesCDecimalPrefix(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  int
	}{
		{input: "", want: 0},
		{input: "abc", want: 0},
		{input: "183abc", want: 183},
		{input: "+183", want: 183},
		{input: "-1", want: -1},
	} {
		if got := mlistAtoi(tc.input); got != tc.want {
			t.Errorf("mlistAtoi(%q) = %d, want %d", tc.input, got, tc.want)
		}
	}
}
