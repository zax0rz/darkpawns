// spec_assign.go — Maps mob/obj/room virtual numbers to special procedure names.
// Source: spec_assign.c
//
// This is the lookup table only. Actual spec proc implementations go in spec_procs.go.

package game

// SpecFunc is the signature for special procedure handlers.
// Real implementations live in spec_procs.go; this defines the interface.
type SpecFunc func(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool

// MobSpecAssign maps mob virtual number (VNum) to spec procedure name.
// Source: assign_mobiles() in spec_assign.c
var MobSpecAssign = map[int]string{
	// General
	1:    "puff",
	4:    "remorter",
	8:    "recharger",
	23:   "roach",
	81:   "conjured",
	82:   "conjured",
	83:   "conjured",
	84:   "conjured",
	85:   "conjured",
	86:   "conjured",
	71:   "paladin", // death dealer

	// Elemental Temple
	1313: "elements_minion",
	1314: "elements_guardian",
	1305: "cleric",
	1307: "magic_user",
	1315: "magic_user",

	// Orc Burrows
	2106: "no_move_west",
	2108: "magic_user",

	// Amber/Arden
	2716: "magic_user",
	2720: "magic_user",
	2732: "magic_user",
	2733: "magic_user",
	2734: "magic_user",
	2747: "cityguard",
	2766: "tattoo4",
	3010: "postmaster",

	// Ironwood Forest
	3119: "thief",   // cutthroat
	3103: "magic_user",
	3113: "magic_user",
	3118: "magic_user",

	// The Lakeshore Tavern
	4101: "magic_user",
	4102: "cleric",

	// Bhyroga Valley
	4202: "magic_user",
	4209: "dragon_breath",

	// Sulfur Fur Mountains
	4704: "magic_user",   // ice mage
	4705: "dragon_breath", // khal'mast

	// Xixieqi
	4813: "guild",
	4821: "guild",
	4825: "guild",
	4914: "fighter",
	4916: "magic_user",
	4919: "magic_user",

	// Xixieqi highlands
	5200: "fighter",

	// Taltos forest
	5503: "magic_user",
	5507: "magic_user",
	5510: "werewolf",

	// Corbiel
	7023: "magic_user",

	// Random Load World
	7900: "breed_killer",
	7901: "fighter",
	7902: "fighter",
	7903: "dracula",
	7909: "rescuer",
	7910: "breed_killer",
	7915: "paladin",
	7979: "eq_thief",
	7969: "magic_user",
	7970: "cleric", // unicorn

	// Kir Drax'in
	8014: "guild_guard",
	8017: "guild_guard",
	8016: "guild_guard",
	8018: "guild_guard",
	8019: "guild_guard",
	8025: "guild_guard",
	8012: "guild",
	8013: "guild",
	8015: "guild",
	8022: "stableboy",
	8023: "prostitute",
	8027: "take_to_jail",
	8059: "take_to_jail",
	8060: "wall_guard_ns",
	8001: "take_to_jail",
	8002: "take_to_jail",
	8020: "take_to_jail",
	8061: "janitor",
	8062: "citizen",
	8063: "fido",
	8081: "magic_user",
	8086: "tattoo1",
	8087: "identifier",
	8088: "jailguard",
	8089: "outofjailguard",
	8092: "butler",
	8095: "mortician",
	8096: "chosen_guard", // old guard
	8024: "guild",        // muntara
	8026: "guild",        // ninja guildmaster
	8083: "guild_guard",  // psionic guild guard

	// road/forest
	9151: "backstabber",

	// Cemetery
	9903: "magic_user",
	9905: "magic_user", // crowley
	9911: "magic_user",

	// Stadium
	10029: "troll",

	// Desert
	11023: "cleric",
	11024: "magic_user",
	11029: "magic_user",
	11030: "magic_user",
	11000: "dragon_breath",
	11001: "dragon_breath",
	11002: "dragon_breath",
	11005: "magic_user",
	11006: "magic_user",
	11007: "magic_user",
	11008: "magic_user",
	11011: "beholder",
	11016: "magic_user",
	11038: "cleric",
	11039: "cleric",

	// Swamp
	12200: "whirlpool",

	// Crystal temple and mini areas
	11706: "magic_user",

	// Haven
	12111: "fighter",   // elven pirate
	12115: "fido",      // seagull
	12118: "eq_thief",  // kender
	12127: "thief",

	// Temple
	14100: "troll",
	14101: "medusa",
	14102: "medusa",
	14103: "snake", // large snake
	14127: "snake",

	// Alaozar
	14202: "cleric",
	14206: "cleric",
	14220: "cleric",
	14219: "magic_user",
	14221: "magic_user",
	14225: "eq_thief", // Oz

	// Ogre stronghold
	14311: "troll",
	14312: "troll",
	14314: "magic_user", // ogre shaman
	14306: "magic_user", // guardian
	14309: "cleric",     // lizard shaman

	// The Grey Keep
	14401: "never_die",
	14405: "teleport_victim",
	14406: "magic_user",
	14407: "fighter",
	14410: "no_move_east",  // silk
	14411: "teleporter",    // master
	14414: "mindflayer",
	14415: "snake",
	14416: "no_get",
	14420: "brain_eater",
	14421: "no_move_west",
	14430: "no_get",
	14432: "brain_eater",
	14435: "magic_user",

	// The Plains
	15108: "fido",

	// Kir Drax'in Guard Training Centre
	16300: "recruiter",
	16308: "no_move_south",

	// Kir-Oshi
	18202: "citizen",
	18203: "fido",
	18213: "tattoo2",
	18218: "thief",
	18219: "zen_master",
	18228: "clerk",
	18215: "cityguard",

	// The Checker Board
	18301: "normal_checker",
	18302: "normal_checker",
	18303: "normal_checker",
	18304: "normal_checker",
	18306: "cuchi",

	// Lighthouse
	15804: "magic_user",
	15807: "magic_user",
	15808: "rescuer",
	15814: "pissedalchemist",

	// Mines
	12848: "magic_user",    // demon knight
	12850: "fighter",       // indy
	12876: "no_move_east",  // guardian
	12877: "cleric",        // soloman

	// Temple (additional)
	14110: "dracula", // Lothar

	// Abandoned city
	18601: "magic_user",
	18603: "magic_user",
	18604: "magic_user",

	// Ghost ship
	19119: "never_die",

	// Fire Pagoda - Shaolin Temple
	19412: "magic_user", // Fire Wizard
	19405: "quan_lo",

	// Darius Elven Camp
	19510: "castle_guard_north",

	// Player Castles
	19601: "castle_guard_north",
	19602: "castle_guard_north",
	19610: "key_seller",
	19626: "castle_guard_down",
	19627: "castle_guard_down",
	19640: "castle_guard_down",
	19641: "castle_guard_down",
	19650: "castle_guard_up",
	19651: "castle_guard_up",
	19690: "castle_guard_north",
	19691: "castle_guard_north",
	19675: "castle_guard_north",
	19676: "castle_guard_north",

	19900: "troll",
	19901: "troll",

	// DMT
	20002: "fighter",
	20003: "magic_user",
	20004: "thief",
	20005: "cleric",
	20008: "magic_user",
	20009: "magic_user",
	20010: "magic_user",
	20011: "fighter",
	20014: "magic_user",
	20018: "cleric",
	20019: "fighter",
	20020: "fighter",
	20023: "magic_user",
	20025: "magic_user",
	20026: "magic_user",
	20027: "dragon_breath",
	20029: "cleric",
	20030: "magic_user",
	20035: "cleric",
	20036: "fighter",
	20041: "cleric",
	20042: "fighter",

	// City of Alaozar
	21200: "cityguard",
	21201: "cityguard",
	21202: "cityguard",
	21203: "cityguard",
	21214: "guild",
	21215: "guild",
	21216: "guild",
	21217: "guild",
	21221: "cleric",     // high priest
	21225: "postmaster",
	21227: "cityguard",
	21228: "cityguard",
	21229: "janitor",
	21242: "thief",
	21244: "tattoo3",
	21246: "con_seller",
}

// ObjSpecAssign maps object virtual number to spec procedure name.
// Source: assign_objects() in spec_assign.c
var ObjSpecAssign = map[int]string{
	50:    "field_object",
	52:    "field_object",
	4001:  "moon_gate",  // blue portal
	4002:  "moon_gate",  // red portal
	8034:  "bank",       // KD atm
	8064:  "gen_board",  // customs house
	8065:  "gen_board",  // chosen
	8096:  "gen_board",  // social board
	8097:  "gen_board",  // freeze board
	8098:  "gen_board",  // immortal board
	8099:  "gen_board",  // mortal board
	19601: "gen_board",  // clan board
	19604: "moon_gate",
	19605: "moon_gate",
	19606: "moon_gate",
	19607: "moon_gate",
	19608: "moon_gate",
	19609: "moon_gate",
	19610: "moon_gate",
	19611: "moon_gate",
	19627: "gen_board", // clan board
	19640: "gen_board",
	19652: "gen_board", // clan board
	19666: "gen_board",
	19677: "gen_board",
	14415: "horn",
	18224: "bank", // kir-oshi atm
}

// RoomSpecAssign maps room virtual number to spec procedure name.
// Source: assign_rooms() in spec_assign.c
var RoomSpecAssign = map[int]string{
	8008:  "pray_for_items",
	8099:  "start_room",
	16300: "newbie_zone_entrance",
	8114:  "assassin",
	8118:  "jail",
	8085:  "dump",
	14305: "carrion",
	18399: "oro_study_room",
	18397: "oro_quarters_room",
	19658: "portal_to_temple",
	20073: "suck_in",
	21223: "dump",
	21235: "pet_shops",

	// Elemental Temple
	1315:  "elements_master_column",
	1326:  "elements_platforms",
	1337:  "elements_platforms",
	1348:  "elements_platforms",
	1359:  "elements_platforms",
	1360:  "elements_load_cylinders",
	1364:  "elements_load_cylinders",
	1380:  "elements_load_cylinders",
	1384:  "elements_load_cylinders",
	1372:  "elements_galeru_column",
	1394:  "elements_galeru_alive",

	// Multi-zone
	1389: "fly_exit_up",
}

// SpecRegistry holds concrete spec proc implementations keyed by name.
// Populated by init() or registration calls in spec_procs.go.
//
// CONTRACT: All spec_proc registrations (via RegisterSpec) must complete
// before the game world Start() is called. SpecRegistry is a plain map
// with no synchronization — concurrent reads after init are safe, but
// any write after game start is a data race. The startup validator
// (AllSpecNames) should be called during world init to catch missing
// registrations early.
var SpecRegistry = map[string]SpecFunc{}

// RegisterSpec registers a special procedure handler by name.
func RegisterSpec(name string, fn SpecFunc) {
	SpecRegistry[name] = fn
}

// GetMobSpec returns the spec function for a mob VNum, or nil.
func GetMobSpec(vnum int) SpecFunc {
	if name, ok := MobSpecAssign[vnum]; ok {
		return SpecRegistry[name]
	}
	return nil
}

// GetObjSpec returns the spec function for an obj VNum, or nil.
func GetObjSpec(vnum int) SpecFunc {
	if name, ok := ObjSpecAssign[vnum]; ok {
		return SpecRegistry[name]
	}
	return nil
}

// GetRoomSpec returns the spec function for a room VNum, or nil.
func GetRoomSpec(vnum int) SpecFunc {
	if name, ok := RoomSpecAssign[vnum]; ok {
		return SpecRegistry[name]
	}
	return nil
}

// IMPROVEMENTS (do not implement — document only):
//
// 1. Direct function pointer on MobInstance: During world load, after all init()
//    calls have run, resolve each mob's VNum to a SpecFunc once and store it
//    directly on the MobInstance struct. GetMobSpec() would then be a single
//    struct field read instead of two map lookups per call. Negligible at current
//    scale but relevant if mob count grows significantly.
//
// 2. Startup validation: Call AllSpecNames() at init time and verify every name
//    has a corresponding entry in SpecRegistry. Currently an unregistered spec
//    silently returns nil, which means a zone's special behavior disappears
//    without any log noise. A startup panic or warning would surface missing
//    registrations immediately.

// AllSpecNames returns a deduplicated set of all spec procedure names
// referenced across mob, obj, and room assignments.
func AllSpecNames() map[string]bool {
	names := make(map[string]bool)
	for _, n := range MobSpecAssign {
		names[n] = true
	}
	for _, n := range ObjSpecAssign {
		names[n] = true
	}
	for _, n := range RoomSpecAssign {
		names[n] = true
	}
	return names
}
