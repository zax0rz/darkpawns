package game

import "testing"

func TestCByteUsesSignedPlainCharConversion(t *testing.T) {
	for _, tc := range []struct {
		in   int
		want int
	}{
		{in: -1, want: -1},
		{in: 127, want: 127},
		{in: 128, want: -128},
		{in: 255, want: -1},
		{in: 256, want: 0},
	} {
		if got := cByte(tc.in); got != tc.want {
			t.Errorf("cByte(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}
