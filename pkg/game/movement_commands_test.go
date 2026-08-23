package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func captureMovementOutput(w *World) map[string]*strings.Builder {
	output := make(map[string]*strings.Builder)
	w.MessageSink = func(name string, message []byte) {
		if output[name] == nil {
			output[name] = &strings.Builder{}
		}
		output[name].Write(message)
	}
	return output
}

func addMovementPlayer(t *testing.T, w *World, name string, room int) *Player {
	t.Helper()
	player := NewPlayer(len(w.GetPlayersInRoom(room))+1, name, room)
	player.SetMove(100)
	player.SetPosition(combat.PosStanding)
	if err := w.AddPlayer(player); err != nil {
		t.Fatalf("AddPlayer(%s): %v", name, err)
	}
	return player
}

func addMovementMount(t *testing.T, w *World, rider *Player, room int) *MobInstance {
	t.Helper()
	mount := NewMob(&parser.Mob{VNum: 9001, Keywords: "pony mount", ShortDesc: "a pony"}, room)
	w.mu.Lock()
	mount.ID = w.nextMobID
	w.nextMobID++
	w.activeMobs[mount.ID] = mount
	w.mu.Unlock()
	mount.SetMountRider(rider.Name)
	mount.SetAffected(affMounted)
	mount.SetFollowing(rider.Name)
	rider.MountName = mount.GetName()
	rider.SetAffect(affMounted, true)
	return mount
}

func TestDoMoveArrivalUsesReverseDirection(t *testing.T) {
	w, actor := newMovementTestWorld(t)
	observer := addMovementPlayer(t, w, "Observer", 1002)
	output := captureMovementOutput(w)

	result := w.DoMove(actor, "north")
	if !result.Success {
		t.Fatal("DoMove failed")
	}
	if got := output[observer.Name].String(); !strings.Contains(got, "TestPlayer arrives from the south.\r\n") {
		t.Fatalf("observer output = %q", got)
	}
}

func TestPerformMoveFollowerPositionAndHide(t *testing.T) {
	t.Run("resting follower stays", func(t *testing.T) {
		w, leader := newMovementTestWorld(t)
		follower := addMovementPlayer(t, w, "Resting", 1001)
		follower.SetFollowing(leader.Name)
		follower.SetPosition(combat.PosResting)

		if !w.DoMove(leader, "north").Success {
			t.Fatal("leader move failed")
		}
		if got := follower.GetRoom(); got != 1001 {
			t.Fatalf("resting follower room = %d, want 1001", got)
		}
	})

	t.Run("sleeping follower stays", func(t *testing.T) {
		w, leader := newMovementTestWorld(t)
		follower := addMovementPlayer(t, w, "Sleeping", 1001)
		follower.SetFollowing(leader.Name)
		follower.SetPosition(combat.PosSleeping)

		if !w.DoMove(leader, "north").Success {
			t.Fatal("leader move failed")
		}
		if got := follower.GetRoom(); got != 1001 {
			t.Fatalf("sleeping follower room = %d, want 1001", got)
		}
	})

	t.Run("standing follower moves and clears hide", func(t *testing.T) {
		w, leader := newMovementTestWorld(t)
		follower := addMovementPlayer(t, w, "Hidden", 1001)
		follower.SetFollowing(leader.Name)
		follower.SetAffect(affHide, true)

		result := w.DoMove(leader, "north")
		if !result.Success {
			t.Fatal("leader move failed")
		}
		if got := follower.GetRoom(); got != 1002 {
			t.Fatalf("standing follower room = %d, want 1002", got)
		}
		if follower.IsAffected(affHide) {
			t.Fatal("follower retained AFF_HIDE")
		}
		if len(result.Followers) != 1 || result.Followers[0] != follower.Name {
			t.Fatalf("moved followers = %#v", result.Followers)
		}
	})

	t.Run("follower directional special blocks movement", func(t *testing.T) {
		parsed := &parser.World{
			Rooms: []parser.Room{
				{VNum: 3001, Name: "East", Exits: map[string]parser.Exit{
					"west": {Direction: "west", ToRoom: 3002},
				}},
				{VNum: 3002, Name: "West", Exits: map[string]parser.Exit{
					"east": {Direction: "east", ToRoom: 3001},
				}},
			},
			Mobs: []parser.Mob{{VNum: 2106, Keywords: "blocker", ShortDesc: "a blocker"}},
		}
		w, err := NewWorld(parsed)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(w.StopAITicker)
		leader := addMovementPlayer(t, w, "Leader", 3001)
		follower := addMovementPlayer(t, w, "Follower", 3001)
		follower.SetFollowing(leader.Name)
		if _, err := w.SpawnMob(2106, 3001); err != nil {
			t.Fatal(err)
		}
		output := captureMovementOutput(w)

		result := w.DoMove(leader, "west")

		if !result.Success || leader.GetRoom() != 3002 {
			t.Fatalf("leader result = %+v, room = %d", result, leader.GetRoom())
		}
		if follower.GetRoom() != 3001 {
			t.Fatalf("follower room = %d, want 3001", follower.GetRoom())
		}
		if got := output[follower.Name].String(); !strings.Contains(got, "blocked by a heavy object") {
			t.Fatalf("follower output = %q", got)
		}
	})
}

func TestPerformMoveRendersLeaderBeforeFollowerRecursion(t *testing.T) {
	w, leader := newMovementTestWorld(t)
	follower := addMovementPlayer(t, w, "Follower", 1001)
	follower.SetFollowing(leader.Name)

	var rendered []string
	w.MovementLook = func(player *Player) {
		rendered = append(rendered, player.Name)
		if player == leader && follower.GetRoom() != 1001 {
			t.Fatalf("follower moved before leader room render: room = %d", follower.GetRoom())
		}
	}

	if !w.DoMove(leader, "north").Success {
		t.Fatal("leader move failed")
	}
	if got, want := strings.Join(rendered, ","), "TestPlayer,Follower"; got != want {
		t.Fatalf("render order = %q, want %q", got, want)
	}
}

func TestMovementFailureMessagesAndMountGate(t *testing.T) {
	t.Run("charmed follower cannot leave master", func(t *testing.T) {
		w, actor := newMovementTestWorld(t)
		master := addMovementPlayer(t, w, "Master", 1001)
		actor.SetFollowing(master.Name)
		actor.SetAffect(affCharm, true)
		output := captureMovementOutput(w)

		result := w.DoMove(actor, "north")

		if result.Success || actor.GetRoom() != 1001 {
			t.Fatalf("result = %+v, room = %d", result, actor.GetRoom())
		}
		if got := output[actor.Name].String(); !strings.Contains(got, "The thought of leaving your master makes you weep.") {
			t.Fatalf("actor output = %q", got)
		}
	})

	t.Run("closed named door", func(t *testing.T) {
		w, actor := newMovementTestWorld(t)
		room := w.GetRoomInWorld(1001)
		exit := room.Exits["north"]
		exit.ExitInfo = parser.ExitIsDoor | parser.ExitClosed
		exit.Keywords = "wooden door"
		room.Exits["north"] = exit
		output := captureMovementOutput(w)

		w.DoMove(actor, "north")

		if got := output[actor.Name].String(); !strings.Contains(got, "The wooden seems to be closed.") {
			t.Fatalf("actor output = %q", got)
		}
	})

	t.Run("unmarked secret door", func(t *testing.T) {
		w, actor := newMovementTestWorld(t)
		room := w.GetRoomInWorld(1001)
		exit := room.Exits["north"]
		exit.ExitInfo = parser.ExitIsDoor | parser.ExitClosed
		exit.Keywords = "secret panel"
		room.Exits["north"] = exit
		output := captureMovementOutput(w)

		w.DoMove(actor, "north")

		if got := output[actor.Name].String(); !strings.Contains(got, "Alas, you cannot go that way...") {
			t.Fatalf("actor output = %q", got)
		}
	})

	t.Run("exhausted follower wording", func(t *testing.T) {
		w, leader := newMovementTestWorld(t)
		follower := addMovementPlayer(t, w, "Follower", 1001)
		follower.SetFollowing(leader.Name)
		follower.SetMove(0)
		output := captureMovementOutput(w)

		w.DoMove(leader, "north")

		if got := output[follower.Name].String(); !strings.Contains(got, "You are too exhausted to follow.") {
			t.Fatalf("follower output = %q", got)
		}
	})

	t.Run("mounted indoor movement refused", func(t *testing.T) {
		w, actor := newMovementTestWorld(t)
		w.GetRoomInWorld(1002).Flags = []string{"indoors"}
		addMovementMount(t, w, actor, 1001)
		output := captureMovementOutput(w)

		result := w.DoMove(actor, "north")

		if result.Success || actor.GetRoom() != 1001 {
			t.Fatalf("result = %+v, room = %d", result, actor.GetRoom())
		}
		if got := output[actor.Name].String(); !strings.Contains(got, "You can't ride in there! Dismount first!") {
			t.Fatalf("actor output = %q", got)
		}
	})
}

func TestMountedMovementPairStateAndAudiences(t *testing.T) {
	w, rider := newMovementTestWorld(t)
	mount := addMovementMount(t, w, rider, 1001)
	origin := addMovementPlayer(t, w, "Origin", 1001)
	destination := addMovementPlayer(t, w, "Destination", 1002)
	output := captureMovementOutput(w)
	riderBefore := rider.GetMove()
	mountBefore := mount.GetMove()

	result := w.DoMove(rider, "north")

	if !result.Success || rider.GetRoom() != 1002 || mount.GetRoom() != 1002 {
		t.Fatalf("result=%+v rider room=%d mount room=%d", result, rider.GetRoom(), mount.GetRoom())
	}
	if got := rider.GetMove(); got != riderBefore {
		t.Fatalf("rider movement = %d, want unchanged %d", got, riderBefore)
	}
	if got, want := mount.GetMove(), mountBefore-2; got != want {
		t.Fatalf("mount movement = %d, want %d", got, want)
	}
	if got := mount.GetMountRider(); got != rider.Name {
		t.Fatalf("mount rider = %q, want %q", got, rider.Name)
	}
	look := w.DoLookRoom(rider, false)
	for _, message := range look.Messages {
		if strings.Contains(message.Format, "a pony") {
			t.Fatalf("rider room observation exposed own mount: %q", message.Format)
		}
	}
	destinationLook := w.DoLookRoom(destination, false)
	foundMountedRider := false
	for _, message := range destinationLook.Messages {
		if strings.Contains(message.Format, "TestPlayer") && strings.Contains(message.Format, "mounted on a pony.") {
			foundMountedRider = true
		}
		if strings.Contains(message.Format, "A pony stands here") {
			t.Fatalf("destination observation listed mounted mob separately: %q", message.Format)
		}
	}
	if !foundMountedRider {
		t.Fatalf("destination observation omitted mounted rider: %#v", destinationLook.Messages)
	}
	if got := output[origin.Name].String(); !strings.Contains(got, "TestPlayer rides north on a pony.\r\n") {
		t.Fatalf("origin output = %q", got)
	}
	if got := output[destination.Name].String(); !strings.Contains(got, "TestPlayer rides in from the south on a pony.\r\n") {
		t.Fatalf("destination output = %q", got)
	}
}

func TestMountedMovementFailureState(t *testing.T) {
	t.Run("invalid pair silently dismounts", func(t *testing.T) {
		w, rider := newMovementTestWorld(t)
		rider.SetFollowing("Master")
		rider.MountName = "a missing pony"
		rider.SetAffect(affMounted, true)
		output := captureMovementOutput(w)

		if w.DoMove(rider, "north").Success {
			t.Fatal("invalid mounted pair moved")
		}
		if rider.IsMounted() || rider.GetMountName() != "" {
			t.Fatal("invalid mounted pair was not cleared")
		}
		if got := rider.GetFollowing(); got != "Master" {
			t.Fatalf("ordinary following relation = %q, want preserved", got)
		}
		if builder := output[rider.Name]; builder != nil && builder.Len() != 0 {
			t.Fatalf("unexpected output = %q", builder.String())
		}
	})

	t.Run("mount position blocks", func(t *testing.T) {
		w, rider := newMovementTestWorld(t)
		mount := addMovementMount(t, w, rider, 1001)
		mount.SetPosition(combat.PosSitting)
		output := captureMovementOutput(w)

		if w.DoMove(rider, "north").Success {
			t.Fatal("sitting mount moved")
		}
		if got := output[rider.Name].String(); !strings.Contains(got, "Your mount is in no position to go ANYWHERE!") {
			t.Fatalf("rider output = %q", got)
		}
	})

	t.Run("mount exhaustion blocks without charging rider", func(t *testing.T) {
		w, rider := newMovementTestWorld(t)
		mount := addMovementMount(t, w, rider, 1001)
		mount.SetMove(0)
		riderBefore := rider.GetMove()
		output := captureMovementOutput(w)

		if w.DoMove(rider, "north").Success {
			t.Fatal("exhausted mount moved")
		}
		if rider.GetMove() != riderBefore || mount.GetMove() != 0 {
			t.Fatalf("movement changed: rider=%d mount=%d", rider.GetMove(), mount.GetMove())
		}
		if got := output[rider.Name].String(); !strings.Contains(got, "Your mount is too exhausted to carry you further.") {
			t.Fatalf("rider output = %q", got)
		}
	})
}

func TestMountedMovementMobPoolRegenerates(t *testing.T) {
	w, rider := newMovementTestWorld(t)
	mount := addMovementMount(t, w, rider, 1001)
	mount.Level = 3
	mount.SetMove(40)

	w.PointUpdate()

	if got, want := mount.GetMove(), 43; got != want {
		t.Fatalf("mount movement after point update = %d, want %d", got, want)
	}
	mount.SetMove(49)
	w.PointUpdate()
	if got, want := mount.GetMove(), 50; got != want {
		t.Fatalf("capped mount movement = %d, want %d", got, want)
	}
}

func TestMountedMovementVerticalArrivalAudiences(t *testing.T) {
	for _, test := range []struct {
		direction string
		origin    string
		arrival   string
	}{
		{direction: "up", origin: "TestPlayer rides up on a pony.\r\n", arrival: "TestPlayer rides in from below on a pony.\r\n"},
		{direction: "down", origin: "TestPlayer rides down on a pony.\r\n", arrival: "TestPlayer rides in from above on a pony.\r\n"},
	} {
		t.Run(test.direction, func(t *testing.T) {
			w, rider := newMovementTestWorld(t)
			w.GetRoomInWorld(1001).Exits = map[string]parser.Exit{
				test.direction: {Direction: test.direction, ToRoom: 1002},
			}
			addMovementMount(t, w, rider, 1001)
			origin := addMovementPlayer(t, w, "Origin", 1001)
			destination := addMovementPlayer(t, w, "Destination", 1002)
			output := captureMovementOutput(w)

			if !w.DoMove(rider, test.direction).Success {
				t.Fatalf("DoMove(%s) failed", test.direction)
			}
			if got := output[origin.Name].String(); !strings.Contains(got, test.origin) {
				t.Fatalf("origin output = %q, want %q", got, test.origin)
			}
			if got := output[destination.Name].String(); !strings.Contains(got, test.arrival) {
				t.Fatalf("destination output = %q, want %q", got, test.arrival)
			}
		})
	}
}

func TestDoMoveMovementCostAndImmortalExemption(t *testing.T) {
	t.Run("mortal pays averaged sector cost", func(t *testing.T) {
		w, actor := newMovementTestWorld(t)
		w.GetRoomInWorld(1001).Sector = SECT_FIELD
		w.GetRoomInWorld(1002).Sector = SECT_DESERT
		actor.SetMove(100)

		if !w.DoMove(actor, "north").Success {
			t.Fatal("DoMove failed")
		}
		if got, want := actor.GetMove(), 95; got != want {
			t.Fatalf("move points = %d, want %d", got, want)
		}
	})

	t.Run("immortal moves free while exhausted", func(t *testing.T) {
		w, actor := newMovementTestWorld(t)
		actor.SetLevel(lvlImmort)
		actor.SetMove(0)

		if !w.DoMove(actor, "north").Success {
			t.Fatal("DoMove failed")
		}
		if got := actor.GetMove(); got != 0 {
			t.Fatalf("move points = %d, want 0", got)
		}
	})
}

func TestDoFollowGuardsAndMessages(t *testing.T) {
	t.Run("charm master guard", func(t *testing.T) {
		w, follower := newMovementTestWorld(t)
		master := addMovementPlayer(t, w, "Master", 1001)
		newLeader := addMovementPlayer(t, w, "Newleader", 1001)
		follower.SetFollowing(master.Name)
		follower.SetAffect(affCharm, true)
		output := captureMovementOutput(w)

		w.DoFollow(follower, newLeader.Name, false)

		if got := follower.GetFollowing(); got != master.Name {
			t.Fatalf("following = %q, want %q", got, master.Name)
		}
		if got := output[follower.Name].String(); !strings.Contains(got, "But you only feel like following Master!") {
			t.Fatalf("follower output = %q", got)
		}
	})

	t.Run("circle rejected", func(t *testing.T) {
		w, first := newMovementTestWorld(t)
		second := addMovementPlayer(t, w, "Second", 1001)
		output := captureMovementOutput(w)
		w.DoFollow(first, second.Name, false)

		w.DoFollow(second, first.Name, false)

		if second.GetFollowing() != "" {
			t.Fatalf("second following = %q, want empty", second.GetFollowing())
		}
		if got := output[second.Name].String(); !strings.Contains(got, "following in loops is not allowed") {
			t.Fatalf("second output = %q", got)
		}
	})

	t.Run("leader and already-following pronouns", func(t *testing.T) {
		w, follower := newMovementTestWorld(t)
		leader := addMovementPlayer(t, w, "Leader", 1001)
		leader.Sex = 1
		output := captureMovementOutput(w)

		w.DoFollow(follower, leader.Name, false)
		w.DoFollow(follower, leader.Name, false)

		if got := output[leader.Name].String(); !strings.Contains(got, "TestPlayer starts following you.") {
			t.Fatalf("leader output = %q", got)
		}
		if got := output[follower.Name].String(); !strings.Contains(got, "You are already following her.") {
			t.Fatalf("follower output = %q", got)
		}
	})

	t.Run("resolves mob leader", func(t *testing.T) {
		w, follower := newMovementTestWorld(t)
		mob := NewMob(&parser.Mob{
			VNum:      9001,
			Keywords:  "grey wolf",
			ShortDesc: "a grey wolf",
		}, 1001)
		mob.ID = 1
		w.mu.Lock()
		w.activeMobs[mob.ID] = mob
		w.mu.Unlock()
		output := captureMovementOutput(w)

		w.DoFollow(follower, "wolf", false)

		if got := follower.GetFollowing(); got != mob.GetName() {
			t.Fatalf("following = %q, want %q", got, mob.GetName())
		}
		if got := output[follower.Name].String(); !strings.Contains(got, "You now follow a grey wolf.") {
			t.Fatalf("follower output = %q", got)
		}
	})
}

func TestPositionCommandsMountAndWake(t *testing.T) {
	t.Run("stand from sitting", func(t *testing.T) {
		w, actor := newMovementTestWorld(t)
		actor.Sex = 1
		actor.SetPosition(combat.PosSitting)
		observer := addMovementPlayer(t, w, "Observer", 1001)
		output := captureMovementOutput(w)

		w.DoStand(actor)

		if actor.GetPosition() != combat.PosStanding {
			t.Fatalf("position = %d, want standing", actor.GetPosition())
		}
		if got := output[observer.Name].String(); !strings.Contains(got, "TestPlayer clambers to her feet.") {
			t.Fatalf("observer output = %q", got)
		}
	})

	t.Run("stand dismounts", func(t *testing.T) {
		w, actor := newMovementTestWorld(t)
		actor.MountName = "pony"
		actor.SetAffect(affMounted, true)
		output := captureMovementOutput(w)

		w.DoStand(actor)

		if actor.IsMounted() {
			t.Fatal("actor remained mounted")
		}
		if got := output[actor.Name].String(); !strings.Contains(got, "You hop off your mount.") {
			t.Fatalf("actor output = %q", got)
		}
	})

	for _, command := range []struct {
		name string
		run  func(*World, *Player)
	}{
		{name: "sit", run: func(w *World, p *Player) { w.DoSit(p) }},
		{name: "rest", run: func(w *World, p *Player) { w.DoRest(p) }},
		{name: "sleep", run: func(w *World, p *Player) { w.DoSleep(p) }},
	} {
		t.Run(command.name+" mounted", func(t *testing.T) {
			w, actor := newMovementTestWorld(t)
			actor.MountName = "pony"
			output := captureMovementOutput(w)

			command.run(w, actor)

			if actor.GetPosition() != combat.PosStanding {
				t.Fatalf("position = %d, want standing", actor.GetPosition())
			}
			if got := output[actor.Name].String(); !strings.Contains(got, "You can't rest while mounted.") {
				t.Fatalf("actor output = %q", got)
			}
		})
	}

	t.Run("magical sleep blocks self wake", func(t *testing.T) {
		w, actor := newMovementTestWorld(t)
		actor.SetPosition(combat.PosSleeping)
		actor.SetAffect(affSleep, true)
		observer := addMovementPlayer(t, w, "Observer", 1001)
		output := captureMovementOutput(w)

		w.DoWake(actor, "")

		if actor.GetPosition() != combat.PosSleeping {
			t.Fatalf("position = %d, want sleeping", actor.GetPosition())
		}
		if got := output[actor.Name].String(); !strings.Contains(got, "You can't wake up!") {
			t.Fatalf("actor output = %q", got)
		}
		if got := output[observer.Name].String(); !strings.Contains(got, "TestPlayer tosses and turns uncomfortably.") {
			t.Fatalf("observer output = %q", got)
		}
	})

	t.Run("wake target", func(t *testing.T) {
		w, actor := newMovementTestWorld(t)
		target := addMovementPlayer(t, w, "Sleeper", 1001)
		target.Sex = 1
		target.SetPosition(combat.PosSleeping)
		observer := addMovementPlayer(t, w, "Observer", 1001)
		output := captureMovementOutput(w)

		w.DoWake(actor, target.Name)

		if target.GetPosition() != combat.PosSitting {
			t.Fatalf("target position = %d, want sitting", target.GetPosition())
		}
		if got := output[actor.Name].String(); !strings.Contains(got, "You wake her up.") {
			t.Fatalf("actor output = %q", got)
		}
		if got := output[target.Name].String(); !strings.Contains(got, "You are awakened by TestPlayer.") {
			t.Fatalf("target output = %q", got)
		}
		if got := output[observer.Name].String(); !strings.Contains(got, "TestPlayer wakes up Sleeper.") {
			t.Fatalf("observer output = %q", got)
		}
	})

	t.Run("magical sleep blocks target wake", func(t *testing.T) {
		w, actor := newMovementTestWorld(t)
		target := addMovementPlayer(t, w, "Sleeper", 1001)
		target.Sex = 1
		target.SetPosition(combat.PosSleeping)
		target.SetAffect(affSleep, true)
		output := captureMovementOutput(w)

		w.DoWake(actor, target.Name)

		if target.GetPosition() != combat.PosSleeping {
			t.Fatalf("target position = %d, want sleeping", target.GetPosition())
		}
		if got := output[actor.Name].String(); !strings.Contains(got, "You can't wake her up!") {
			t.Fatalf("actor output = %q", got)
		}
	})
}

func TestDoEnterAndLeave(t *testing.T) {
	newWorld := func(t *testing.T) (*World, *Player) {
		t.Helper()
		parsed := &parser.World{Rooms: []parser.Room{
			{VNum: 2001, Name: "Outside", Exits: map[string]parser.Exit{
				"north": {Direction: "north", ToRoom: 2002, Keywords: "doorway"},
			}},
			{VNum: 2002, Name: "Inside", Flags: []string{"indoors"}, Exits: map[string]parser.Exit{
				"south": {Direction: "south", ToRoom: 2001, Keywords: "doorway"},
			}},
		}}
		w, err := NewWorld(parsed)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(w.StopAITicker)
		return w, addMovementPlayer(t, w, "Entrant", 2001)
	}

	t.Run("named door exact match", func(t *testing.T) {
		w, actor := newWorld(t)
		if result := w.DoEnter(actor, "doorway"); !result.Success || actor.GetRoom() != 2002 {
			t.Fatalf("result = %+v, room = %d", result, actor.GetRoom())
		}
	})

	t.Run("named door rejects prefix", func(t *testing.T) {
		w, actor := newWorld(t)
		output := captureMovementOutput(w)
		if result := w.DoEnter(actor, "door"); result.Success || actor.GetRoom() != 2001 {
			t.Fatalf("result = %+v, room = %d", result, actor.GetRoom())
		}
		if got := output[actor.Name].String(); !strings.Contains(got, "There is no door here.") {
			t.Fatalf("actor output = %q", got)
		}
	})

	t.Run("automatic enter and leave", func(t *testing.T) {
		w, actor := newWorld(t)
		if result := w.DoEnter(actor, ""); !result.Success || actor.GetRoom() != 2002 {
			t.Fatalf("enter result = %+v, room = %d", result, actor.GetRoom())
		}
		if result := w.DoLeave(actor); !result.Success || actor.GetRoom() != 2001 {
			t.Fatalf("leave result = %+v, room = %d", result, actor.GetRoom())
		}
	})
}
