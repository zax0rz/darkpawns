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

// checkLevel checks if a session's player has at least the required level.
func checkLevel(s *Session, level int) bool {
	if s.player == nil {
		return false
	}
	return s.player.Level >= level
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
// SECURITY NOTE: This is intentionally cosmetic-only.
//
// The original C MUD's switch command fully swapped the session's player
// reference, allowing the wizard to run commands as the target character.
// That design has significant security implications:
//
//   - The wizard could execute any command with the target's permissions,
//     including commands the target's level shouldn't have access to.
//   - Inventory manipulation, save data corruption, and privilege escalation
//     are all possible if the swap isn't handled carefully.
//   - If the target player reconnects during a switch, ownership of the
//     session becomes ambiguous.
//
// Full body switching would require:
//   1. Swapping Session.player to the target Player pointer
//   2. Updating world.RemovePlayer/AddPlayer for both characters
//   3. Preventing the target from receiving commands during the switch
//   4. Auditing command execution to ensure permission isolation
//   5. Coordinating with the session-takeover logic (M-28) for edge cases
//
// TODO(M-16): Implement a safe player reference swap with:
//   - A permission wrapper that gates commands by the *original* wizard level
//   - Save-state snapshots before and after the switch
//   - A timeout that auto-returns if the wizard disconnects mid-switch
//   - Logging all commands executed while switched for audit trail
//
// cmdSwitch transfers the wizard's control to a different character.
// Expected behavior (from original C):
// - Save the current character state
// - Load the target character
// - Attach the wizard's session to the new character
