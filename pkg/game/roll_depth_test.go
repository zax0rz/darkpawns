package game

import "testing"

func TestRollMaximumUsesCArgumentParser(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want uint32
	}{
		{name: "empty defaults", arg: "", want: 100},
		{name: "zero defaults", arg: "0", want: 100},
		{name: "malformed defaults", arg: "abc", want: 100},
		{name: "decimal prefix", arg: "6junk", want: 6},
		{name: "fill word skipped", arg: "the 6", want: 6},
		{name: "leading plus", arg: "+6", want: 6},
		{name: "negative converted to unsigned", arg: "-5", want: 4294967291},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rollMaximum(tt.arg); got != tt.want {
				t.Fatalf("rollMaximum(%q) = %d, want %d", tt.arg, got, tt.want)
			}
		})
	}
}

func TestRollNumberUsesCUnsignedConversion(t *testing.T) {
	const maxRoll = uint32(4294967291)
	called := false
	got := rollNumber(maxRoll, func(from, to int) int {
		called = true
		if from != 1 || to != -5 {
			t.Fatalf("number bounds = (%d, %d), want (1, -5)", from, to)
		}
		return -3
	})
	if !called {
		t.Fatal("rollNumber did not draw")
	}
	if got != 4294967293 {
		t.Fatalf("rollNumber result = %d, want 4294967293", got)
	}
}
