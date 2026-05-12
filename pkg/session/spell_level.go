//lint:file-ignore U1000 Game logic port — not yet wired to command registry.
package session

import "github.com/zax0rz/darkpawns/pkg/game"

// spellLearnEntry maps a spell/skill number to the level at which it is learned.
// Spell and skill numbers match src/spells.h exactly.
type spellLearnEntry struct {
	Num   int
	Level int
}

// classSpells maps each class to the set of spells/skills they can learn and at what level.
// Indexed by Class constant (game.ClassMageUser=0 .. game.ClassMystic=11).
// Faithfully translated from src/class.c:init_spell_levels() — C source is authoritative.
var classSpells = [12][]spellLearnEntry{
	// MAGES (CLASS_MAGIC_USER=0)
	game.ClassMageUser: {
		{32, 1}, // magic missile
		{50, 1}, // infravision
		{75, 2}, // acid blast
		{19, 2}, // detect invis
		{20, 2}, // detect magic
		{8, 3},  // chill touch
		{29, 4}, // invisible
		{5, 5},  // burning hands
		{39, 6}, // strength
		{37, 7}, // shocking grasp
		{38, 8}, // sleep
		{30, 9}, // lightning bolt
		{4, 9},  // blindness
		{21, 10}, // detect poison
		{10, 11}, // color spray
		{51, 12}, // waterwalk
		{44, 12}, // sense life
		{25, 13}, // energy drain
		{17, 14}, // curse
		{26, 15}, // fireball
		{58, 20}, // hellfire
		{65, 21}, // metalskin
		{102, 22}, // water breathe
		{59, 24}, // enchant armor
		{93, 25}, // disintegrate
		{24, 26}, // enchant weapon
		{66, 28}, // invulnerability
	},

	// CLERICS (CLASS_CLERIC=1)
	game.ClassCleric: {
		{185, 1}, // turn
		{16, 1},  // cure light
		{12, 2},  // create food
		{13, 2},  // create water
		{21, 3},  // detect poison
		{18, 4},  // detect align
		{14, 4},  // cure blind
		{3, 5},   // bless
		{1, 6},   // armor
		{4, 6},   // blindness
		{34, 8},  // prot from evil
		{95, 8},  // prot from good
		{15, 9},  // cure critic
		{101, 10}, // coc
		{40, 10}, // summon
		{43, 10}, // remove poison
		{47, 11}, // holy shield
		{42, 12}, // word of recall
		{33, 13}, // poison
		{22, 14}, // dispel evil
		{46, 14}, // dispel good
		{36, 15}, // sanctuary
		{28, 16}, // heal
		{48, 22}, // group heal
		{52, 23}, // mass heal
		{68, 25}, // invigorate
		{35, 26}, // remove curse
		{67, 27}, // vitality
		{49, 29}, // group recall
	},

	// THIEVES (CLASS_THIEF=2)
	game.ClassThief: {
		{151, 1}, // peek
		{131, 1}, // backstab
		{169, 1}, // compare
		{138, 2}, // sneak
		{139, 3}, // steal
		{170, 3}, // palm
		{135, 4}, // pick lock
		{133, 5}, // hide
		{167, 7}, // appraise
		{180, 8}, // detect
		{144, 9},  // trip
		{152, 11}, // subdue
		{173, 15}, // circle
		{174, 18}, // groinrip
		{143, 20}, // cutthroat
		{184, 27}, // disembowel
	},

	// WARRIORS (CLASS_WARRIOR=3)
	game.ClassWarrior: {
		{134, 1}, // kick
		{132, 3}, // bash
		{137, 4}, // rescue
		{149, 5}, // retreat
		{142, 7}, // bearhug
		{171, 8}, // berserk
		{140, 9}, // track
		{187, 12}, // sleeper
		{172, 13}, // parry
		{141, 15}, // headbutt
		{146, 17}, // slug
		{145, 20}, // smackheads
		{147, 23}, // charge
	},

	// PSIONICS (CLASS_PSIONIC=9)
	game.ClassPsionic: {
		{61, 1},  // mindpoke
		{168, 1}, // flesh alter
		{73, 2},  // psyshield
		{69, 3},  // lesser perception
		{84, 4},  // mindsight
		{71, 5},  // mind attack
		{63, 6},  // chameleon
		{72, 7},  // adrenaline
		{64, 8},  // levitate
		{62, 9},  // mindblast
		{74, 11}, // change density
		{99, 13}, // dream travel
		{2, 15},  // teleport
		{70, 18}, // great perception
		{76, 20}, // dominate
		{79, 22}, // mirror image
		{77, 23}, // cell adjustment
		{90, 25}, // mental lapse
		{100, 26}, // psiblast
		{82, 28}, // mind bar
	},

	// NINJA (CLASS_NINJA=8)
	game.ClassNinja: {
		{155, 1}, // strike
		{159, 2}, // kk kyo
		{153, 3}, // stealth
		{156, 4}, // serpent kick
		{157, 5}, // escape
		{166, 5}, // kk sha
		{154, 7}, // kabuki
		{164, 8}, // kk zai
		{161, 9}, // kk kai
		{189, 11}, // tiger punch
		{163, 12}, // kk retsu
		{152, 13}, // subdue
		{160, 15}, // kk toh
		{186, 17}, // evasion
		{158, 18}, // kk rin
		{188, 20}, // dragon kick
		{162, 22}, // kk jin
		{165, 25}, // kk zhen
		{143, 26}, // cutthroat
		{190, 28}, // neckbreak
		{91, 29},  // smokescreen
		{83, 30},  // soul leech
	},

	// MAGUS (CLASS_MAGUS=4)
	game.ClassMagus: {
		{32, 1},  // magic missile
		{50, 1},  // infravision
		{75, 2},  // acid blast
		{19, 2},  // detect invis
		{20, 2},  // detect magic
		{8, 3},   // chill touch
		{29, 4},  // invisible
		{5, 5},   // burning hands
		{39, 6},  // strength
		{37, 7},  // shocking grasp
		{38, 8},  // sleep
		{30, 9},  // lightning bolt
		{4, 9},   // blindness
		{21, 10}, // detect poison
		{10, 11}, // color spray
		{44, 12}, // sense life
		{51, 12}, // waterwalk
		{25, 13}, // energy drain
		{17, 14}, // curse
		{26, 15}, // fireball
		{87, 20}, // gate
		{58, 20}, // hellfire
		{65, 21}, // metalskin
		{53, 22}, // fly
		{102, 22}, // water breathe
		{57, 23}, // group invis
		{59, 24}, // enchant armor
		{96, 23}, // flamestrike
		{93, 25}, // disintegrate
		{24, 26}, // enchant weapon
		{92, 27}, // disrupt
		{66, 28}, // invulnerability
		{105, 29}, // conjure elemental
		{41, 30}, // meteor swarm
	},

	// AVATARS (CLASS_AVATAR=5)
	game.ClassAvatar: {
		{185, 1}, // turn
		{16, 1},  // cure light
		{12, 2},  // create food
		{13, 2},  // create water
		{21, 3},  // detect poison
		{18, 4},  // detect align
		{14, 4},  // cure blind
		{3, 5},   // bless
		{1, 5},   // armor
		{4, 6},   // blindness
		{33, 7},  // poison
		{34, 8},  // prot from evil
		{95, 8},  // prot from good
		{15, 9},  // cure critic
		{101, 10}, // coc
		{40, 10}, // summon
		{43, 10}, // remove poison
		{47, 11}, // holy shield
		{42, 12}, // word of recall
		{88, 13}, // intellect
		{22, 14}, // dispel evil
		{46, 14}, // dispel good
		{36, 15}, // sanctuary
		{28, 16}, // heal
		{48, 22}, // group heal
		{52, 23}, // mass heal
		{68, 25}, // invigorate
		{35, 26}, // remove curse
		{67, 27}, // vitality
		{49, 29}, // group recall
		{81, 30}, // divine intervention
	},

	// ASSASSINS (CLASS_ASSASSIN=6)
	game.ClassAssassin: {
		{151, 1}, // peek
		{131, 1}, // backstab
		{169, 1}, // compare
		{4, 2},   // blindness
		{138, 2}, // sneak
		{139, 3}, // steal
		{170, 3}, // palm
		{135, 4}, // pick lock
		{175, 4}, // sharpen
		{133, 5}, // hide
		{180, 5}, // detect
		{167, 7}, // appraise
		{173, 8}, // circle
		{144, 9},  // trip
		{174, 10}, // groinrip
		{181, 12}, // shadow
		{152, 15}, // subdue
		{91, 16},  // smokescreen
		{186, 17}, // evasion
		{143, 20}, // cutthroat
		{184, 25}, // disembowel
	},

	// PALADINS (CLASS_PALADIN=7)
	game.ClassPaladin: {
		{134, 1}, // kick
		{132, 3}, // bash
		{137, 3}, // rescue
		{175, 4}, // sharpen
		{18, 5},  // detect align
		{149, 5}, // retreat
		{142, 7}, // bearhug
		{171, 8}, // berserk
		{140, 9}, // track
		{89, 10}, // lay hands
		{187, 12}, // sleeper
		{172, 13}, // parry
		{141, 15}, // headbutt
		{22, 14}, // dispel evil
		{46, 14}, // dispel good
		{146, 17}, // slug
		{145, 20}, // smackheads
		{147, 23}, // charge
		{34, 25}, // prot from evil
		{95, 25}, // prot from good
		{177, 30}, // disarm
	},

	// MYSTICS (CLASS_MYSTIC=11)
	game.ClassMystic: {
		{61, 1},  // mindpoke
		{168, 1}, // flesh alter
		{73, 2},  // psyshield
		{69, 3},  // lesser perception
		{84, 4},  // mindsight
		{71, 5},  // mind attack
		{63, 6},  // chameleon
		{72, 7},  // adrenaline
		{64, 8},  // levitate
		{62, 9},  // mindblast
		{74, 11}, // change density
		{99, 13}, // dream travel
		{2, 15},  // teleport
		{70, 16}, // great perception
		{76, 19}, // dominate
		{77, 21}, // cell adjustment
		{79, 22}, // mirror image
		{90, 24}, // mental lapse
		{100, 25}, // psiblast
		{82, 26}, // mind bar
		{80, 28}, // mass dominate
		{97, 30}, // haste
	},

	// RANGERS (CLASS_RANGER=10)
	game.ClassRanger: {
		{134, 1}, // kick
		{176, 2}, // scrounge
		{132, 3}, // bash
		{175, 4}, // sharpen
		{137, 5}, // rescue
		{149, 6}, // retreat
		{179, 6}, // first aid
		{142, 7}, // bearhug
		{180, 8}, // detect
		{140, 9}, // track
		{133, 10}, // hide
		{44, 11}, // sense life
		{187, 12}, // sleeper
		{172, 13}, // parry
		{141, 15}, // headbutt
		{146, 17}, // slug
		{192, 19}, // scout
		{147, 22}, // charge
		{191, 23}, // ambush
		{148, 26}, // shoot
		{177, 30}, // disarm
	},
}
