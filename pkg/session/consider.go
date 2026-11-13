package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/command"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// cmdConsider compares the player against a visible character in the room.
// Source: act.informative.c do_consider() lines 2330-2431.
func cmdConsider(s *Session, args []string) error {
	s.manager.world.DoConsider(s.player, strings.Join(args, " "))
	return nil
}

func cmdConsiderBody(ctx command.ExecutionContext, args []string) error {
	s := commandSessionFromContext(ctx)
	body := ctx.Body
	if s == nil || body == nil {
		return nil
	}
	if _, ok := body.(*game.Player); ok {
		return cmdConsider(s, args)
	}
	s.manager.world.DoConsider(body, strings.Join(args, " "))
	return nil
}
