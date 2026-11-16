package game

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/engine"
)

const (
	skillNumScrounge = 176
	skillNumDetect   = 180
)

// CharacterScrounge ports do_scrounge for the switched mobile body. C's
// zero-skill refusal still installs a two-round ch->wait (src/new_cmds2.c:64-75).
func CharacterScrounge(ctx CharacterCommandContext, _ []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	prob := ctx.World.DummyMobSkill(skillNumScrounge)
	if prob == 0 {
		mobileSend(ctx, mob, "You can't seem to find anything edible.\r\n")
		mob.SetWaitStatePulses(engine.PULSE_VIOLENCE * 2)
		return CharacterCommandHandled
	}
	if mob.IsMountedMob() {
		mobileSend(ctx, mob, "Dismount first!\r\n")
		return CharacterCommandHandled
	}
	room := ctx.World.GetRoomInWorld(mob.GetRoom())
	if room == nil {
		return CharacterCommandHandled
	}

	percent := dprng.Number(1, 101)
	foodVNum, find := 0, true
	switch room.Sector {
	case SECT_FOREST:
		foodVNum, find = 28, false
	case SECT_FIELD, SECT_HILLS:
		foodVNum, find = 29, false
	case SECT_DESERT:
		foodVNum, find = 30, false
	case SECT_MOUNTAIN:
		foodVNum = 31
	case SECT_WATER_SWIM, SECT_WATER_NOSWIM, SECT_UNDERWATER:
		foodVNum, find = 27, false
	default:
		mobileSend(ctx, mob, "You need to be in the wilderness to scrounge!\r\n")
	}

	if foodVNum != 0 {
		if percent < prob {
			proto, exists := ctx.World.objs[foodVNum]
			if exists {
				obj := ctx.World.NewObjectFromProto(proto, mob.GetRoom())
				if obj != nil {
					if err := ctx.World.MoveObjectToMobInventoryFront(obj, mob); err == nil {
						message := "You capture and kill $p."
						if find {
							message = "You find $p."
						}
						ActWithPrivateSink(ctx.World, true, mob, nil, obj, nil, message, "", ToChar, mob, ctx.PrivateOutput)
						mob.SetWaitStatePulses(engine.PULSE_VIOLENCE * 2)
					} else {
						ctx.World.ExtractObject(obj, mob.GetRoom())
					}
				}
			}
		} else {
			mobileSend(ctx, mob, "You can't seem to find anything edible.\r\n")
			mob.SetWaitStatePulses(engine.PULSE_VIOLENCE * 2)
		}
	}
	Act(ctx.World, true, mob, nil, nil, nil, "$n searches for something to eat.", "", ToRoom)
	return CharacterCommandHandled
}

// CharacterDetect ports do_detect for a switched mobile. Elf bodies bypass the
// shared zero-skill gate, then take the roll, failure message, and 21-pulse
// wait in C (src/new_cmds2.c:498-521; src/spells.h:228).
func CharacterDetect(ctx CharacterCommandContext, _ []string) CharacterCommandResult {
	mob, ok := ctx.Actor.(*MobInstance)
	if !ok || ctx.World == nil {
		return CharacterCommandUnsupported
	}
	proto := mob.Proto()
	if ctx.World.DummyMobSkill(skillNumDetect) == 0 && (proto == nil || proto.Race != RaceElf) {
		mobileSend(ctx, mob, "Yeah, right.\r\n")
		return CharacterCommandHandled
	}
	if mob.IsAffected(affBlind) {
		mobileSend(ctx, mob, "You're fucking blind, you can't find anything!!\r\n")
		return CharacterCommandHandled
	}
	room := ctx.World.GetRoomInWorld(mob.GetRoom())
	if room == nil {
		return CharacterCommandHandled
	}

	prob := ctx.World.DummyMobSkill(skillNumDetect)
	mobileSend(ctx, mob, "You carefully check the room...\r\n")
	if prob <= dprng.Number(1, 101) {
		mobileSend(ctx, mob, "You can't seem to find anything.\r\n")
		mob.SetWaitStatePulses(engine.PULSE_VIOLENCE + 1)
		return CharacterCommandHandled
	}
	for _, direction := range dirs {
		if exit, exists := room.Exits[direction]; exists && strings.Contains(exit.Keywords, "secret") {
			where := fmt.Sprintf("the %s wall", direction)
			switch direction {
			case "up":
				where = "the ceiling"
			case "down":
				where = "the floor"
			}
			mobileSend(ctx, mob, fmt.Sprintf("You notice something funny about %s.\r\n", where))
			ctx.World.SetRoomFlagBit(mob.GetRoom(), roomFlagSecretMark)
		}
	}
	return CharacterCommandHandled
}
