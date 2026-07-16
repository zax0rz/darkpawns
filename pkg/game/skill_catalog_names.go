package game

import "github.com/zax0rz/darkpawns/pkg/spells"

// SkillCatalogName returns the canonical Dark Pawns display name for a
// skill/spell number. The table lives in pkg/spells so casting and practice
// cannot drift onto different name-by-number mappings.
func SkillCatalogName(num int) string {
	return spells.GetSpellName(num)
}

func skillCatalogSize() int {
	return spells.SkillCatalogSize()
}
