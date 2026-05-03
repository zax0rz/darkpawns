package session

import (
	"strings"
)

// Wizard level constants — matching Dark Pawns C source scale mapped to Go codebase.
// Original C: LVL_IMMORT=31, LVL_GOD=34, LVL_GRGOD=38, LVL_IMPL=40
// Go codebase uses higher scale: 50/60/61.
const (
	LVL_IMMORT = 50
	LVL_GOD    = 60
	LVL_GRGOD  = 61
	LVL_IMPL   = 61
)

// getEffectiveLevel returns the level that should be used for permission checks.
// When a wizard is switched into another body, their original wizard level is used
// so they cannot escalate beyond their own authority. (M-16)
// When a player is under a forced command, their own level is used (force safety).
func getEffectiveLevel(s *Session) int {
	if s.player == nil {
		return 0
	}
	if s.isSwitched && s.switchedOriginalLevel > 0 {
		return s.switchedOriginalLevel
	}
	if s.IsForced && s.ForcedPrivilegeLevel > 0 {
		return s.ForcedPrivilegeLevel
	}
	return s.player.Level
}

// checkLevel checks if a session's player has at least the required level.
// Uses getEffectiveLevel to ensure switched wizards are gated by their original level.
func checkLevel(s *Session, level int) bool {
	return getEffectiveLevel(s) >= level
}

// findSessionByName searches all sessions for a player by name (case-insensitive).
func findSessionByName(m *Manager, name string) *Session {
	name = strings.ToLower(name)
	for _, sess := range m.sessions {
		if sess.player != nil && strings.ToLower(sess.player.Name) == name {
			return sess
		}
	}
	return nil
}

// broadcastToRoomText sends a text message to all players in a given room.
func broadcastToRoomText(s *Session, roomVNum int, msg string) {
	if s.manager != nil && s.manager.world != nil {
		s.manager.BroadcastToRoom(roomVNum, []byte(msg), "")
	}
}

// ---------------------------------------------------------------------------
// goto — teleport to any room (LVL_IMMORT)
// ---------------------------------------------------------------------------
func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ---------------------------------------------------------------------------
// switch — switch into another character's body (LVL_GRGOD)
// ---------------------------------------------------------------------------
// M-16: Implemented with permission gating by original wizard level,
// auto-return on disconnect, and audit logging. Toggle: calling switch
// while already switched returns to original body.
//
// Security: checkLevel uses getEffectiveLevel() which returns the wizard's
// original level even when switched — no privilege escalation.
//
// TODO(future): Save-state snapshots before/after switch (requires DB
// transaction support in the persistence layer).
