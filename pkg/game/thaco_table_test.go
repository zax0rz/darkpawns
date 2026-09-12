package game

import "testing"

// thacoBeforeKeying is the complete numeric fixture captured from the Go
// tables at origin/main 2ea3474002e5f055395ac9d90251e22bb84eabf6 before this
// refactor. It is independently cross-checked against src/class.c:297-371;
// it does not use production class constants or read the production table.
var thacoBeforeKeying = [12][41]int{
	/* MAGE */ {100, 20, 20, 20, 19, 19, 19, 18, 18, 18, 17, 17, 17, 16, 16, 16, 15, 15, 15, 14, 14, 14, 13, 13, 13, 12, 12, 12, 11, 11, 11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
	/* CLERIC */ {100, 20, 20, 20, 18, 18, 18, 16, 16, 16, 14, 14, 14, 12, 12, 12, 10, 10, 10, 8, 8, 8, 6, 6, 6, 4, 4, 4, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	/* THIEF */ {100, 20, 20, 19, 19, 18, 18, 17, 17, 16, 16, 15, 15, 14, 13, 13, 12, 12, 11, 11, 10, 10, 9, 9, 8, 8, 7, 7, 6, 6, 5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	/* WARRIOR */ {100, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	/* MAGUS */ {100, 20, 20, 20, 19, 19, 19, 18, 18, 18, 17, 17, 17, 16, 16, 16, 15, 15, 15, 14, 14, 14, 13, 13, 13, 12, 12, 12, 11, 11, 11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
	/* AVATAR */ {100, 20, 20, 20, 18, 18, 18, 16, 16, 16, 14, 14, 14, 12, 12, 12, 10, 10, 10, 8, 8, 8, 6, 6, 6, 4, 4, 4, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	/* ASSASSIN */ {100, 20, 20, 19, 19, 18, 18, 17, 17, 16, 16, 15, 15, 14, 13, 13, 12, 12, 11, 11, 10, 10, 9, 9, 8, 8, 7, 7, 6, 6, 5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	/* PALADIN */ {100, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	/* NINJA */ {100, 20, 20, 19, 19, 18, 18, 17, 17, 16, 16, 15, 15, 14, 13, 13, 12, 12, 11, 11, 10, 10, 9, 9, 8, 8, 7, 7, 6, 6, 5, 5, 4, 4, 3, 3, 3, 3, 3, 3, 3},
	/* PSIONIC */ {100, 20, 20, 19, 18, 18, 17, 16, 16, 16, 15, 15, 14, 14, 14, 13, 12, 12, 10, 10, 9, 9, 8, 8, 7, 7, 6, 5, 5, 4, 4, 3, 3, 3, 2, 2, 1, 1, 1, 1, 1},
	/* RANGER */ {100, 20, 19, 18, 17, 16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
	/* MYSTIC */ {100, 20, 20, 20, 19, 19, 19, 18, 18, 18, 17, 17, 17, 16, 16, 16, 15, 15, 15, 14, 14, 14, 13, 13, 13, 12, 12, 12, 11, 11, 11, 10, 10, 10, 9, 9, 9, 9, 9, 9, 9},
}

var thacoClassIDs = [...]int{
	ClassMageUser, ClassCleric, ClassThief, ClassWarrior,
	ClassMagus, ClassAvatar, ClassAssassin, ClassPaladin,
	ClassNinja, ClassPsionic, ClassRanger, ClassMystic,
}

func TestTHAC0TablePreservesEveryCell(t *testing.T) {
	for class := range thacoBeforeKeying {
		for level := range thacoBeforeKeying[class] {
			if got, want := thaco[class][level], thacoBeforeKeying[class][level]; got != want {
				t.Errorf("thaco[%d][%d] = %d, want %d", class, level, got, want)
			}
		}
	}
}

func TestTHAC0ClassIDsMatchCOrder(t *testing.T) {
	for cClass, goClass := range thacoClassIDs {
		if goClass != cClass {
			t.Errorf("Go class identifier %d = %d, want C class index %d", cClass, goClass, cClass)
		}
	}
}

func TestNewCharacterTHAC0ReaderPreservesLevelOneLookup(t *testing.T) {
	for class, wantRow := range thacoBeforeKeying {
		p := RestoreCharacterWithStats(class+1, "THAC0", class, RaceHuman, CharStats{})
		if got, want := p.THAC0, wantRow[1]; got != want {
			t.Errorf("class %d level 1 THAC0 = %d, want %d", class, got, want)
		}
	}
	for _, class := range []int{-1, 12} {
		p := RestoreCharacterWithStats(100+class, "THAC0", class, RaceHuman, CharStats{})
		if p.THAC0 != 20 {
			t.Errorf("invalid class %d THAC0 = %d, want constructor default 20", class, p.THAC0)
		}
	}
}
