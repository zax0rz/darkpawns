package session

import (
	"log/slog"
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
		follower, ok := s.manager.world.GetPlayer(followerName)
		if !ok {
			continue
		}
		_ = s.manager.world.OnPlayerEnterRoom(follower, follower.GetRoom(), s.manager.combatEngine)
		if followerSession, ok := s.manager.GetSession(followerName); ok {
			if err := cmdMovementLook(followerSession); err != nil {
				slog.Error("movement look failed for follower", "follower", followerName, "error", err)
			}
			followerSession.markDirty(VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems, VarMove)
		}
	}

	if s.manager.world.OnPlayerEnterRoom(s.player, result.NewRoomVNum, s.manager.combatEngine) {
		s.sendText("You are attacked!")
	}
	s.markDirty(VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems, VarMove)
	return cmdMovementLook(s)
}
