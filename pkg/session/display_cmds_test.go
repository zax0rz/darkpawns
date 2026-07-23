package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestFindExp(t *testing.T) {
	tests := []struct {
		name  string
		class int
		level int
		want  int
	}{
		// Class modifiers at default level 13: base(900000) + 0 + 169000 + int(modifier * 130000)
		{name: "mage level 13", class: game.ClassMageUser, level: 13, want: 1108000},     // 0.3*130000=39000
		{name: "cleric level 13", class: game.ClassCleric, level: 13, want: 1121000},     // 0.4*130000=52000
		{name: "warrior level 13", class: game.ClassWarrior, level: 13, want: 1160000},   // 0.7*130000=91000
		{name: "thief level 13", class: game.ClassThief, level: 13, want: 1082000},       // 0.1*130000=13000
		{name: "magus level 13", class: game.ClassMagus, level: 13, want: 1264000},       // 1.5*130000=195000
		{name: "mystic level 13", class: game.ClassMystic, level: 13, want: 1264000},     // 1.5*130000=195000
		{name: "avatar level 13", class: game.ClassAvatar, level: 13, want: 1277000},     // 1.6*130000=208000
		{name: "assassin level 13", class: game.ClassAssassin, level: 13, want: 1225000}, // 1.2*130000=156000
		{name: "paladin level 13", class: game.ClassPaladin, level: 13, want: 1316000},   // 1.9*130000=247000
		{name: "ranger level 13", class: game.ClassRanger, level: 13, want: 1316000},     // 1.9*130000=247000
		{name: "ninja level 13", class: game.ClassNinja, level: 13, want: 1147000},       // 0.6*130000=78000
		{name: "psionic level 13", class: game.ClassPsionic, level: 13, want: 1147000},   // 0.6*130000=78000
		{name: "default level 13", class: 99, level: 13, want: 1199000},                  // 1.0*130000=130000

		// Level <= 0 always returns 1
		{name: "level 0", class: game.ClassWarrior, level: 0, want: 1},
		{name: "level -1", class: game.ClassWarrior, level: -1, want: 1},

		// Hardcoded levels
		{name: "level 1", class: game.ClassWarrior, level: 1, want: 1500},
		{name: "level 2", class: game.ClassWarrior, level: 2, want: 3000},
		{name: "level 3", class: game.ClassWarrior, level: 3, want: 6000},
		{name: "level 4", class: game.ClassWarrior, level: 4, want: 11000},
		{name: "level 5", class: game.ClassWarrior, level: 5, want: 21000},
		{name: "level 6", class: game.ClassWarrior, level: 6, want: 42000},
		{name: "level 7", class: game.ClassWarrior, level: 7, want: 80000},
		{name: "level 8", class: game.ClassWarrior, level: 8, want: 155000},
		{name: "level 9", class: game.ClassWarrior, level: 9, want: 300000},
		{name: "level 10", class: game.ClassWarrior, level: 10, want: 450000},
		{name: "level 11", class: game.ClassWarrior, level: 11, want: 650000},
		{name: "level 12", class: game.ClassWarrior, level: 12, want: 870000},

		// Level 20 — formula: 900000 + ((20-13) * 20 * 20000) + (20 * 20 * 1000) + modifier*10000*20
		// 900000 + (7*20*20000) + (400*1000) + (0.7*10000*20)
		// 900000 + 2800000 + 400000 + 140000 = 4240000
		{name: "warrior level 20", class: game.ClassWarrior, level: 20, want: 4240000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findExp(tt.class, tt.level); got != tt.want {
				t.Errorf("findExp(%d, %d) = %d, want %d", tt.class, tt.level, got, tt.want)
			}
		})
	}
}
