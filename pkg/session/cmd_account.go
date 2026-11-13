package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/command"
)

// cmdPrompt is the C prompt alias for do_display.
// Source: src/interpreter.c:619 -> src/act.other.c:1024-1082.
func cmdPrompt(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecDisplay(player, mob, strings.Join(args, " "))
	return nil
}
