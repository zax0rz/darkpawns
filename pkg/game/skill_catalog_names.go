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
// entries use their lower-cased catalog name directly. These C catalog entries
// differ from their command gameplay keys: the Ninja escape skill is displayed
// as "escape of the mongoose", while do_retreat reads SkillEscape; kuji-kiri
// Jin is displayed as "kuji-kiri jin", while do_kuji_kiri reads SkillKkJin.
// Keep those translations at the catalog boundary so skillset, practice, and
// character bootstrap all address the same C skill slot (R1/R5e).
func SkillStorageName(num int) string {
	name := strings.ToLower(SkillCatalogName(num))
	switch name {
	case "escape of the mongoose":
		return SkillEscape
	case "kuji-kiri jin":
		return SkillKkJin
	}
	return name
}

func skillCatalogSize() int {
	return spells.SkillCatalogSize()
}
