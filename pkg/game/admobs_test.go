package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestAdjustMobPrototypesMatchesCFormula(t *testing.T) {
	parsed := &parser.World{Mobs: []parser.Mob{
		{VNum: 1, Level: 0, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: -1}, Damage: parser.DiceRoll{Num: 2, Sides: 3, Plus: -1}},
		{VNum: 10, Level: 10, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: -1}, Damage: parser.DiceRoll{Num: 2, Sides: 3, Plus: -1}},
		{VNum: 11, Level: 11, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: -1}, Damage: parser.DiceRoll{Num: 2, Sides: 3, Plus: -1}},
		{VNum: 22, Level: 22, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: -1}, Damage: parser.DiceRoll{Num: 2, Sides: 3, Plus: -1}},
		{VNum: 23, Level: 23, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: -1}, Damage: parser.DiceRoll{Num: 2, Sides: 3, Plus: -1}},
		{VNum: 30, Level: 30, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: -1}, Damage: parser.DiceRoll{Num: 2, Sides: 3, Plus: -1}},
		{VNum: 31, Level: 31, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: -1}, Damage: parser.DiceRoll{Num: 2, Sides: 3, Plus: -1}},
		{VNum: 40, Level: 40, HP: parser.DiceRoll{Num: 1, Sides: 1, Plus: -1}, Damage: parser.DiceRoll{Num: 2, Sides: 3, Plus: -1}},
	}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	if got := w.AdjustMobPrototypes(); got != len(parsed.Mobs) {
		t.Fatalf("adjusted mob count = %d, want %d", got, len(parsed.Mobs))
	}

	want := map[int]struct {
		damroll int
		addHP   int
	}{
		1:  {damroll: 0, addHP: 10},
		10: {damroll: 5, addHP: 110},
		11: {damroll: 7, addHP: 120},
		22: {damroll: 14, addHP: 230},
		23: {damroll: 15, addHP: 253},
		30: {damroll: 20, addHP: 414},
		31: {damroll: 20, addHP: 997},
		40: {damroll: 26, addHP: 6244},
	}
	for vnum, expected := range want {
		mob, ok := w.GetMobPrototype(vnum)
		if !ok {
			t.Fatalf("mob %d missing after adjustment", vnum)
		}
		if mob.Damage.Plus != expected.damroll || mob.HP.Plus != expected.addHP {
			t.Errorf("mob %d adjusted to damroll=%d addHP=%d, want damroll=%d addHP=%d", vnum, mob.Damage.Plus, mob.HP.Plus, expected.damroll, expected.addHP)
		}
		if mob.Damage.Num != 2 || mob.Damage.Sides != 3 || mob.HP.Num != 1 || mob.HP.Sides != 1 {
			t.Errorf("mob %d changed dice shape: HP=%+v damage=%+v", vnum, mob.HP, mob.Damage)
		}
	}
}
