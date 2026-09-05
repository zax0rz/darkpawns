package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdMove(s *Session, direction string) error {
	return finishMovementCommand(s, s.manager.world.DoMove(s.player, direction))
}

func cmdEnter(s *Session, args []string) error {
	return finishMovementCommand(s, s.manager.world.DoEnter(s.player, strings.Join(args, " ")))
}

func cmdLeave(s *Session) error {
	return finishMovementCommand(s, s.manager.world.DoLeave(s.player))
}

// finishMovementCommand is the transport adapter around game-owned movement.
// The game transaction owns validation, messages, follower dragging, scripts,
// and room mutation; sessions only publish room observations and agent vars.
func finishMovementCommand(s *Session, result game.MoveResult) error {
	if !result.Success {
		return nil
	}

	for _, followerName := range result.Followers {
		if followerSession, ok := s.manager.GetSession(followerName); ok {
			followerSession.markDirty(VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems, VarMove)
		}
	}

	// C has no room-entry aggro path: aggressive mobs attack from their own
	// mobile_activity tick (mobact.c, PULSE_MOBILE), gated on AWAKE — a mob
	// never engages merely because a player walked in. The invented entry
	// hook even engaged sleeping mobs; removed per R4.
	s.markDirty(VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems, VarMove)
	return nil
}
