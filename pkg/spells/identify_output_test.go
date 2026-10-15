package spells

import (
	"strings"
	"testing"

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
