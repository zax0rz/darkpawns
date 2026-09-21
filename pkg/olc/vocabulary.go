package olc

// VocabularyEntry is one numbered OLC menu entry. Value is the value accepted
// by the editor operation; Label is the byte-faithful menu spelling.
type VocabularyEntry struct {
	Value int    `json:"value"`
	Label string `json:"label"`
}

// FlagVocabulary joins the world-file token and the telnet display label at
// the bit that owns both. The storage spelling is used when applying a flag;
// the display spelling is used by the menus and schema.
type FlagVocabulary struct {
	Bit     int
	Storage string
	Label   string
}

var RoomFlagNames = []string{
	"DARK", "DEATH", "!MOB", "INDOORS", "PEACEFUL", "SOUNDPROOF", "!TRACK",
	"!MAGIC", "TUNNEL", "PRIVATE", "GODROOM", "HOUSE", "HCRSH", "ATRIUM",
	"OLC", "*", "NEUTRAL", "BFR", "REGENROOM", "NO_WHO_ROOM", "**",
	"FLOW_NORTH", "FLOW_SOUTH", "FLOW_EAST", "FLOW_WEST", "FLOW_UP",
	"FLOW_DOWN", "ARENA",
}

var SectorNames = []string{
	"Inside", "City", "Field", "Forest", "Hills", "Mountains", "Water (Swim)",
	"Water (No Swim)", "Underwater", "In Flight", "Desert", "Fire", "Earth",
	"Wind", "Water", "Swamp",
}

var RoomScriptFlagNames = []string{"NONE", "ENTER", "ONPULSE", "ONDROP", "ONGET", "ONCMD"}

// ExitDoorFlagNames are the three states accepted by OpSetExitDoorFlags.
// Exit editing is bespoke in the web surface, but the telnet menu and any
// future ExitEditor must still share these byte-faithful labels.
var ExitDoorFlagNames = []string{"No door", "Closeable door", "Pickproof"}

var GenderNames = []string{"Neutral", "Male", "Female"}

var PositionNames = []string{
	"Dead", "Mortally wounded", "Incapacitated", "Stunned", "Sleeping",
	"Resting", "Sitting", "Fighting", "Standing",
}

var AttackNames = []string{
	"hit", "sting", "whip", "slash", "bite", "bludgeon", "crush", "pound",
	"claw", "maul", "thrash", "pierce", "blast", "punch", "stab",
}

// MobActionFlags is the 25-bit MEDIT menu. Storage tokens are the parser and
// world-file spellings; labels are the C menu spellings. The parser has one
// additional reserved EXTRACT bit, exposed separately below because it is not
// an editable MEDIT bit.
var MobActionFlags = []FlagVocabulary{
	{Bit: 0, Storage: "SPEC", Label: "SPEC"},
	{Bit: 1, Storage: "SENTINEL", Label: "SENTINEL"},
	{Bit: 2, Storage: "SCAVENGER", Label: "SCAVENGER"},
	{Bit: 3, Storage: "ISNPC", Label: "ISNPC"},
	{Bit: 4, Storage: "AWARE", Label: "AWARE"},
	{Bit: 5, Storage: "AGGRESSIVE", Label: "AGGR"},
	{Bit: 6, Storage: "STAY_ZONE", Label: "STAY-ZONE"},
	{Bit: 7, Storage: "WIMPY", Label: "WIMPY"},
	{Bit: 8, Storage: "AGGR_EVIL", Label: "AGGR_EVIL"},
	{Bit: 9, Storage: "AGGR_GOOD", Label: "AGGR_GOOD"},
	{Bit: 10, Storage: "AGGR_NEUTRAL", Label: "AGGR_NEUTRAL"},
	{Bit: 11, Storage: "MEMORY", Label: "MEMORY"},
	{Bit: 12, Storage: "HELPER", Label: "HELPER"},
	{Bit: 13, Storage: "NOCHARM", Label: "!CHARM"},
	{Bit: 14, Storage: "NOSUMMON", Label: "!SUMMN"},
	{Bit: 15, Storage: "NOSLEEP", Label: "!SLEEP"},
	{Bit: 16, Storage: "NOBASH", Label: "!BASH"},
	{Bit: 17, Storage: "NOBLIND", Label: "!BLIND"},
	{Bit: 18, Storage: "HUNTER", Label: "HUNTER"},
	{Bit: 19, Storage: "AGGR24", Label: "AGGR24"},
	{Bit: 20, Storage: "RANDZON", Label: "RNDLD_ZONE"},
	{Bit: 21, Storage: "MOUNTABLE", Label: "MOUNTABLE"},
	{Bit: 22, Storage: "RARE", Label: "RARE"},
	{Bit: 23, Storage: "LOOTS", Label: "LOOTS"},
	{Bit: 24, Storage: "OKGIVE", Label: "OKGIVE"},
}

var (
	MobActionStorageNames = append(flagStorageNames(MobActionFlags), "EXTRACT")
	MobActionDisplayNames = flagDisplayNames(MobActionFlags)
)

var MobAffectFlags = []FlagVocabulary{
	{Bit: 0, Storage: "BLIND", Label: "BLIND"},
	{Bit: 1, Storage: "INVISIBLE", Label: "INVIS"},
	{Bit: 2, Storage: "DETECT_ALIGN", Label: "DET-ALIGN"},
	{Bit: 3, Storage: "DETECT_INVIS", Label: "DET-INVIS"},
	{Bit: 4, Storage: "DETECT_MAGIC", Label: "DET-MAGIC"},
	{Bit: 5, Storage: "SENSE_LIFE", Label: "SENSE-LIFE"},
	{Bit: 6, Storage: "WATERWALK", Label: "WATERWALK"},
	{Bit: 7, Storage: "SANCTUARY", Label: "SANCT"},
	{Bit: 8, Storage: "GROUP", Label: "GROUP"},
	{Bit: 9, Storage: "CURSE", Label: "CURSE"},
	{Bit: 10, Storage: "INFRAVISION", Label: "INFRA"},
	{Bit: 11, Storage: "POISON", Label: "POISON"},
	{Bit: 12, Storage: "PROTECT_EVIL", Label: "PROT-EVIL"},
	{Bit: 13, Storage: "PROTECT_GOOD", Label: "PROT-GOOD"},
	{Bit: 14, Storage: "SLEEP", Label: "SLEEP"},
	{Bit: 15, Storage: "NOTRACK", Label: "!TRACK"},
	{Bit: 16, Storage: "FLESH_ALTER", Label: "FLESH-ALTER"},
	{Bit: 17, Storage: "DODGE", Label: "DODGE"},
	{Bit: 18, Storage: "SNEAK", Label: "SNEAK"},
	{Bit: 19, Storage: "HIDE", Label: "HIDE"},
	{Bit: 20, Storage: "BERSERK", Label: "BERSERK"},
	{Bit: 21, Storage: "CHARM", Label: "CHARM"},
	{Bit: 22, Storage: "FOLLOW", Label: "FOLLOW"},
	{Bit: 23, Storage: "WIMPY", Label: "WIMPY"},
	{Bit: 24, Storage: "KUJI_KIRI", Label: "KUJI-KIRI"},
	{Bit: 25, Storage: "CUTTHROAT", Label: "CUTTHROAT"},
	{Bit: 26, Storage: "FLY", Label: "FLY"},
	{Bit: 27, Storage: "WEREWOLF", Label: "WEREWOLF"},
	{Bit: 28, Storage: "VAMPIRE", Label: "VAMPIRE"},
	{Bit: 29, Storage: "MOUNT", Label: "MOUNTED"},
	{Bit: 30, Storage: "INVULN", Label: "INVULN"},
	{Bit: 31, Storage: "FLAMING", Label: "FLAMING"},
	{Bit: 32, Storage: "NOTHING", Label: "NOTHING"},
	{Bit: 33, Storage: "HASTE", Label: "HASTE"},
	{Bit: 34, Storage: "SLOW", Label: "SLOW"},
	{Bit: 35, Storage: "DREAM", Label: "DREAM"},
	{Bit: 36, Storage: "WATERBREATHE", Label: "WATERBREATHE"},
}

var (
	MobAffectStorageNames = append(flagStorageNames(MobAffectFlags), "METALSKIN", "ROBBED")
	MobAffectDisplayNames = flagDisplayNames(MobAffectFlags)
)

var MobScriptFlagNames = []string{
	"NONE", "BRIBE", "GREET", "ONGIVE", "SOUND", "DEATH", "ONPULSE (ALL)",
	"ONPULSE (PC)", "FIGHT", "ONCMD",
}

var MobRaceNames = []string{
	"Human", "Elf", "Dwarf", "Kender", "Centaur", "Rakshasa", "Troll",
	"Lycanthrope", "Vampire", "Undead", "Dragon", "Demon", "Horse", "Reptile",
	"Arachnid", "Rodent", "Other", "Vegetable", "Giant", "Demi-god", "Ogre",
	"Insect", "Mammal", "Fish", "Avian", "Magical Construct", "Amphibian",
	"Humanoid", "Faery", "Ssaur", "Minotaur",
}

var ItemTypeNames = []string{
	"UNDEFINED", "LIGHT", "SCROLL", "WAND", "STAFF", "WEAPON", "FIRE WEAPON",
	"MISSILE", "TREASURE", "ARMOR", "POTION", "WORN", "OTHER", "TRASH", "TRAP",
	"CONTAINER", "NOTE", "LIQ CONTAINER", "KEY", "FOOD", "MONEY", "PEN", "BOAT",
	"FOUNTAIN",
}

var ItemExtraFlagNames = []string{
	"GLOW", "HUM", "!RENT", "!DONATE", "!INVIS", "INVIS", "MAGIC", "!DROP",
	"BLESS", "!GOOD", "!EVIL", "!NEU", "!MAGE", "!CLE", "!THI", "!WAR", "!SELL",
	"NAMED", "!PSI", "!NIN", "!PAL", "!MAGUS", "!ASS", "!AVA", "RARE", "!LOCATE",
	"!RAN", "!MYS", "TWOHANDS",
}

var ItemWearFlagNames = []string{
	"TAKE", "FINGER", "NECK", "BODY", "HEAD", "LEGS", "FEET", "HANDS", "ARMS",
	"SHIELD", "ABOUT", "WAIST", "WRIST", "WIELD", "HOLD", "THROW", "ABLEGS", "FACE",
	"HOVER",
}

var ApplyTypeNames = []string{
	"NONE", "STR", "DEX", "INT", "WIS", "CON", "CHA", "CLASS", "LEVEL", "AGE",
	"CHAR_WEIGHT", "CHAR_HEIGHT", "MAXMANA", "MAXHIT", "MAXMOVE", "GOLD", "EXP", "ARMOR",
	"HITROLL", "DAMROLL", "SAVING_PARA", "SAVING_ROD", "SAVING_PETRI", "SAVING_BREATH",
	"SAVING_SPELL", "RACE_HATE", "HIT_REGEN", "MANA_REGEN", "MOVE_REGEN", "PERM_SPELL",
}

var ContainerFlagNames = []string{"CLOSEABLE", "PICKPROOF", "CLOSED", "LOCKED"}

var LiquidNames = []string{
	"water", "beer", "wine", "ale", "dark ale", "whiskey", "lemonade", "firebreather",
	"local speciality", "slime mold juice", "milk", "tea", "coffee", "blood", "salt water",
	"clear water",
}

var ObjectAffectFlagNames = []string{
	"BLIND", "INVIS", "DET-ALIGN", "DET-INVIS", "DET-MAGIC", "SENSE-LIFE",
	"WATERWALK", "SANCT", "GROUP", "CURSE", "INFRA", "POISON", "PROT-EVIL", "PROT-GOOD",
	"SLEEP", "!TRACK", "FLESH-ALTER", "DODGE", "SNEAK", "HIDE", "BERSERK", "CHARM",
	"FOLLOW", "WIMPY", "KUJI-KIRI", "CUTTHROAT", "FLY", "WEREWOLF", "VAMPIRE", "MOUNTED",
	"INVULN", "FLAMING", "NOTHING", "HASTE", "SLOW", "DREAM", "WATERBREATHE",
}

var SpellNames = []string{
	"!RESERVED!", "holy ward", "shift reality", "bless", "blindness", "burning hands",
	"call lightning", "charm person", "chill touch", "clone", "color spray", "control weather",
	"create food", "create water", "cure blind", "cure critic", "cure light", "curse",
	"detect alignment", "detect invisibility", "detect magic", "detect poison", "dispel evil",
	"earthquake", "enchant weapon", "energy drain", "fireball", "harm", "heal", "invisibility",
	"lightning bolt", "locate object", "flame arrow", "poison", "protection from evil",
	"remove curse", "sanctuary", "shocking grasp", "sleep", "strength", "summon", "meteor swarm",
	"word of recall", "remove poison", "sense life", "animate dead", "dispel good", "holy shield",
	"group heal", "group recall", "infravision", "waterwalk", "mass heal", "fly", "lycanthropy",
	"vampirism", "sobriety", "group invisibility", "hellfire", "enchant armor", "identify",
	"mind poke", "mind blast", "chameleon", "levitate", "metalskin", "globe of invulnerability",
	"vitality", "invigorate", "lesser perception", "greater perception", "mind attack",
	"adrenaline boost", "psychic shield", "change density", "acid blast", "dominate", "cell adjustment",
	"zen", "mirror image", "mass dominate", "divine intervention", "mind bar", "soul leech", "mindsight",
	"transparency", "know align", "gate", "word of intellect", "lay hands", "mental lapse", "smokescreen",
	"ray of disruption", "disintegration", "calliope", "protection from good", "flame strike", "haste",
	"slow", "dream travel", "psiblast", "glyph of summoning", "waterbreathe", "!drowning!",
}

var ObjectScriptFlagNames = []string{"NONE", "ONCMD", "ONPULSE"}

var ShopFlagNames = []string{"WILL_FIGHT", "USES_BANK"}

var ShopTradeNames = []string{"Good", "Evil", "Neutral", "Magic User", "Cleric", "Thief", "Warrior"}

var ZoneEquipmentNames = []string{
	"Used as light", "Worn on right finger", "Worn on left finger",
	"First worn around Neck", "Second worn around Neck", "Worn on body", "Worn on head",
	"Worn on legs", "Worn on feet", "Worn on hands", "Worn on arms", "Worn as shield",
	"Worn about body", "Worn around waist", "Worn around right wrist", "Worn around left wrist",
	"Wielded", "Held", "Wielded for throwing", "Worn about legs", "Worn on face", "Hovering near head",
}

var ZoneDirections = []string{"north", "east", "south", "west", "up", "down", "\n"}

func flagStorageNames(flags []FlagVocabulary) []string {
	result := make([]string, len(flags))
	for i, flag := range flags {
		result[i] = flag.Storage
	}
	return result
}

func flagDisplayNames(flags []FlagVocabulary) []string {
	result := make([]string, len(flags))
	for i, flag := range flags {
		result[i] = flag.Label
	}
	return result
}

func VocabularyOptions(names []string) []VocabularyEntry {
	options := make([]VocabularyEntry, len(names))
	for i, name := range names {
		options[i] = VocabularyEntry{Value: i, Label: name}
	}
	return options
}

func FlagOptions(flags []FlagVocabulary) []VocabularyEntry {
	options := make([]VocabularyEntry, len(flags))
	for i, flag := range flags {
		options[i] = VocabularyEntry{Value: flag.Bit, Label: flag.Label}
	}
	return options
}
