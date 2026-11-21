//lint:file-ignore U1000 Game logic port — not yet wired to command registry.
package session

import (
	"strings"
)

func cmdStand(s *Session) error {
	s.manager.world.DoStand(s.player)
	return nil
}

func cmdSit(s *Session) error {
	s.manager.world.DoSit(s.player)
	return nil
}

func cmdRest(s *Session) error {
	s.manager.world.DoRest(s.player)
	return nil
}

func cmdSleep(s *Session) error {
	s.manager.world.DoSleep(s.player)
	return nil
}

func cmdWake(s *Session, args []string) error {
	s.manager.world.DoWake(s.player, strings.Join(args, " "))
	return nil
}
