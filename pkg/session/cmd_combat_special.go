package session

import (
	"fmt"
	"strings"
)

// cmdOrder — order a pet or follower to perform a command (LVL_IMMORT).
func cmdOrder(s *Session, args []string) error {
	if !checkLevel(s, LVL_IMMORT) {
		s.Send("Huh?!?")
		return nil
	}
	if len(args) < 2 {
		s.Send("Order whom to do what?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	orderCmd := strings.Join(args[1:], " ")

	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	for _, mob := range s.manager.world.GetMobsInRoom(room.VNum) {
		if strings.Contains(strings.ToLower(mob.GetName()), targetName) ||
			strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) {
			s.Send(fmt.Sprintf("%s obeys your order: %s", mob.GetShortDesc(), orderCmd))
			broadcastCombatMsg(s, room.VNum, "order",
				fmt.Sprintf("%s orders %s to '%s'.", s.player.Name, mob.GetShortDesc(), orderCmd))
			return nil
		}
	}

	s.Send("No follower by that name here.")
	return nil
}
