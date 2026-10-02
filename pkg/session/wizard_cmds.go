package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// Wizard level constants — from src/config.c. LVL_IMMORT aliases the
// canonical value in pkg/game; the remaining authority levels stay local and
// numeric because they are outside this bounded single-source slice.
const (
	LVL_IMMORT = game.LVL_IMMORT
	LVL_GOD    = 34
	LVL_LEGEND = 35
	LVL_HIGOD  = 36
	LVL_GRGOD  = 38
	LVL_IMPL   = 40
)

// getEffectiveLevel returns the level that should be used for permission checks.
// PC switch commands use the acting body's level (src/interpreter.c:909-914).
// The existing NPC adapter remains C1's separate frontier.
// When a player is under a forced command, their own level is used (force safety).
func getEffectiveLevel(s *Session) int {
	if s.manager != nil {
		s.manager.mu.RLock()
		defer s.manager.mu.RUnlock()
	}
	if s.player == nil {
		return 0
	}
	if s.isSwitched && s.switchedPlayer != nil {
		return s.player.GetLevel()
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
// Uses the acting PC level; explicit original-level consumers (snoop) stay separate.
func checkLevel(s *Session, level int) bool {
	return getEffectiveLevel(s) >= level
}

// findSessionByName searches all sessions for a player by name (case-insensitive).
func findSessionByName(m *Manager, name string) *Session {
	if attached, involved := m.switchDescriptorByName(name); involved {
		return attached
	}
	name = strings.ToLower(name)
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, sess := range m.sessions {
		if sess.player != nil && strings.ToLower(sess.player.Name) == name {
			return sess
		}
	}
	return nil
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
