package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// ---------------------------------------------------------------------------
// Social commands (ported from act.other.c / act.social.c)
// ---------------------------------------------------------------------------

// cmdDream handles the 'dream' command — players see their dream state.
// Shows dream text while sleeping.
func cmdDream(s *Session, args []string) error {
	_ = args

	pos := s.player.GetPosition()
	if pos != combat.PosSleeping {
		s.Send("You are awake.")
		return nil
	}

	// In the original MUD, dream is called during point_update for sleeping chars
	// and shows random dream messages. For now, show a simple status.
	s.Send("You are lost in a dream...")

	return nil
}

// cmdInsult insults a target in the room.
// Find target in room, send insult emote.
func cmdInsult(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Insult whom?")
		return nil
	}

	targetName := strings.ToLower(args[0])
	roomVNum := s.player.GetRoomVNum()

	// Look for target in the room (players first, then mobs)
	found := false

	players := s.manager.world.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if strings.Contains(strings.ToLower(p.Name), targetName) {
			s.Send(fmt.Sprintf("You insult %s.", p.Name))
			broadcastToRoomText(s, roomVNum,
				fmt.Sprintf("%s insults %s.", s.player.Name, p.Name))
			found = true
			break
		}
	}

	if !found {
		mobs := s.manager.world.GetMobsInRoom(roomVNum)
		for _, m := range mobs {
			if strings.Contains(strings.ToLower(m.GetShortDesc()), targetName) {
				s.Send(fmt.Sprintf("You insult %s.", m.GetShortDesc()))
				broadcastToRoomText(s, roomVNum,
					fmt.Sprintf("%s insults %s.", s.player.Name, m.GetShortDesc()))
				found = true
				break
			}
		}
	}

	if !found {
		s.Send("They aren't here.")
	}

	return nil
}
