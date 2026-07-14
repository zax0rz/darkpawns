package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Door commands are deliberately thin session adapters. Container and exit
// lookup, the C precondition ladder, mutations, and messages all live in game.
func cmdOpen(s *Session, args []string) error {
	s.manager.world.DoOpen(s.player, strings.Join(args, " "))
	return nil
}

func cmdClose(s *Session, args []string) error {
	s.manager.world.DoClose(s.player, strings.Join(args, " "))
	return nil
}

func cmdLock(s *Session, args []string) error {
	s.manager.world.DoLock(s.player, strings.Join(args, " "))
	return nil
}

func cmdUnlock(s *Session, args []string) error {
	s.manager.world.DoUnlock(s.player, strings.Join(args, " "))
	return nil
}

func cmdPick(s *Session, args []string) error {
	s.manager.world.DoPick(s.player, strings.Join(args, " "))
	return nil
}

func cmdKnock(s *Session, args []string) error {
	dir := ""
	if len(args) > 0 {
		dir = resolveDirection(strings.ToLower(args[0]))
	}
	if dir == "" {
		s.Send("Knock on what?  Try north, south, east, west, up, or down.")
		return nil
	}

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return nil
	}
	exit, exists := room.Exits[dir]
	if !exists || exit.ExitInfo&parser.ExitIsDoor == 0 {
		s.Send("There is nothing to knock on in that direction.")
		return nil
	}

	doorDesc := dir
	if exit.Keywords != "" {
		doorDesc = exit.Keywords
	}
	s.Send(fmt.Sprintf("You knock on the %s.", doorDesc))
	doorBroadcast(s, fmt.Sprintf("%s knocks on the %s.", s.player.Name, doorDesc))
	if exit.ToRoom > 0 {
		s.manager.BroadcastToRoom(exit.ToRoom, []byte("Someone knocks on the door from the other side."), "")
	}
	return nil
}
