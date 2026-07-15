package session

import "strings"

// cmdConsider compares the player against a visible character in the room.
// Source: act.informative.c do_consider() lines 2330-2431.
func cmdConsider(s *Session, args []string) error {
	s.manager.world.DoConsider(s.player, strings.Join(args, " "))
	return nil
}
