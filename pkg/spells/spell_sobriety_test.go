package spells

import (
	"strings"
	"testing"
)

type sobrietyStateChar struct {
	*mockSpellsChar
	drunk int
}

func (c *sobrietyStateChar) SetCondition(condition, value int) {
	if condition == 0 { // C's DRUNK condition index
		c.drunk = value
	}
}

func TestCastSobrietyClearsDrunkAndUpdatesPosition(t *testing.T) {
	for _, tc := range []struct {
		name     string
		hp       int
		position int
		wantPos  int
	}{
		{name: "sleeping with positive hp stays sleeping", hp: 20, position: int(PosSleeping), wantPos: int(PosSleeping)},
		{name: "stunned with positive hp stands", hp: 20, position: int(PosStunned), wantPos: int(PosStanding)},
		{name: "standing stays standing", hp: 20, position: int(PosStanding), wantPos: int(PosStanding)},
		{name: "incapacitating hp sets incap position", hp: -4, position: int(PosStanding), wantPos: int(PosIncap)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			victim := &sobrietyStateChar{
				mockSpellsChar: &mockSpellsChar{name: "Target", hp: tc.hp, position: tc.position},
				drunk:          7,
			}
			caster := &mockSpellsChar{name: "Caster"}

			ExecuteManualSpell(SpellSobriety, 20, caster, victim, nil, "", nil)

			if victim.drunk != 0 {
				t.Errorf("DRUNK condition = %d, want 0", victim.drunk)
			}
			if victim.position != tc.wantPos {
				t.Errorf("position = %d, want %d", victim.position, tc.wantPos)
			}
			const wantText = "You are splashed in the face with HOT coffee, but feel much more sober.\n\r"
			if got := strings.Join(victim.messages, ""); got != wantText {
				t.Errorf("victim text = %q, want %q", got, wantText)
			}
		})
	}
}
