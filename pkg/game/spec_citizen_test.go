package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestSpecCitizen_EntryGates(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*MobInstance, *Player)
	}{
		{name: "player-triggered", setup: func(_ *MobInstance, _ *Player) {}},
		{name: "command", setup: func(_ *MobInstance, _ *Player) {}},
		{name: "sleeping", setup: func(mob *MobInstance, _ *Player) { mob.SetPosition(combat.PosSleeping) }},
		{name: "negative-hp", setup: func(mob *MobInstance, _ *Player) { mob.CurrentHP = -1 }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, player, lastMsg := newSpecProcTestWorld(t)
			mob := newSpecProcTestMob(t, w, 1001, 10)
			tt.setup(mob, player)
			lastMsg() // discard spawn output

			previous := citizenNumber
			citizenNumber = func(from, to int) int {
				t.Fatalf("citizen gate %s consumed RNG draw (%d,%d)", tt.name, from, to)
				return 0
			}
			t.Cleanup(func() { citizenNumber = previous })

			ch := (*Player)(nil)
			cmd := ""
			switch tt.name {
			case "player-triggered":
				ch = player
			case "command":
				cmd = "look"
			}
			if specCitizen(w, ch, mob, cmd, "") {
				t.Fatalf("%s gate returned true, want false", tt.name)
			}
			if got := lastMsg(); got != "" {
				t.Fatalf("%s gate output = %q, want empty", tt.name, got)
			}
		})
	}
}

func TestSpecCitizen_StandingRecovery(t *testing.T) {
	for _, tt := range []struct {
		name string
		pos  int
		want string
	}{
		{name: "sitting", pos: combat.PosSitting, want: "A test mob clambers to its feet.\r\n"},
		{name: "resting", pos: combat.PosResting, want: "A test mob stops resting, and clambers on its feet.\r\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w, _, lastMsg := newSpecProcTestWorld(t)
			mob := newSpecProcTestMob(t, w, 1001, 10)
			mob.SetPosition(tt.pos)
			mob.SetFighting("Tester")
			lastMsg()

			previous := citizenNumber
			citizenNumber = func(from, to int) int {
				t.Fatalf("fighting citizen consumed RNG draw (%d,%d)", from, to)
				return 0
			}
			t.Cleanup(func() { citizenNumber = previous })

			if specCitizen(w, nil, mob, "", "") {
				t.Fatalf("%s recovery returned true, want C's false", tt.name)
			}
			if got := mob.GetPosition(); got != combat.PosStanding {
				t.Fatalf("%s position = %d, want standing", tt.name, got)
			}
			if got := lastMsg(); got != tt.want {
				t.Fatalf("%s output = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestSpecCitizen_Sayings(t *testing.T) {
	wants := map[int]string{
		1:  "A test mob jingles some change in its pocket.\r\n",
		2:  "A test mob stares into the sky.\r\nA test mob says, 'Looks like rain. *sigh*'\r\n",
		3:  "A test mob glances at you out of the corner of its eye.\r\n",
		4:  "A test mob mumbles something about the price of a crappy loaf of bread.\r\n",
		5:  "A test mob kicks a pebble out of the road.\r\n",
		6:  "A test mob looks at you and shouts 'Repent! The end is near!'\r\n",
		7:  "A test mob eyes your coin purse.\r\n",
		8:  "A test mob looks around for the cityguards just before giving you the bird.\r\n",
		9:  "",
		10: "",
	}

	for roll, want := range wants {
		t.Run(string(rune('0'+roll)), func(t *testing.T) {
			w, _, lastMsg := newSpecProcTestWorld(t)
			mob := newSpecProcTestMob(t, w, 1001, 10)
			lastMsg()

			previous := citizenNumber
			calls := 0
			citizenNumber = func(from, to int) int {
				calls++
				switch calls {
				case 1:
					if from != 0 || to != 19 {
						t.Errorf("outer draw bounds = (%d,%d), want (0,19)", from, to)
					}
					return 0
				case 2:
					if from != 1 || to != 10 {
						t.Errorf("inner draw bounds = (%d,%d), want (1,10)", from, to)
					}
					return roll
				default:
					t.Fatalf("unexpected citizen draw %d: (%d,%d)", calls, from, to)
					return 0
				}
			}
			t.Cleanup(func() { citizenNumber = previous })

			if specCitizen(w, nil, mob, "", "") {
				t.Fatalf("inner roll %d returned true, want C's false", roll)
			}
			if got := lastMsg(); got != want {
				t.Fatalf("inner roll %d output = %q, want %q", roll, got, want)
			}
		})
	}
}

func TestSpecCitizen_SilentAndDrawOrder(t *testing.T) {
	for _, tt := range []struct {
		name  string
		rolls []int
		calls int
	}{
		{name: "outer-nonzero", rolls: []int{19}, calls: 1},
		{name: "inner-nine", rolls: []int{0, 9}, calls: 2},
		{name: "inner-ten", rolls: []int{0, 10}, calls: 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w, _, lastMsg := newSpecProcTestWorld(t)
			mob := newSpecProcTestMob(t, w, 1001, 10)
			lastMsg()

			previous := citizenNumber
			calls := 0
			citizenNumber = func(from, to int) int {
				if from != 0 && from != 1 {
					t.Errorf("unexpected draw bounds = (%d,%d)", from, to)
				}
				if calls >= len(tt.rolls) {
					t.Fatalf("unexpected draw %d", calls+1)
				}
				wantFrom, wantTo := 0, 19
				if calls > 0 {
					wantFrom, wantTo = 1, 10
				}
				if from != wantFrom || to != wantTo {
					t.Errorf("draw %d bounds = (%d,%d), want (%d,%d)", calls+1, from, to, wantFrom, wantTo)
				}
				roll := tt.rolls[calls]
				calls++
				return roll
			}
			t.Cleanup(func() { citizenNumber = previous })

			if specCitizen(w, nil, mob, "", "") {
				t.Fatalf("%s returned true, want false", tt.name)
			}
			if calls != tt.calls {
				t.Fatalf("%s consumed %d draws, want %d", tt.name, calls, tt.calls)
			}
			if got := lastMsg(); got != "" {
				t.Fatalf("%s output = %q, want empty", tt.name, got)
			}
		})
	}
}
