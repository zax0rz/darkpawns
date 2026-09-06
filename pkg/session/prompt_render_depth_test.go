package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestPromptTextPreservesCFieldOrdering(t *testing.T) {
	tests := []struct {
		name    string
		flags   []int
		invis   int
		infobar int
		want    string
	}{
		{name: "bare", want: "> "},
		{
			name:  "vitals-in-C-order",
			flags: []int{game.PrfDisphp, game.PrfDispmmana, game.PrfDispmove},
			want:  "101H 202M 303V > ",
		},
		{
			name:    "infobar-owns-vitals",
			flags:   []int{game.PrfDisphp, game.PrfDispmmana, game.PrfDispmove},
			infobar: InfobarOn,
			want:    "> ",
		},
		{
			name:  "invis-precedes-vitals",
			flags: []int{game.PrfDisphp, game.PrfDispmmana, game.PrfDispmove},
			invis: 31,
			want:  "i31 101H 202M 303V > ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := makeTestManager(t)
			s := makeCommandTestSession(t, m, "Promptfields", 1, 1001)
			s.player.Health, s.player.Mana, s.player.Move = 101, 202, 303
			s.infobarMode = tt.infobar
			s.player.SetInvisLevel(tt.invis)
			for _, bit := range tt.flags {
				s.player.SetPlrFlag(bit, true)
			}

			if got := s.promptText(); got != tt.want {
				t.Fatalf("prompt = %q, want %q", got, tt.want)
			}
		})
	}
}
