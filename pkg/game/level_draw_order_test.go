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

// TestNewCharacterConstructorConsumesZeroLevelDraws — DP-1212 regression: the
// shared constructor (newCharacter → NewCharacterWithStats/NewCharacter) must
// NOT call AdvanceLevel, because that consumed 2 phantom draws on the God path
// (C skips do_start for an already-leveled God — interpreter.c:2214). The
// constructor now sets base stats only; AdvanceLevel is the caller's job.
func TestNewCharacterConstructorConsumesZeroLevelDraws(t *testing.T) {
	t.Chdir(t.TempDir())
	oldNumber := levelNumber
	t.Cleanup(func() { levelNumber = oldNumber })

	var got []levelDraw
	levelNumber = func(from, to int) int {
		got = append(got, levelDraw{from: from, to: to})
		return from
	}

	// Construct a warrior — previously this drew number(11,14)+number(1,4); now
	// it draws nothing.
	p := NewCharacterWithStats(1, "God", ClassWarrior, RaceHuman, 0, CharStats{Str: 15, Con: 14, Wis: 10, Dex: 12, Int: 10, Cha: 9})
	if len(got) != 0 {
		t.Fatalf("constructor consumed %d level-draws (%+v), want 0 — AdvanceLevel must not run in the constructor (DP-1212)", len(got), got)
	}
	// The base stats stand (no level-1 bonus applied).
	if p.MaxHealth != 10 {
		t.Errorf("constructor MaxHealth = %d, want 10 (base only; no AdvanceLevel)", p.MaxHealth)
	}
}

// TestBootstrapFirstPlayerGodConsumesZeroLevelDraws — the God bootstrap
// (init_char first-player block) is pure assignment: level 40, max_hit 500,
// every skill 100, conditions -1. It must draw nothing (R3 — matches C's
// skipped do_start for the already-LVL_IMPL God).
func TestBootstrapFirstPlayerGodConsumesZeroLevelDraws(t *testing.T) {
	t.Chdir(t.TempDir())
	oldNumber := levelNumber
	t.Cleanup(func() { levelNumber = oldNumber })

	var got []levelDraw
	levelNumber = func(from, to int) int {
		got = append(got, levelDraw{from: from, to: to})
		return from
	}

	p := NewCharacterWithStats(1, "FirstGod", ClassWarrior, RaceHuman, 0, CharStats{Str: 15, Con: 14, Wis: 10, Dex: 12, Int: 10, Cha: 9})
	BootstrapFirstPlayerGod(p)

	if len(got) != 0 {
		t.Fatalf("BootstrapFirstPlayerGod consumed %d level-draws (%+v), want 0 (pure assignment, DP-1212)", len(got), got)
	}
	if p.Level != LVL_IMPL || p.GetMaxHP() != 500 {
		t.Errorf("God stats wrong: level=%d maxHP=%d, want 40/500", p.Level, p.GetMaxHP())
	}
}

// TestGodThenMortalCreationDrawSequence — the regression guard for the whole
// DP-1212 class: a God (0 draws) then a mortal warrior (2 draws: number(11,14)
// then number(1,4)) = 2 total, not 4. This is the stream position C expects.
func TestGodThenMortalCreationDrawSequence(t *testing.T) {
	t.Chdir(t.TempDir())
	oldNumber := levelNumber
	t.Cleanup(func() { levelNumber = oldNumber })

	var got []levelDraw
	levelNumber = func(from, to int) int {
		got = append(got, levelDraw{from: from, to: to})
		return from
	}

	// God: constructor (0 draws) + bootstrap (0 draws).
	god := NewCharacterWithStats(1, "FirstGod", ClassWarrior, RaceHuman, 0, CharStats{Str: 15, Con: 14, Wis: 10, Dex: 12, Int: 10, Cha: 9})
	BootstrapFirstPlayerGod(god)
	if len(got) != 0 {
		t.Fatalf("God creation drew %d, want 0 (DP-1212)", len(got))
	}

	// Mortal: constructor (0 draws) + explicit AdvanceLevel (2 draws).
	mortal := NewCharacterWithStats(2, "Hero", ClassWarrior, RaceHuman, 0, CharStats{Str: 15, Con: 14, Wis: 10, Dex: 12, Int: 10, Cha: 9})
	mortal.AdvanceLevel()

	want := []levelDraw{{11, 14}, {1, 4}} // warrior: HP then move
	if !slices.Equal(got, want) {
		t.Fatalf("God+mortal draw sequence = %+v, want %+v (God 0 + mortal 2 = 2 total, not 4)", got, want)
	}
}
