// Package session manages WebSocket connections and player sessions.
package session

import (
	"github.com/zax0rz/darkpawns/pkg/moderation"
)

// ModerationAdapter wraps a moderation.Manager to satisfy session.ModerationChecker.
type ModerationAdapter struct {
	mod *moderation.Manager
}

// NewModerationAdapter creates a new ModerationAdapter.
func NewModerationAdapter(mod *moderation.Manager) *ModerationAdapter {
	return &ModerationAdapter{mod: mod}
}

// CheckPreCommand checks if a player is muted or banned before executing a command.
// Only blocks for non-communication commands (let communication flow through CheckMessage).
func (a *ModerationAdapter) CheckPreCommand(playerName string, command string) (string, bool) {
	if a.mod == nil {
		return "", false
	}

	// Check ban
	if a.mod.IsBanned(playerName) {
		return "You have been banned from the game.", true
	}

	// Check mute - block non-info/communication commands
	if a.mod.IsMuted(playerName) {
		// Allow certain commands even when muted
		allowed := map[string]bool{
			"report":    true,
			"who":       true,
			"where":     true,
			"look":      true,
			"l":         true,
			"score":     true,
			"sc":        true,
			"inventory": true,
			"i":         true,
			"equipment": true,
			"eq":        true,
			"quit":      true,
			"help":      true,
			"commands":  true,
			"cmds":      true,
			"password":  true,
			"prompt":    true,
			"save":      true,
		}
		if !allowed[command] {
			return "You are muted and cannot use this command.", true
		}
	}

	return "", false
}

// CheckMessage filters a message for word filters and returns the filtered version.
func (a *ModerationAdapter) CheckMessage(playerName string, message string) (string, bool) {
	if a.mod == nil {
		return message, false
	}
	filtered, _, shouldBlock := a.mod.CheckMessage(playerName, message)
	return filtered, shouldBlock
}

// RecordMessage records a message timestamp for spam detection.
func (a *ModerationAdapter) RecordMessage(playerName string) {
	if a.mod != nil {
		a.mod.RecordMessage(playerName)
	}
}

// IsMuted checks if a player is muted.
func (a *ModerationAdapter) IsMuted(playerName string) bool {
	if a.mod == nil {
		return false
	}
	return a.mod.IsMuted(playerName)
}
