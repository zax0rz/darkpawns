package game

import "testing"

// The generated table must match Dark Pawns' src/spell_parser.c spells[] by number
// (the same numbering classSpells / spells.h uses). Spot-check the anchors the
// practice catalog depends on + the unused/reserved masking.
func TestSkillCatalogName(t *testing.T) {
	cases := map[int]string{
		1:   "holy ward",     // DP-custom slot 1 (NOT stock "armor")
		132: "bash",          // SKILL_BASH
		134: "kick",          // SKILL_KICK — the L1 warrior catalog entry
		171: "berserk",       // SKILL_BERSERK
		147: "charge",        // SKILL_CHARGE
		5:   "burning hands", // an offensive spell
	}
	for num, want := range cases {
		if got := SkillCatalogName(num); got != want {
			t.Errorf("SkillCatalogName(%d) = %q, want %q", num, got, want)
		}
	}
	// Reserved/unused/out-of-range slots resolve to "".
	if got := SkillCatalogName(0); got != "" {
		t.Errorf("SkillCatalogName(0 reserved) = %q, want \"\"", got)
	}
	if got := SkillCatalogName(9999); got != "" {
		t.Errorf("SkillCatalogName(out-of-range) = %q, want \"\"", got)
	}
}
