package session

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func newGateDepthFixture(t *testing.T, room int) (*game.World, *Manager, *Session) {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 1001, Name: "Outside the grove", Description: "A quiet road."},
			{VNum: 4000, Name: "The Void", Description: "An empty void."},
			{VNum: 4001, Name: "New Moon Grove", Description: "A grove beneath a new moon."},
			{VNum: 4002, Name: "Quarter Moon Grove", Description: "A grove beneath a quarter moon."},
			{VNum: 4004, Name: "Three Quarter Grove", Description: "A grove beneath a three-quarter moon."},
			{VNum: 4005, Name: "Full Moon Grove", Description: "A grove beneath a full moon."},
			{VNum: 4006, Name: "Waning Grove", Description: "A grove beneath a waning moon."},
			{VNum: 4007, Name: "Half Waning Grove", Description: "A grove beneath a half-waning moon."},
			{VNum: 4008, Name: "Quarter Waning Grove", Description: "A grove beneath a quarter-waning moon."},
		},
		Objs: []parser.Obj{
			{VNum: game.BluePortalVNum, Keywords: "portal gate", ShortDesc: "a blue portal", LongDesc: "A blue portal shimmers here."},
			{VNum: game.RedPortalVNum, Keywords: "portal gate", ShortDesc: "a red portal", LongDesc: "A red portal shimmers here."},
		},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	m := newTestManager(t, w, nil)
	m.WireCombatCallbacks()
	actor := makeTestSession(t, m, "Gateactor", room, true)
	actor.player.SetPosition(combat.PosStanding)
	actor.player.SetSex(game.SexMale)
	actor.player.Class = game.ClassMagus
	actor.player.Stats.Int = 10
	actor.player.Stats.Wis = 10
	actor.player.CopyBaseAttributes()
	actor.player.SetLevel(30)
	actor.player.SetMana(1000)
	actor.player.SetSkill("gate", 100)
	registerTestSession(t, m, actor, "gateactor")
	return w, m, actor
}

func addGateObserver(t *testing.T, m *Manager, name string, room int) *Session {
	t.Helper()
	peer := makeTestSession(t, m, name, room, true)
	peer.player.SetPosition(combat.PosStanding)
	registerTestSession(t, m, peer, strings.ToLower(name))
	return peer
}

func allowGateCast(t *testing.T) {
	t.Helper()
	original := castNumber
	castNumber = func(minValue, _ int) int { return minValue }
	t.Cleanup(func() { castNumber = original })
}

func TestGatePortalCommandsUseAssignedObjectSpecial(t *testing.T) {
	w, m, actor := newGateDepthFixture(t, 4001)
	origin := addGateObserver(t, m, "Gateorigin", 4001)
	destination := addGateObserver(t, m, "Gatedestination", 4004)
	portal, err := w.SpawnObject(game.BluePortalVNum, 4001)
	if err != nil {
		t.Fatalf("SpawnObject blue portal: %v", err)
	}
	w.AddItemToRoom(portal, 4001)

	if err := executeCommand(actor, "look", []string{"portal"}, false); err != nil {
		t.Fatalf("look portal: %v", err)
	}
	if got := strings.Join(drainSessionText(t, actor), ""); !strings.Contains(got, "You see a shimmering light... you could ENTER the gate.\r\n") {
		t.Fatalf("look portal output = %q", got)
	}

	roller := combat.NewScriptedRoller([]int{0, 0}) // C number(0, 3) selects Exit1.
	combat.WithRoller(roller, func() {
		if err := executeCommand(actor, "enter", []string{"portal"}, false); err != nil {
			t.Fatalf("enter portal: %v", err)
		}
	})
	if actor.player.GetRoomVNum() != 4004 {
		t.Fatalf("destination room = %d, want C's Exit1 room 4004", actor.player.GetRoomVNum())
	}
	if roller.Index != 1 {
		t.Fatalf("blue portal draws = %d, want one number(0, 3)", roller.Index)
	}
	if got := strings.Join(drainSessionText(t, actor), ""); !strings.Contains(got, "You enter the portal and are transferred...\n\r") {
		t.Fatalf("actor transfer output = %q", got)
	}
	if got := strings.Join(drainSessionText(t, origin), ""); got != "Gateactor enters the gate which suddenly starts glowing brightly,\r\nobscuring your view of him!\r\n" {
		t.Fatalf("origin audience = %q", got)
	}
	if got := strings.Join(drainSessionText(t, destination), ""); got != "With a sound like a raging river, a hole in space opens up and Gateactor steps out.\r\n" {
		t.Fatalf("destination audience = %q", got)
	}
}

func TestGateSpellManualDispatchOutputsAndTimer(t *testing.T) {
	allowGateCast(t)
	t.Run("invalid room", func(t *testing.T) {
		_, _, actor := newGateDepthFixture(t, 1001)
		if err := executeCommand(actor, "cast", []string{"'gate'"}, false); err != nil {
			t.Fatalf("cast gate: %v", err)
		}
		if got := strings.Join(drainSessionText(t, actor), ""); !strings.Contains(got, "The magic flows through you, but nothing else happens.\r\n") {
			t.Fatalf("invalid-room output = %q", got)
		}
		if objects := actor.manager.world.GetItemsInRoom(1001); len(objects) != 0 {
			t.Fatalf("invalid-room objects = %d, want none", len(objects))
		}
	})

	for _, tc := range []struct {
		name  string
		level int
		timer int
	}{
		{name: "exactly level 30", level: 30, timer: 3},
		{name: "above level 30", level: 31, timer: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, m, actor := newGateDepthFixture(t, 4001)
			actor.player.SetLevel(tc.level)
			peer := addGateObserver(t, m, "Gategrovepeer", 4001)
			if err := executeCommand(actor, "cast", []string{"'gate'"}, false); err != nil {
				t.Fatalf("cast gate: %v", err)
			}
			if got := strings.Join(drainSessionText(t, actor), ""); !strings.Contains(got, "The magick flows through you, then out into the world, changing it....\r\n") {
				t.Fatalf("actor output = %q", got)
			}
			if got := strings.Join(drainSessionText(t, peer), ""); got != "Gateactor utters the words, 'ti'.\r\nA shimmering red portal fades into existence.\r\n" {
				t.Fatalf("room audience = %q", got)
			}
			objects := w.GetItemsInRoom(4001)
			if len(objects) != 1 || objects[0].GetVNum() != game.RedPortalVNum || objects[0].GetValue(2) != tc.timer {
				t.Fatalf("created portal state = %#v, want one red portal with timer %d", objects, tc.timer)
			}
		})
	}

	t.Run("portal collision", func(t *testing.T) {
		w, m, actor := newGateDepthFixture(t, 4001)
		peer := addGateObserver(t, m, "Gatecollisionpeer", 4001)
		startExp, startDeaths := actor.player.GetExp(), actor.player.Deaths
		portal, err := w.SpawnObject(game.BluePortalVNum, 4001)
		if err != nil {
			t.Fatalf("SpawnObject blue portal: %v", err)
		}
		w.AddItemToRoom(portal, 4001)
		if err := executeCommand(actor, "cast", []string{"'gate'"}, false); err != nil {
			t.Fatalf("cast gate: %v", err)
		}
		actorText := strings.Join(drainSessionText(t, actor), "")
		if !strings.Contains(actorText, "The magick flows through you, then out into the world, changing it....\r\n") {
			t.Fatalf("collision actor prelude = %q", actorText)
		}
		if !strings.Contains(actorText, "In your final moments, the only thing you can feel is a\r\nwave of cosmic energy coursing through you, tearing your soul to shreds.\r\n") {
			t.Fatalf("collision actor finale = %q", actorText)
		}
		if got := strings.Join(drainSessionText(t, peer), ""); got != "Gateactor utters the words, 'ti'.\r\nAs you watch the red portal slowly fade into existence, \r\nthe existing portal pulses once, then begins to expand, \r\nconsuming the new portal, and then the entire grove.\r\nThe fabric of time and space warps and stretches\r\naround you, and flashes of white light explode in the back of your mind.\r\n" {
			t.Fatalf("collision room output = %q", got)
		}
		if actor.player.GetExp() != startExp || actor.player.Deaths != startDeaths {
			t.Fatalf("raw_kill applied death penalties: exp %d->%d deaths %d->%d", startExp, actor.player.GetExp(), startDeaths, actor.player.Deaths)
		}
		if objects := w.GetItemsInRoom(4001); len(objects) != 1 || !objects[0].IsCorpse || objects[0].GetLongDesc() != "A blasted corpse lies here in pieces." {
			t.Fatalf("collision room objects = %#v, want only the TYPE_BLAST corpse", objects)
		}
	})
}
