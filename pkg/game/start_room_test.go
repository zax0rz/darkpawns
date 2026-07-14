package game

import "testing"

func TestNewbieHometownRoom(t *testing.T) {
	tests := []struct {
		name     string
		hometown int
		want     int
	}{
		{name: "Kir Drax'in", hometown: 1, want: 8162},
		{name: "Kir-Oshi", hometown: 2, want: 18201},
		{name: "Alaozar", hometown: 3, want: 21202},
		{name: "unset", hometown: 0, want: MortalStartRoom},
		{name: "unknown", hometown: 99, want: MortalStartRoom},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewbieHometownRoom(tt.hometown); got != tt.want {
				t.Errorf("NewbieHometownRoom(%d) = %d, want %d", tt.hometown, got, tt.want)
			}
		})
	}
}
