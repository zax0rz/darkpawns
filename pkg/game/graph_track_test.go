package game

// do_track port tests — one per gate and branch of src/graph.c:187-258.
// Every draw the handler makes goes through the trackNumber seam, so each
// test pins the exact draw sequence (R3 draw parity).

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// newTrackWorld builds the track fixture:
//
//	100 Crossroads --north--> 101 North Room   (open)
//	100 Crossroads --east---> 102 East Room    (open)
//	100 Crossroads --down---> 300 Cellar       (door, CLOSED)
//	200 Island (no exits, unreachable)
//	mob 2001 "beggar" (spawnable)
func newTrackWorld(t *testing.T) *World {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 100, Name: "Crossroads", Zone: 1, Exits: map[string]parser.Exit{
				"north": {Direction: "north", ToRoom: 101},
				"east":  {Direction: "east", ToRoom: 102},
				"down": {
					Direction: "down", ToRoom: 300, DoorState: 1,
					ExitInfo: parser.ExitIsDoor | parser.ExitClosed,
				},
			}},
			{VNum: 101, Name: "North Room", Zone: 1, Exits: map[string]parser.Exit{
				"south": {Direction: "south", ToRoom: 100},
			}},
			{VNum: 102, Name: "East Room", Zone: 1, Exits: map[string]parser.Exit{
				"west": {Direction: "west", ToRoom: 100},
			}},
			{VNum: 200, Name: "Island", Zone: 1},
			{VNum: 300, Name: "Cellar", Zone: 1, Exits: map[string]parser.Exit{
				"up": {
					Direction: "up", ToRoom: 100, DoorState: 1,
					ExitInfo: parser.ExitIsDoor | parser.ExitClosed,
				},
			}},
		},
		Mobs: []parser.Mob{
			{VNum: 2001, Keywords: "beggar", ShortDesc: "a ragged beggar"},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	return w
}

// newTracker returns a warrior with the track skill at 75 in roomVNum and
// captures everything the world sends to them.
func newTracker(t *testing.T, w *World, roomVNum int) (*Player, *strings.Builder) {
	t.Helper()
	var out strings.Builder
	w.MessageSink = func(playerName string, msg []byte) {
		if playerName == "Egon" {
			out.Write(msg)
		}
	}
	p := NewPlayer(1, "Egon", roomVNum)
	p.Class = ClassWarrior
	p.SetSkill("track", 75)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	return p, &out
}

// stubTrackNumber feeds doTrack's draws from a fixed sequence and fails the
// test on any extra draw.
func stubTrackNumber(t *testing.T, draws ...int) {
	t.Helper()
	old := trackNumber
	i := 0
	trackNumber = func(lo, hi int) int {
		if i >= len(draws) {
			t.Fatalf("trackNumber draw %d past sequence %v", i, draws)
		}
		v := draws[i]
		i++
		if v < lo || v > hi {
			t.Fatalf("trackNumber draw %d = %d outside [%d,%d]", i-1, v, lo, hi)
		}
		return v
	}
	t.Cleanup(func() { trackNumber = old })
}

func newVictim(t *testing.T, w *World, roomVNum int) *Player {
	t.Helper()
	p := NewPlayer(2, "Vic", roomVNum)
	if err := w.AddPlayer(p); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	return p
}

func TestTrackClassGate(t *testing.T) {
	w := newTrackWorld(t)
	p, out := newTracker(t, w, 100)
	p.Class = ClassMageUser

	w.doTrack(p, nil, "track", "vic")
	if got, want := out.String(), "You have no idea how.\r\n"; got != want {
		t.Fatalf("non-tracker class output = %q, want %q", got, want)
	}
}

func TestTrackSkillGate(t *testing.T) {
	w := newTrackWorld(t)
	p, out := newTracker(t, w, 100)
	p.SetSkill("track", 0)

	w.doTrack(p, nil, "track", "vic")
	if got, want := out.String(), "You have no idea how.\r\n"; got != want {
		t.Fatalf("skill-less output = %q, want %q", got, want)
	}
}

func TestTrackArgumentAndLookupGates(t *testing.T) {
	t.Run("no argument", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		w.doTrack(p, nil, "track", "")
		if got, want := out.String(), "Whom are you trying to track?\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})

	t.Run("target not found", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		w.doTrack(p, nil, "track", "nobody")
		if got, want := out.String(), "You can't sense a trail to them from here.\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})
}

func TestTrackNotrackAndSentinel(t *testing.T) {
	t.Run("AFF_NOTRACK player", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		vic := newVictim(t, w, 101)
		vic.SetAffect(affNoTrack, true)

		w.doTrack(p, nil, "track", "vic")
		if got, want := out.String(), "You sense no trail.\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})

	t.Run("sentinel mob", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		mob, err := w.SpawnMob(2001, 101)
		if err != nil {
			t.Fatalf("SpawnMob failed: %v", err)
		}
		mob.SetMobFlag(MobFlagSentinel)

		w.doTrack(p, nil, "track", "beggar")
		if got, want := out.String(), "You sense no trail.\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})
}

func TestTrackEvasionRoll(t *testing.T) {
	t.Run("roll at or under evasion hides the trail", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		vic := newVictim(t, w, 102)
		vic.SetSkill("evasion", 50)
		stubTrackNumber(t, 50) // number(1,151) == 50 <= 50

		w.doTrack(p, nil, "track", "vic")
		if got, want := out.String(), "You sense no trail.\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})

	t.Run("roll over evasion continues to the path", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		vic := newVictim(t, w, 102) // east of the tracker
		vic.SetSkill("evasion", 50)
		stubTrackNumber(t, 51, 0) // evasion 51 > 50; num 0 < 75 success

		w.doTrack(p, nil, "track", "vic")
		if got, want := out.String(), "You sense a trail east from here!\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})
}

func TestTrackBFSSentinels(t *testing.T) {
	t.Run("BFS_ALREADY_THERE", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		newVictim(t, w, 100)

		w.doTrack(p, nil, "track", "vic")
		if got, want := out.String(), "You're already in the same room!!\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})

	t.Run("BFS_NO_PATH names the victim by sex", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		vic := newVictim(t, w, 200) // Island: unreachable
		vic.SetSex(SexFemale)

		w.doTrack(p, nil, "track", "vic")
		if got, want := out.String(), "You can't sense a trail to her from here.\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})

	t.Run("BFS_ERROR", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		newVictim(t, w, 101)
		p.SetRoom(999999) // src room outside the world

		w.doTrack(p, nil, "track", "vic")
		if got, want := out.String(), "Hmm.. something seems to be wrong.\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})
}

func TestTrackSuccessNamesDirection(t *testing.T) {
	for _, tc := range []struct {
		victimRoom int
		want       string
	}{
		{101, "You sense a trail north from here!\r\n"},
		{102, "You sense a trail east from here!\r\n"},
	} {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		newVictim(t, w, tc.victimRoom)
		stubTrackNumber(t, 0) // num 0 < skill 75: success, no random direction

		w.doTrack(p, nil, "track", "vic")
		if got := out.String(); got != tc.want {
			t.Fatalf("victim in %d: output = %q, want %q", tc.victimRoom, got, tc.want)
		}
	}
}

func TestTrackFailureRollPicksRandomDirection(t *testing.T) {
	w := newTrackWorld(t)
	p, out := newTracker(t, w, 100)
	newVictim(t, w, 102) // true path is east
	// num 100 >= skill 75: failure; the retry loop then draws 0 (north),
	// an open exit — so the tracker is sent the WRONG direction. That is
	// C's behavior (src/graph.c:241-251), not a bug to smooth over.
	stubTrackNumber(t, 100, 0)

	w.doTrack(p, nil, "track", "vic")
	if got, want := out.String(), "You sense a trail north from here!\r\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestTrackNoWayOut(t *testing.T) {
	t.Run("BFS paths through the closed door but CAN_GO fails", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		newVictim(t, w, 300)  // Cellar, below, behind the closed door
		stubTrackNumber(t, 0) // success roll; dir stays the BFS step (down)

		w.doTrack(p, nil, "track", "vic")
		if got, want := out.String(), "There doesn't seem to be any way out of here!\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})

	t.Run("failure retry exhausts ten draws with no open exit", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		newVictim(t, w, 300)
		// Failure roll, then ten draws of 5 (down): closed every time.
		stubTrackNumber(t, 100, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5)

		w.doTrack(p, nil, "track", "vic")
		if got, want := out.String(), "There doesn't seem to be any way out of here!\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})
}

func TestTrackRainPenalty(t *testing.T) {
	restore := func() { weatherInfo.Sky = SkyCloudless }
	t.Cleanup(restore)

	t.Run("rain adds 35 to the roll", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		p.SetSkill("track", 100)
		newVictim(t, w, 102)
		weatherInfo.Sky = SkyRaining
		// num 80 < 100 clear; 80+35 = 115 >= 100 in rain: failure branch,
		// whose direction draw (0, north) is open — a different line than
		// the success branch's BFS answer (east).
		stubTrackNumber(t, 80, 0)

		w.doTrack(p, nil, "track", "vic")
		if got, want := out.String(), "You sense a trail north from here!\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})

	t.Run("indoor tracker gets no rain penalty", func(t *testing.T) {
		w := newTrackWorld(t)
		p, out := newTracker(t, w, 100)
		p.SetSkill("track", 100)
		newVictim(t, w, 102)
		weatherInfo.Sky = SkyRaining
		// Make the tracker's room ROOM_INDOORS|SECT_INSIDE (bit 3, word 0).
		if room, ok := w.GetRoom(100); ok {
			room.Flags = []string{"8", "0", "0", "0"}
			room.Sector = SECT_INSIDE
		}
		stubTrackNumber(t, 80) // 80 < 100: success, BFS east

		w.doTrack(p, nil, "track", "vic")
		if got, want := out.String(), "You sense a trail east from here!\r\n"; got != want {
			t.Fatalf("output = %q, want %q", got, want)
		}
	})
}
