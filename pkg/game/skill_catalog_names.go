package game

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/spells"
)

// SkillCatalogName returns the canonical Dark Pawns display name for a
// skill/spell number. The table lives in pkg/spells so casting and practice
// cannot drift onto different name-by-number mappings.
func SkillCatalogName(num int) string {
	return spells.GetSpellName(num)
}

// SkillStorageName returns the Go skill key for a C spells[] entry. Most
// entries use their lower-cased catalog name directly. The Ninja escape skill
// is the one C catalog entry whose display name differs from the command's
// gameplay key: spells[157] is "escape of the mongoose", while do_retreat
// reads SKILL_ESCAPE. Keep that translation at the catalog boundary so
// skillset, practice, and character bootstrap all address the same skill.
func SkillStorageName(num int) string {
	name := strings.ToLower(SkillCatalogName(num))
	if name == "escape of the mongoose" {
		return SkillEscape
	}
	return name
}

func skillCatalogSize() int {
	return spells.SkillCatalogSize()
}
