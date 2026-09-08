package game

import "testing"

// Pin InvalidClass anti-class bits to the C ITEM_ANTI_* positions in
// src/structs.h:480-495 (see Linear DP-1257). World data and save files
// store C-numbered extra-flag bits; drift here silently inverts every
// class-restriction gate.
func TestInvalidClassAntiClassBitsMatchCStructs(t *testing.T) {
	// Class order: character.go:13-24
	classes := []int{
		ClassMageUser, ClassCleric, ClassThief, ClassWarrior,
		ClassMagus, ClassAvatar, ClassAssassin, ClassPaladin,
		ClassNinja, ClassPsionic, ClassRanger, ClassMystic,
	}
	// C ITEM_ANTI_* bit positions: src/structs.h:480-495
	cAntiBits := []int{12, 13, 14, 15, 21, 23, 22, 20, 19, 18, 26, 27}

	for i, class := range classes {
		bits := uint32(1) << cAntiBits[i]
		if !InvalidClass(class, bits, false, false) {
			t.Errorf("class %d: InvalidClass false with C ANTI bit %d set", class, cAntiBits[i])
		}
		// No other single anti-class bit should block this class.
		for j, other := range cAntiBits {
			if j == i {
				continue
			}
			if InvalidClass(class, uint32(1)<<other, false, false) {
				t.Errorf("class %d: InvalidClass true with only class-%d ANTI bit %d set", class, classes[j], other)
			}
		}
	}
}
