package session

import (
	"strings"
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
