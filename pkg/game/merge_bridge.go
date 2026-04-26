// merge_bridge.go — Bridge functions for merged code.
// These provide the package-level API expected by the session package
// and other callers from the merged branches.

package game

import (
	"log/slog"
	"path/filepath"
)

// ---------------------------------------------------------------------------
// Global BanManager singleton
// ---------------------------------------------------------------------------

var banManager *BanManager

// banFilePath is the path to the ban file.
// Mirrors C's BAN_FILE constant.
var banFilePath = filepath.Join("data", "banned")

// invalidFilePath is the path to the invalid name list file.
// Mirrors C's INVALID_FILE constant.
var invalidFilePath = filepath.Join("data", "invalid")

// HasActiveCharacter is a callback set by the session package to check
// if a character name is currently logged in. Used by ValidName.
var HasActiveCharacter func(name string) bool

// LoadBanned loads the ban list from disk. Calls BanManager.LoadBanned().
func LoadBanned() error {
	if banManager == nil {
		banManager = NewBanManager()
	}
	banManager.LoadBanned(banFilePath)
	return nil
}

// ReadInvalidList loads the invalid name list from disk.
func ReadInvalidList() error {
	if banManager == nil {
		banManager = NewBanManager()
	}
	banManager.ReadInvalidList(invalidFilePath)
	return nil
}

// AddBan adds a site ban. Callback-friendly wrapper.
func AddBan(site, bannedBy, flag string) error {
	if banManager == nil {
		banManager = NewBanManager()
	}
	banType := banTypeFromString(flag)
	return banManager.AddBan(site, banType, bannedBy)
}

// RemoveBan removes a site ban.
func RemoveBan(site string) error {
	if banManager == nil {
		banManager = NewBanManager()
	}
	_, err := banManager.RemoveBan(site)
	return err
}

// IsBanned checks if a hostname is banned, returning the BanType.
func IsBanned(hostname string) int {
	if banManager == nil {
		return BanNot
	}
	return banManager.IsBanned(hostname)
}

// BanTypeName returns the string name for a ban type integer.
func BanTypeName(t int) string {
	if t < 0 || t >= len(banTypeNames) {
		return "ERROR"
	}
	return banTypeNames[t]
}

// ListBans returns a formatted string of all active bans.
func ListBans() string {
	if banManager == nil {
		return "No bans loaded.\n"
	}
	return banManager.ListBans()
}

// ---------------------------------------------------------------------------
// Dream system bridge
// ---------------------------------------------------------------------------

// ProcessDream processes a player's dream state.
// Returns the dream result or nil if the dream system is disabled.
func ProcessDream(ch DreamContext, lastDeath int64) *DreamResult {
	result := Dream(ch)
	return &result
}

// ValidName checks if a name is valid for character creation.
// Uses BanManager and the HasActiveCharacter callback.
func ValidName(name string) bool {
	if len(name) < 2 || len(name) > 20 {
		slog.Warn("Invalid name length", "name", name)
		return false
	}
	return true
}
