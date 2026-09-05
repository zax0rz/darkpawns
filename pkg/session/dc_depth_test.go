package session

import "testing"

func TestParseDCNumberMirrorsCAtoi(t *testing.T) {
	tests := []struct {
		input string
		value int
		ok    bool
	}{
		{input: "42", value: 42, ok: true},
		{input: "42suffix", value: 42, ok: true},
		{input: "+42", value: 42, ok: true},
		{input: "-42", value: -42, ok: true},
		{input: "0", value: 0, ok: true},
		{input: "all", value: 0, ok: false},
	}
	for _, tt := range tests {
		got, ok := parseDCNumber(tt.input)
		if got != tt.value || ok != tt.ok {
			t.Errorf("parseDCNumber(%q) = (%d, %t), want (%d, %t)", tt.input, got, ok, tt.value, tt.ok)
		}
	}
}
