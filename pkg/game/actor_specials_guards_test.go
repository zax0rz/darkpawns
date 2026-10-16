package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// guardActorWorld returns a world whose actor body is a descriptor-less mobile
// in room 1001, the room's peer output sink (the player Zach), and a private
// sink standing in for a switched body's descriptor.
func guardActorWorld(t *testing.T) (*World, *Player, *MobInstance, *strings.Builder, *strings.Builder) {
	t.Helper()
	w, player, actor, peer := newNPCCommandWorld(t)
	private := &strings.Builder{}
	return w, player, actor, peer, private
}

func guardActorCtx(actor *MobInstance, private *strings.Builder) CharacterSpecialContext {
	return CharacterSpecialContext{
		Actor:         actor,
		PrivateOutput: func(message string) { private.WriteString(message) },
	}
}

// TestNoMoveGuardActorPortsMatchC pins the four SPECIAL(no_move_*) bodies for an
// NPC actor: the directional block with C's exact bytes, and a plain decline for
// every other command (src/spec_procs2.c:2033-2111, and the unassigned
// no_move_north at :2054-2073).
func TestNoMoveGuardActorPortsMatchC(t *testing.T) {
	tests := []struct {
		name        string
		special     string
		guardDesc   string
		cmd         string
		actorLevel  int
		actorHunts  bool
		guardPos    int
		want        SpecialResult
		wantPeer    string
		wantPrivate string
	}{
		{
			name: "west block", special: "no_move_west", guardDesc: "the western guard", cmd: "west",
			actorLevel: 10, guardPos: combat.PosStanding, want: SpecialHandled,
			wantPeer:    "The western guard humiliates an ogre, and blocks its way.\r\n",
			wantPrivate: "The western guard humiliates you and blocks your way.\r\n",
		},
		{
			name: "south block", special: "no_move_south", guardDesc: "the southern guard", cmd: "south",
			actorLevel: 10, guardPos: combat.PosStanding, want: SpecialHandled,
			wantPeer: "The southern guard blocks an ogre's way.\r\nThe southern guard says 'Thou shalt not pass.'\r\n",
			// C's act() order is TO_NOTVICT, TO_VICT, TO_ROOM: a body with a
			// descriptor receives the victim line and then the room line.
			wantPrivate: "The southern guard blocks your way.\r\nThe southern guard says 'Thou shalt not pass.'\r\n",
		},
		{
			name: "north block", special: "no_move_north", guardDesc: "the northern guard", cmd: "north",
			actorLevel: 10, guardPos: combat.PosStanding, want: SpecialHandled,
			wantPeer:    "The northern guard blocks an ogre's way.\r\nThe northern guard says 'I cannot let you pass.'\r\n",
			wantPrivate: "The northern guard blocks your way.\r\nThe northern guard says 'I cannot let you pass.'\r\n",
		},
		{name: "other direction", special: "no_move_west", guardDesc: "the western guard", cmd: "east", actorLevel: 10, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "non-movement command", special: "no_move_west", guardDesc: "the western guard", cmd: "emote", actorLevel: 10, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "immortal actor", special: "no_move_west", guardDesc: "the western guard", cmd: "west", actorLevel: LVL_IMMORT, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "hunting actor", special: "no_move_west", guardDesc: "the western guard", cmd: "west", actorLevel: 10, actorHunts: true, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "sleeping guard", special: "no_move_west", guardDesc: "the western guard", cmd: "west", actorLevel: 10, guardPos: combat.PosSleeping, want: SpecialDeclined},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, _, actor, peer, private := guardActorWorld(t)
			guard := newSpecProcTestMob(t, w, actor.GetRoom(), 10)
			actor.SetLevel(test.actorLevel)
			guard.SetPosition(test.guardPos)
			if test.actorHunts {
				actor.SetHunting("a quarry")
			}
			actorProto := *actor.Proto()
			actorProto.ShortDesc = "an ogre"
			actor.SetProto(&actorProto)
			guardProto := *guard.Proto()
			guardProto.ShortDesc = test.guardDesc
			guard.SetProto(&guardProto)

			got := DispatchCharacterSpecial(test.special, w, guardActorCtx(actor, private), guard, test.cmd, "")
			if got != test.want {
				t.Fatalf("%s result = %v, want %v", test.special, got, test.want)
			}
			if got, want := peer.String(), test.wantPeer; got != want {
				t.Errorf("peer output = %q, want %q", got, want)
			}
			if got, want := private.String(), test.wantPrivate; got != want {
				t.Errorf("descriptor output = %q, want %q", got, want)
			}
		})
	}
}

// TestCastleGuardActorPortsMatchC pins the three castle-guard bodies for an NPC
// actor (src/spec_procs2.c:2134-2258): the guarded direction blocks a body that
// owns no house, and every other command declines. C's grouped-owner arm needs
// AFF_GROUP on both the actor and its master plus a house the master owns.
func TestCastleGuardActorPortsMatchC(t *testing.T) {
	tests := []struct {
		name        string
		special     string
		cmd         string
		actorLevel  int
		guardPos    int
		want        SpecialResult
		wantPeer    string
		wantPrivate string
	}{
		{
			name: "down block", special: "castle_guard_down", cmd: "down", actorLevel: 10, guardPos: combat.PosStanding,
			want:        SpecialHandled,
			wantPeer:    "The castle guard blocks an ogre's path.\r\nThe castle guard states, 'Thou shalt not pass.'\r\n",
			wantPrivate: "The castle guard blocks your way.\r\nThe castle guard states, 'Thou shalt not pass.'\r\n",
		},
		{
			name: "up block", special: "castle_guard_up", cmd: "up", actorLevel: 10, guardPos: combat.PosStanding,
			want:        SpecialHandled,
			wantPeer:    "The castle guard blocks an ogre's path.\r\nThe castle guard states, 'Thou shalt not pass.'\r\n",
			wantPrivate: "The castle guard blocks your way.\r\nThe castle guard states, 'Thou shalt not pass.'\r\n",
		},
		{
			name: "north block", special: "castle_guard_north", cmd: "north", actorLevel: 10, guardPos: combat.PosStanding,
			want:        SpecialHandled,
			wantPeer:    "The castle guard blocks an ogre's path.\r\nThe castle guard states, 'Thou shalt not pass.'\r\n",
			wantPrivate: "The castle guard blocks your way.\r\nThe castle guard states, 'Thou shalt not pass.'\r\n",
		},
		{name: "other direction", special: "castle_guard_down", cmd: "north", actorLevel: 10, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "non-movement command", special: "castle_guard_down", cmd: "emote", actorLevel: 10, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "immortal actor", special: "castle_guard_down", cmd: "down", actorLevel: LVL_IMMORT, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "sleeping guard", special: "castle_guard_down", cmd: "down", actorLevel: 10, guardPos: combat.PosSleeping, want: SpecialDeclined},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, _, actor, peer, private := guardActorWorld(t)
			guard := newSpecProcTestMob(t, w, actor.GetRoom(), 10)
			actor.SetLevel(test.actorLevel)
			guard.SetPosition(test.guardPos)
			actorProto := *actor.Proto()
			actorProto.ShortDesc = "an ogre"
			actor.SetProto(&actorProto)
			guardProto := *guard.Proto()
			guardProto.ShortDesc = "the castle guard"
			guard.SetProto(&guardProto)

			got := DispatchCharacterSpecial(test.special, w, guardActorCtx(actor, private), guard, test.cmd, "")
			if got != test.want {
				t.Fatalf("%s result = %v, want %v", test.special, got, test.want)
			}
			if got, want := peer.String(), test.wantPeer; got != want {
				t.Errorf("peer output = %q, want %q", got, want)
			}
			if got, want := private.String(), test.wantPrivate; got != want {
				t.Errorf("descriptor output = %q, want %q", got, want)
			}
		})
	}
}

// TestCastleGuardActorPortsGroupedOwnerPasses covers C's second arm: an actor
// grouped with the house owner is let through, and the command is still not
// consumed — the move continues.
func TestCastleGuardActorPortsGroupedOwnerPasses(t *testing.T) {
	w, player, actor, peer, private := guardActorWorld(t)
	guard := newSpecProcTestMob(t, w, actor.GetRoom(), 10)
	actor.SetLevel(10)
	actor.SetAffected(affGroup)
	player.SetAffect(affGroup, true)
	actor.SetFollowingBody(player)

	roomVNum := actor.GetRoom() + 2 // C's world[ch->in_room].number+2 for the down guard.
	w.HouseControl = append(w.HouseControl, HouseControl{VNum: roomVNum, Owner: int64(player.GetID())})
	t.Cleanup(func() { w.HouseControl = w.HouseControl[:len(w.HouseControl)-1] })

	actorProto := *actor.Proto()
	actorProto.ShortDesc = "an ogre"
	actor.SetProto(&actorProto)
	guardProto := *guard.Proto()
	guardProto.ShortDesc = "the castle guard"
	guard.SetProto(&guardProto)

	got := DispatchCharacterSpecial("castle_guard_down", w, guardActorCtx(actor, private), guard, "down", "")
	if got != SpecialDeclined {
		t.Fatalf("grouped-owner result = %v, want declined (C falls through after the pass acts)", got)
	}
	if want := "The castle guard moves aside and allows you to pass.\r\n"; private.String() != want {
		t.Errorf("descriptor output = %q, want %q", private.String(), want)
	}
	if want := "The castle guard moves aside and allows an ogre to pass.\r\n"; peer.String() != want {
		t.Errorf("peer output = %q, want %q", peer.String(), want)
	}
}

// TestChosenGuardActorPortsMatchC pins SPECIAL(chosen_guard),
// src/spec_procs2.c:2113-2132, for an NPC actor.
func TestChosenGuardActorPortsMatchC(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		actorLevel  int
		guardPos    int
		want        SpecialResult
		wantPeer    string
		wantPrivate string
	}{
		{
			name: "south block", cmd: "south", actorLevel: 10, guardPos: combat.PosStanding, want: SpecialHandled,
			wantPeer:    "The chosen guard blocks an ogre's way.\r\nThe chosen guard says 'Thou shalt not pass.'\r\n",
			wantPrivate: "The chosen guard blocks your way.\r\nThe chosen guard says 'Thou shalt not pass.'\r\n",
		},
		{name: "other direction", cmd: "north", actorLevel: 10, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "non-movement command", cmd: "emote", actorLevel: 10, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "immortal actor", cmd: "south", actorLevel: LVL_IMMORT, guardPos: combat.PosStanding, want: SpecialDeclined},
		{name: "sleeping guard", cmd: "south", actorLevel: 10, guardPos: combat.PosSleeping, want: SpecialDeclined},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, _, actor, peer, private := guardActorWorld(t)
			guard := newSpecProcTestMob(t, w, actor.GetRoom(), 10)
			actor.SetLevel(test.actorLevel)
			guard.SetPosition(test.guardPos)
			actorProto := *actor.Proto()
			actorProto.ShortDesc = "an ogre"
			actor.SetProto(&actorProto)
			guardProto := *guard.Proto()
			guardProto.ShortDesc = "the chosen guard"
			guard.SetProto(&guardProto)

			got := DispatchCharacterSpecial("chosen_guard", w, guardActorCtx(actor, private), guard, test.cmd, "")
			if got != test.want {
				t.Fatalf("chosen_guard result = %v, want %v", got, test.want)
			}
			if got, want := peer.String(), test.wantPeer; got != want {
				t.Errorf("peer output = %q, want %q", got, want)
			}
			if got, want := private.String(), test.wantPrivate; got != want {
				t.Errorf("descriptor output = %q, want %q", got, want)
			}
		})
	}
}

// TestJailGuardActorPortsMatchC pins SPECIAL(jailguard) and
// SPECIAL(outofjailguard), src/spec_procs.c:1763-1796, for an NPC actor. The
// block is scoped to the guard's own room and direction; the test world has no
// room 8117/8118, so the blocking cases move the actor to that vnum.
func TestJailGuardActorPortsMatchC(t *testing.T) {
	tests := []struct {
		name        string
		special     string
		roomVNum    int
		cmd         string
		actorLevel  int
		want        SpecialResult
		wantPeer    string
		wantPrivate string
	}{
		{
			name: "jail north block", special: "jailguard", roomVNum: 8118, cmd: "north", actorLevel: 10,
			want: SpecialHandled,
			// C's act root is the actor, so the room line excludes the actor
			// itself; only the guard's send_to_char reaches the actor. The
			// world has no other body in room 8118, so no peer sees it here.
			wantPrivate: "The guard stops you from leaving with one flabby hand.\r\n",
		},
		{
			name: "out of jail south block", special: "outofjailguard", roomVNum: 8117, cmd: "south", actorLevel: 10,
			want:        SpecialHandled,
			wantPrivate: "The guard stops you from entering with one quick jerk of your collar.\r\n",
		},
		{name: "wrong direction", special: "jailguard", cmd: "south", actorLevel: 10, want: SpecialDeclined},
		{name: "non-movement command", special: "jailguard", cmd: "emote", actorLevel: 10, want: SpecialDeclined},
		{name: "immortal actor", special: "jailguard", cmd: "north", actorLevel: LVL_IMMORT, want: SpecialDeclined},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, _, actor, peer, private := guardActorWorld(t)
			guard := newSpecProcTestMob(t, w, actor.GetRoom(), 10)
			actor.SetLevel(test.actorLevel)
			actorProto := *actor.Proto()
			actorProto.ShortDesc = "an ogre"
			actor.SetProto(&actorProto)

			if test.roomVNum != 0 {
				w.rooms[test.roomVNum] = &parser.Room{VNum: test.roomVNum, Name: "The Jail", Zone: 1}
				if err := w.MobTransfer(actor, test.roomVNum); err != nil {
					t.Fatalf("transfer actor: %v", err)
				}
			}

			got := DispatchCharacterSpecial(test.special, w, guardActorCtx(actor, private), guard, test.cmd, "")
			if got != test.want {
				t.Fatalf("%s result = %v, want %v", test.special, got, test.want)
			}
			if got, want := peer.String(), test.wantPeer; got != want {
				t.Errorf("peer output = %q, want %q", got, want)
			}
			if got, want := private.String(), test.wantPrivate; got != want {
				t.Errorf("descriptor output = %q, want %q", got, want)
			}
		})
	}
}

// TestGuildGuardActorPortsFleeArm pins the flee/escape/retreat arm of
// SPECIAL(guild_guard), src/spec_procs.c:573-583, for an NPC actor. C roots the
// act at the actor, so the actor sees only the send_to_char line.
func TestGuildGuardActorPortsFleeArm(t *testing.T) {
	w, _, actor, peer, private := guardActorWorld(t)
	guard := newSpecProcTestMob(t, w, actor.GetRoom(), 10)
	actorProto := *actor.Proto()
	actorProto.ShortDesc = "an ogre"
	actor.SetProto(&actorProto)

	got := DispatchCharacterSpecial("guild_guard", w, guardActorCtx(actor, private), guard, "flee", "")
	if got != SpecialHandled {
		t.Fatalf("guild_guard flee = %v, want handled", got)
	}
	if want := "An ogre tries to flee inside the guild but the guard block its way!\r\n\r\n"; peer.String() != want {
		t.Errorf("peer output = %q, want %q", peer.String(), want)
	}
	if want := "You try to flee inside the guild but the guard stops you!\r\n"; private.String() != want {
		t.Errorf("descriptor output = %q, want %q", private.String(), want)
	}
}

// TestGuildGuardActorPortsDoorwayAndDeclines pins the guild_info arm of
// SPECIAL(guild_guard), src/spec_procs.c:585-603, for an NPC actor: the
// "wrong class" test is `IS_NPC(ch) || GET_CLASS(ch) != guild_info[i][0]`, so a
// mobile is always at the wrong class. The test world has no guild room, so a
// synthetic row points at the actor's own room.
func TestGuildGuardActorPortsDoorwayAndDeclines(t *testing.T) {
	w, _, actor, peer, private := guardActorWorld(t)
	guard := newSpecProcTestMob(t, w, actor.GetRoom(), 10)
	actorProto := *actor.Proto()
	actorProto.ShortDesc = "an ogre"
	actor.SetProto(&actorProto)

	if got := DispatchCharacterSpecial("guild_guard", w, guardActorCtx(actor, private), guard, "emote", ""); got != SpecialDeclined {
		t.Fatalf("guild_guard emote = %v, want declined", got)
	}
	if got := DispatchCharacterSpecial("guild_guard", w, guardActorCtx(actor, private), guard, "north", ""); got != SpecialDeclined {
		t.Fatalf("guild_guard north outside a guild room = %v, want declined", got)
	}

	GuildInfo = append(GuildInfo, GuildInfoEntry{Class: ClassWarrior, Room: actor.GetRoom(), Direction: 0})
	t.Cleanup(func() { GuildInfo = GuildInfo[:len(GuildInfo)-1] })

	if got := DispatchCharacterSpecial("guild_guard", w, guardActorCtx(actor, private), guard, "north", ""); got != SpecialHandled {
		t.Fatalf("guild_guard north at the guild doorway = %v, want handled", got)
	}
	if want := "The guard humiliates an ogre, and blocks its way.\r\n"; peer.String() != want {
		t.Errorf("peer output = %q, want %q", peer.String(), want)
	}
	if want := "The guard humiliates you, and blocks your way.\r\n"; private.String() != want {
		t.Errorf("descriptor output = %q, want %q", private.String(), want)
	}
}

// TestNewbieZoneEntranceActorPortsLevelGate pins SPECIAL(newbie_zone_entrance),
// src/spec_procs.c:2262-2274, for an NPC actor: the actor's own level decides,
// and C's write reaches only a body with a descriptor.
func TestNewbieZoneEntranceActorPortsLevelGate(t *testing.T) {
	tests := []struct {
		name        string
		cmd         string
		actorLevel  int
		want        SpecialResult
		wantPrivate string
	}{
		{
			name: "above newbie level blocked", cmd: "south", actorLevel: newbieLevel, want: SpecialHandled,
			wantPrivate: "Nah, you're too much of a badass to go in there!\r\n",
		},
		{name: "below newbie level passes", cmd: "south", actorLevel: newbieLevel - 1, want: SpecialDeclined},
		{name: "immortal passes", cmd: "south", actorLevel: LVL_IMMORT, want: SpecialDeclined},
		{name: "other direction", cmd: "north", actorLevel: newbieLevel, want: SpecialDeclined},
		{name: "non-movement command", cmd: "emote", actorLevel: newbieLevel, want: SpecialDeclined},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, _, actor, _, private := guardActorWorld(t)
			actor.SetLevel(test.actorLevel)

			got := DispatchCharacterSpecial("newbie_zone_entrance", w, guardActorCtx(actor, private), nil, test.cmd, "")
			if got != test.want {
				t.Fatalf("newbie_zone_entrance result = %v, want %v", got, test.want)
			}
			if got, want := private.String(), test.wantPrivate; got != want {
				t.Errorf("descriptor output = %q, want %q", got, want)
			}
		})
	}
}

// TestWallGuardNSActorPortsDelegateLikeC pins SPECIAL(wall_guard_ns),
// src/spec_procs2.c:2260-2300, for an NPC actor: C hands every typed command to
// fighter(ch, ch, 0, NULL), whose first gate declines a body that is not already
// fighting. The revert control is the old blanket "unsupported".
func TestWallGuardNSActorPortsDelegateLikeC(t *testing.T) {
	w, _, actor, _, private := guardActorWorld(t)
	guard := newSpecProcTestMob(t, w, actor.GetRoom(), 10)

	for _, cmd := range []string{"emote", "say", "north"} {
		if got := DispatchCharacterSpecial("wall_guard_ns", w, guardActorCtx(actor, private), guard, cmd, ""); got != SpecialDeclined {
			t.Errorf("wall_guard_ns %q = %v, want declined", cmd, got)
		}
	}
}

// TestMovementSpecialBlocksMobUsesTheActorLanes is the integration pin behind
// the playtest report: a room holding a declining special and a blocking guard
// lets a mobile move in the unguarded direction and stops it in the guarded one.
func TestMovementSpecialBlocksMobUsesTheActorLanes(t *testing.T) {
	w, _, actor, _ := newNPCCommandWorld(t)
	decliner := newSpecProcTestMob(t, w, actor.GetRoom(), 10)
	guard := newSpecProcTestMob(t, w, actor.GetRoom(), 10)

	previousDecliner, hadDecliner := MobSpecAssign[decliner.GetVNum()]
	previousGuard, hadGuard := MobSpecAssign[guard.GetVNum()]
	// cityguard declines every typed command and predates this round's table
	// (PR #2069), so the integration case stands on the pre-round-8 tree too.
	MobSpecAssign[decliner.GetVNum()] = "cityguard"
	MobSpecAssign[guard.GetVNum()] = "castle_guard_down"
	t.Cleanup(func() {
		if hadDecliner {
			MobSpecAssign[decliner.GetVNum()] = previousDecliner
		} else {
			delete(MobSpecAssign, decliner.GetVNum())
		}
		if hadGuard {
			MobSpecAssign[guard.GetVNum()] = previousGuard
		} else {
			delete(MobSpecAssign, guard.GetVNum())
		}
	})

	if w.movementSpecialBlocksMob(actor, "north") {
		t.Error("a declining special blocked an unguarded direction")
	}
	if !w.movementSpecialBlocksMob(actor, "down") {
		t.Error("the castle guard did not block its own direction")
	}
}

// TestNoMoveGuardPlayerLaneMatchesC pins the same C body for a player actor. The
// immortal, hunting and sleeping-guard gates were missing from the player lane,
// so a leader at LVL_IMMORT was refused where C lets them pass
// (src/spec_procs2.c:1823-1842, :2033-2052, :2094-2111). The revert control is
// the gate-free legacy block.
func TestNoMoveGuardPlayerLaneMatchesC(t *testing.T) {
	tests := []struct {
		name        string
		special     string
		cmd         string
		playerLevel int
		guardPos    int
		want        bool
	}{
		{name: "mortal blocked west", special: "no_move_west", cmd: "west", playerLevel: 10, guardPos: combat.PosStanding, want: true},
		{name: "mortal blocked south", special: "no_move_south", cmd: "south", playerLevel: 10, guardPos: combat.PosStanding, want: true},
		{name: "mortal blocked east", special: "no_move_east", cmd: "east", playerLevel: 10, guardPos: combat.PosStanding, want: true},
		{name: "immortal passes", special: "no_move_south", cmd: "south", playerLevel: LVL_IMMORT, guardPos: combat.PosStanding},
		{name: "sleeping guard passes", special: "no_move_south", cmd: "south", playerLevel: 10, guardPos: combat.PosSleeping},
		{name: "other direction passes", special: "no_move_south", cmd: "north", playerLevel: 10, guardPos: combat.PosStanding},
		{name: "non-movement command passes", special: "no_move_south", cmd: "emote", playerLevel: 10, guardPos: combat.PosStanding},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			w, player, _, _ := newNPCCommandWorld(t)
			guard := newSpecProcTestMob(t, w, player.GetRoom(), 10)
			player.Level = test.playerLevel
			guard.SetPosition(test.guardPos)

			got := DispatchCharacterSpecial(test.special, w, CharacterSpecialContext{Actor: player}, guard, test.cmd, "")
			if want := boolToResult(test.want); got != want {
				t.Fatalf("%s %s at level %d = %v, want %v", test.special, test.cmd, test.playerLevel, got, want)
			}
		})
	}
}

func boolToResult(handled bool) SpecialResult {
	if handled {
		return SpecialHandled
	}
	return SpecialDeclined
}
