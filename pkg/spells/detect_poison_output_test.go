package spells

import (
	"strings"
	"testing"
)

func TestCastDetectPoisonOutputBranches(t *testing.T) {
	t.Run("self target", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			aff  uint64
			want string
		}{
			{name: "poisoned", aff: 1 << 11, want: "You can sense poison in your blood.\r\n"},
			{name: "healthy", want: "You feel healthy.\r\n"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				caster := &mockSpellsChar{name: "Poisoncheck", aff: tc.aff}
				ExecuteManualSpell(SpellDetectPoison, 30, caster, caster, nil, "", nil)
				if got := strings.Join(caster.messages, ""); got != tc.want {
					t.Fatalf("self detect-poison output = %q, want %q", got, tc.want)
				}
				if caster.aff != tc.aff {
					t.Errorf("self detect-poison changed affects: got %d, want %d", caster.aff, tc.aff)
				}
			})
		}
	})

	t.Run("other character uses C subject pronoun", func(t *testing.T) {
		for _, tc := range []struct {
			name string
			sex  int
			aff  uint64
			want string
		}{
			{name: "male poisoned", sex: 0, aff: 1 << 11, want: "You sense that he is poisoned.\r\n"},
			{name: "female poisoned", sex: 1, aff: 1 << 11, want: "You sense that she is poisoned.\r\n"},
			{name: "neutral poisoned", sex: 2, aff: 1 << 11, want: "You sense that it is poisoned.\r\n"},
			{name: "male healthy", sex: 0, want: "You sense that he is healthy.\r\n"},
			{name: "female healthy", sex: 1, want: "You sense that she is healthy.\r\n"},
			{name: "neutral healthy", sex: 2, want: "You sense that it is healthy.\r\n"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				caster := &mockSpellsChar{name: "Poisoncheck"}
				victim := &mockSpellsChar{name: "Target", sex: tc.sex, aff: tc.aff}
				ExecuteManualSpell(SpellDetectPoison, 30, caster, victim, nil, "", nil)
				if got := strings.Join(caster.messages, ""); got != tc.want {
					t.Fatalf("other-character detect-poison output = %q, want %q", got, tc.want)
				}
				if caster.aff != 0 || victim.aff != tc.aff {
					t.Errorf("character detect-poison changed affects: caster=%d victim=%d, want 0/%d", caster.aff, victim.aff, tc.aff)
				}
			})
		}
	})

	t.Run("food and drink use C short description", func(t *testing.T) {
		for _, tc := range []struct {
			name   string
			poison int
			want   string
		}{
			{name: "safe", want: "You sense that a loaf of bread is safe for consumption.\r\n"},
			{name: "contaminated", poison: 1, want: "You sense that a loaf of bread has been contaminated.\r\n"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				caster := &mockSpellsChar{name: "Poisoncheck"}
				obj := &identifyOutputObject{typeFlag: 19, values: [4]int{0, 0, 0, tc.poison}, short: "a loaf of bread"}
				originalValues := obj.values
				ExecuteManualSpell(SpellDetectPoison, 30, caster, nil, obj, "", nil)
				if got := strings.Join(caster.messages, ""); got != tc.want {
					t.Fatalf("food detect-poison output = %q, want %q", got, tc.want)
				}
				if obj.values != originalValues {
					t.Errorf("food detect-poison changed values: got %v, want %v", obj.values, originalValues)
				}
			})
		}
	})

	t.Run("non-consumable object", func(t *testing.T) {
		caster := &mockSpellsChar{name: "Poisoncheck"}
		obj := &identifyOutputObject{typeFlag: 5, short: "a sword"}
		originalValues := obj.values
		ExecuteManualSpell(SpellDetectPoison, 30, caster, nil, obj, "", nil)
		const want = "You sense that it should not be consumed.\r\n"
		if got := strings.Join(caster.messages, ""); got != want {
			t.Fatalf("non-consumable detect-poison output = %q, want %q", got, want)
		}
		if obj.values != originalValues {
			t.Errorf("non-consumable detect-poison changed values: got %v, want %v", obj.values, originalValues)
		}
	})
}
