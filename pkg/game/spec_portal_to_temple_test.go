package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type portalToTempleTestWorld struct {
	w        *World
	actor    *Player
	peer     *Player
	messages map[string]string
}

func newPortalToTempleTestWorld(t *testing.T) portalToTempleTestWorld {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 19658, Name: "Tower", Description: "A tall tower."},
		{VNum: 8008, Name: "Temple of the Cross", Description: "A quiet temple."},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(w.StopAITicker)

	messages := make(map[string]string)
	w.MessageSink = func(name string, msg []byte) { messages[name] += string(msg) }
	actor := NewPlayer(1, "PortalHero", 19658)
	actor.SetPosition(combat.PosStanding)
	actor.Stats.Int = 10
	actor.Stats.Wis = 10
	if err := w.AddPlayer(actor); err != nil {
		t.Fatalf("AddPlayer actor: %v", err)
	}
	peer := NewPlayer(2, "PortalWitness", 19658)
	peer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(peer); err != nil {
		t.Fatalf("AddPlayer peer: %v", err)
	}
	return portalToTempleTestWorld{w: w, actor: actor, peer: peer, messages: messages}
}

func clearPortalToTempleMessages(messages map[string]string) {
	for name := range messages {
		messages[name] = ""
	}
}

func TestSpecPortalToTemple_EntryAndExactArgumentGates(t *testing.T) {
	a := newPortalToTempleTestWorld(t)
	for _, tc := range []struct {
		name string
		cmd  string
		arg  string
	}{
		{name: "wrong command", cmd: "look", arg: "setchswayno"},
		{name: "trailing argument byte", cmd: "say", arg: "setchswayno "},
		{name: "wrong argument", cmd: "say", arg: "nope"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearPortalToTempleMessages(a.messages)
			a.actor.SetRoom(19658)
			if specPortalToTemple(a.w, a.actor, nil, tc.cmd, tc.arg) {
				t.Fatal("portal special unexpectedly handled command")
			}
			if got := a.messages[a.actor.Name] + a.messages[a.peer.Name]; got != "" {
				t.Fatalf("gate output = %q, want empty", got)
			}
			if got := a.actor.GetRoomVNum(); got != 19658 {
				t.Fatalf("actor room = %d, want 19658", got)
			}
		})
	}
}

func TestSpecPortalToTemple_SayAudienceTeleportAndLandingLook(t *testing.T) {
	a := newPortalToTempleTestWorld(t)

	if !specPortalToTemple(a.w, a.actor, nil, "say", "setchswayno") {
		t.Fatal("say portal command was not handled")
	}
	actor := a.messages[a.actor.Name]
	peer := a.messages[a.peer.Name]
	for _, want := range []string{
		"You say 'setchswayno'\r\n",
		"With a blinding flash of light and a crack of thunder, you are teleported...\r\n",
		"Temple of the Cross\r\n",
		"A quiet temple.\r\n",
	} {
		if !strings.Contains(actor, want) {
			t.Errorf("actor output %q missing %q", actor, want)
		}
	}
	if got, want := peer, "PortalHero says, 'setchswayno'\r\n"+
		"With a blinding flash of light and a crack of thunder, PortalHero disappears!\r\n"; got != want {
		t.Errorf("peer output = %q, want %q", got, want)
	}
	if strings.Contains(actor, "PortalHero disappears!") {
		t.Error("actor received the TO_ROOM disappearance message")
	}
	if got := a.actor.GetRoomVNum(); got != 8008 {
		t.Fatalf("actor room = %d, want 8008", got)
	}
	if got := a.peer.GetRoomVNum(); got != 19658 {
		t.Fatalf("peer room = %d, want 19658", got)
	}
}

func TestSpecPortalToTemple_ApostropheAlias(t *testing.T) {
	a := newPortalToTempleTestWorld(t)
	if !specPortalToTemple(a.w, a.actor, nil, "'", "setchswayno") {
		t.Fatal("apostrophe portal command was not handled")
	}
	if got := a.actor.GetRoomVNum(); got != 8008 {
		t.Fatalf("actor room = %d, want 8008", got)
	}
	if got := a.messages[a.actor.Name]; !strings.Contains(got, "You say 'setchswayno'\r\n") {
		t.Fatalf("apostrophe alias did not use do_say output: %q", got)
	}
}
