package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/command"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func contextArgument(ctx command.ExecutionContext, args []string) string {
	if ctx.RawArgs != "" {
		return ctx.RawArgs
	}
	return strings.Join(args, " ")
}

func splitCommunicationTarget(argument string) (string, string) {
	argument = strings.TrimLeft(argument, cCommandWhitespace)
	if argument == "" {
		return "", ""
	}
	if split := strings.IndexAny(argument, cCommandWhitespace); split >= 0 {
		return strings.ToLower(argument[:split]), strings.TrimLeft(argument[split:], cCommandWhitespace)
	}
	return strings.ToLower(argument), ""
}

func cmdRaceSayBody(ctx command.ExecutionContext, args []string) error {
	s := commandSessionFromContext(ctx)
	if mob, ok := ctx.Body.(*game.MobInstance); ok {
		ctx.World.NPCDoRaceSay(mob, sanitizeMessage(contextArgument(ctx, args)))
		return nil
	}
	if ctx.RawArgs != "" {
		return cmdRaceSayText(s, ctx.RawArgs)
	}
	return cmdRaceSay(s, args)
}

func cmdGtellBody(ctx command.ExecutionContext, args []string) error {
	s := commandSessionFromContext(ctx)
	argument := contextArgument(ctx, args)
	if mob, ok := ctx.Body.(*game.MobInstance); ok {
		argument = sanitizeMessage(argument)
		if len(args) > 0 {
			filtered, block := filterCommMessage(s, argument)
			if block {
				s.Send("Your message was blocked.")
				return nil
			}
			argument = filtered
		}
		ctx.World.GroupSayBody(mob, argument)
		return nil
	}
	if ctx.RawArgs != "" {
		return cmdGtellText(s, ctx.RawArgs)
	}
	return cmdGtell(s, args)
}

func cmdChannelBody(ctx command.ExecutionContext, args []string, channel string) error {
	s := commandSessionFromContext(ctx)
	if mob, ok := ctx.Body.(*game.MobInstance); ok {
		message := sanitizeMessage(contextArgument(ctx, args))
		if len(args) > 0 {
			filtered, block := filterCommMessage(s, message)
			if block {
				s.Send("Your message was blocked.")
				return nil
			}
			message = filtered
		}
		ctx.World.NPCGenCommCommand(mob, message, channel)
		return nil
	}
	return cmdChannel(s, args, channel)
}

func cmdHollerBody(ctx command.ExecutionContext, args []string) error {
	return cmdChannelBody(ctx, args, "holler")
}

func cmdGratsBody(ctx command.ExecutionContext, args []string) error {
	return cmdChannelBody(ctx, args, "grats")
}

func cmdNewbieChannelBody(ctx command.ExecutionContext, args []string) error {
	return cmdChannelBody(ctx, args, "newbie")
}

func cmdAskBody(ctx command.ExecutionContext, args []string) error {
	s := commandSessionFromContext(ctx)
	mob, ok := ctx.Body.(*game.MobInstance)
	if !ok {
		return cmdAsk(s, args)
	}
	argument := contextArgument(ctx, args)
	targetName, message := splitCommunicationTarget(argument)
	if message != "" {
		message = sanitizeMessage(message)
		filtered, block := filterCommMessage(s, message)
		if block {
			mob.SendMessage("Your message was blocked.\r\n")
			return nil
		}
		argument = targetName + " " + filtered
	}
	ctx.World.DoSpecCommMobile(mob, argument, true)
	return nil
}

func cmdWhisperBody(ctx command.ExecutionContext, args []string) error {
	s := commandSessionFromContext(ctx)
	mob, ok := ctx.Body.(*game.MobInstance)
	if !ok {
		return cmdWhisper(s, args)
	}
	argument := contextArgument(ctx, args)
	targetName, message := splitCommunicationTarget(argument)
	if message != "" {
		message = sanitizeMessage(message)
		filtered, block := filterCommMessage(s, message)
		if block {
			mob.SendMessage("Your message was blocked.\r\n")
			return nil
		}
		argument = targetName + " " + filtered
	}
	ctx.World.DoSpecCommMobile(mob, argument, false)
	return nil
}

func cmdQsayBody(ctx command.ExecutionContext, args []string) error {
	s := commandSessionFromContext(ctx)
	if ctx.Body != nil && ctx.Body.IsNPC() {
		// NPC player_specials point at C's zeroed dummy_mob, so PRF_QUEST is
		// clear and do_qcomm returns this exact refusal (src/act.comm.c:1306).
		ctx.SendPrivate("You aren't even part of the quest!\r\n")
		return nil
	}
	if ctx.RawArgs != "" {
		return cmdQsayText(s, ctx.RawArgs)
	}
	return cmdQsay(s, args)
}

func cmdCTellBody(ctx command.ExecutionContext, args []string) error {
	s := commandSessionFromContext(ctx)
	if mob, ok := ctx.Body.(*game.MobInstance); ok {
		message := sanitizeMessage(contextArgument(ctx, args))
		if len(args) > 0 {
			filtered, block := filterCommMessage(s, message)
			if block {
				mob.SendMessage("Your message was blocked.\r\n")
				return nil
			}
			message = filtered
		}
		ctx.World.NPCCTellCommand(mob, message)
		return nil
	}
	return cmdCTell(s, args)
}

func cmdInsultBody(ctx command.ExecutionContext, args []string) error {
	s := commandSessionFromContext(ctx)
	if mob, ok := ctx.Body.(*game.MobInstance); ok {
		game.DoInsult(ctx.World, mob, contextArgument(ctx, args))
		return nil
	}
	return cmdInsult(s, args)
}

func cmdThinkBody(ctx command.ExecutionContext, args []string) error {
	s := commandSessionFromContext(ctx)
	mob, ok := ctx.Body.(*game.MobInstance)
	if !ok {
		return cmdThink(s, args)
	}
	argument := contextArgument(ctx, args)
	trimmed := strings.TrimLeft(argument, cCommandWhitespace)
	if trimmed == "" {
		game.Act(ctx.World, false, mob, nil, nil, nil,
			"$n thinks about life, the universe, and everything.", "", game.ToRoom)
		mob.SendMessage("You think about life, the universe, and everything.\n\r")
		return nil
	}
	if mob.GetInt() == 0 {
		mob.SendMessage("You try to think aloud, but cannot!\r\n")
		return nil
	}
	trimmed = game.DeleteANSIControls(trimmed)
	roomText := "$n thinks . o O ( " + trimmed + " )"
	game.Act(ctx.World, false, mob, nil, nil, nil, roomText, "", game.ToRoom)
	mob.SendMessage(fmt.Sprintf("You think . o O ( %s )\n\r", trimmed))
	return nil
}
