package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestZresetZoneIndexUsesCZoneTablePosition(t *testing.T) {
	zones := []*parser.Zone{
		{Number: 0},
		{Number: 12},
		{Number: 80},
	}

	tests := []struct {
		name   string
		number int
		want   int
	}{
		{name: "first zone", number: 0, want: 0},
		{name: "sparse zone", number: 12, want: 1},
		{name: "last zone", number: 80, want: 2},
		{name: "missing zone", number: 999, want: -1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := zresetZoneIndex(zones, tt.number); got != tt.want {
				t.Fatalf("zresetZoneIndex(%d) = %d, want %d", tt.number, got, tt.want)
			}
		})
	}
}
