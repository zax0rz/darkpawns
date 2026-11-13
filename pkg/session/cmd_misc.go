package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/command"
)

func cmdSave(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecSave(player, mob)
	return nil
}

// cmdReport shows a report of surroundings.
func cmdReport(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecReport(player, mob, strings.Join(args, " "))
	return nil
}

// cmdSplit splits gold with the group.
func cmdSplit(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecSplit(player, mob, strings.Join(args, " "))
	return nil
}

// cmdWimpy sets the wimpy threshold.
func cmdWimpy(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecWimpy(player, mob, strings.Join(args, " "))
	return nil
}

// cmdDisplay sets display preferences.
func cmdDisplay(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecDisplay(player, mob, strings.Join(args, " "))
	return nil
}

// cmdDns manages the C DNS cache and pages the list branch through the
// session descriptor, matching C page_string(ch->desc, ...).
func cmdDns(s *Session, args []string) error {
	if page := s.manager.world.ExecDns(s.player, strings.Join(args, " ")); page != "" {
		PageString(s, page)
	}
	return nil
}

// cmdTransform transforms the player's appearance.
func cmdTransform(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecTransform(player, mob, strings.Join(args, " "))
	return nil
}

// cmdRide rides a mount.
func cmdRide(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecRide(player, mob, strings.Join(args, " "))
	return nil
}

// cmdDismount dismounts from a mount.
func cmdDismount(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecDismount(player, mob, strings.Join(args, " "))
	return nil
}

// cmdYank yanks someone from a mount or chair.
func cmdYank(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecYank(player, mob, strings.Join(args, " "))
	return nil
}

// cmdPeek peeks at another player's inventory.
func cmdPeek(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecPeek(player, mob, strings.Join(args, " "))
	return nil
}

// cmdRecall recalls to the home city.
func cmdRecall(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecRecall(player, mob, strings.Join(args, " "))
	return nil
}

// cmdAppraise appraises an item's value.
func cmdAppraise(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecAppraise(player, mob, strings.Join(args, " "))
	return nil
}

// cmdScout scouts ahead for danger.
func cmdScout(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecScout(player, mob, strings.Join(args, " "))
	return nil
}

// cmdPour pours liquid between containers or out onto the ground.
func cmdPour(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecPour(player, mob, strings.Join(args, " "))
	return nil
}

// cmdFill fills a container from a room fountain.
func cmdFill(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecFill(player, mob, strings.Join(args, " "))
	return nil
}

// cmdRoll rolls a random number.
func cmdRoll(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecRoll(player, mob, strings.Join(args, " "))
	return nil
}

// cmdVisible makes the player visible.
func cmdVisible(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecVisible(player, mob, strings.Join(args, " "))
	return nil
}

// cmdTrack senses the direction of a tracked target.
func cmdTrack(s *Session, args []string) error {
	s.manager.world.ExecTrack(s.player, strings.Join(args, " "))
	return nil
}

// cmdInactive toggles inactive status.
func cmdInactive(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecInactive(player, mob, strings.Join(args, " "))
	return nil
}

// cmdAuto toggles auto-attack mode.
func cmdAuto(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecAuto(player, mob, strings.Join(args, " "))
	return nil
}

// cmdAFK toggles away-from-keyboard status.
func cmdAFK(ctx command.ExecutionContext, args []string) error {
	player, mob := executionBodyParts(ctx)
	ctx.World.ExecAFK(player, mob, strings.Join(args, " "))
	return nil
}

// cmdNotHere is C's do_not_here fallback for commands whose real behavior is
// supplied by a room special procedure (act.other.c:206-211).
func cmdNotHere(s *Session, _ []string) error {
	s.Send("Sorry, but you cannot do that here!\r\n")
	return nil
}

// cmdClan — player-facing clan management (ported from clan.c)
func cmdClan(s *Session, args []string) error {
	s.manager.world.ExecClanCommand(s.player, strings.Join(args, " "))
	return nil
}

// cmdHcontrol — admin house control (ported from house.c)
func cmdHcontrol(s *Session, args []string) error {
	s.manager.world.Hcontrol(s.player, strings.Join(args, " "))
	return nil
}

// cmdHouse — player-facing house management (ported from house.c)
func cmdHouse(s *Session, args []string) error {
	s.manager.world.DoHouse(s.player, strings.Join(args, " "))
	return nil
}

// cmdBan handles the "ban" admin command (ported from ban.c do_ban). The
// producer fires inside DoBan, before the acknowledgement it writes through
// ack (ban.c:205-209).
func cmdBan(s *Session, args []string) error {
	name, invis := s.mudlogActor()
	s.manager.world.ExecBan(name, invis, strings.Join(args, " "), s.sendText)
	return nil
}

// cmdUnban handles the "unban" admin command (ported from ban.c do_unban).
// C acknowledges before it logs (ban.c:237-243), so the ack writer is passed
// into DoUnban.
func cmdUnban(s *Session, args []string) error {
	name, invis := s.mudlogActor()
	// Terminated acks ride Send: sendText's MsgText render appends another
	// terminator, which the missing prompt frame used to cancel (PF-063
	// cluster 5 class).
	s.manager.world.ExecUnban(name, invis, strings.Join(args, " "), s.Send)
	return nil
}

// cmdWhod handles the "whod" admin command (ported from whod.c do_whod).
func cmdWhod(s *Session, args []string) error {
	// whod's acks are C-terminated (some LFCR, whod.c): Send renders them
	// verbatim; sendText appended a second terminator.
	msg := s.manager.world.ExecWhod(s.player, strings.Join(args, " "), s.Send)
	if msg != "" {
		s.Send(msg)
	}
	return nil
}
