package game

// mob_hunt_voice_depth_test.go — DP-1440 defect 4: hunt_victim's taunts
// (src/graph.c:262-404). Every case drives the real hunt step and reads each
// player's bytes plus review's output.

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// huntVoiceWorld is a two-room world (1001 north of 1002) with a hunting
// mobile in 1001, the hunted player in 1002, and a listener beside the hunter.
func huntVoiceWorld(t *testing.T) (*World, *MobInstance, *Player, *Player, map[string]string) {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Hunter Room", Zone: 1, Exits: map[string]parser.Exit{
			"north": {Direction: "north", ToRoom: 1002},
		}},
		{VNum: 1002, Name: "Middle Room", Zone: 1, Exits: map[string]parser.Exit{
			"south": {Direction: "south", ToRoom: 1001},
			"north": {Direction: "north", ToRoom: 1004},
		}},
		{VNum: 1004, Name: "Prey Room", Zone: 1, Exits: map[string]parser.Exit{
			"south": {Direction: "south", ToRoom: 1002},
		}},
		{VNum: 1003, Name: "Far Room", Zone: 2},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	w.StopAITicker()
	t.Cleanup(w.StopAITicker)
	out := map[string]string{}
	w.MessageSink = func(name string, msg []byte) { out[name] += string(msg) }

	mob := NewMob(&parser.Mob{
		VNum: 9701, Keywords: "stalker", ShortDesc: "the dune stalker",
		Race: 1, Level: 20, Position: 8, DefaultPos: 8,
	}, 1001)
	w.mu.Lock()
	w.activeMobs[mob.ID] = mob
	w.mu.Unlock()

	prey := NewPlayer(1, "Prey", 1004)
	prey.SetPosition(8)
	if err := w.AddPlayer(prey); err != nil {
		t.Fatalf("AddPlayer prey: %v", err)
	}
	watcher := NewPlayer(2, "Watcher", 1001)
	watcher.SetPosition(8)
	if err := w.AddPlayer(watcher); err != nil {
		t.Fatalf("AddPlayer watcher: %v", err)
	}
	return w, mob, prey, watcher, out
}

// scriptHuntDraws pins the hunt's number() draws to a fixed sequence.
func scriptHuntDraws(t *testing.T, results ...int) {
	t.Helper()
	previous := huntNumber
	idx := 0
	huntNumber = func(from, to int) int {
		if idx >= len(results) {
			t.Fatalf("unexpected extra hunt draw number(%d,%d)", from, to)
		}
		v := results[idx]
		idx++
		return v
	}
	t.Cleanup(func() { huntNumber = previous })
}

// TestHuntVictimSayReachesTheRoom is DP-1440's "says go to the mob, whose
// SendMessage is a no-op" defect: C's do_say (src/graph.c:282) is an act to the
// room.
func TestHuntVictimSayReachesTheRoom(t *testing.T) {
	w, mob, _, _, out := huntVoiceWorld(t)
	// A hunting target that is not in the world takes the "prey is gone" arm.
	mob.SetHunting("Nobodyhere")
	scriptHuntDraws(t)

	w.huntVictim(mob)

	want := "The dune stalker exclaims, 'Damn!  My prey is gone!!'\r\n"
	if got := out["Watcher"]; !strings.Contains(got, want) {
		t.Fatalf("hunter's room bytes = %q, want %q among them", got, want)
	}
	if got := out[mob.GetShortDesc()]; got != "" {
		t.Fatalf("the mobile received its own say: %q", got)
	}
}

// TestHuntVictimEvasionSayReachesTheRoom covers hunt_victim's second say arm
// (src/graph.c:293), which the evasion gate reaches.
func TestHuntVictimEvasionSayReachesTheRoom(t *testing.T) {
	w, mob, prey, _, out := huntVoiceWorld(t)
	prey.SetSkill("evasion", 100)
	mob.SetHunting(prey.Name)
	// number(1,151) below evasion, then the first number(0,6) fires.
	scriptHuntDraws(t, 1, 0)

	w.huntVictim(mob)

	want := "The dune stalker exclaims, 'Where the hell did my prey go?!'\r\n"
	if got := out["Watcher"]; !strings.Contains(got, want) {
		t.Fatalf("hunter's room bytes = %q, want %q among them", got, want)
	}
}

// TestHuntVictimTellUsesTheTellPath covers the tell arm: C prepends the prey's
// name as do_tell's target (src/graph.c:359-361), and perform_tell sets
// GET_LAST_TELL so reply works (src/act.comm.c:894).
func TestHuntVictimTellUsesTheTellPath(t *testing.T) {
	w, mob, prey, _, out := huntVoiceWorld(t)
	mob.SetHunting(prey.Name)
	scriptHuntDraws(t, 2) // case 2: "<name> Run to your momma, pansy!"

	w.huntVictim(mob)

	want := "The dune stalker tells you, 'Run to your momma, pansy!'\r\n"
	if got := out["Prey"]; !strings.Contains(got, want) {
		t.Fatalf("prey bytes = %q, want %q among them", got, want)
	}
	if got := prey.GetLastTeller(); got != mob.GetShortDesc() {
		t.Fatalf("GET_LAST_TELL = %q, want %q so reply reaches the hunter", got, mob.GetShortDesc())
	}
}

// TestHuntVictimGossipIsGlobalAndRecorded covers the gossip arms: C uses
// do_gen_comm(SCMD_GOSSIP) (src/graph.c:369, :373, :377), which is global and
// records review (src/act.comm.c:1275-1276). Case 4's text is
// "I hear %s thinks %s's bad." with HSSH (src/graph.c:367-368).
func TestHuntVictimGossipIsGlobalAndRecorded(t *testing.T) {
	w, mob, prey, watcher, out := huntVoiceWorld(t)
	mob.SetHunting(prey.Name)
	scriptHuntDraws(t, 4)

	w.huntVictim(mob)

	want := "The dune stalker gossips, 'I hear Prey thinks he's bad.'\r\n"
	for _, name := range []string{"Watcher", "Prey"} {
		if got := out[name]; !strings.Contains(got, want) {
			t.Fatalf("%s bytes = %q, want the global gossip %q", name, got, want)
		}
	}
	if got := w.ReviewGossip(watcher); !strings.Contains(got, "I hear Prey thinks he's bad.") {
		t.Fatalf("review = %q, want the taunt recorded", got)
	}
}

// TestHuntVictimAuctionAndShoutReachEveryListener covers the auction arm
// (src/graph.c:357, :388) and the shout arm (:398), which C sends through
// do_gen_comm with that channel's own audience.
func TestHuntVictimAuctionAndShoutReachEveryListener(t *testing.T) {
	t.Run("auction is global", func(t *testing.T) {
		w, mob, prey, _, out := huntVoiceWorld(t)
		mob.SetHunting(prey.Name)
		scriptHuntDraws(t, 9) // "How much will I get for the head of %s?"

		w.huntVictim(mob)

		want := "The dune stalker auctions, 'How much will I get for the head of Prey?'\r\n"
		for _, name := range []string{"Watcher", "Prey"} {
			if got := out[name]; !strings.Contains(got, want) {
				t.Fatalf("%s bytes = %q, want %q", name, got, want)
			}
		}
	})

	t.Run("shout is zone limited", func(t *testing.T) {
		w, mob, prey, _, out := huntVoiceWorld(t)
		far := NewPlayer(3, "Faraway", 1003)
		far.SetPosition(8)
		if err := w.AddPlayer(far); err != nil {
			t.Fatalf("AddPlayer far: %v", err)
		}
		mob.SetHunting(prey.Name)
		scriptHuntDraws(t, 12) // "Damn it!"

		w.huntVictim(mob)

		want := "The dune stalker shouts, 'Damn it!'\r\n"
		if got := out["Watcher"]; !strings.Contains(got, want) {
			t.Fatalf("same-zone bytes = %q, want %q", got, want)
		}
		if got := out["Faraway"]; strings.Contains(got, want) {
			t.Fatalf("a listener in another zone heard the shout: %q", got)
		}
	})
}

// TestExecuteMobCommandUsesTheMobileSpeechPaths is DP-1440's class audit
// (R5c): the legacy VNum-based mob command adapter's say, emote, gossip, tell,
// shout and auction cases must reach the same sinks as the native mobile
// commands, not hand-built lines of their own.
func TestExecuteMobCommandUsesTheMobileSpeechPaths(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command string
		want    string
		who     string
	}{
		{name: "say", command: "say hello there", want: "The dune stalker says, 'hello there'\r\n", who: "Watcher"},
		{name: "emote", command: "emote waves.", want: "The dune stalker waves.\r\n", who: "Watcher"},
		{name: "gossip", command: "gossip a rumor", want: "The dune stalker gossips, 'a rumor'\r\n", who: "Prey"},
		{name: "tell", command: "tell Prey run!", want: "The dune stalker tells you, 'run!'\r\n", who: "Prey"},
		{name: "shout", command: "shout rally", want: "The dune stalker shouts, 'rally'\r\n", who: "Watcher"},
		{name: "auction", command: "auction a blade", want: "The dune stalker auctions, 'a blade'\r\n", who: "Prey"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, mob, _, _, out := huntVoiceWorld(t)

			w.executeMobCommand(mob.GetVNum(), tc.command)

			if got := out[tc.who]; !strings.Contains(got, tc.want) {
				t.Fatalf("%s bytes = %q, want %q among them", tc.who, got, tc.want)
			}
		})
	}
}
