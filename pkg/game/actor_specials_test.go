package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

func TestCharacterSpecialRegistryKeepsPlayerLaneAndMarksNPCGaps(t *testing.T) {
	const missingName = "test_actor_special_missing_entry"
	delete(SpecRegistry, missingName)
	if got := DispatchCharacterSpecial(missingName, nil, CharacterSpecialContext{}, nil, "look", ""); got != SpecialUnsupported {
		t.Fatalf("missing native special result = %v, want unsupported", got)
	}
	const emptyName = "test_actor_special_empty_entry"
	delete(SpecRegistry, emptyName)
	RegisterSpec(emptyName, nil)
	t.Cleanup(func() { delete(SpecRegistry, emptyName) })
	if got := DispatchCharacterSpecial(emptyName, nil, CharacterSpecialContext{}, nil, "look", ""); got != SpecialUnsupported {
		t.Fatalf("empty native special result = %v, want unsupported", got)
	}

	const name = "test_actor_special_registry_entry"
	delete(SpecRegistry, name)
	t.Cleanup(func() { delete(SpecRegistry, name) })

	legacyCalls := 0
	RegisterSpec(name, func(_ *World, _ *Player, _ *MobInstance, _, _ string) bool {
		legacyCalls++
		return true
	})
	w, player, mob, _ := newNPCCommandWorld(t)

	if got := DispatchCharacterSpecial(name, w, CharacterSpecialContext{Actor: mob}, nil, "look", ""); got != SpecialUnsupported {
		t.Fatalf("legacy-only NPC special result = %v, want unsupported", got)
	}
	if legacyCalls != 0 {
		t.Fatalf("legacy *Player callback invoked for NPC actor %d times", legacyCalls)
	}

	RegisterActorSpec(name, func(_ *World, _ CharacterSpecialContext, _ *MobInstance, _, _ string) SpecialResult {
		return SpecialHandled
	})
	if got := DispatchCharacterSpecial(name, w, CharacterSpecialContext{Actor: mob}, nil, "look", ""); got != SpecialHandled {
		t.Fatalf("actor-aware NPC special result = %v, want handled", got)
	}
	if legacyCalls != 0 {
		t.Fatalf("NPC actor lane called legacy callback %d times", legacyCalls)
	}
	if got := DispatchCharacterSpecial(name, w, CharacterSpecialContext{Actor: player}, nil, "look", ""); got != SpecialHandled {
		t.Fatalf("player special result = %v, want handled", got)
	}
	if legacyCalls != 1 {
		t.Fatalf("player callback count = %d, want 1", legacyCalls)
	}

	if got := DispatchCharacterSpecial("guild", w, CharacterSpecialContext{Actor: mob}, nil, "practice", ""); got != SpecialDeclined {
		t.Fatalf("C-rejected NPC guild special result = %v, want declined", got)
	}
	for _, name := range []string{"citizen", "cleric", "dragon_breath"} {
		if got := DispatchCharacterSpecial(name, w, CharacterSpecialContext{Actor: mob}, nil, "look", ""); got != SpecialDeclined {
			t.Errorf("nonempty command-time NPC %s result = %v, want declined", name, got)
		}
		if got := DispatchCharacterSpecial(name, w, CharacterSpecialContext{Actor: mob}, nil, "", ""); got != SpecialUnsupported {
			t.Errorf("empty command-time NPC %s result = %v, want unsupported", name, got)
		}
	}
	if got := DispatchCharacterSpecial("tattoo1", w, CharacterSpecialContext{Actor: mob}, nil, "list", ""); got != SpecialUnsupported {
		t.Fatalf("branch-local NPC tattoo special result = %v, want unsupported", got)
	}
}

func TestNoMoveEastActorSpecialCGuardsAndOutput(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		actorLevel  int
		actorHunts  bool
		actorBlind  bool
		actorInvis  bool
		actorSleep  bool
		guardPos    int
		want        SpecialResult
		wantPeer    string
		wantPrivate string
	}{
		{name: "matching mortal movement", cmd: "east", actorLevel: 10, guardPos: combat.PosStanding, want: SpecialHandled, wantPeer: "The eastern guard humiliates an ogre, and blocks its way.\r\n", wantPrivate: "The eastern guard humiliates you and blocks your way.\r\n"},
		{name: "invisible actor remains hidden and excluded by body identity", cmd: "east", actorLevel: 10, actorInvis: true, guardPos: combat.PosStanding, want: SpecialHandled, wantPeer: "The eastern guard humiliates someone, and blocks its way.\r\n", wantPrivate: "The eastern guard humiliates you and blocks your way.\r\n"},
		{name: "blind switched actor sees guard as someone", cmd: "east", actorLevel: 10, actorBlind: true, guardPos: combat.PosStanding, want: SpecialHandled, wantPeer: "The eastern guard humiliates an ogre, and blocks its way.\r\n", wantPrivate: "Someone humiliates you and blocks your way.\r\n"},
		{name: "sleeping switched actor gets no descriptor output", cmd: "east", actorLevel: 10, actorSleep: true, guardPos: combat.PosStanding, want: SpecialHandled, wantPeer: "The eastern guard humiliates an ogre, and blocks its way.\r\n"},
		{name: "immortal actor", cmd: "east", actorLevel: LVL_IMMORT, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "hunting actor", cmd: "east", actorLevel: 10, actorHunts: true, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "sleeping guard", cmd: "east", actorLevel: 10, guardPos: combat.PosSleeping, want: SpecialDeclined},
		{name: "non-movement command", cmd: "look", actorLevel: 10, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "other movement direction", cmd: "west", actorLevel: 10, guardPos: combat.PosStanding, want: SpecialDeclined},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, _, actor, peerOutput := newNPCCommandWorld(t)
			guard := newSpecProcTestMob(t, w, actor.GetRoom(), 10)
			actor.SetLevel(test.actorLevel)
			guard.SetPosition(test.guardPos)
			if test.actorBlind {
				actor.SetAffected(affBlind)
			}
			if test.actorInvis {
				actor.SetAffected(affInvisible)
			}
			if test.actorSleep {
				actor.SetPosition(combat.PosSleeping)
			}
			if test.actorHunts {
				actor.SetHunting("a quarry")
			}

			actorProto := *actor.Proto()
			actorProto.ShortDesc = "an ogre"
			actor.SetProto(&actorProto)
			guardProto := *guard.Proto()
			guardProto.ShortDesc = "the eastern guard"
			guard.SetProto(&guardProto)

			var private strings.Builder
			ctx := CharacterSpecialContext{
				Actor: actor,
				PrivateOutput: func(message string) {
					private.WriteString(message)
				},
			}
			got := DispatchCharacterSpecial("no_move_east", w, ctx, guard, test.cmd, "")
			if got != test.want {
				t.Fatalf("special result = %v, want %v", got, test.want)
			}
			if test.want == SpecialHandled {
				if got, want := peerOutput.String(), test.wantPeer; got != want {
					t.Errorf("peer output = %q, want %q", got, want)
				}
				if got, want := private.String(), test.wantPrivate; got != want {
					t.Errorf("descriptor output = %q, want %q", got, want)
				}
				return
			}
			if got := peerOutput.String(); got != "" {
				t.Errorf("declined special emitted peer output %q", got)
			}
			if got := private.String(); got != "" {
				t.Errorf("declined special emitted descriptor output %q", got)
			}
		})
	}
}

func TestNoMoveEastPlayerLegacyLaneRemainsActive(t *testing.T) {
	w, player, _, peerOutput := newNPCCommandWorld(t)
	guard := newSpecProcTestMob(t, w, player.GetRoom(), 10)

	result := DispatchCharacterSpecial("no_move_east", w, CharacterSpecialContext{Actor: player}, guard, "east", "")
	if result != SpecialHandled {
		t.Fatalf("player legacy result = %v, want handled", result)
	}
	if got, want := peerOutput.String(), "A test mob humiliates you and blocks your way.\r\n"; got != want {
		t.Fatalf("player output = %q, want %q", got, want)
	}
}

func TestNoMoveEastActorWithoutDescriptorOnlyEmitsToPeers(t *testing.T) {
	w, _, actor, peerOutput := newNPCCommandWorld(t)
	guard := newSpecProcTestMob(t, w, actor.GetRoom(), 10)
	actorProto := *actor.Proto()
	actorProto.ShortDesc = "an ogre"
	actor.SetProto(&actorProto)
	guardProto := *guard.Proto()
	guardProto.ShortDesc = "the eastern guard"
	guard.SetProto(&guardProto)

	result := DispatchCharacterSpecial("no_move_east", w, CharacterSpecialContext{Actor: actor}, guard, "east", "")
	if result != SpecialHandled {
		t.Fatalf("descriptor-less NPC result = %v, want handled", result)
	}
	if got, want := peerOutput.String(), "The eastern guard humiliates an ogre, and blocks its way.\r\n"; got != want {
		t.Fatalf("peer output = %q, want %q", got, want)
	}
}
