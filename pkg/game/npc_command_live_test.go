package game

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/dprng"
)

func init() {
	testWorldInit = wireLiveNPCDispatchForTest
}

// wireLiveNPCDispatchForTest installs the same live Character* / NPC*Command
// handlers that pkg/session registers into cmdRegistry on production boot, so
// pkg/game unit tests exercise the live command bodies without reintroducing a
// production fallback table (DP-1596).
func wireLiveNPCDispatchForTest(w *World) {
	if w == nil {
		return
	}
	w.CharacterCommandDirectExecFunc = func(body combat.Combatant, commandName, argument string, subcmd int) CharacterCommandResult {
		mob, ok := body.(*MobInstance)
		if !ok {
			return CharacterCommandUnsupported
		}
		commandName = strings.ToLower(strings.TrimSpace(commandName))
		ctx := CharacterCommandContext{
			World:      w,
			Actor:      mob,
			RawArgs:    argument,
			DirectCall: true,
			Subcmd:     subcmd,
		}
		args := strings.Fields(argument)
		switch commandName {
		case "say", "'":
			w.NPCSayCommand(mob, argument)
		case "tell":
			w.NPCTellCommand(mob, argument)
		case "emote", ":":
			w.NPCEmoteCommand(mob, argument)
		case "echo":
			w.NPCEchoCommand(mob, argument)
		case "gecho":
			w.NPCGechoCommand(mob, argument)
		case "gossip":
			w.NPCGossipCommand(mob, argument)
		case "shout":
			w.NPCShoutCommand(mob, argument)
		case "auction":
			w.NPCAuctionCommand(mob, argument)
		case "stand":
			return CharacterStand(ctx)
		case "sit":
			return CharacterSit(ctx)
		case "rest":
			return CharacterRest(ctx)
		case "sleep":
			return CharacterSleep(ctx)
		case "wake":
			return CharacterWake(ctx)
		case "flee":
			return CharacterFlee(ctx)
		case "follow":
			return CharacterFollow(ctx)
		case "give":
			w.NPCGiveCommand(mob, argument)
		case "get":
			return CharacterGet(ctx)
		case "wear":
			return CharacterWear(ctx)
		case "rescue", "headbutt", "parry", "bash", "charge", "disarm", "backstab":
			return CharacterCombatSkill(ctx, commandName, args)
		case "berserk":
			return CharacterBerserk(ctx, args)
		case "behead":
			return CharacterBehead(ctx, args)
		case "assist":
			w.NPCAssistCommand(mob, argument)
		case "look":
			return w.NPCLookCommand(mob, argument)
		case "ride", "mount":
			return CharacterMount(ctx)
		case "dismount":
			return CharacterDismount(ctx)
		default:
			if _, isSocial := socialMinPosition[commandName]; isSocial {
				w.NPCSocialCommand(mob, commandName, argument)
				return CharacterCommandHandled
			}
			return CharacterCommandUnsupported
		}
		return CharacterCommandHandled
	}

	w.CharacterCommandExecFunc = func(body combat.Combatant, line string) CharacterCommandResult {
		mob, ok := body.(*MobInstance)
		if !ok {
			return CharacterCommandUnsupported
		}
		if dprng.Number(0, 3) == 0 {
			mob.RemoveAffected(affHide)
		}
		line = strings.TrimLeft(line, " ")
		if line == "" {
			return CharacterCommandHandled
		}
		var word, rest string
		if c := line[0]; !isASCIIAlpha(c) {
			word, rest = line[:1], line[1:]
		} else {
			word, rest = anyOneArg(line)
		}
		word = strings.ToLower(word)
		pos := mob.GetPosition()
		switch word {
		case "say", "'", "give", "stand":
			if pos < combat.PosResting {
				return CharacterCommandHandled
			}
		case "tell":
			if pos < combat.PosMortally {
				return CharacterCommandHandled
			}
		case "emote", ":":
			if pos < combat.PosResting || mob.GetLevel() < 1 {
				return CharacterCommandHandled
			}
		default:
			if minPos, isSocial := socialMinPosition[word]; isSocial && pos < minPos {
				return CharacterCommandHandled
			}
		}
		return w.CharacterCommandDirectExecFunc(mob, word, rest, 0)
	}
}
