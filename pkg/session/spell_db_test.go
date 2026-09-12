package session

import "testing"

// spellDBBaselineEntry is independent of spellData so this fixture remains a
// pre-keying comparison rather than a restatement of the production literal.
// It was captured from origin/main e961be906 before the representation change.
type spellDBBaselineEntry struct {
	SpellNum   int
	Name       string
	ManaMax    int
	ManaMin    int
	ManaChange int
	MinLevel   [12]int
}

// spellDBBeforeKeying is the complete set of present records from the
// pre-refactor map. The exhaustive test below also checks every omitted index
// in the C MAX_SPELLS range, including reserved and unused sentinels.
var spellDBBeforeKeying = map[int]spellDBBaselineEntry{
	1:   {1, "armor", 30, 15, 3, [12]int{}},
	2:   {2, "teleport", 60, 50, 3, [12]int{}},
	3:   {3, "bless", 36, 10, 2, [12]int{}},
	4:   {4, "blindness", 35, 25, 1, [12]int{}},
	5:   {5, "burning hands", 45, 20, 5, [12]int{}},
	6:   {6, "call lightning", 68, 52, 5, [12]int{}},
	7:   {7, "charm", 75, 50, 5, [12]int{}},
	8:   {8, "chill touch", 35, 15, 5, [12]int{}},
	9:   {9, "clone", 80, 65, 5, [12]int{}},
	10:  {10, "color spray", 58, 38, 4, [12]int{}},
	11:  {11, "control weather", 75, 25, 5, [12]int{}},
	12:  {12, "create food", 35, 10, 5, [12]int{}},
	13:  {13, "create water", 35, 10, 5, [12]int{}},
	14:  {14, "cure blind", 35, 5, 5, [12]int{}},
	15:  {15, "cure critical", 70, 40, 5, [12]int{}},
	16:  {16, "cure light", 30, 10, 2, [12]int{}},
	17:  {17, "curse", 80, 50, 2, [12]int{}},
	18:  {18, "detect alignment", 20, 10, 2, [12]int{}},
	19:  {19, "detect invis", 20, 10, 2, [12]int{}},
	20:  {20, "detect magic", 20, 10, 2, [12]int{}},
	21:  {21, "detect poison", 20, 10, 2, [12]int{}},
	22:  {22, "dispel evil", 95, 65, 5, [12]int{}},
	23:  {23, "earthquake", 70, 50, 5, [12]int{}},
	24:  {24, "enchant weapon", 200, 150, 10, [12]int{}},
	25:  {25, "energy drain", 60, 45, 5, [12]int{}},
	26:  {26, "fireball", 70, 50, 2, [12]int{}},
	27:  {27, "harm", 105, 75, 5, [12]int{}},
	28:  {28, "heal", 90, 80, 3, [12]int{}},
	29:  {29, "invisible", 45, 45, 1, [12]int{}},
	30:  {30, "lightning bolt", 54, 34, 4, [12]int{}},
	31:  {31, "locate object", 25, 20, 1, [12]int{}},
	32:  {32, "flame arrow", 30, 15, 5, [12]int{}},
	33:  {33, "poison", 50, 40, 2, [12]int{}},
	34:  {34, "protect evil", 50, 50, 1, [12]int{}},
	35:  {35, "remove curse", 45, 45, 1, [12]int{}},
	36:  {36, "sanctuary", 110, 85, 2, [12]int{}},
	37:  {37, "shocking grasp", 55, 35, 5, [12]int{}},
	38:  {38, "sleep", 40, 35, 1, [12]int{}},
	39:  {39, "strength", 35, 30, 1, [12]int{}},
	40:  {40, "summon", 90, 70, 1, [12]int{}},
	41:  {41, "meteor swarm", 180, 170, 5, [12]int{}},
	42:  {42, "recall", 50, 50, 1, [12]int{}},
	43:  {43, "remove poison", 40, 30, 1, [12]int{}},
	44:  {44, "sense life", 30, 20, 1, [12]int{}},
	45:  {45, "animate dead", 120, 100, 10, [12]int{}},
	46:  {46, "dispel good", 95, 65, 5, [12]int{}},
	47:  {47, "holy shield", 90, 65, 5, [12]int{}},
	48:  {48, "group heal", 210, 150, 5, [12]int{}},
	49:  {49, "group recall", 155, 125, 5, [12]int{}},
	50:  {50, "infravision", 25, 25, 1, [12]int{}},
	51:  {51, "waterwalk", 80, 55, 1, [12]int{}},
	52:  {52, "mass heal", 130, 100, 1, [12]int{}},
	53:  {53, "fly", 100, 80, 5, [12]int{}},
	54:  {54, "calliope", 100, 50, 10, [12]int{}},
	55:  {55, "vampirism", 1, 1, 1, [12]int{}},
	56:  {56, "sobriety", 35, 20, 5, [12]int{}},
	57:  {57, "group invis", 135, 135, 1, [12]int{}},
	58:  {58, "hellfire", 200, 150, 10, [12]int{}},
	59:  {59, "enchant armor", 150, 130, 10, [12]int{}},
	60:  {60, "identify", 125, 100, 10, [12]int{}},
	61:  {61, "mindpoke", 30, 15, 5, [12]int{}},
	62:  {62, "mindblast", 70, 40, 2, [12]int{}},
	63:  {63, "chameleon", 50, 30, 5, [12]int{}},
	64:  {64, "levitate", 90, 70, 5, [12]int{}},
	65:  {65, "metalskin", 75, 60, 1, [12]int{}},
	66:  {66, "invulnerability", 85, 85, 1, [12]int{}},
	67:  {67, "vitality", 110, 100, 1, [12]int{}},
	68:  {68, "invigorate", 110, 95, 1, [12]int{}},
	69:  {69, "lesser perception", 40, 30, 1, [12]int{}},
	70:  {70, "greater perception", 65, 45, 1, [12]int{}},
	71:  {71, "mind attack", 55, 25, 1, [12]int{}},
	72:  {72, "adrenaline", 35, 30, 1, [12]int{}},
	73:  {73, "psyshield", 30, 20, 1, [12]int{}},
	74:  {74, "change density", 70, 55, 1, [12]int{}},
	75:  {75, "acid blast", 35, 20, 1, [12]int{}},
	76:  {76, "dominate", 75, 50, 5, [12]int{}},
	77:  {77, "cell adjustment", 85, 75, 1, [12]int{}},
	78:  {78, "zen", 70, 60, 4, [12]int{}},
	79:  {79, "mirror image", 150, 130, 5, [12]int{}},
	80:  {80, "mass dominate", 220, 150, 10, [12]int{}},
	81:  {81, "divine int", 290, 290, 1, [12]int{}},
	82:  {82, "mind bar", 115, 100, 1, [12]int{}},
	83:  {83, "soul leech", 60, 55, 1, [12]int{}},
	84:  {84, "mindsight", 70, 60, 1, [12]int{}},
	85:  {85, "transparency", 35, 25, 1, [12]int{}},
	86:  {86, "know alignment", 20, 20, 1, [12]int{}},
	87:  {87, "gate", 95, 95, 1, [12]int{}},
	88:  {88, "intellect", 60, 60, 1, [12]int{}},
	89:  {89, "lay hands", 90, 90, 1, [12]int{}},
	90:  {90, "mental lapse", 100, 90, 1, [12]int{}},
	91:  {91, "smokescreen", 100, 100, 1, [12]int{}},
	92:  {92, "disrupt", 175, 165, 1, [12]int{}},
	93:  {93, "disintegrate", 120, 120, 1, [12]int{}},
	94:  {94, "calliope", 100, 50, 10, [12]int{}},
	95:  {95, "protect good", 50, 50, 1, [12]int{}},
	96:  {96, "flamestrike", 105, 100, 1, [12]int{}},
	97:  {97, "haste", 140, 140, 1, [12]int{}},
	98:  {98, "slow", 80, 50, 2, [12]int{}},
	99:  {99, "dream travel", 60, 45, 1, [12]int{}},
	100: {100, "psiblast", 180, 150, 10, [12]int{}},
	101: {101, "call of chaos", 90, 70, 1, [12]int{}},
	102: {102, "water breathe", 92, 58, 6, [12]int{}},
	105: {105, "conjure elemental", 165, 145, 1, [12]int{}},
}

func TestSpellDBPreservesCompletePreKeyingTable(t *testing.T) {
	const baselineMaxCastSpell = 130

	if maxCastSpell != baselineMaxCastSpell {
		t.Fatalf("maxCastSpell = %d, want unchanged C MAX_SPELLS range %d", maxCastSpell, baselineMaxCastSpell)
	}
	if len(spellDB) != len(spellDBBeforeKeying) {
		t.Fatalf("spellDB length = %d, want pre-keying length %d", len(spellDB), len(spellDBBeforeKeying))
	}

	for spellNum := 0; spellNum <= baselineMaxCastSpell; spellNum++ {
		want, wantPresent := spellDBBeforeKeying[spellNum]
		got, gotPresent := spellDB[spellNum]
		if gotPresent != wantPresent {
			t.Errorf("spellDB[%d] presence = %v, want %v", spellNum, gotPresent, wantPresent)
			continue
		}
		if !gotPresent {
			continue
		}
		if got == nil {
			t.Errorf("spellDB[%d] = nil, want complete record", spellNum)
			continue
		}
		gotSnapshot := spellDBBaselineEntry{
			SpellNum:   got.SpellNum,
			Name:       got.Name,
			ManaMax:    got.ManaMax,
			ManaMin:    got.ManaMin,
			ManaChange: got.ManaChange,
			MinLevel:   got.MinLevel,
		}
		if gotSnapshot != want {
			t.Errorf("spellDB[%d] changed: got %#v, want %#v", spellNum, gotSnapshot, want)
		}
	}

	for spellNum := range spellDB {
		if spellNum < 0 || spellNum > baselineMaxCastSpell {
			t.Errorf("spellDB contains out-of-range key %d", spellNum)
		}
	}
}
