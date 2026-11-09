package game

import (
	"reflect"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestDoDreamMatchesCOrderAudienceAndVisibility(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1001, Name: "Dream Room", Zone: 1}},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() { w.StopAITicker() })

	actor := NewPlayer(1, "Dreamer", 1001)
	awake := NewPlayer(2, "Awake", 1001)
	sleeper := NewPlayer(3, "Sleeper", 1001)
	blind := NewPlayer(4, "Blind", 1001)
	actor.SetPosition(combat.PosSleeping)
	sleeper.SetPosition(combat.PosSleeping)
	blind.SetAffect(affBlind, true)
	for _, p := range []*Player{actor, awake, sleeper, blind} {
		if err := w.AddPlayer(p); err != nil {
			t.Fatalf("AddPlayer(%s) failed: %v", p.Name, err)
		}
	}

	var events []string
	w.MessageSink = func(name string, message []byte) {
		events = append(events, name+":"+string(message))
	}

	DoDream(w, actor)

	want := []string{
		"Awake:Dreamer dreams of running naked through a field of tulips.\r\n",
		"Dreamer:You dream of running naked through a field of tulips.\r\n",
	}
	if !reflect.DeepEqual(events, want) {
		t.Fatalf("dream events = %#v, want %#v", events, want)
	}
}

func TestDreamPointUpdateOutputBranches(t *testing.T) {
	type testCase struct {
		name        string
		level       int
		deathAge    time.Duration
		dreaming    bool
		sleepAffect bool
		chanceMax   int
		want        []string
		wantPos     int
		wantRoom    int
	}

	cases := []testCase{
		{
			name: "death under one day wakes through C do_wake", level: 1, deathAge: 12 * time.Hour, chanceMax: 5,
			want: []string{
				"Dreamer:You see the visions of your own death and wake up screaming!\r\n",
				"Awake:Dreamer wakes up screaming, with a look of death in his eyes.\r\n",
				"Dreamer:You awaken, and sit up.\r\n",
				"Awake:Dreamer awakens.\r\n",
			}, wantPos: combat.PosSitting, wantRoom: 1001,
		},
		{
			name: "death one to two days", level: 1, deathAge: 36 * time.Hour, chanceMax: 5,
			want: []string{
				"Dreamer:In your dreams you keep seeing a dark figure hunched over your corpse.\r\n",
				"Awake:Dreamer shivers in his sleep.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 1001,
		},
		{
			name: "death two to three days", level: 1, deathAge: 60 * time.Hour, chanceMax: 5,
			want: []string{
				"Dreamer:You toss and turn as a dark cloud hovers over your dreams.\r\n",
				"Awake:Dreamer tosses and turns in his sleep, must be a bad dream.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 1001,
		},
		{
			name: "death three to five days", level: 1, deathAge: 96 * time.Hour, chanceMax: 5,
			want: []string{
				"Dreamer:You sleep uneasily, as if something looms over your past\r\n",
				"Awake:Dreamer grunts in his sleep.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 1001,
		},
		{
			name: "level zero to five", level: 1, chanceMax: 15,
			want: []string{
				"Dreamer:You have dreams of showing this world what you are really made of.\r\n",
				"Awake:Dreamer smiles in his sleep.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 1001,
		},
		{
			name: "level six to ten", level: 6, chanceMax: 15,
			want: []string{
				"Dreamer:You have a pleasant dream of safe travels to far places and a hero's welcome when you return.\r\n",
				"Awake:Dreamer begins to hum a happy ditty in his sleep.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 1001,
		},
		{
			name: "level eleven to twenty", level: 11, chanceMax: 15,
			want: []string{
				"Dreamer:You dream of your conquest of the world.\r\n",
				"Awake:Dreamer begins to grin in his sleep.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 1001,
		},
		{
			name: "level twenty-one to twenty-eight", level: 21, chanceMax: 15,
			want: []string{
				"Dreamer:You dream of slaying the dark creatures of the night.\r\n",
				"Awake:Dreamer smirks in his sleep.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 1001,
		},
		{
			name: "level twenty-nine to thirty", level: 29, chanceMax: 15,
			want: []string{
				"Dreamer:You have a fantastic dream of one day attaining immortality.\r\n",
				"Awake:Dreamer looks like he is having big dreams.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 1001,
		},
		{
			name: "immortal level", level: LVLImmort, chanceMax: 15,
			want: []string{
				"Dreamer:You have big, grand dreams of the power of the Gods.\r\n",
				"Awake:Dreamer glows in his sleep.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 1001,
		},
		{
			name: "level above immortal wakes through C do_wake", level: LVLImmort + 1, chanceMax: 15,
			want: []string{
				"Dreamer:You toss and turn under the constant fear of the wrath of Orodreth :-)\r\n",
				"Dreamer:You find yourself wide awake!\r\n",
				"Awake:Dreamer awakens with fear in his eyes.\r\n",
				"Dreamer:You awaken, and sit up.\r\n",
				"Awake:Dreamer awakens.\r\n",
			}, wantPos: combat.PosSitting, wantRoom: 1001,
		},
		{
			name: "level above immortal remains magically asleep", level: LVLImmort + 1, sleepAffect: true, chanceMax: 15,
			want: []string{
				"Dreamer:You toss and turn under the constant fear of the wrath of Orodreth :-)\r\n",
				"Dreamer:You find yourself wide awake!\r\n",
				"Awake:Dreamer awakens with fear in his eyes.\r\n",
				"Dreamer:You can't wake up!\r\n",
				"Awake:Dreamer tosses and turns uncomfortably.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 1001,
		},
		{
			name: "normal dream travel", level: 1, dreaming: true, chanceMax: 15,
			want: []string{
				"Dreamer:You have a dream that shows you penitent before an altar. \r\n",
				"Awake:The sleeping body of Dreamer fades from existence.\r\n",
				"Destination:The sleeping body of Dreamer fades into existence.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 8004,
		},
		{
			name: "bad dream travel", level: 1, deathAge: 12 * time.Hour, dreaming: true, chanceMax: 15,
			want: []string{
				"Dreamer:You have a dream that shows you penitent before an altar. \r\n",
				"Awake:The sleeping body of Dreamer fades from existence.\r\n",
				"Destination:The sleeping body of Dreamer fades into existence.\r\n",
			}, wantPos: combat.PosSleeping, wantRoom: 8004,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w, err := NewWorld(&parser.World{Rooms: []parser.Room{
				{VNum: 1001, Name: "Source", Zone: 1},
				{VNum: 8004, Name: "Destination", Zone: 8},
			}})
			if err != nil {
				t.Fatalf("NewWorld failed: %v", err)
			}
			t.Cleanup(func() {
				w.StopAITicker()
				dprng.ResetStream(1)
			})

			dreamer := NewPlayer(1, "Dreamer", 1001)
			dreamer.Sex = SexMale
			dreamer.Level = tc.level
			dreamer.SetPosition(combat.PosSleeping)
			dreamer.SetCondition(CondFull, -1)
			dreamer.SetCondition(CondDrunk, -1)
			dreamer.SetCondition(CondThirst, -1)
			if tc.deathAge > 0 {
				dreamer.SetLastDeath(time.Now().Unix() - int64(tc.deathAge/time.Second))
			}
			if tc.dreaming {
				dreamer.SetAffect(AFF_DREAM_BIT, true)
			}
			if tc.sleepAffect {
				dreamer.SetAffect(AffSleep, true)
			}

			source := NewPlayer(2, "Awake", 1001)
			destination := NewPlayer(3, "Destination", 8004)
			for _, player := range []*Player{dreamer, source, destination} {
				if err := w.AddPlayer(player); err != nil {
					t.Fatalf("AddPlayer(%s) failed: %v", player.Name, err)
				}
			}

			var events []string
			w.MessageSink = func(name string, message []byte) {
				events = append(events, name+":"+string(message))
			}

			seed := dreamPointUpdateSeed(t, tc.chanceMax)
			dprng.ResetStream(seed)
			w.PointUpdate()

			if !reflect.DeepEqual(events, tc.want) {
				t.Fatalf("point_update events = %#v, want %#v", events, tc.want)
			}
			if got := dreamer.GetPosition(); got != tc.wantPos {
				t.Fatalf("position = %d, want %d", got, tc.wantPos)
			}
			if dreamer.RoomVNum != tc.wantRoom {
				t.Fatalf("room = %d, want %d", dreamer.RoomVNum, tc.wantRoom)
			}
			if got := dreamer.IsAffected(AFF_DREAM_BIT); got {
				t.Fatalf("AFF_DREAM remained set after travel: %t", got)
			}
		})
	}
}

func dreamPointUpdateSeed(t *testing.T, max int) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 10000; seed++ {
		if dprng.New(seed).Number(0, max) == 0 {
			return seed
		}
	}
	t.Fatalf("no seed produced Number(0,%d) == 0", max)
	return 0
}

// fakeDreamContext records the side effects DreamTravelFn performs, so a draw
// test can detect an unexpected fire.
type fakeDreamContext struct {
	traveled bool
}

func (f *fakeDreamContext) GetLevel() int       { return 1 }
func (f *fakeDreamContext) GetLastDeath() int64 { return 0 }
func (f *fakeDreamContext) SetLastDeath(int64)  {}
func (f *fakeDreamContext) HasAffect(int) bool  { return false }
func (f *fakeDreamContext) RemoveAffect(int)    {}
func (f *fakeDreamContext) SendToChar(string)   {}
func (f *fakeDreamContext) SendToRoom(string)   {}
func (f *fakeDreamContext) WakeUp()             {}
func (f *fakeDreamContext) MoveToRoom(int)      { f.traveled = true }
func (f *fakeDreamContext) CurrentTime() int64  { return 0 }

// dreamTravelOracleDraws simulates C's draw order for dream_travel()
// (src/dream.c:195, 208): number(0,15) is evaluated first in both branches, so
// every entry costs a first draw and, when the first branch does not fire, a
// second draw. It reports the draws consumed before a fire, or the whole pass.
func dreamTravelOracleDraws(seed uint32, subcmd int) (draws int, fired bool) {
	s := dprng.New(seed)
	for i := 0; i <= NumDreams; i++ {
		draws++
		if s.Number(0, 15) == 0 && subcmd == 0 && DreamTravelTable[i].Subcmd == 0 {
			return draws, true
		}
		draws++
		if s.Number(0, 15) == 0 && subcmd != 0 {
			return draws, true
		}
	}
	return draws, false
}

func dreamTravelNonFiringSeed(t *testing.T, subcmd int) uint32 {
	t.Helper()
	for seed := uint32(1); seed < 200000; seed++ {
		if _, fired := dreamTravelOracleDraws(seed, subcmd); !fired {
			return seed
		}
	}
	t.Fatalf("no non-firing seed for subcmd %d", subcmd)
	return 0
}

// TestDreamTravelDrawOrderMatchesC pins R3: a non-firing dream_travel pass draws
// number(0,15) twice per entry (2*(NUM_DREAMS+1) = 18), in C's order, for both
// subcmd values. The ported code checked the flags before drawing, so it drew
// once for an eligible entry and never for the rest; reverting the flag/draw
// order leaves the shared per-hour stream short and turns this test red.
func TestDreamTravelDrawOrderMatchesC(t *testing.T) {
	const drawsPerEntry = 2
	wantTotal := drawsPerEntry * (NumDreams + 1)
	for _, subcmd := range []int{0, 1} {
		seed := dreamTravelNonFiringSeed(t, subcmd)
		wantDraws, fired := dreamTravelOracleDraws(seed, subcmd)
		if fired || wantDraws != wantTotal {
			t.Fatalf("oracle seed %d subcmd %d: fired=%t draws=%d, want non-firing/%d",
				seed, subcmd, fired, wantDraws, wantTotal)
		}

		ref := dprng.New(seed)
		for i := 0; i < wantDraws; i++ {
			ref.Number(0, 15)
		}
		wantNext := ref.Number(0, 15)

		dprng.ResetStream(seed)
		t.Cleanup(func() { dprng.ResetStream(1) })
		ch := &fakeDreamContext{}
		DreamTravelFn(ch, subcmd)
		if ch.traveled {
			t.Fatalf("subcmd %d: the port fires for seed %d while C's draw order "+
				"leaves the sleeper dreaming — the port is not drawing in C's order "+
				"(src/dream.c:195,208)", subcmd, seed)
		}
		gotNext := dprng.Number(0, 15)
		if gotNext != wantNext {
			t.Fatalf("subcmd %d: DreamTravelFn fired for seed %d here while C's draw "+
				"order leaves the sleeper dreaming, so the next draw is %d, want %d "+
				"(C draws number(0,15) first and twice per non-firing entry, "+
				"src/dream.c:195,208)", subcmd, seed, gotNext, wantNext)
		}
	}
}
