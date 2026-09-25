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

// catalogSkillKeys maps C spells[] display names (lowercased) to the Go skill
// keys the command handlers read, where the two differ. Keeping every
// translation at this one boundary means skillset, practice, remort and the
// first-player God all store the key the handler looks up (R1/R5e). Before
// DP-1342 only the escape and kuji-kiri entries were here and skillset kept a
// private switch for some of the rest, so the God and practice stored
// "tiger punch" where do_tiger_punch reads tiger_punch.
var catalogSkillKeys = map[string]string{
	"escape of the mongoose": SkillEscape,
	"kuji-kiri rin":          SkillKkRin,
	"kuji-kiri kyo":          SkillKkKyo,
	"kuji-kiri toh":          SkillKkToh,
	"kuji-kiri kai":          SkillKkKai,
	"kuji-kiri jin":          SkillKkJin,
	"kuji-kiri retsu":        SkillKkRetsu,
	"kuji-kiri zai":          SkillKkZai,
	"kuji-kiri zhen":         SkillKkZhen,
	"kuji-kiri sha":          SkillKkSha,
	"pick lock":              SkillPickLock,    // SKILL_PICK_LOCK 135
	"strike of revenge":      SkillStrike,      // SKILL_STRIKE 155
	"serpent kick":           SkillSerpentKick, // SKILL_SERPENT_KICK 156
	"flesh alter":            SkillFleshAlter,  // SKILL_FLESH_ALTER 168
	"aid":                    SkillFirstAid,    // SKILL_FIRST_AID 179
	"search":                 SkillDetect,      // SKILL_DETECT 180
	"dragon kick":            SkillDragonKick,  // SKILL_DRAGON_KICK 188
	"tiger punch":            SkillTigerPunch,  // SKILL_TIGER_PUNCH 189
}

// SkillStorageName returns the Go skill key for a C spells[] entry: its
// lowercased catalog name, or the key in catalogSkillKeys.
func SkillStorageName(num int) string {
	name := strings.ToLower(SkillCatalogName(num))
	if key, ok := catalogSkillKeys[name]; ok {
		return key
	}
	return name
}

// canonicalSkillKey maps a saved skill name to the key handlers read. Saves
// written before DP-1342 hold some skills under their catalog display name
// ("tiger punch"); those load under the handler's key.
func canonicalSkillKey(name string) string {
	if key, ok := catalogSkillKeys[name]; ok {
		return key
	}
	return name
}

func skillCatalogSize() int {
	return spells.SkillCatalogSize()
}
