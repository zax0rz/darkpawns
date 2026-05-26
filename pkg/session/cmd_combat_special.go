package session

import (
	"fmt"
	"strings"
)

// cmdOrder — order a charmed pet or follower to perform a command.
func cmdOrder(s *Session, args []string) error {
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
		if !strings.HasPrefix(strings.ToLower(mob.GetName()), targetName) &&
			!strings.HasPrefix(strings.ToLower(mob.GetShortDesc()), targetName) {
			continue
		}
		if mob.GetFollowing() != s.player.Name {
			s.Send(fmt.Sprintf("%s isn't following you.", mob.GetShortDesc()))
			return nil
		}
		s.Send(fmt.Sprintf("%s obeys your order.", mob.GetShortDesc()))
		broadcastCombatMsg(s, room.VNum, "order",
			fmt.Sprintf("%s orders %s to '%s'.", s.player.Name, mob.GetShortDesc(), orderCmd))
		s.manager.world.ExecMobCommand(mob.GetVNum(), orderCmd)
		return nil
	}

	s.Send("No follower by that name here.")
	return nil
}
