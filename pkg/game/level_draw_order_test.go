package game

import (
	"fmt"
	"slices"
	"testing"
)

type levelDraw struct {
	from int
	to   int
}

func TestAdvanceLevelDrawOrderByClass(t *testing.T) {
	t.Chdir(t.TempDir())
	oldNumber := levelNumber
	t.Cleanup(func() { levelNumber = oldNumber })

	const level = 1
	tests := []struct {
		class int
		want  []levelDraw
	}{
		{ClassMageUser, []levelDraw{{4, 8}, {level, 3 * level}, {1, 3}}},
		{ClassCleric, []levelDraw{{5, 9}, {level, 3 * level}, {1, 3}}},
		{ClassThief, []levelDraw{{7, 13}, {1, 4}}},
		{ClassWarrior, []levelDraw{{11, 14}, {1, 4}}},
		{ClassMagus, []levelDraw{{5, 9}, {level, 3 * level}, {1, 3}}},
		{ClassAvatar, []levelDraw{{6, 11}, {level, 3 * level}, {1, 3}}},
		{ClassAssassin, []levelDraw{{8, 14}, {level, 2 * level}, {1, 4}}},
		{ClassPaladin, []levelDraw{{level, 2 * level}, {12, 16}, {1, 4}}},
		{ClassNinja, []levelDraw{{8, 13}, {level, 2 * level}, {1, 4}}},
		{ClassPsionic, []levelDraw{{4, 8}, {level, 2 * level}, {1, 4}}},
		{ClassRanger, []levelDraw{{13, 16}, {2, 4}}},
		{ClassMystic, []levelDraw{{5, 9}, {level, 2 * level}, {1, 4}}},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("class_%d", test.class), func(t *testing.T) {
			var got []levelDraw
			levelNumber = func(from, to int) int {
				got = append(got, levelDraw{from: from, to: to})
				return from
			}

			player := NewPlayer(test.class+1, fmt.Sprintf("Class%d", test.class), MortalStartRoom)
			player.Class = test.class
			player.Level = level
			player.Stats = CharStats{Con: 10, Wis: 10}
			player.MaxHealth = 10
			player.MaxMana = 100
			player.MaxMove = 82
			player.AdvanceLevel()

			if !slices.Equal(got, test.want) {
				t.Fatalf("draws = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestAdvanceLevelWarriorMovementUsesSecondDraw(t *testing.T) {
	t.Chdir(t.TempDir())
	oldNumber := levelNumber
	t.Cleanup(func() { levelNumber = oldNumber })

	returns := []int{11, 3}
	levelNumber = func(from, to int) int {
		value := returns[0]
		returns = returns[1:]
		if value < from || value > to {
			t.Fatalf("scripted value %d outside number(%d,%d)", value, from, to)
		}
		return value
	}

	player := NewPlayer(1, "Warrior", MortalStartRoom)
	player.Class = ClassWarrior
	player.Level = 1
	player.Stats = CharStats{Con: 10, Wis: 10}
	player.MaxHealth = 10
	player.MaxMana = 100
	player.MaxMove = 82
	player.AdvanceLevel()

	if player.MaxMove != 85 {
		t.Fatalf("warrior max move = %d, want base 82 + movement draw 3", player.MaxMove)
	}
	if len(returns) != 0 {
		t.Fatalf("unused scripted draws: %v", returns)
	}
}

func TestNewCharacterWithStatsPreservesAcceptedModel(t *testing.T) {
	t.Chdir(t.TempDir())
	oldNumber := levelNumber
	t.Cleanup(func() { levelNumber = oldNumber })
	levelNumber = func(from, to int) int { return from }

	wantStats := CharStats{Str: 17, Int: 9, Wis: 10, Dex: 15, Con: 14, Cha: 8}
	player := NewCharacterWithStats(1, "Newbie", ClassWarrior, RaceKender, 1, wantStats)

	if player.Stats != wantStats {
		t.Fatalf("accepted stats replaced: got %+v, want %+v", player.Stats, wantStats)
	}
	if player.Sex != 1 {
		t.Fatalf("sex = %d, want female", player.Sex)
	}
	if player.AC != 100 {
		t.Fatalf("newbie AC = %d, want C base 100", player.AC)
	}
}
