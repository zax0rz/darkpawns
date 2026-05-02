package session

import "github.com/zax0rz/darkpawns/pkg/game"

// spellLearnEntry maps a spell number to the level at which it is learned.
type spellLearnEntry struct {
	SpellNum int
	Level    int
}

// classSpells maps each class to the set of spells they can learn and at what level.
// Indexed by Class constant (game.ClassMageUser=0 .. game.ClassMystic=11).
// Derived from class.c spell_level() calls.
var classSpells = [12][]spellLearnEntry{
	// MAGE (game.ClassMageUser=0) — primary arcane caster, gets all standard spells
	game.ClassMageUser: {
		{1, 1},    // armor
		{32, 1},   // magic missile
		{20, 1},   // detect magic
		{19, 2},   // detect invis
		{31, 3},   // locate object
		{50, 3},   // infravision
		{8, 3},    // chill touch
		{38, 4},   // sleep
		{5, 3},    // burning hands
		{37, 5},   // shocking grasp
		{10, 6},   // color spray
		{29, 7},   // invisible
		{39, 7},   // strength
		{30, 8},   // lightning bolt
		{26, 9},   // fireball
		{33, 9},   // poison
		{4, 10},   // blindness
		{17, 11},  // curse
		{7, 12},   // charm
		{36, 13},  // sanctuary
		{40, 14},  // summon
		{42, 15},  // recall
		{9, 15},   // clone
		{2, 16},   // teleport
		{23, 17},  // earthquake
		{6, 18},   // call lightning
		{46, 20},  // dispel good
		{22, 20},  // dispel evil
		{25, 22},  // energy drain
		{24, 24},  // enchant weapon
		{58, 25},  // hellfire
		{53, 26},  // fly
		{59, 28},  // enchant armor
		{60, 30},  // identify
		{41, 32},  // meteor swarm
		{79, 34},  // mirror image
		{97, 35},  // haste
		{98, 36},  // slow
		{99, 38},  // dream travel
		{105, 40}, // conjure elemental
		{100, 43}, // psiblast
		{101, 45}, // call of chaos
		{80, 48},  // mass dominate
		{61, 4},   // mindpoke
		{62, 10},  // mindblast
		{63, 8},   // chameleon
		{64, 12},  // levitate
		{65, 14},  // metalskin
		{66, 20},  // invulnerability
		{67, 22},  // vitality
		{68, 24},  // invigorate
		{69, 5},   // lesser perception
		{70, 15},  // greater perception
		{71, 8},   // mind attack
		{72, 10},  // adrenaline
		{73, 12},  // psyshield
		{74, 14},  // change density
		{75, 6},   // acid blast
		{76, 16},  // dominate
		{77, 18},  // cell adjustment
		{78, 20},  // zen
		{102, 20}, // water breathe
		{51, 22},  // waterwalk
		{54, 5},   // calliope
		{45, 20},  // protect good
	},

	// CLERIC (game.ClassCleric=1) — divine/protection caster
	game.ClassCleric: {
		{16, 1},   // cure light
		{12, 1},   // create food
		{13, 1},   // create water
		{1, 2},    // armor
		{3, 2},    // bless
		{14, 3},   // cure blind
		{15, 3},   // cure critical
		{43, 3},   // remove poison
		{44, 4},   // sense life
		{28, 5},   // heal
		{18, 5},   // detect alignment
		{20, 4},   // detect magic
		{21, 4},   // detect poison
		{34, 6},   // protect evil
		{35, 6},   // remove curse
		{36, 8},   // sanctuary
		{22, 10},  // dispel evil
		{46, 15},  // dispel good
		{42, 11},  // recall
		{27, 12},  // harm
		{52, 14},  // mass heal
		{47, 15},  // holy shield
		{48, 18},  // group heal
		{49, 20},  // group recall
		{11, 20},  // control weather
		{67, 22},  // vitality
		{68, 24},  // invigorate
		{77, 25},  // cell adjustment
		{78, 28},  // zen
		{53, 30},  // fly
		{6, 16},   // call lightning
		{23, 22},  // earthquake
		{96, 25},  // flamestrike
		{50, 5},   // infravision
		{17, 8},   // curse
		{4, 10},   // blindness
		{33, 12},  // poison
		{51, 15},  // waterwalk
		{102, 18}, // water breathe
		{41, 30},  // meteor swarm
		{105, 35}, // conjure elemental
		{97, 32},  // haste
		{98, 34},  // slow
		{56, 5},   // sobriety
	},

	// THIEF (game.ClassThief=2) — limited utility spells
	game.ClassThief: {
		{8, 10},  // chill touch
		{29, 15}, // invisible
		{33, 15}, // poison
		{4, 20},  // blindness
		{19, 10}, // detect invis
		{50, 10}, // infravision
		{38, 20}, // sleep
		{7, 25},  // charm
		{63, 18}, // chameleon
		{61, 15}, // mindpoke
		{69, 12}, // lesser perception
		{71, 20}, // mind attack
		{64, 25}, // levitate
		{72, 22}, // adrenaline
		{2, 30},  // teleport
		{42, 25}, // recall
		{99, 35}, // dream travel
	},

	// WARRIOR (game.ClassWarrior=3) — combat spells
	game.ClassWarrior: {
		{1, 5},    // armor
		{39, 8},   // strength
		{5, 10},   // burning hands
		{37, 12},  // shocking grasp
		{8, 10},   // chill touch
		{30, 15},  // lightning bolt
		{26, 18},  // fireball
		{36, 20},  // sanctuary
		{34, 10},  // protect evil
		{22, 20},  // dispel evil
		{36, 20},  // sanctuary
		{50, 5},   // infravision
		{25, 25},  // energy drain
		{65, 20},  // metalskin
		{66, 25},  // invulnerability
		{72, 15},  // adrenaline
		{74, 22},  // change density
		{58, 30},  // hellfire
		{41, 35},  // meteor swarm
		{53, 28},  // fly
		{97, 30},  // haste
		{75, 15},  // acid blast
		{62, 25},  // mindblast
		{78, 35},  // zen
		{100, 40}, // psiblast
	},

	// MAGUS (game.ClassMagus=4) — blend of mage/warrior, spells at +5 levels vs mage
	game.ClassMagus: {
		{1, 6},    // armor
		{32, 6},   // magic missile
		{20, 6},   // detect magic
		{19, 7},   // detect invis
		{50, 8},   // infravision
		{8, 8},    // chill touch
		{38, 9},   // sleep
		{5, 8},    // burning hands
		{37, 10},  // shocking grasp
		{10, 11},  // color spray
		{29, 12},  // invisible
		{39, 12},  // strength
		{30, 13},  // lightning bolt
		{26, 14},  // fireball
		{4, 15},   // blindness
		{36, 18},  // sanctuary
		{40, 19},  // summon
		{42, 20},  // recall
		{2, 21},   // teleport
		{6, 22},   // call lightning
		{25, 24},  // energy drain
		{53, 26},  // fly
		{58, 30},  // hellfire
		{65, 20},  // metalskin
		{66, 25},  // invulnerability
		{75, 10},  // acid blast
		{41, 35},  // meteor swarm
		{97, 35},  // haste
		{98, 36},  // slow
		{100, 43}, // psiblast
		{101, 45}, // call of chaos
		{105, 40}, // conjure elemental
	},

	// AVATAR (game.ClassAvatar=5) — priest/caster hybrid
	game.ClassAvatar: {
		{16, 1},  // cure light
		{12, 1},  // create food
		{13, 2},  // create water
		{1, 6},   // armor
		{3, 5},   // bless
		{14, 7},  // cure blind
		{15, 8},  // cure critical
		{43, 8},  // remove poison
		{44, 9},  // sense life
		{28, 10}, // heal
		{20, 5},  // detect magic
		{18, 8},  // detect alignment
		{34, 10}, // protect evil
		{35, 10}, // remove curse
		{36, 15}, // sanctuary
		{22, 15}, // dispel evil
		{42, 15}, // recall
		{47, 18}, // holy shield
		{48, 22}, // group heal
		{49, 25}, // group recall
		{52, 20}, // mass heal
		{50, 7},  // infravision
		{17, 12}, // curse
		{4, 15},  // blindness
		{53, 28}, // fly
		{96, 28}, // flamestrike
		{67, 25}, // vitality
		{77, 28}, // cell adjustment
		{78, 30}, // zen
		{97, 32}, // haste
		{6, 20},  // call lightning
	},

	// ASSASSIN (game.ClassAssassin=6) — poison, invis, dark arts
	game.ClassAssassin: {
		{33, 5},  // poison
		{29, 10}, // invisible
		{4, 15},  // blindness
		{8, 10},  // chill touch
		{19, 8},  // detect invis
		{50, 8},  // infravision
		{38, 15}, // sleep
		{17, 12}, // curse
		{63, 15}, // chameleon
		{61, 12}, // mindpoke
		{69, 10}, // lesser perception
		{71, 18}, // mind attack
		{72, 15}, // adrenaline
		{2, 25},  // teleport
		{42, 20}, // recall
		{25, 28}, // energy drain
		{94, 20}, // soul leech
		{75, 15}, // acid blast
		{99, 30}, // dream travel
	},

	// PALADIN (game.ClassPaladin=7) — holy knight spells
	game.ClassPaladin: {
		{1, 5},   // armor
		{16, 3},  // cure light
		{3, 8},   // bless
		{14, 10}, // cure blind
		{15, 12}, // cure critical
		{28, 15}, // heal
		{34, 8},  // protect evil
		{35, 10}, // remove curse
		{22, 15}, // dispel evil
		{36, 20}, // sanctuary
		{39, 10}, // strength
		{47, 18}, // holy shield
		{48, 25}, // group heal
		{42, 20}, // recall
		{50, 5},  // infravision
		{53, 28}, // fly
		{97, 30}, // haste
		{65, 18}, // metalskin
		{66, 25}, // invulnerability
		{72, 15}, // adrenaline
		{74, 22}, // change density
		{78, 30}, // zen
	},

	// NINJA (game.ClassNinja=8) — utility/support spells
	game.ClassNinja: {
		{19, 8},  // detect invis
		{50, 6},  // infravision
		{8, 10},  // chill touch
		{29, 12}, // invisible
		{38, 15}, // sleep
		{33, 12}, // poison
		{4, 15},  // blindness
		{63, 12}, // chameleon
		{61, 10}, // mindpoke
		{69, 8},  // lesser perception
		{71, 15}, // mind attack
		{72, 12}, // adrenaline
		{64, 20}, // levitate
		{2, 25},  // teleport
		{42, 20}, // recall
		{99, 30}, // dream travel
		{53, 28}, // fly
		{97, 30}, // haste
		{98, 28}, // slow
	},

	// PSIONIC (game.ClassPsionic=9) — mental/mind spells
	game.ClassPsionic: {
		{61, 1},   // mindpoke
		{62, 5},   // mindblast
		{69, 3},   // lesser perception
		{70, 10},  // greater perception
		{71, 4},   // mind attack
		{72, 6},   // adrenaline
		{73, 5},   // psyshield
		{74, 8},   // change density
		{75, 6},   // acid blast
		{76, 12},  // dominate
		{77, 14},  // cell adjustment
		{78, 16},  // zen
		{7, 15},   // charm
		{4, 18},   // blindness
		{17, 15},  // curse
		{38, 12},  // sleep
		{33, 16},  // poison
		{82, 5},   // mind bar
		{94, 10},  // soul leech
		{93, 8},   // mindsight
		{80, 20},  // mass dominate
		{100, 25}, // psiblast
		{99, 22},  // dream travel
		{2, 25},   // teleport
		{42, 20},  // recall
		{36, 20},  // sanctuary
		{67, 18},  // vitality
		{68, 20},  // invigorate
		{65, 15},  // metalskin
		{66, 22},  // invulnerability
		{25, 24},  // energy drain
		{101, 30}, // call of chaos
	},

	// RANGER (game.ClassRanger=10) — nature/woodland spells
	game.ClassRanger: {
		{16, 3},   // cure light
		{1, 5},    // armor
		{3, 8},    // bless
		{14, 8},   // cure blind
		{15, 10},  // cure critical
		{28, 12},  // heal
		{6, 10},   // call lightning
		{43, 8},   // remove poison
		{44, 6},   // sense life
		{50, 4},   // infravision
		{39, 8},   // strength
		{51, 15},  // waterwalk
		{53, 20},  // fly
		{102, 15}, // water breathe
		{11, 18},  // control weather
		{42, 15},  // recall
		{36, 20},  // sanctuary
		{23, 22},  // earthquake
		{67, 22},  // vitality
		{97, 28},  // haste
		{98, 25},  // slow
		{99, 25},  // dream travel
		{2, 28},   // teleport
		{105, 30}, // conjure elemental
	},

	// MYSTIC (game.ClassMystic=11) — blend of psionic + divine
	game.ClassMystic: {
		{61, 3},   // mindpoke
		{69, 3},   // lesser perception
		{71, 5},   // mind attack
		{72, 6},   // adrenaline
		{73, 8},   // psyshield
		{16, 4},   // cure light
		{28, 10},  // heal
		{14, 8},   // cure blind
		{62, 10},  // mindblast
		{75, 8},   // acid blast
		{70, 12},  // greater perception
		{82, 8},   // mind bar
		{93, 10},  // mindsight
		{94, 12},  // soul leech
		{76, 15},  // dominate
		{77, 16},  // cell adjustment
		{78, 18},  // zen
		{74, 12},  // change density
		{67, 16},  // vitality
		{68, 18},  // invigorate
		{36, 18},  // sanctuary
		{4, 20},   // blindness
		{17, 18},  // curse
		{80, 22},  // mass dominate
		{100, 28}, // psiblast
		{42, 20},  // recall
		{53, 25},  // fly
		{99, 25},  // dream travel
		{101, 30}, // call of chaos
		{97, 28},  // haste
		{98, 30},  // slow
		{65, 18},  // metalskin
		{66, 25},  // invulnerability
	},
}
