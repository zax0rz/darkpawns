// Package session manages WebSocket connections and player sessions.
package session

import (
	"log/slog"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

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

// GetPlayer returns the player associated with this session
