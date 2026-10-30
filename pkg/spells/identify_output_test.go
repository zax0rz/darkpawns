package spells

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type identifyOutputObject struct {
	typeFlag int
	values   [4]int
	short    string
	weight   int
	cost     int
	affects  []parser.ObjAffect
}

func (o *identifyOutputObject) GetTypeFlag() int               { return o.typeFlag }
func (o *identifyOutputObject) GetValue(index int) int         { return o.values[index] }
func (o *identifyOutputObject) GetAffects() []parser.ObjAffect { return o.affects }
func (o *identifyOutputObject) GetName() string                { return o.short }
func (o *identifyOutputObject) GetShortDesc() string           { return o.short }
func (o *identifyOutputObject) GetWeight() int                 { return o.weight }
func (o *identifyOutputObject) GetCost() int                   { return o.cost }
func (o *identifyOutputObject) GetExtraFlags() [4]int          { return [4]int{} }

func TestCastIdentifyObjectOutputBranches(t *testing.T) {
	tests := []struct {
		name string
		obj  identifyOutputObject
		want string
	}{
		{
			name: "scroll double spacing",
			obj: identifyOutputObject{
				typeFlag: 2, short: "a return scroll", weight: 2, cost: 1000,
				values: [4]int{0, 1, 3, 0},
			},
			want: "This SCROLL casts:  holy ward bless\r\n",
		},
		{
			name: "potion spell list",
			obj: identifyOutputObject{
				typeFlag: 10, short: "a potion", values: [4]int{0, 1, 3, 0},
			},
			want: "This POTION casts:  holy ward bless\r\n",
		},
		{
			name: "wand spell and charges",
			obj: identifyOutputObject{
				typeFlag: 3, short: "a wand", values: [4]int{0, 2, 1, 1},
			},
			want: "This WAND casts: holy ward\r\nIt has 2 maximum charges and 1 remaining.\r\n",
		},
		{
			name: "staff singular charge",
			obj: identifyOutputObject{
				typeFlag: 4, short: "a staff", values: [4]int{0, 1, 1, 3},
			},
			want: "This STAFF casts: bless\r\nIt has 1 maximum charge and 1 remaining.\r\n",
		},
		{
			name: "weapon average damage",
			obj: identifyOutputObject{
				typeFlag: 5, short: "a sword", values: [4]int{0, 2, 7, 0},
			},
			want: "Damage Dice is '2D7' for an average per-round damage of 8.0.\r\n",
		},
		{
			name: "armor class apply",
			obj: identifyOutputObject{
				typeFlag: 9, short: "a breastplate", values: [4]int{3, 0, 0, 0},
			},
			want: "AC-apply is 3\r\n",
		},
		{
			name: "object affect header and detail",
			obj: identifyOutputObject{
				typeFlag: 5, short: "a ring",
				affects: []parser.ObjAffect{{Location: 1, Modifier: -2}},
			},
			want: "Can affect you as :\r\n   Affects: STR By -2\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actor := &mockSpellsChar{name: "Identifier"}
			ExecuteManualSpell(SpellIdentify, 30, actor, nil, &tt.obj, "", nil)
			if got := strings.Join(actor.messages, ""); !strings.Contains(got, tt.want) {
				t.Fatalf("identify output %q does not contain exact branch bytes %q", got, tt.want)
			}
		})
	}
}

func TestCastIdentifyLowLevelCharacterRefusal(t *testing.T) {
	caster := &mockSpellsChar{name: "Identifier", level: 30}
	target := &mockSpellsChar{name: "New player", level: 5}

	ExecuteManualSpell(SpellIdentify, caster.level, caster, target, nil, "", nil)

	const want = "You cannot identify them yet.\r\n"
	if got := strings.Join(caster.messages, ""); got != want {
		t.Fatalf("identify refusal output = %q, want exactly %q", got, want)
	}
}

type identifyReportTarget struct {
	mockSpellsChar
	str, strAdd, intStat, wis, dex, con, cha   int
	ac, hitroll, damroll, mana, height, weight int
	age                                        [4]int
}

func (p *identifyReportTarget) GetAgeParts() (int, int, int, int) {
	return p.age[0], p.age[1], p.age[2], p.age[3]
}
func (p *identifyReportTarget) GetHeight() int  { return p.height }
func (p *identifyReportTarget) GetWeight() int  { return p.weight }
func (p *identifyReportTarget) GetMana() int    { return p.mana }
func (p *identifyReportTarget) GetAC() int      { return p.ac }
func (p *identifyReportTarget) GetHitroll() int { return p.hitroll }
func (p *identifyReportTarget) GetDamroll() int { return p.damroll }
func (p *identifyReportTarget) GetStr() int     { return p.str }
func (p *identifyReportTarget) GetStrAdd() int  { return p.strAdd }
func (p *identifyReportTarget) GetInt() int     { return p.intStat }
func (p *identifyReportTarget) GetWis() int     { return p.wis }
func (p *identifyReportTarget) GetDex() int     { return p.dex }
func (p *identifyReportTarget) GetCon() int     { return p.con }
func (p *identifyReportTarget) GetCha() int     { return p.cha }

func TestCastIdentifyCharacterReportMatchesCBytes(t *testing.T) {
	caster := &mockSpellsChar{name: "Identifier", level: 30}
	target := &identifyReportTarget{
		mockSpellsChar: mockSpellsChar{name: "Target", level: 12, hp: 87},
		str:            18,
		strAdd:         42,
		intStat:        13,
		wis:            14,
		dex:            15,
		con:            16,
		cha:            17,
		ac:             -4,
		hitroll:        6,
		damroll:        8,
		mana:           99,
		height:         177,
		weight:         143,
		age:            [4]int{19, 2, 3, 4},
	}

	ExecuteManualSpell(SpellIdentify, caster.level, caster, target, nil, "", nil)

	const want = "Name: Target\r\n" +
		"Target is 19 years, 2 months, 3 days and 4 hours old.\r\n" +
		"Height 177 cm, Weight 143 pounds\r\n" +
		"Level: 12, Hits: 87, Mana: 99\r\n" +
		"AC: -4, Hitroll: 6, Damroll: 8\r\n" +
		"Str: 18/42, Int: 13, Wis: 14, Dex: 15, Con: 16, Cha: 17\r\n"
	if got := strings.Join(caster.messages, ""); got != want {
		t.Fatalf("identify character report = %q, want exact C bytes %q", got, want)
	}
}

type identifyCombatant struct {
	combat.Combatant
	char *mockSpellsChar
}

func (c *identifyCombatant) GetName() string        { return c.char.GetName() }
func (c *identifyCombatant) IsNPC() bool            { return c.char.IsNPC() }
func (c *identifyCombatant) SendMessage(msg string) { c.char.SendMessage(msg) }

type identifyHitSpy struct {
	attacker combat.Combatant
	defender combat.Combatant
}

// ExecuteSpellCounterattack is the retaliation seam every spell refusal site
// routes through now (npcRetaliate); MobFirstHit was the older name-based
// lane this spy watched before the unification.
func (s *identifyHitSpy) ExecuteSpellCounterattack(victim, caster interface{}) {
	if a, ok := victim.(combat.Combatant); ok {
		s.attacker = a
	}
	if d, ok := caster.(combat.Combatant); ok {
		s.defender = d
	}
}

func TestCastIdentifyNPCRefusalMakesCOpeningHit(t *testing.T) {
	caster := &identifyCombatant{char: &mockSpellsChar{name: "Identifier", level: 30}}
	target := &identifyCombatant{char: &mockSpellsChar{name: "Test", level: 27, npc: true}}
	world := &identifyHitSpy{}

	ExecuteManualSpell(SpellIdentify, 30, caster, target, nil, "", world)

	if got := strings.Join(caster.char.messages, ""); got != "The magicks fail horribly!\r\n" {
		t.Fatalf("NPC refusal output = %q, want exact C bytes", got)
	}
	if world.attacker != target || world.defender != caster {
		t.Fatalf("opening hit = (%T, %T), want victim as attacker and caster as defender", world.attacker, world.defender)
	}
}
