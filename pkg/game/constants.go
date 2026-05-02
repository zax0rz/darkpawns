//nolint:unused // Game logic port — not yet wired to command registry.
// Ported from src/constants.c
// Data tables: name arrays, stat tables, string constants

package game

// sendBufSize is the buffer size for player send channels.
// Must be consistent across all session/player creation paths.
const sendBufSize = 256 //nolint:unused // buffer size constant

// Phase names (phases[])
var Phases = []string{
	"New Moon",
	"Waxing Crescent",
	"First Quarter",
	"Waxing Gibbous",
	"Full Moon",
	"Waning Gibbous",
	"Last Quarter",
	"Waning Crescent",
}

// Hometown names (hometowns[])
var Hometowns = []string{
	"Kalaman",
	"Solace",
	"Port Storm",
	"Tarsis",
	"Tarmin Keep",
	"Highpeak",
	"Gwynned",
	"Crystalmir",
	"Kaolyn",
	"Erstwhile Temple",
	"Port Balifor",
}

// Ability score names (abil_names[])
var AbilityNames = []string{
	"Strength",
	"Intelligence",
	"Wisdom",
	"Dexterity",
	"Constitution",
	"Charisma",
}

// Crowd size descriptions (crowd_size[])
var CrowdSize = []string{
	"nobody",
	"a few people",
	"a small crowd",
	"a crowd",
	"a large crowd",
}

// Direction names (dirs[])
var DirectionNames = []string{
	"north",
	"east",
	"south",
	"west",
	"up",
	"down",
}

// Mobile race names (mob_races[])
var MobRaceNames = []string{
	"Human",
	"Elf",
	"Dwarf",
	"Kender",
	"Minotaur",
	"Rakshasa",
	"Ssaur",
	"Half-Elf",
	"Half-Ogre",
	"Gnome",
	"Goblin",
	"Barbarian",
	"Halfling",
	"Wolf",
	"Bear",
	"Evil",
	"Animal",
	"Flying",
	"Feline",
	"Canine",
	"Draconian",
	"Dragon",
	"Draconian Highlord",
	"Undead",
	"Insect",
	"Ogre",
	"Orc",
	"Hobgoblin",
	"Spider",
	"Elemental",
	"Golem",
	"Snake",
	"Demon",
	"Scorpion",
	"Centaur",
	"Dwarf",
	"Elf",
	"Gnome",
	"Human",
	"Kender",
	"Half-Elf",
	"Half-Ogre",
	"Goblin",
	"Barbarian",
	"Halfling",
	"Pixie",
	"Giant",
	"Troll",
	"Fish",
}

// Intelligent races bitvector (intelligent_races[])
var IntelligentRaces = []int{
	1 << 0, // RACE_HUMAN
	1 << 1, // RACE_ELF
	1 << 2, // RACE_DWARF
	1 << 3, // RACE_KENDER
	1 << 4, // RACE_MINOTAUR
	1 << 5, // RACE_RAKSHASA
	1 << 6, // RACE_SSAUR
	1 << 7, // RACE_HALFELF
	1 << 8, // RACE_HALFOGRE
	1 << 9, // RACE_GNOME
	1 << 10, // RACE_GOBLIN
	1 << 11, // RACE_BARBARIAN
	1 << 12, // RACE_HALFLING
}

// Room flag names (room_bits[])
var RoomBitNames = []string{
	"DARK",
	"DEATH",
	"NO_MOB",
	"INDOORS",
	"PEACEFUL",
	"SOUNDPROOF",
	"NO_TRACK",
	"NO_MAGIC",
	"TUNNEL",
	"PRIVATE",
	"GODROOM",
	"NO_FIGHT",
	"HOUSE",
	"ALWAYS_LIGHT",
	"ATRIUM",
	"TELEPORT",
	"ARENA",
	"DEATH_TRAP",
	"NO_SAVE",
	"NO_CHARGE",
	"LAW",
	"WILD",
}

// Exit flag names (exit_bits[])
var ExitBitNames = []string{
	"ISDOOR",
	"CLOSED",
	"LOCKED",
	"CAN_SEE",
	"PICKPROOF",
	"NO_MOB",
	"HIDDEN",
	"SECRET",
}

// Sector type names (sector_types[])
var SectorTypeNames = []string{
	"Inside",
	"City",
	"Field",
	"Forest",
	"Hills",
	"Mountain",
	"Swim",
	"Water",
	"Air",
	"Desert",
	"Underground",
	"Ocean",
	"Tundra",
}

// Gender names (genders[])
var GenderNames = []string{
	"neutral",
	"male",
	"female",
}

// Position type names (position_types[])
var PositionNames = []string{
	"dead",
	"mortally wounded",
	"incapacitated",
	"stunned",
	"sleeping",
	"resting",
	"sitting",
	"fighting",
	"standing",
}

// Player flag names (player_bits[])
var PlayerBitNames = []string{
	"KILLER",
	"THIEF",
	"FROZEN",
	"DONTSET",
	"WRITE",
	"FORCE",
	"COMPRESS",
	"AFK",
	"DOORHACK",
	"TAG",
}

// Action flag names (action_bits[])
var ActionBitNames = []string{
	"SPEC",
	"SENTINEL",
	"SCAVENGER",
	"ISNPC",
	"NICE",
	"AGGRESSIVE",
	"GREEDY",
	"STAY_ZONE",
	"WIMPY",
	"FOLLOW",
	"PURSUE",
	"DEADLY",
	"POLYSELF",
	"META_AGG",
	"GUARD",
	"AUCTION",
	"CHARITABLE",
	"MOUNT",
	"INVISIBLE",
}

// Preference flag names (preference_bits[])
var PreferenceBitNames = []string{
	"BRIEF",
	"COMPACT",
	"SHORT",
	"NOTITLE",
	"NO_INTRO",
	"NO_GOSSIP",
	"NO_GAINS",
	"NO_HASSLE",
	"NO_TELL",
	"OLC",
	"NO_ANSI",
	"NO_GOAL",
	"ROOMFLAGS",
	"OREMOTE",
	"DISPHP",
	"DISPMANA",
	"DISPMOVE",
	"NO_AUTOLOOT",
	"AUTOEXIT",
	"NO_LIGHT",
	"NOREPEAT",
	"TELLWRAP",
	"EMAIL",
	"MAP",
	"RENT",
	"HEAR",
	"LOCATE",
	"AFK",
	"WIZARD",
	"ANONYMOUS",
	"STOP_BLEED_MSG",
	"NO_AUTOOFFER",
	"PATHWAY",
	"JUMP_ONCE",
}

// Affected bit names (affected_bits[])
var AffectedBitNames = []string{
	"BLIND",
	"CHARM",
	"CURSE",
	"POISON",
	"PROTECT_EVIL",
	"PROTECT_GOOD",
	"SLEEP",
	"NO_FLIGHT",
	"FLYING",
	"TRUE_SIGHT",
	"INFRARED",
	"WATERWALK",
	"SANCTUARY",
	"GROUP",
	"HASTE",
	"SLOW",
	"PLAGUE",
	"WEAKEN",
	"INVISIBLE",
	"DETECT_ALIGN",
	"DETECT_INVIS",
	"DETECT_MAGIC",
	"SENSE_LIFE",
	"SENSE_PSY",
	"SHIELD",
	"WEB",
	"BERSERK",
	"BLADE",
	"BLUR",
	"FIRESHIELD",
	"ICESHIELD",
	"SHOCKSHIELD",
	"BARKSKIN",
	"LEVITATE",
	"DETECT_INV",
}

// Connected state names (connected_types[])
var ConnectedTypeNames = []string{
	"Playing",
	"Get Name",
	"Get Password",
	"Get New Password",
	"Get Confirm Password",
	"Get Sex",
	"Get Class",
	"Get Race",
	"Roll Stats",
	"Get Hometown",
	"Get Align",
	"Get Menu",
	"Read Motd",
	"Disconnecting",
	"Get Name - Menu",
}

// Where descriptions (where[])
var WhereNames = []string{
	"<used as light>    ",
	"<worn on finger>   ",
	"<worn on finger>   ",
	"<worn around neck> ",
	"<worn around neck> ",
	"<worn on body>     ",
	"<worn on head>     ",
	"<worn on legs>     ",
	"<worn on feet>     ",
	"<worn on hands>    ",
	"<worn on arms>     ",
	"<worn as shield>   ",
	"<worn about body>  ",
	"<worn around waist>",
	"<worn on wrist>    ",
	"<worn on wrist>    ",
	"<wielded>          ",
	"<held in off-hand> ",
	"<held>             ",
	"<worn on ear>      ",
	"<worn on face>     ",
	"<double-wielded>   ",
}

// Equipment position names (equipment_types[])
var EquipmentTypes = []string{
	"Light",
	"Finger Right",
	"Finger Left",
	"Neck 1",
	"Neck 2",
	"Body",
	"Head",
	"Legs",
	"Feet",
	"Hands",
	"Arms",
	"Shield",
	"About",
	"Waist",
	"Wrist Right",
	"Wrist Left",
	"Wield",
	"Off Hand",
	"Held",
	"Ear",
	"Face",
	"Double Wield",
}

// Item type names (item_types[])
var ItemTypeNames = []string{
	"UNDEFINED",
	"LIGHT",
	"SCROLL",
	"WAND",
	"STAFF",
	"WEAPON",
	"FIRE WEAPON",
	"MISSILE",
	"TREASURE",
	"ARMOR",
	"POTION",
	"WORN",
	"OTHER",
	"TRASH",
	"TRAP",
	"CONTAINER",
	"NOTE",
	"LIQ CONTAINER",
	"KEY",
	"FOOD",
	"MONEY",
	"PEN",
	"BOAT",
	"FOUNTAIN",
	"SPELLBOOK",
	"BOARD",
	"PORTAL",
	"ROOM KEY",
	"LOCK PICK",
	"PIPE",
	"HORN",
	"BATTLE HORN",
	"HERB CONTAINER",
	"POISON",
	"CARRIAGE",
	"BANDAGE",
	"AMMO",
}

// Wear flag names (wear_bits[])
var WearBitNames = []string{
	"TAKE",
	"FINGER",
	"NECK",
	"BODY",
	"HEAD",
	"LEGS",
	"FEET",
	"HANDS",
	"ARMS",
	"SHIELD",
	"ABOUT",
	"WAIST",
	"WRIST",
	"WIELD",
	"HOLD",
	"EAR",
	"FACE",
	"LIGHT",
	"THROW",
}

// Extra flag names (extra_bits[])
var ExtraBitNames = []string{
	"GLOW",
	"HUM",
	"TIMER",
	"INVISIBLE",
	"MAGIC",
	"NODROP",
	"BLESS",
	"ANTI_GOOD",
	"ANTI_EVIL",
	"ANTI_NEUTRAL",
	"NORENT",
	"NODONATE",
	"NOSACRIFICE",
	"ROTTEN",
	"CRUSHED",
	"ANTI_CLERIC",
	"ANTI_MAGIC_USER",
	"ANTI_THIEF",
	"ANTI_WARRIOR",
	"ANTI_MAGUS",
	"ANTI_AVATAR",
	"ANTI_ASSASSIN",
	"ANTI_PALADIN",
	"ANTI_NINJA",
	"ANTI_PSIONIC",
	"ANTI_RANGER",
	"ANTI_MYSTIC",
	"BROKEN",
	"ANTI_MALE",
	"ANTI_FEMALE",
	"ANTI_KENDER",
	"BURIED",
	"TATTOO",
}

// Apply type names (apply_types[])
var ApplyTypeNames = []string{
	"NONE",
	"STR",
	"DEX",
	"INT",
	"WIS",
	"CON",
	"CHA",
	"CLASS",
	"LEVEL",
	"AGE",
	"CHAR_WEIGHT",
	"CHAR_HEIGHT",
	"MANA",
	"HIT",
	"MOVE",
	"GOLD",
	"EXP",
	"ARMOR",
	"HITROLL",
	"DAMROLL",
	"SAVING_PARA",
	"SAVING_ROD",
	"SAVING_PETRI",
	"SAVING_BREATH",
	"SAVING_SPELL",
}

// Container flag names (container_bits[])
var ContainerBitNames = []string{
	"CLOSEABLE",
	"PICKPROOF",
	"CLOSED",
	"LOCKED",
}

// Spell wear-off messages (spell_wear_off_msg[])
var SpellWearOffMessages = []string{
	"You feel less protected.",
	"Your detect alignment wears off.",
	"Your detect evil wears off.",
	"Your detect invisibility wears off.",
	"Your detect magic wears off.",
	"You feel less aware.",
	"Your infrared vision fades.",
	"You feel a cloak of protection disappear.",
	"The shroud of darkness fades away.",
	"The light around you fades...",
	"Your courage fails.",
	"You feel less aware of your surroundings.",
	"You feel less aware of your surroundings.",
	"You feel less aware of your surroundings.",
	"Your sense of the astral plane fades.",
	"Your divination spell ends.",
	"The divination ends.",
	"Your precognition fades.",
	"Your detect psionics fades.",
	"Your telepathic powers fade.",
	"Your clairvoyance fades.",
	"The psychic crush leaves your mind.",
	"Your shield fades away.",
	"The web dissolves!",
	"Your berserking ends.",
	"Your bladethirst ends.",
	"Your blur fades.",
	"The fire shield around you fades.",
	"The ice shield around you melts.",
	"The shock shield around you crackles and fades.",
	"Your barkskin wears off.",
	"You slowly drift downwards.",
	"Your water breathing wears off.",
	"Your flying wears off.",
	"Your mind blank wears off.",
	"Your immunity wears off.",
	"You feel a protection from elements fade.",
	"You feel less protected from mental attacks.",
	"The mental barrier around you fades.",
	"Your statis field ends.",
	"Your telekinesis ends.",
	"Your Aura of Glory fades.",
	"The roots retract into the ground.",
	"Your death ward fades.",
	"Your battle frenzy ends.",
	"You no longer feel chameleon-like.",
	"The shadow form fades away.",
	"Your word of recall fades.",
	"Your endurance fades.",
	"Your stone skin crumbles away.",
	"Your iron skin crumbles away.",
	"You feel less holy.",
	"You feel less power over serpents.",
	"The smoke screen clears.",
	"Your cloud of darkness dissipates.",
	"You feel less holy.",
	"Your evil protection fades.",
	"Your good protection fades.",
	"The camouflage fades.",
	"Your protection from scrying fades.",
	"You are no longer hidden in mist.",
	"The wall of fog dissipates.",
	"The entangling vines wither.",
	"You are no longer shielded from acid.",
	"You are no longer shielded from cold.",
	"You are no longer shielded from fire.",
	"You are no longer shielded from lightning.",
	"Your poison protection fades.",
	"Your anti-magic shell dissolves.",
	"The wall of thorns withers.",
	"The lava wall hardens.",
	"The wall of force shimmers.",
	"The iris closes.",
	"The prismatic sphere fades.",
	"The prismatic wall fades.",
	"The temporal wall dissolves.",
	"The null field collapses.",
	"Your temporal fugue fades.",
}

// NPC class type names (npc_class_types[])
var NpcClassTypeNames = []string{
	"None",
	"Mage",
	"Cleric",
	"Warrior",
	"Thief",
}

// Reverse direction lookup (rev_dir[])
var ReverseDirection = []int{
	2, // north -> south
	3, // east -> west
	0, // south -> north
	1, // west -> east
	5, // up -> down
	4, // down -> up
}

// Movement loss by sector type (movement_loss[])
// Ported from src/constants.c movement_loss[] — per-sector move point cost.
var MovementLoss = []int{
	2, // INSIDE
	2, // CITY
	3, // FIELD
	4, // FOREST
	5, // HILLS
	7, // MOUNTAIN
	5, // WATER_SWIM
	6, // WATER_NOSWIM
	2, // AIR
	8, // DESERT
	4, // UNDERGROUND
	6, // OCEAN
	6, // TUNDRA
}

// Day names (weekdays[])
var WeekdayNames = []string{
	"the Feast of the Just",
	"the Day of the Dark",
	"the Day of the White",
	"the Day of the Red",
	"the Day of the Black",
	"the Day of the Blue",
	"the Day of the Green",
}

// Month names (month_name[])
var MonthNames = []string{
	"New Year Tide",
	"Winter Deep",
	"Snow Melt",
	"Spring Dawning",
	"Green Field",
	"Flower Blooms",
	"High Sun",
	"Harvest Tide",
	"Fruit Picking",
	"Leaf Fall",
	"First Ice",
	"Dark Tide",
}

// Sharp damage bonus table (sharp[])
var SharpDamage = []int{
	0, 1, 1, 1, 2, 2, 2, 3, 3, 3,
	4, 4, 4, 5, 5, 5, 6, 6, 6, 7,
	7, 7, 8, 8, 8, 9, 9, 9, 10, 10,
	10, 11, 11, 11, 12, 12, 12, 13, 13, 13,
	14, 14, 14, 15, 15, 15, 16, 16, 16, 17,
}
