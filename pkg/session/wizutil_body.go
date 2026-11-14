package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// mobWizutilBody is do_wizutil for a mobile body (src/act.wizard.c:2077-2219).
// PLR_CHOSEN is NPC-guarded (src/utils.h:234), so the body gate is the body
// level; the target comes from the shared get_char_vis port and the
// squelch/pardon mutations run on the player target.
func mobWizutilBody(subcmd string) characterActorHandler {
	return func(ctx game.CharacterCommandContext, _ string, args []string) game.CharacterCommandResult {
		mob, ok := ctx.Actor.(*game.MobInstance)
		if !ok || ctx.World == nil {
			return game.CharacterCommandUnsupported
		}
		if mob.GetLevel() < LVL_IMMORT {
			mob.SendMessage("Huh?!?\r\n")
			return game.CharacterCommandHandled
		}
		argument := ctx.RawArgs
		if argument == "" && len(args) > 0 {
			argument = strings.Join(args, " ")
		}
		name, _ := game.OneArgument(argument)
		if name == "" {
			mob.SendMessage("Yes, but for whom?!?\r\n")
			return game.CharacterCommandHandled
		}
		resolved, found := ctx.World.GetCharVis(mob, name)
		if !found || resolved.Player == nil {
			if found && resolved.Mob != nil {
				mob.SendMessage("You can't do that to a mob!\r\n")
				return game.CharacterCommandHandled
			}
			mob.SendMessage("There is no such player.\r\n")
			return game.CharacterCommandHandled
		}
		target := resolved.Player
		if target.GetLevel() > mob.GetLevel() && target.GetLevel() > game.LVL_IMMORT {
			mob.SendMessage("Hmmm...you'd better not.\r\n")
			return game.CharacterCommandHandled
		}
		// C GET_NAME(ch) is the short description for a mobile.
		actorName := mob.GetName()
		switch subcmd {
		case "mute":
			newState := target.GetFlags()&(1<<game.PlrNoshout) == 0
			target.SetPlrFlag(game.PlrNoshout, newState)
			onoff := "OFF"
			if newState {
				onoff = "ON"
			}
			game.MudLog(fmt.Sprintf("(GC) Squelch %s for %s by %s.", onoff, target.GetName(), actorName),
				game.MudlogBrief, game.LVL_GOD, true)
			mob.SendMessage(fmt.Sprintf("(GC) Squelch %s for %s by %s.\r\n", onoff, target.GetName(), actorName))
		case "pardon":
			if target.GetFlags()&(1<<game.PlrOutlaw) == 0 {
				mob.SendMessage("Your victim is not flagged.\r\n")
				return game.CharacterCommandHandled
			}
			target.SetPlrFlag(game.PlrOutlaw, false)
			mob.SendMessage("Pardoned.\r\n")
			target.SendMessage("You have been pardoned by the Gods!\r\n")
			game.MudLog(fmt.Sprintf("(GC) %s pardoned by %s", target.GetName(), actorName),
				game.MudlogBrief, game.LVL_GOD, true)
		}
		return game.CharacterCommandHandled
	}
}
