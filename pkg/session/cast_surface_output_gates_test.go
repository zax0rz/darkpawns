package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestCastSurfaceOutputGates(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		args          []string
		class         int
		spell         string
		level         int
		skill         int
		mana          int
		peaceful      bool
		missingTarget bool
		want          string
	}{
		{
			name:    "normal class level gate",
			command: "cast",
			args:    []string{"'acid", "blast'"},
			class:   game.ClassMageUser,
			spell:   "acid blast",
			level:   1,
			skill:   100,
			mana:    100,
			want:    "You do not know that spell!\r\n",
		},
		{
			name:    "psionic level gate",
			command: "will",
			args:    []string{"'psychic", "shield'"},
			class:   game.ClassPsionic,
			spell:   "psychic shield",
			level:   1,
			skill:   100,
			mana:    100,
			want:    "You are not learned in that power!\r\n",
		},
		{
			name:    "normal class unfamiliar spell",
			command: "cast",
			args:    []string{"'acid", "blast'"},
			class:   game.ClassMageUser,
			spell:   "acid blast",
			level:   2,
			skill:   0,
			mana:    100,
			want:    "You are unfamiliar with that spell.\r\n",
		},
		{
			name:    "psionic unfamiliar power",
			command: "will",
			args:    []string{"'psychic", "shield'"},
			class:   game.ClassPsionic,
			spell:   "psychic shield",
			level:   2,
			skill:   0,
			mana:    100,
			want:    "You are unfamiliar with that power.\r\n",
		},
		{
			name:     "peaceful room gate",
			command:  "cast",
			args:     []string{"'acid", "blast'"},
			class:    game.ClassMageUser,
			spell:    "acid blast",
			level:    2,
			skill:    100,
			mana:     100,
			peaceful: true,
			want:     "This room just has such a peaceful, easy feeling..\r\n",
		},
		{
			name:    "psionic violent self-target",
			command: "will",
			args:    []string{"'mind", "poke'", "Caster"},
			class:   game.ClassPsionic,
			spell:   "mind poke",
			level:   1,
			skill:   100,
			mana:    100,
			want:    "Exerting that power on yourself could be harmful!\r\n",
		},
		{
			name:          "psionic missing target",
			command:       "will",
			args:          []string{"'mind", "poke'", "Absent"},
			class:         game.ClassPsionic,
			spell:         "mind poke",
			level:         1,
			skill:         100,
			mana:          100,
			missingTarget: true,
			want:          "Okay.\r\nCannot find the target of your will!\r\n",
		},
		{
			name:    "psionic insufficient mana",
			command: "will",
			args:    []string{"'mind", "poke'", "Target"},
			class:   game.ClassPsionic,
			spell:   "mind poke",
			level:   10,
			skill:   100,
			mana:    0,
			want:    "You haven't the energy to will that power!\r\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager := makeTestManager(t)
			if test.peaceful {
				manager = makeGateTestManager(t, true)
			}
			caster := makeTestSession(t, manager, "Caster", 1001, true)
			registerInWorld(t, caster)
			prepareCaster(caster, test.class, test.spell, test.skill)
			caster.player.SetLevel(test.level)
			caster.player.SetMana(test.mana)

			if test.missingTarget {
				target := makeTestSession(t, manager, "Target", 1001, true)
				registerInWorld(t, target)
			}
			if test.name == "psionic insufficient mana" {
				target := makeTestSession(t, manager, "Target", 1001, true)
				registerInWorld(t, target)
			}

			if err := ExecuteCommand(caster, test.command, test.args); err != nil {
				t.Fatalf("%s dispatch: %v", test.command, err)
			}
			if got := strings.Join(drainSessionText(t, caster), ""); got != test.want {
				t.Fatalf("output = %q, want %q", got, test.want)
			}
		})
	}
}
