package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newChannelWorld(t *testing.T) (*World, *Player, *Player, *Player, map[string]*strings.Builder) {
	t.Helper()
	parsed := &parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Town Square", Zone: 1},
		{VNum: 1002, Name: "Soundproof", Zone: 1, Flags: []string{"32", "0", "0", "0"}},
		{VNum: 2001, Name: "Other Zone", Zone: 2},
	}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	actor := channelPlayer(t, w, 1, "Actor", 1001)
	local := channelPlayer(t, w, 2, "Local", 1001)
	remote := channelPlayer(t, w, 3, "Remote", 2001)
	output := make(map[string]*strings.Builder)
	w.MessageSink = func(name string, message []byte) {
		if output[name] == nil {
			output[name] = &strings.Builder{}
		}
		output[name].Write(message)
	}
	return w, actor, local, remote, output
}

func channelPlayer(t *testing.T, w *World, id int, name string, room int) *Player {
	t.Helper()
	player := NewPlayer(id, name, room)
	player.Level = levelCanShout
	player.Stats.Int = 10
	player.Stats.Wis = 10
	player.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(player); err != nil {
		t.Fatal(err)
	}
	return player
}

func channelOutput(output map[string]*strings.Builder, name string) string {
	if output[name] == nil {
		return ""
	}
	return output[name].String()
}

func TestDoChannelSenderGatesMatchC(t *testing.T) {
	t.Run("minimum levels", func(t *testing.T) {
		for _, channel := range []string{"gossip", "shout"} {
			t.Run(channel, func(t *testing.T) {
				w, actor, local, _, output := newChannelWorld(t)
				actor.Level = 1
				w.DoChannel(actor, "hello", channel)
				want := "You must be at least level 2 before you can " + channel + ".\r\n"
				if got := channelOutput(output, actor.Name); got != want {
					t.Fatalf("actor output = %q, want %q", got, want)
				}
				if got := channelOutput(output, local.Name); got != "" {
					t.Fatalf("recipient received blocked channel: %q", got)
				}
			})
		}
	})

	t.Run("noshout", func(t *testing.T) {
		w, actor, _, _, output := newChannelWorld(t)
		actor.SetPlrFlag(PlrNoshout, true)
		w.DoChannel(actor, "hello", "gossip")
		if got := channelOutput(output, actor.Name); got != "You cannot gossip!!\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("soundproof", func(t *testing.T) {
		w, actor, _, _, output := newChannelWorld(t)
		actor.SetRoom(1002)
		w.DoChannel(actor, "hello", "shout")
		if got := channelOutput(output, actor.Name); got != "The walls seem to absorb your words.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("sender has channel off", func(t *testing.T) {
		w, actor, _, _, output := newChannelWorld(t)
		actor.SetPlrFlag(PrfNoGossip, true)
		w.DoChannel(actor, "hello", "gossip")
		if got := channelOutput(output, actor.Name); got != "You aren't even on the channel!\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("norepeat uses C okay text", func(t *testing.T) {
		w, actor, _, _, output := newChannelWorld(t)
		actor.SetPlrFlag(PrfNoRepeat, true)
		w.DoChannel(actor, "hello", "gossip")
		if got := channelOutput(output, actor.Name); got != "Okay.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})
}

func TestDoChannelFanoutMatchesC(t *testing.T) {
	t.Run("gossip is global with exact wording", func(t *testing.T) {
		w, actor, local, remote, output := newChannelWorld(t)
		w.DoChannel(actor, "hello world", "gossip")
		if got := channelOutput(output, actor.Name); got != "You gossip, 'hello world'\r\n" {
			t.Fatalf("actor output = %q", got)
		}
		for _, target := range []*Player{local, remote} {
			if got := channelOutput(output, target.Name); got != "Actor gossips, 'hello world'\r\n" {
				t.Fatalf("%s output = %q", target.Name, got)
			}
		}
	})

	t.Run("recipient eligibility", func(t *testing.T) {
		w, actor, offChannel, writing, output := newChannelWorld(t)
		offChannel.SetPlrFlag(PrfNoGossip, true)
		writing.SetPlrFlag(PlrWriting, true)
		soundproof := channelPlayer(t, w, 4, "Soundproof", 1002)
		eligible := channelPlayer(t, w, 5, "Eligible", 2001)
		w.DoChannel(actor, "hello", "gossip")
		for _, target := range []*Player{offChannel, writing, soundproof} {
			if got := channelOutput(output, target.Name); got != "" {
				t.Fatalf("%s received gossip: %q", target.Name, got)
			}
		}
		if got := channelOutput(output, eligible.Name); got != "Actor gossips, 'hello'\r\n" {
			t.Fatalf("eligible output = %q", got)
		}
	})

	t.Run("shout is zone limited and requires resting hearers", func(t *testing.T) {
		w, actor, local, remote, output := newChannelWorld(t)
		sleeping := channelPlayer(t, w, 4, "Sleeping", 1001)
		sleeping.SetPosition(combat.PosSleeping)
		w.DoChannel(actor, "hello", "shout")
		if got := channelOutput(output, local.Name); got != "Actor shouts, 'hello'\r\n" {
			t.Fatalf("local output = %q", got)
		}
		for _, target := range []*Player{remote, sleeping} {
			if got := channelOutput(output, target.Name); got != "" {
				t.Fatalf("%s received shout: %q", target.Name, got)
			}
		}
	})
}

func TestDoChannelFamilyGatesAndHoller(t *testing.T) {
	t.Run("shout deafness is recipient-only", func(t *testing.T) {
		w, actor, local, _, output := newChannelWorld(t)
		actor.SetPlrFlag(PrfDeaf, true)
		w.DoChannel(actor, "hello", "shout")
		if got := channelOutput(output, actor.Name); got != "You shout, 'hello'\r\n" {
			t.Fatalf("actor output = %q", got)
		}
		if got := channelOutput(output, local.Name); got != "Actor shouts, 'hello'\r\n" {
			t.Fatalf("local output = %q", got)
		}
	})

	t.Run("recipient channel flags are per-channel", func(t *testing.T) {
		for _, test := range []struct {
			name string
			flag int
			cmd  string
			verb string
		}{
			{name: "gossip", flag: PrfNoGossip, cmd: "gossip", verb: "gossip"},
			{name: "auction", flag: PrfNoAuctions, cmd: "auction", verb: "auction"},
			{name: "grats", flag: PrfNoGratz, cmd: "grats", verb: "congrat"},
		} {
			t.Run(test.name, func(t *testing.T) {
				w, actor, recipient, _, output := newChannelWorld(t)
				recipient.SetPlrFlag(test.flag, true)
				w.DoChannel(actor, "hello", test.cmd)
				if got := channelOutput(output, actor.Name); got != "You "+test.verb+", 'hello'\r\n" {
					t.Fatalf("actor output = %q", got)
				}
				if got := channelOutput(output, recipient.Name); got != "" {
					t.Fatalf("recipient output = %q", got)
				}
			})
		}
	})

	t.Run("holler is global and costs twenty move", func(t *testing.T) {
		w, actor, local, remote, output := newChannelWorld(t)
		actor.SetMove(50)
		w.DoChannel(actor, "hello", "holler")
		if got := channelOutput(output, actor.Name); got != "You holler, 'hello'\r\n" {
			t.Fatalf("actor output = %q", got)
		}
		for _, target := range []*Player{local, remote} {
			if got := channelOutput(output, target.Name); got != "Actor hollers, 'hello'\r\n" {
				t.Fatalf("%s output = %q", target.Name, got)
			}
		}
		if got := actor.GetMove(); got != 30 {
			t.Fatalf("move = %d, want 30", got)
		}
	})

	t.Run("holler exhaustion is a sender gate", func(t *testing.T) {
		w, actor, local, _, output := newChannelWorld(t)
		actor.SetMove(hollerMoveCost - 1)
		w.DoChannel(actor, "hello", "holler")
		if got := channelOutput(output, actor.Name); got != "You're too exhausted to holler.\r\n" {
			t.Fatalf("actor output = %q", got)
		}
		if got := channelOutput(output, local.Name); got != "" {
			t.Fatalf("recipient output = %q", got)
		}
	})

	t.Run("minimum level applies to every requested channel", func(t *testing.T) {
		for _, channel := range []string{"gossip", "shout", "auction", "grats", "holler"} {
			t.Run(channel, func(t *testing.T) {
				w, actor, local, _, output := newChannelWorld(t)
				actor.Level = levelCanShout - 1
				w.DoChannel(actor, "hello", channel)
				want := "You must be at least level 2 before you can " + communicationChannels[channel].verb + ".\r\n"
				if got := channelOutput(output, actor.Name); got != want {
					t.Fatalf("actor output = %q, want %q", got, want)
				}
				if got := channelOutput(output, local.Name); got != "" {
					t.Fatalf("recipient output = %q", got)
				}
			})
		}
	})
}

func TestSocialTableMapsCHideAndVictimPositionFields(t *testing.T) {
	dance := Socials["dance"]
	if dance == nil {
		t.Fatal("dance social missing")
	}
	if !dance.hidesInvisibleActor() || dance.minimumVictimPosition() != combat.PosStanding {
		t.Fatalf("dance gates = hide %t, victim position %d; want true/%d", dance.hidesInvisibleActor(), dance.minimumVictimPosition(), combat.PosStanding)
	}

	w, actor, target, _, output := newChannelWorld(t)
	target.SetPosition(combat.PosSleeping)
	DoAction(w, actor, "dance", target.Name)
	if got := channelOutput(output, actor.Name); got != "Local is not in a proper position for that.\r\n" {
		t.Fatalf("actor output = %q", got)
	}
}

func TestDoActionSleepingActorReceivesMessages(t *testing.T) {
	t.Run("position-fail message reaches sleeping actor", func(t *testing.T) {
		w, actor, target, _, output := newChannelWorld(t)
		actor.SetPosition(combat.PosSleeping)
		target.SetPosition(combat.PosSleeping)
		DoAction(w, actor, "dance", target.Name)
		// Sleeping actor cannot see the target, so $N resolves to "someone".
		if got := channelOutput(output, actor.Name); got != "Someone is not in a proper position for that.\r\n" {
			t.Fatalf("actor output = %q, want position-fail message", got)
		}
	})

	t.Run("char-found message reaches sleeping actor", func(t *testing.T) {
		w, actor, target, _, output := newChannelWorld(t)
		actor.SetPosition(combat.PosSleeping)
		DoAction(w, actor, "dance", target.Name)
		// Sleeping actor cannot see the target, so $M resolves to "him".
		if got := channelOutput(output, actor.Name); got != "You lead him to the dancefloor.\r\n" {
			t.Fatalf("actor output = %q, want char-found message", got)
		}
	})
}
