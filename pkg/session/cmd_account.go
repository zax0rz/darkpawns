package session

import (
	"strings"
)

// cmdPrompt is the C prompt alias for do_display.
// Source: src/interpreter.c:619 -> src/act.other.c:1024-1082.
func cmdPrompt(s *Session, args []string) error {
	s.manager.world.ExecDisplay(s.player, strings.Join(args, " "))
	return nil
}
