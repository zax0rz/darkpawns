package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newDirectedSpeechWorld(t *testing.T) (*World, *Player, *Player, map[string]*strings.Builder) {
	t.Helper()
	parsed := &parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Common Room"},
		{VNum: 1002, Name: "Elsewhere"},
		{VNum: 1003, Name: "Soundproof", Flags: []string{"32", "0", "0", "0"}},
	}}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	actor := directedSpeechPlayer(t, w, 1, "Actor", 1001)
	target := directedSpeechPlayer(t, w, 2, "Target", 1001)
	target.Sex = 1
	output := make(map[string]*strings.Builder)
	w.MessageSink = func(name string, message []byte) {
		if output[name] == nil {
			output[name] = &strings.Builder{}
		}
		output[name].Write(message)
	}
	return w, actor, target, output
}

func directedSpeechPlayer(t *testing.T, w *World, id int, name string, room int) *Player {
	t.Helper()
	player := NewPlayer(id, name, room)
	player.Stats.Int = 10
	player.Stats.Wis = 10
	if err := w.AddPlayer(player); err != nil {
		t.Fatal(err)
	}
	return player
}

func directedOutput(output map[string]*strings.Builder, name string) string {
	if output[name] == nil {
		return ""
	}
	return output[name].String()
}

func TestDoSayCBehavior(t *testing.T) {
	t.Run("mute", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		actor.Stats.Int = 0
		w.DoSay(actor, "hello")
		if got := directedOutput(output, actor.Name); got != "You are too stupid to communicate with language!\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("noshout", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		actor.SetPlrFlag(PlrNoshout, true)
		w.DoSay(actor, "hello")
		if got := directedOutput(output, actor.Name); got != "You cannot speak!\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("empty", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		w.DoSay(actor, "")
		if got := directedOutput(output, actor.Name); got != "Yes, but WHAT do you want to say?\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("drunk room text and sober self echo", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		actor.SetCondition(CondDrunk, 11)
		w.DoSay(actor, "what is the test")
		if got := directedOutput(output, target.Name); got != "Actor says, 'wha' ish th' theshth'\r\n" {
			t.Fatalf("target output = %q", got)
		}
		if got := directedOutput(output, actor.Name); got != "You say 'what is the test'\r\n" {
			t.Fatalf("actor output = %q", got)
		}
	})

	t.Run("punctuation and norepeat", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		actor.SetPlrFlag(PrfNoRepeat, true)
		w.DoSay(actor, "really?")
		if got := directedOutput(output, actor.Name); got != "Ok.\r\n" {
			t.Fatalf("actor output = %q", got)
		}
		if got := directedOutput(output, target.Name); got != "Actor asks, 'really?'\r\n" {
			t.Fatalf("target output = %q", got)
		}
	})
}

func TestDoTellEligibilityAndDelivery(t *testing.T) {
	t.Run("usage", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		w.DoTell(actor, "Target")
		if got := directedOutput(output, actor.Name); got != "Who do you wish to tell what??\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("not found and not visible", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		target.SetAffect(affInvisible, true)
		w.DoTell(actor, "Target hello")
		if got := directedOutput(output, actor.Name); got != "No-one by that name here.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("self", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		actor.SetAffect(affInvisible, true)
		w.DoTell(actor, "Actor hello")
		if got := directedOutput(output, actor.Name); got != "You try to tell yourself something.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("sender notell", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		actor.SetPlrFlag(PrfNotell, true)
		w.DoTell(actor, "Target hello")
		if got := directedOutput(output, actor.Name); got != "You can't tell other people while you have notell on.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("sender noshout", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		actor.SetPlrFlag(PlrNoshout, true)
		w.DoTell(actor, "Target hello")
		if got := directedOutput(output, actor.Name); got != "You cannot tell anyone anything!\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("sender soundproof", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		actor.SetRoom(1003)
		w.DoTell(actor, "Target hello")
		if got := directedOutput(output, actor.Name); got != "The walls seem to absorb your words.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("target notell", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		target.SetPlrFlag(PrfNotell, true)
		w.DoTell(actor, "Target hello")
		if got := directedOutput(output, actor.Name); got != "She can't hear you.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("target soundproof", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		target.SetRoom(1003)
		w.DoTell(actor, "Target hello")
		if got := directedOutput(output, actor.Name); got != "She can't hear you.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("target writing", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		target.SetPlrFlag(PlrWriting, true)
		w.DoTell(actor, "Target hello")
		if got := directedOutput(output, actor.Name); got != "She's writing a message right now; try again later.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("target linkless", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		target.SetLinkless(true)
		w.DoTell(actor, "Target hello")
		if got := directedOutput(output, actor.Name); got != "She's linkless at the moment.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("delivery strips ansi and records reply", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		w.DoTell(actor, "Target &Rhello")
		if got := directedOutput(output, actor.Name); got != "You tell Target, 'Rhello'\r\n" {
			t.Fatalf("actor output = %q", got)
		}
		if got := directedOutput(output, target.Name); got != "Actor tells you, 'Rhello'\r\n" {
			t.Fatalf("target output = %q", got)
		}
		if got := target.GetLastTeller(); got != actor.Name {
			t.Fatalf("last teller = %q", got)
		}
	})

	t.Run("norepeat", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		actor.SetPlrFlag(PrfNoRepeat, true)
		w.DoTell(actor, "Target hello")
		if got := directedOutput(output, actor.Name); got != "Okay.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})
}

func TestDoReplyCGates(t *testing.T) {
	t.Run("none", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		w.DoReply(actor, "hello")
		if got := directedOutput(output, actor.Name); got != "You have no-one to reply to!\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("empty", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		actor.SetLastTeller(target.Name)
		w.DoReply(actor, "")
		if got := directedOutput(output, actor.Name); got != "What is your reply?\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("sender notell does not block reply", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		actor.SetLastTeller(target.Name)
		actor.SetPlrFlag(PrfNotell, true)
		w.DoReply(actor, "hello")
		if got := directedOutput(output, target.Name); got != "Actor tells you, 'hello'\r\n" {
			t.Fatalf("target output = %q", got)
		}
	})

	t.Run("target notell", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		actor.SetLastTeller(target.Name)
		target.SetPlrFlag(PrfNotell, true)
		w.DoReply(actor, "hello")
		if got := directedOutput(output, actor.Name); got != "They can't hear you.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("target no longer playing", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		actor.SetLastTeller(target.Name)
		w.RemovePlayer(target.Name)
		w.DoReply(actor, "hello")
		if got := directedOutput(output, actor.Name); got != "They are no longer playing.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("target writing", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		actor.SetLastTeller(target.Name)
		target.SetPlrFlag(PlrWriting, true)
		w.DoReply(actor, "hello")
		if got := directedOutput(output, actor.Name); got != "They are writing now, try later.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("sender noshout", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		actor.SetLastTeller(target.Name)
		actor.SetPlrFlag(PlrNoshout, true)
		w.DoReply(actor, "hello")
		if got := directedOutput(output, actor.Name); got != "You cannot tell anyone anything!\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("sender soundproof", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		actor.SetLastTeller(target.Name)
		actor.SetRoom(1003)
		w.DoReply(actor, "hello")
		if got := directedOutput(output, actor.Name); got != "The walls seem to absorb your words.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("target soundproof", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		actor.SetLastTeller(target.Name)
		target.SetRoom(1003)
		w.DoReply(actor, "hello")
		if got := directedOutput(output, actor.Name); got != "She can't hear you.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})
}

func TestDoSpecCommCBehavior(t *testing.T) {
	t.Run("ask all audiences and ansi stripping", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		observer := directedSpeechPlayer(t, w, 3, "Observer", 1001)
		w.DoSpecComm(actor, "Target what &Ris it", true)
		if got := directedOutput(output, actor.Name); got != "You ask Target, 'what Ris it'\r\n" {
			t.Fatalf("actor output = %q", got)
		}
		if got := directedOutput(output, target.Name); got != "Actor asks you, 'what Ris it'\r\n" {
			t.Fatalf("target output = %q", got)
		}
		if got := directedOutput(output, observer.Name); got != "Actor asks Target a question.\r\n" {
			t.Fatalf("observer output = %q", got)
		}
	})

	t.Run("self by character name", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		w.DoSpecComm(actor, "Actor muttering", false)
		if got := directedOutput(output, actor.Name); got != "You can't get your mouth close enough to your ear...\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("not found", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		w.DoSpecComm(actor, "Nobody hello", true)
		if got := directedOutput(output, actor.Name); got != "No-one by that name here.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	for _, tc := range []struct {
		name string
		ask  bool
		want string
	}{
		{name: "whisper usage", want: "Whom do you want to whisper to.. and what??\r\n"},
		{name: "ask usage", ask: true, want: "Whom do you want to ask.. and what??\r\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, actor, _, output := newDirectedSpeechWorld(t)
			w.DoSpecComm(actor, "", tc.ask)
			if got := directedOutput(output, actor.Name); got != tc.want {
				t.Fatalf("output = %q", got)
			}
		})
	}

	t.Run("noshout precedes usage", func(t *testing.T) {
		w, actor, _, output := newDirectedSpeechWorld(t)
		actor.SetPlrFlag(PlrNoshout, true)
		w.DoSpecComm(actor, "", false)
		if got := directedOutput(output, actor.Name); got != "Sorry, you cannot do that.\r\n" {
			t.Fatalf("output = %q", got)
		}
	})

	t.Run("norepeat", func(t *testing.T) {
		w, actor, target, output := newDirectedSpeechWorld(t)
		actor.SetPlrFlag(PrfNoRepeat, true)
		w.DoSpecComm(actor, "Target hello", false)
		if got := directedOutput(output, actor.Name); got != "Okay.\r\n" {
			t.Fatalf("actor output = %q", got)
		}
		if got := directedOutput(output, target.Name); got != "Actor whispers to you, 'hello'\r\n" {
			t.Fatalf("target output = %q", got)
		}
	})
}
