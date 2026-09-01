package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// cmdOrder — order a charmed pet or follower to perform a command.
func cmdOrder(s *Session, args []string) error {
	if len(args) < 2 {
		s.Send("Order who to do what?\r\n")
		return nil
	}

	targetName := strings.ToLower(args[0])
	orderCmd := strings.Join(args[1:], " ")
	world := s.manager.world
	roomVNum := s.player.GetRoom()

	// C do_order resolves a room character before checking the special
	// "followers" pseudo-target (act.offensive.c:304-306). This includes the
	// actor, players, and mobs through get_char_room_vis (handler.c:1276-1300).
	target, found := world.ResolveCharInRoom(s.player, targetName)
	if !found && !isFollowersTarget(targetName) {
		s.Send("That person isn't here.\r\n")
		return nil
	}
	if found && target.Player == s.player {
		s.Send("You obviously suffer from skitzofrenia.\r\n")
		return nil
	}

	// A charmed actor cannot issue orders (act.offensive.c:310-317).
	if s.player.IsAffected(game.AffCharm) {
		s.Send("Your superior would not aprove of you giving orders.\r\n")
		return nil
	}

	if found {
		victim := targetActor(target)
		if victim == nil {
			return fmt.Errorf("order target is not an actor")
		}
		game.Act(world, false, victim, s.player, nil, nil,
			fmt.Sprintf("$N orders you to '%s'", orderCmd), "", game.ToChar)
		game.Act(world, false, s.player, victim, nil, nil,
			"$n gives $N an order.", "", game.ToRoom)

		if orderFollowing(victim) != s.player.Name || !world.IsCharmedI(victim) {
			game.Act(world, false, victim, nil, nil, nil,
				"$n has an indifferent look.", "", game.ToRoom)
			return nil
		}

		s.Send("Okay.\r\n")
		executeOrder(world, victim, orderCmd)
		return nil
	}

	// C's followers arm emits the room announcement first, then dispatches each
	// same-room charmed follower, and acknowledges only if one was found
	// (act.offensive.c:337-357).
	game.Act(world, false, s.player, nil, nil, nil,
		fmt.Sprintf("$n issues the order '%s'.", orderCmd), "", game.ToRoom)
	foundFollower := false
	for _, follower := range world.GetFollowerActors(s.player.Name) {
		if follower.GetRoom() != roomVNum || !world.IsCharmedI(follower) {
			continue
		}
		foundFollower = true
		executeOrder(world, follower, orderCmd)
	}
	if foundFollower {
		s.Send("Okay.\r\n")
	} else {
		s.Send("Nobody here is a loyal subject of yours!\r\n")
	}
	return nil
}

func isFollowersTarget(argument string) bool {
	return argument != "" && len(argument) <= len("followers") && "followers"[:len(argument)] == argument
}

func targetActor(target game.CharTarget) game.Actor {
	if target.Player != nil {
		return target.Player
	}
	if target.Mob != nil {
		return target.Mob
	}
	return nil
}

func orderFollowing(actor game.Actor) string {
	switch value := actor.(type) {
	case *game.Player:
		return value.GetFollowing()
	case *game.MobInstance:
		return value.GetFollowing()
	default:
		return ""
	}
}

func executeOrder(world *game.World, follower game.Actor, command string) {
	switch value := follower.(type) {
	case *game.Player:
		if world.CommandExecFunc != nil {
			world.CommandExecFunc(value, command)
		}
	case *game.MobInstance:
		world.ExecMobCommand(value.GetVNum(), command)
	}
}

// cmdOrgasm implements do_otouch (src/new_cmds.c:666-724). The command is
// intentionally kept beside the other combat-special session commands even
// though C registers it as a top-level immortal utility.
func cmdOrgasm(s *Session, args []string) error {
	// C has this handler-level guard in addition to the interpreter row's
	// LVL_IMMORT gate. It remains reachable for NPC command re-entry.
	if s.player.GetLevel() < game.LVL_IMMORT {
		s.Send("Come again?\n\r")
		return nil
	}

	targetName, _ := game.OneArgument(strings.Join(args, " "))
	if targetName == "" {
		s.Send("Yeah, ok, but who?\n\r")
		return nil
	}

	target, found := s.manager.world.ResolveCharInRoom(s.player, targetName)
	if !found {
		s.Send("Humm...seems that person doesn't need your help.\n\r")
		return nil
	}
	victim := targetActor(target)
	if victim == nil {
		return fmt.Errorf("orgasm target is not an actor")
	}

	if victim == s.player {
		s.Send("Yes, it WILL fall off if you don't stop that.\n\r")
		game.Act(s.manager.world, false, s.player, victim, nil, nil,
			"$n just can't stop playing with $Mself!\n\r", "", game.ToNotVict)
		return nil
	}

	world := s.manager.world
	game.Act(world, false, s.player, victim, nil, nil,
		"You touch $N, who slumps over in orgasm!", "", game.ToChar)
	game.Act(world, false, victim, s.player, nil, nil,
		"A wonderful feeling spreads throughout your entire body as $N touches you in all the right places...", "", game.ToChar)
	victim.SendMessage("You explode with a breath-taking orgasm!!\r\n")
	game.Act(world, false, s.player, victim, nil, nil,
		"$N collapses in a quivering, moaning heap as $n touches them..", "", game.ToNotVict)
	addOrgasmHealth(victim)

	if victim.GetSex() == game.SexMale {
		victim.SendMessage("God, you need some beer and some food.\n\r")
		game.Act(world, false, s.player, victim, nil, nil,
			"A dark stain appears under $N's armor and slowly spreads down his leg.", "", game.ToNotVict)
		if player, ok := victim.(*game.Player); ok && player.GetCondition(game.CondFull) >= 0 {
			player.SetCondition(game.CondFull, 0)
		}
	}

	if victim.GetSex() == game.SexFemale && s.player.GetSex() == game.SexMale {
		victim.SendMessage("Once is never enough, you beg for more!\n\r")
		game.Act(world, false, s.player, victim, nil, nil,
			"$N pleads for more and $n deftly glides a hand beneath her panties!", "", game.ToNotVict)
	}
	return nil
}

func addOrgasmHealth(victim game.Actor) {
	switch value := victim.(type) {
	case *game.Player:
		value.SetHealth(value.GetHealth() + 2)
	case *game.MobInstance:
		value.SetHealth(value.GetHealth() + 2)
	}
}
