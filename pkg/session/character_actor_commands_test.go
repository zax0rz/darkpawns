package session

import (
	"reflect"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

type actorPositionFixture struct {
	world    *game.World
	actor    *game.MobInstance
	observer *game.Player
	roomOut  strings.Builder
}

func newActorPositionFixture(t *testing.T, position int) *actorPositionFixture {
	t.Helper()
	parsed := &parser.World{
		Rooms: []parser.Room{{VNum: 1000, Name: "Void"}, {VNum: 1001, Name: "Hall"}},
		Mobs: []parser.Mob{{
			VNum: 99001, Keywords: "guard trainee", ShortDesc: "a guard trainee",
			Sex: 0, Position: position, DefaultPos: combat.PosStanding,
		}},
	}
	w, err := game.NewWorld(parsed)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(w.StopAITicker)
	observer := game.NewPlayer(1, "Observer", 1001)
	observer.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(observer); err != nil {
		t.Fatal(err)
	}
	actor, err := w.SpawnMob(99001, 1001)
	if err != nil {
		t.Fatal(err)
	}
	actor.SetPosition(position)
	f := &actorPositionFixture{world: w, actor: actor, observer: observer}
	w.MessageSink = func(name string, message []byte) {
		if name == observer.GetName() {
			f.roomOut.Write(message)
		}
	}
	return f
}

func callRegisteredActorCommand(t *testing.T, f *actorPositionFixture, name string, private *strings.Builder) game.CharacterCommandResult {
	t.Helper()
	entry, ok := cmdRegistry.Lookup(name)
	if !ok || entry.ActorHandler == nil {
		t.Fatalf("%s has no registered actor callback", name)
	}
	playerHandler := entry.Handler
	result := entry.ActorHandler(game.CharacterCommandContext{
		Actor: f.actor, World: f.world,
		PrivateOutput: func(line string) { private.WriteString(line) },
	}, name, nil)
	if reflect.ValueOf(entry.Handler).Pointer() != reflect.ValueOf(playerHandler).Pointer() {
		t.Fatalf("%s actor callback changed the player handler", name)
	}
	return result
}

func TestRegisteredActorSitAndStandRoomBytesAndState(t *testing.T) {
	tests := []struct {
		name         string
		command      string
		position     int
		wantPosition int
		wantPrivate  string
		wantRoom     string
	}{
		{"sit standing", "sit", combat.PosStanding, combat.PosSitting, "You sit down.\r\n", "A guard trainee sits down.\r\n"},
		{"sit resting", "sit", combat.PosResting, combat.PosSitting, "You stop resting, and sit up.\r\n", "A guard trainee stops resting.\r\n"},
		{"stand sitting", "stand", combat.PosSitting, combat.PosStanding, "You stand up.\r\n", "A guard trainee clambers to its feet.\r\n"},
		{"stand resting", "stand", combat.PosResting, combat.PosStanding, "You stop resting, and stand up.\r\n", "A guard trainee stops resting, and clambers on its feet.\r\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newActorPositionFixture(t, test.position)
			var private strings.Builder
			if got := callRegisteredActorCommand(t, f, test.command, &private); got != game.CharacterCommandHandled {
				t.Fatalf("actor callback result = %v, want handled", got)
			}
			if got := private.String(); got != test.wantPrivate {
				t.Errorf("private bytes = %q, want %q", got, test.wantPrivate)
			}
			if got := f.roomOut.String(); got != test.wantRoom {
				t.Errorf("room bytes = %q, want %q", got, test.wantRoom)
			}
			if got := f.actor.GetPosition(); got != test.wantPosition {
				t.Errorf("actor position = %d, want %d", got, test.wantPosition)
			}
		})
	}
}

func TestRegisteredActorPositionRefusalsAndMountedDistinction(t *testing.T) {
	tests := []struct {
		name        string
		command     string
		position    int
		mounted     bool
		wantResult  game.CharacterCommandResult
		wantPrivate string
	}{
		{"stand already standing", "stand", combat.PosStanding, false, game.CharacterCommandHandled, "You are already standing.\r\n"},
		{"stand fighting", "stand", combat.PosFighting, false, game.CharacterCommandHandled, "Do you not consider fighting as standing?\r\n"},
		{"sit already sitting", "sit", combat.PosSitting, false, game.CharacterCommandHandled, "You're sitting already.\r\n"},
		{"sit fighting", "sit", combat.PosFighting, false, game.CharacterCommandHandled, "Sit down while fighting? are you MAD?\r\n"},
		{"sit mounted", "sit", combat.PosStanding, true, game.CharacterCommandHandled, "You can't rest while mounted.\r\n"},
		{"stand mounted dismount unported", "stand", combat.PosStanding, true, game.CharacterCommandUnsupported, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newActorPositionFixture(t, test.position)
			if test.mounted {
				f.actor.SetAffected(game.AffMount)
			}
			var private strings.Builder
			got := callRegisteredActorCommand(t, f, test.command, &private)
			if got != test.wantResult {
				t.Fatalf("actor callback result = %v, want %v", got, test.wantResult)
			}
			if got := private.String(); got != test.wantPrivate {
				t.Errorf("private bytes = %q, want %q", got, test.wantPrivate)
			}
			if got := f.roomOut.String(); got != "" {
				t.Errorf("room bytes = %q, want none", got)
			}
			if got := f.actor.GetPosition(); got != test.position {
				t.Errorf("actor position = %d, want unchanged %d", got, test.position)
			}
			if got := f.actor.IsAffected(game.AffMount); got != test.mounted {
				t.Errorf("AFF_MOUNT = %v, want unchanged %v", got, test.mounted)
			}
		})
	}
}

func TestRegisteredActorRoomVisibilityMatchesSitAndStandActFlags(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		position int
		wantRoom string
	}{
		{"standing sit uses visible substitution without hiding recipient", "sit", combat.PosStanding, "Someone sits down.\r\n"},
		{"resting sit suppresses invisible actor", "sit", combat.PosResting, ""},
		{"default sit suppresses invisible actor", "sit", combat.PosMortally, ""},
		{"stand suppresses invisible actor", "stand", combat.PosSitting, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f := newActorPositionFixture(t, test.position)
			f.actor.SetAffected(game.AffInvisible)
			var private strings.Builder
			if got := callRegisteredActorCommand(t, f, test.command, &private); got != game.CharacterCommandHandled {
				t.Fatalf("actor callback result = %v, want handled", got)
			}
			if got := f.roomOut.String(); got != test.wantRoom {
				t.Errorf("room bytes = %q, want %q", got, test.wantRoom)
			}
		})
	}
}

func TestRegisteredActorCallbacksWithoutPrivateDescriptorOnlyShowRoomAct(t *testing.T) {
	f := newActorPositionFixture(t, combat.PosStanding)
	entry, ok := cmdRegistry.Lookup("sit")
	if !ok || entry.ActorHandler == nil {
		t.Fatal("sit has no registered actor callback")
	}
	result := entry.ActorHandler(game.CharacterCommandContext{Actor: f.actor, World: f.world}, "sit", nil)
	if result != game.CharacterCommandHandled {
		t.Fatalf("actor callback result = %v, want handled", result)
	}
	if got := f.roomOut.String(); got != "A guard trainee sits down.\r\n" {
		t.Fatalf("room bytes = %q", got)
	}
	if got := f.actor.GetPosition(); got != combat.PosSitting {
		t.Fatalf("actor position = %d, want sitting", got)
	}
}
