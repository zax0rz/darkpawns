// Package session manages WebSocket connections and player sessions.
package session

import (
	"context"
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/game/systems"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// ctx returns the connection context, falling back to context.Background if nil.
func (s *Session) ctx() context.Context {
	if s.sessionCtx == nil {
		return context.Background()
	}
	return s.sessionCtx
}

// logAttrs returns standard structured logging attributes for the session.
func (s *Session) logAttrs(extra ...slog.Attr) []interface{} {
	playerName := s.playerName
	if playerName == "" {
		playerName = "guest"
	}
	attrs := []slog.Attr{
		slog.String("player", playerName),
		slog.String("session", s.sessionID()),
	}
	if s.player != nil {
		attrs = append(attrs, slog.Int("room", s.player.GetRoom()))
	}
	if s.charCreating {
		attrs = append(attrs, slog.String("stage", s.charStage))
	}

	res := make([]interface{}, 0, len(attrs)+len(extra))
	for _, a := range attrs {
		res = append(res, a)
	}
	for _, e := range extra {
		res = append(res, e)
	}
	return res
}

func getExitNames(exits map[string]parser.Exit) []string {
	var names []string
	for dir := range exits {
		names = append(names, dir)
	}
	return names
}

func getDoorInfo(dm *systems.DoorManager, roomVNum int, exits map[string]parser.Exit) []DoorInfo {
	if dm == nil {
		return nil
	}
	var doors []DoorInfo
	for dir := range exits {
		door, ok := dm.GetDoor(roomVNum, dir)
		if !ok {
			continue
		}
		if !door.CanSee() {
			continue
		}
		doors = append(doors, DoorInfo{
			Direction: dir,
			Closed:    door.Closed,
			Locked:    door.Locked,
		})
	}
	if len(doors) == 0 {
		return nil
	}
	return doors
}

// GetPlayer returns the player associated with this session
