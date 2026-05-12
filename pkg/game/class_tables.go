// class_tables.go — static data tables from class.c
// Source: src/class.c
//
// Contains: pc_class_types, menus, prac_params, guild_info, ParseClass,
// InvalidClass, and corrected FindClassBitvector.
// Tables already ported elsewhere (not duplicated here):
//   - ClassAbbrevs, ClassNames      → character.go
//   - Titles                         → limits.go
//   - thaco                          → combat/formulas.go
//   - FindExp                        → limits_exp.go
//   - backstabMult                   → combat/fight_core.go, skill_combat.go
//   - RollRealAbils, DoStart         → character.go
//   - AdvanceLevel                   → level.go
//   - classSpells (init_spell_levels)→ session/spell_level.go

package game

// PCClassTypes is the full display name for each PC class.
// Source: class.c pc_class_types[]
var PCClassTypes = []string{
	"Magic User", // ClassMageUser = 0
	"Cleric",     // ClassCleric   = 1
	"Thief",      // ClassThief    = 2
	"Warrior",    // ClassWarrior  = 3
	"Magus",      // ClassMagus    = 4
	"Avatar",     // ClassAvatar   = 5
	"Assassin",   // ClassAssassin = 6
	"Paladin",    // ClassPaladin  = 7
	"Ninja",      // ClassNinja    = 8
	"Psionic",    // ClassPsionic  = 9
	"Ranger",     // ClassRanger   = 10
	"Mystic",     // ClassMystic   = 11
}

// ClassMenu is the default class selection menu shown during character creation.
// Source: class.c class_menu
const ClassMenu = "\r\n" +
	"Select a class:\r\n" +
	"  [C]leric     - Healers and warriors of the gods\r\n" +
	"  [T]hief      - Stealthy, quick-fingered, lock-picking back-stabbers\r\n" +
	"  [W]arrior    - Fierce, battle-trained fighters\r\n" +
	"  [M]agic-user - Spell-casters trained in the art of magick\r\n" +
	"  Ps[i]onic    - Fighters endowed with the powers of the mind"

// HumanClassMenu is the class selection menu shown for human characters.
// Source: class.c human_class_menu
const HumanClassMenu = "\r\n" +
	"Select a class:\r\n" +
	"  [C]leric     - Healers and warriors of the gods\r\n" +
	"  [T]hief      - Stealthy, quick-fingered, lock-picking back-stabbers\r\n" +
	"  [W]arrior    - Fierce, battle-trained fighters\r\n" +
	"  [M]agic-user - Spell-casters trained in the art of magick\r\n" +
	"  [N]inja      - Stealthy, magick-endowed warriors from the orient\r\n" +
	"  Ps[i]onic    - Fighters endowed with the powers of the mind"

// HometownMenu is the hometown selection menu shown during character creation.
// Source: class.c hometown_menu
const HometownMenu = "\r\n" +
	"Choose your home town:\r\n" +
	"  [K]ir Drax'in  - The Main City. New players should choose this.\r\n" +
	"  Kir-[O]shi     - The Port City.\r\n" +
	"  [A]laozar      - The Holy City.\r\n"

// Practice type constants — from class.c #define SPELL/SKILL/BOTH
const (
	PracTypeSpell = 0 // SPELL
	PracTypeSkill = 1 // SKILL
	PracTypeBoth  = 2 // BOTH
)

// PracParams contains practice parameters for each class.
// [0] = learned level (max % skill before "already learned")
// [1] = max gain per practice
// [2] = min gain per practice
// [3] = practice type (SPELL/SKILL/BOTH — controls "spells" vs "skills" wording)
// Source: class.c prac_params[4][NUM_CLASSES]
var PracParams = [4][12]int{
	// MAG CLE  THE  WAR  MAGU AVA  ASS  PAL  NIN  PSI  RAN  MYS
	{95, 95, 85, 80, 95, 95, 85, 80, 85, 95, 80, 95},  // learned level
	{100, 100, 25, 25, 100, 100, 25, 25, 25, 100, 25, 100}, // max per prac
	{25, 25, 0, 0, 25, 25, 0, 0, 0, 25, 0, 25},       // min per prac
	{PracTypeSpell, PracTypeSpell, PracTypeSkill, PracTypeSkill, PracTypeSpell, PracTypeBoth, PracTypeSkill, PracTypeBoth, PracTypeBoth, PracTypeBoth, PracTypeSkill, PracTypeBoth}, // prac type
}

// GuildInfoEntry represents one row of the guild_info table.
// Source: class.c guild_info[][3]
type GuildInfoEntry struct {
	Class     int // Class constant (or -1 for sentinel)
	Room      int // Guild room vnum
	Direction int // SCMD_NORTH/SOUTH/EAST/WEST
}

// GuildInfo controls which rooms guildguards allow each class to enter.
// The last entry is a sentinel with Class=-1.
// Source: class.c guild_info[][3]
var GuildInfo = []GuildInfoEntry{
	{ClassMageUser, 8014, 0}, // SCMD_NORTH
	{ClassThief, 8028, 0},    // SCMD_NORTH
	{ClassCleric, 8027, 1},   // SCMD_SOUTH
	{ClassWarrior, 8015, 0},  // SCMD_NORTH
	{ClassPsionic, 8518, 3},  // SCMD_WEST
	{ClassNinja, 8525, 1},    // SCMD_SOUTH
	{-1, -1, -1},             // sentinel
}

// ParseClass interprets a class selection letter and returns the class constant.
// Returns ClassUndefined (-1) for unknown letters.
// Source: class.c parse_class()
func ParseClass(arg byte) int {
	switch arg {
	case 'm', 'M':
		return ClassMageUser
	case 'c', 'C':
		return ClassCleric
	case 'w', 'W':
		return ClassWarrior
	case 't', 'T':
		return ClassThief
	case 'a', 'A':
		return ClassMagus
	case 'v', 'V':
		return ClassAvatar
	case 's', 'S':
		return ClassAssassin
	case 'p', 'P':
		return ClassPaladin
	case 'n', 'N':
		return ClassNinja
	case 'i', 'I':
		return ClassPsionic
	case 'r', 'R':
		return ClassRanger
	case 'y', 'Y':
		return ClassMystic
	default:
		return -1 // CLASS_UNDEFINED
	}
}

// FindClassBitvector maps a class selection letter to a bitvector (power of two).
// Each class occupies a sequential bit: mage=bit0, cleric=bit1, ..., mystic=bit11.
// Source: class.c find_class_bitvector()
func FindClassBitvector(arg byte) int64 {
	switch arg {
	case 'm':
		return 1 << 0 // mage
	case 'c':
		return 1 << 1 // cleric
	case 't':
		return 1 << 2 // thief
	case 'w':
		return 1 << 3 // warrior
	case 'a':
		return 1 << 4 // magus
	case 'v':
		return 1 << 5 // avatar
	case 's':
		return 1 << 6 // assassin
	case 'p':
		return 1 << 7 // paladin
	case 'n':
		return 1 << 8 // ninja
	case 'i':
		return 1 << 9 // psionic
	case 'r':
		return 1 << 10 // ranger
	case 'y':
		return 1 << 11 // mystic
	default:
		return 0
	}
}

// InvalidClass checks if a piece of equipment is unusable by a character's class.
// Returns true if the object has an ITEM_ANTI_{class} bitvector matching the character's class,
// or if a cleric tries to wield a slashing weapon, or if a thief/assassin/ninja tries to use a shield.
// Source: class.c invalid_class()
func InvalidClass(chClass int, objAntiClassBits uint32, isWieldedSlashWeapon bool, isShield bool) bool {
	// Check ITEM_ANTI_{class} bitvectors
	// Bit indices match the ExtraBitNames order in constants.go:
	// 15=ANTI_CLERIC, 16=ANTI_MAGIC_USER, 17=ANTI_THIEF, 18=ANTI_WARRIOR,
	// 19=ANTI_MAGUS, 20=ANTI_AVATAR, 21=ANTI_ASSASSIN, 22=ANTI_PALADIN,
	// 23=ANTI_NINJA, 24=ANTI_PSIONIC, 25=ANTI_RANGER, 26=ANTI_MYSTIC
	if chClass >= 0 && chClass < 12 {
		// Map class to the anti-class bit index
		// The bit order in ExtraBitNames: ANTI_CLERIC=15, ANTI_MAGIC_USER=16, ANTI_THIEF=17, ANTI_WARRIOR=18,
		// ANTI_MAGUS=19, ANTI_AVATAR=20, ANTI_ASSASSIN=21, ANTI_PALADIN=22, ANTI_NINJA=23, ANTI_PSIONIC=24, ANTI_RANGER=25, ANTI_MYSTIC=26
		antiBitMap := [12]int{
			16, // ClassMageUser → bit 16 (ANTI_MAGIC_USER)
			15, // ClassCleric   → bit 15 (ANTI_CLERIC)
			17, // ClassThief    → bit 17 (ANTI_THIEF)
			18, // ClassWarrior  → bit 18 (ANTI_WARRIOR)
			19, // ClassMagus    → bit 19 (ANTI_MAGUS)
			20, // ClassAvatar   → bit 20 (ANTI_AVATAR)
			21, // ClassAssassin → bit 21 (ANTI_ASSASSIN)
			22, // ClassPaladin  → bit 22 (ANTI_PALADIN)
			23, // ClassNinja    → bit 23 (ANTI_NINJA)
			24, // ClassPsionic  → bit 24 (ANTI_PSIONIC)
			25, // ClassRanger   → bit 25 (ANTI_RANGER)
			26, // ClassMystic   → bit 26 (ANTI_MYSTIC)
		}
		bit := antiBitMap[chClass]
		if objAntiClassBits&(1<<bit) != 0 {
			return true
		}
	}

	// Clerics cannot wield slashing weapons
	if isWieldedSlashWeapon && chClass == ClassCleric {
		return true
	}

	// Thieves, Assassins, and Ninjas cannot use shields
	if isShield && (chClass == ClassThief || chClass == ClassAssassin || chClass == ClassNinja) {
		return true
	}

	return false
}
