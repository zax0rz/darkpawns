package session

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestWizardSetOutputGates(t *testing.T) {
	cases := []struct {
		name string
		args []string
		npc  bool
		impl bool
		want string
	}{
		{
			name: "PC-only field refuses NPC",
			args: []string{"mob", "cleaner", "brief", "on"},
			npc:  true,
			want: "You can't do that to a beast!\r\n",
		},
		{
			name: "invalid binary value",
			args: []string{"Hero", "brief", "maybe"},
			want: "Value must be on or off.\r\n",
		},
		{
			name: "invalid sex value",
			args: []string{"Hero", "sex", "other"},
			want: "Must be 'male', 'female', or 'neutral'.\r\n",
		},
		{
			name: "invalid condition value",
			args: []string{"Hero", "drunk", "tipsy"},
			want: "Must be 'off' or a value from 0 to 48.\r\n",
		},
		{
			name: "level must be below actor",
			args: []string{"Hero", "level", "55"},
			want: "You can't do that.\r\n",
		},
		{
			name: "missing player-only target",
			args: []string{"player", "Nobody", "level", "1"},
			want: "There is no such player.\r\n",
		},
		{
			name: "invalid room value",
			args: []string{"Hero", "room", "999999"},
			impl: true,
			want: "No room exists with that number.\r\n",
		},
		{
			name: "invalid class value",
			args: []string{"Hero", "class", "other"},
			want: "That is not a class.\r\n",
		},
		{
			name: "invalid race value",
			args: []string{"Hero", "race", "other"},
			want: "That is not a race.\r\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var wiz *Session
			if tc.npc {
				parsed := &parser.World{
					Rooms: []parser.Room{{VNum: 1001, Name: "Room A", Zone: 1}},
					Mobs: []parser.Mob{{
						VNum:      9001,
						Keywords:  "cleaner",
						ShortDesc: "a cleaner",
						LongDesc:  "A cleaner is here.",
						Level:     10,
						HP:        parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
					}},
				}
				world, err := game.NewWorld(parsed)
				if err != nil {
					t.Fatalf("NewWorld: %v", err)
				}
				manager := newTestManager(t, world, nil)
				wiz = makeTestSession(t, manager, "God", 1001, true)
				wiz.player.SetLevel(LVL_GRGOD)
				if _, err := world.SpawnMob(9001, 1001); err != nil {
					t.Fatalf("SpawnMob: %v", err)
				}
			} else {
				wiz, _ = makeSetTestSession(t)
			}
			if tc.impl {
				wiz.player.SetLevel(LVL_IMPL)
			}

			if err := ExecuteCommand(wiz, "set", tc.args); err != nil {
				t.Fatalf("set dispatch: %v", err)
			}
			if got := readSessionText(t, wiz); got != tc.want {
				t.Fatalf("output = %q, want %q", got, tc.want)
			}
		})
	}
}
