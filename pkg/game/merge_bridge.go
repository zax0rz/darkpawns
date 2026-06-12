// merge_bridge.go — Bridge functions for merged code.
// These provide the package-level API expected by the session package
// and other callers from the merged branches.

package game

import (
	"path/filepath"
)

// ---------------------------------------------------------------------------
// Global BanManager singleton
// ---------------------------------------------------------------------------

var banManager *BanManager

// banFilePath is the path to the ban file.
// Mirrors C's BAN_FILE constant: lib/etc/banned
var banFilePath = filepath.Join("lib", "etc", "banned")

// invalidFilePath is the path to the invalid name list file.
// Mirrors C's INVALID_FILE (xnames): lib/text/xnames
var invalidFilePath = filepath.Join("lib", "text", "xnames")

// SetBanFilePaths updates the ban and invalid-name file paths to match the
// deployed world directory layout (e.g. /opt/darkpawns/lib).
// Must be called before LoadBanned/ReadInvalidList.
func SetBanFilePaths(worldDir string) {
	banFilePath = filepath.Join(worldDir, "etc", "banned")
	invalidFilePath = filepath.Join(worldDir, "text", "xnames")
}

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
		return false
	}
	if banManager != nil && !banManager.ValidName(name) {
		return false
	}
	// Check if character is already online (DP-554)
	if HasActiveCharacter != nil && HasActiveCharacter(name) {
		return false
	}
	return true
}

// ValidNameNoActive checks if a name is valid for character creation without
// checking if the character is currently online.
func ValidNameNoActive(name string) bool {
	if len(name) < 2 || len(name) > 20 {
		return false
	}
	if banManager != nil && !banManager.ValidName(name) {
		return false
	}
	return true
}
