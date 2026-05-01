package session

import (
	"strings"
)

func cmdSave(s *Session, args []string) error {
	s.manager.world.ExecSave(s.player)
	return nil
}

// cmdReport shows a report of surroundings.
func cmdReport(s *Session, args []string) error {
	s.manager.world.ExecReport(s.player, strings.Join(args, " "))
	return nil
}

// cmdSplit splits gold with the group.
func cmdSplit(s *Session, args []string) error {
	s.manager.world.ExecSplit(s.player, strings.Join(args, " "))
	return nil
}

// cmdWimpy sets the wimpy threshold.
func cmdWimpy(s *Session, args []string) error {
	s.manager.world.ExecWimpy(s.player, strings.Join(args, " "))
	return nil
}

// cmdDisplay sets display preferences.
func cmdDisplay(s *Session, args []string) error {
	s.manager.world.ExecDisplay(s.player, strings.Join(args, " "))
	return nil
}

// cmdTransform transforms the player's appearance.
func cmdTransform(s *Session, args []string) error {
	s.manager.world.ExecTransform(s.player, strings.Join(args, " "))
	return nil
}

// cmdRide rides a mount.
func cmdRide(s *Session, args []string) error {
	s.manager.world.ExecRide(s.player, strings.Join(args, " "))
	return nil
}

// cmdDismount dismounts from a mount.
func cmdDismount(s *Session, args []string) error {
	s.manager.world.ExecDismount(s.player, strings.Join(args, " "))
	return nil
}

// cmdYank yanks someone from a mount or chair.
func cmdYank(s *Session, args []string) error {
	s.manager.world.ExecYank(s.player, strings.Join(args, " "))
	return nil
}

// cmdPeek peeks at another player's inventory.
func cmdPeek(s *Session, args []string) error {
	s.manager.world.ExecPeek(s.player, strings.Join(args, " "))
	return nil
}

// cmdRecall recalls to the home city.
func cmdRecall(s *Session, args []string) error {
	s.manager.world.ExecRecall(s.player, strings.Join(args, " "))
	return nil
}

// cmdStealth enters stealth mode.
func cmdStealth(s *Session, args []string) error {
	s.manager.world.ExecStealth(s.player, strings.Join(args, " "))
	return nil
}

// cmdAppraise appraises an item's value.
func cmdAppraise(s *Session, args []string) error {
	s.manager.world.ExecAppraise(s.player, strings.Join(args, " "))
	return nil
}

// cmdScout scouts ahead for danger.
func cmdScout(s *Session, args []string) error {
	s.manager.world.ExecScout(s.player, strings.Join(args, " "))
	return nil
}

// cmdRoll rolls a random number.
func cmdRoll(s *Session, args []string) error {
	s.manager.world.ExecRoll(s.player, strings.Join(args, " "))
	return nil
}

// cmdVisible makes the player visible.
func cmdVisible(s *Session, args []string) error {
	s.manager.world.ExecVisible(s.player, strings.Join(args, " "))
	return nil
}

// cmdInactive toggles inactive status.
func cmdInactive(s *Session, args []string) error {
	s.manager.world.ExecInactive(s.player, strings.Join(args, " "))
	return nil
}

// cmdAuto toggles auto-attack mode.
func cmdAuto(s *Session, args []string) error {
	s.manager.world.ExecAuto(s.player, strings.Join(args, " "))
	return nil
}

// cmdGenTog toggles a general option.
func cmdGenTog(s *Session, args []string) error {
	s.manager.world.ExecGenTog(s.player, strings.Join(args, " "))
	return nil
}

// cmdBug reports a bug.
func cmdBug(s *Session, args []string) error {
	s.manager.world.ExecGenWrite(s.player, "bug", strings.Join(args, " "))
	return nil
}

// cmdTypo reports a typo.
func cmdTypo(s *Session, args []string) error {
	s.manager.world.ExecGenWrite(s.player, "typo", strings.Join(args, " "))
	return nil
}

// cmdIdea submits an idea.
func cmdIdea(s *Session, args []string) error {
	s.manager.world.ExecGenWrite(s.player, "idea", strings.Join(args, " "))
	return nil
}

// cmdTodo submits a todo suggestion.
func cmdTodo(s *Session, args []string) error {
	s.manager.world.ExecGenWrite(s.player, "todo", strings.Join(args, " "))
	return nil
}

// cmdAFK toggles away-from-keyboard status.
func cmdAFK(s *Session, args []string) error {
	s.manager.world.ExecAFK(s.player, strings.Join(args, " "))
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

// cmdBan handles the "ban" admin command (ported from ban.c do_ban).
func cmdBan(s *Session, args []string) error {
	msg := s.manager.world.ExecBan(s.player, strings.Join(args, " "))
	s.sendText(msg)
	return nil
}

// cmdUnban handles the "unban" admin command (ported from ban.c do_unban).
func cmdUnban(s *Session, args []string) error {
	msg := s.manager.world.ExecUnban(s.player, strings.Join(args, " "))
	s.sendText(msg)
	return nil
}

// cmdWhod handles the "whod" admin command (ported from whod.c do_whod).
func cmdWhod(s *Session, args []string) error {
	msg := s.manager.world.ExecWhod(s.player, strings.Join(args, " "))
	s.sendText(msg)
	return nil
}
