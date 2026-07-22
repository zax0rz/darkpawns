package session

import (
	"fmt"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestFindExp(t *testing.T) {
	tests := []struct {
		class int
		level int
		want  int
	}{
		{class: 0, level: -1, want: 1},
		{class: 0, level: 0, want: 1},
		{class: 0, level: 1, want: 1500},
		{class: 0, level: 2, want: 3000},
		{class: 0, level: 3, want: 6000},
		{class: 0, level: 4, want: 11000},
		{class: 0, level: 5, want: 21000},
		{class: 0, level: 6, want: 42000},
		{class: 0, level: 7, want: 80000},
		{class: 0, level: 8, want: 155000},
		{class: 0, level: 9, want: 300000},
		{class: 0, level: 10, want: 450000},
		{class: 0, level: 11, want: 650000},
		{class: 0, level: 12, want: 870000},
		{class: game.ClassMageUser, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(0.3*10000*20)},
		{class: game.ClassCleric, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(0.4*10000*20)},
		{class: game.ClassWarrior, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(0.7*10000*20)},
		{class: game.ClassThief, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(0.1*10000*20)},
		{class: game.ClassMagus, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(1.5*10000*20)},
		{class: game.ClassMystic, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(1.5*10000*20)},
		{class: game.ClassAvatar, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(1.6*10000*20)},
		{class: game.ClassAssassin, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(1.2*10000*20)},
		{class: game.ClassPaladin, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(1.9*10000*20)},
		{class: game.ClassRanger, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(1.9*10000*20)},
		{class: game.ClassNinja, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(0.6*10000*20)},
		{class: game.ClassPsionic, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(0.6*10000*20)},
		{class: 999, level: 20, want: 900000 + ((20 - 13) * 20 * 20000) + (20 * 20 * 1000) + int(1.0*10000*20)},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("class%d_lvl%d", tt.class, tt.level), func(t *testing.T) {
			if got := findExp(tt.class, tt.level); got != tt.want {
				t.Errorf("findExp(%d, %d) = %d, want %d", tt.class, tt.level, got, tt.want)
			}
		})
	}
}
