package game

import (
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestBreedKillerLowGrowlOutputGates(t *testing.T) {
	var growlSeed, quietSeed uint32
	for seed := uint32(1); seed <= 256 && (growlSeed == 0 || quietSeed == 0); seed++ {
		probe := dprng.New(seed)
		if probe.Number(0, 5) == 0 {
			growlSeed = seed
		} else {
			quietSeed = seed
		}
	}
	if growlSeed == 0 || quietSeed == 0 {
		t.Fatal("could not find both low-growl draw outcomes")
	}

	for _, test := range []struct {
		name      string
		seed      uint32
		wantGrowl bool
	}{
		{name: "zero roll emits growl", seed: growlSeed, wantGrowl: true},
		{name: "nonzero roll stays quiet", seed: quietSeed},
	} {
		t.Run(test.name, func(t *testing.T) {
			w, err := NewWorld(&parser.World{
				Rooms: []parser.Room{{VNum: 1001, Name: "Nightbreed Street"}},
				Mobs: []parser.Mob{{
					VNum:        7900,
					Keywords:    "nightbreed guard",
					ShortDesc:   "a nightbreed guard",
					LongDesc:    "A nightbreed guard is here.\r\n",
					Level:       10,
					Race:        1,
					HP:          parser.DiceRoll{Num: 1, Sides: 1, Plus: 100},
					ActionFlags: []string{"spec"},
				}},
			})
			if err != nil {
				t.Fatalf("NewWorld: %v", err)
			}
			t.Cleanup(func() {
				w.StopAITicker()
				dprng.ResetStream(1)
			})
			if name, assigned := GetMobSpecName(7900); !assigned || name != "breed_killer" {
				t.Fatalf("mob 7900 special = %q, assigned %v; want breed_killer", name, assigned)
			}

			outputs := make(map[string]string)
			w.MessageSink = func(name string, message []byte) { outputs[name] += string(message) }
			nightbreed := NewPlayer(1, "Nightbreed", 1001)
			nightbreed.SetLevel(LVL_IMMORT)
			nightbreed.SetPlrFlag(PrfNohassle, true)
			nightbreed.SetAffect(affVampire, true)
			witness := NewPlayer(2, "Witness", 1001)
			if err := w.AddPlayer(nightbreed); err != nil {
				t.Fatalf("AddPlayer nightbreed: %v", err)
			}
			if err := w.AddPlayer(witness); err != nil {
				t.Fatalf("AddPlayer witness: %v", err)
			}
			guard, err := w.SpawnMob(7900, 1001)
			if err != nil {
				t.Fatalf("SpawnMob guard: %v", err)
			}

			reference := dprng.New(test.seed)
			roll := reference.Number(0, 5)
			wantNext := reference.Number(0, 999)
			if (roll == 0) != test.wantGrowl {
				t.Fatalf("seed %d roll = %d, want growl %v", test.seed, roll, test.wantGrowl)
			}
			dprng.ResetStream(test.seed)
			w.MobileActivity()
			gotNext := dprng.Number(0, 999)
			if gotNext != wantNext {
				t.Fatalf("next draw = %d, want %d after exactly one number(0, 5)", gotNext, wantNext)
			}

			wantOutput := ""
			if test.wantGrowl {
				wantOutput = "You hear a low growl in the back of a nightbreed guard's throat.\r\n"
			}
			for _, recipient := range []string{"Nightbreed", "Witness"} {
				if got := outputs[recipient]; got != wantOutput {
					t.Errorf("%s bytes = %q, want %q", recipient, got, wantOutput)
				}
			}
			if guard.GetFightingBody() != nil {
				t.Fatal("nohassle nightbreed target started combat")
			}
			if guard.GetPosition() <= combat.PosSleeping {
				t.Fatalf("guard position = %d, expected it to remain awake", guard.GetPosition())
			}
		})
	}
}
