package validation

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	playerNameRegex      = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	maxPlayerNameLength  = 32
	minPlayerNameLength  = 2
	creationNameReserved = map[string]struct{}{
		"in": {}, "from": {}, "with": {}, "the": {}, "on": {}, "at": {}, "to": {},
		"a": {}, "an": {}, "self": {}, "me": {}, "all": {}, "room": {},
		"someone": {}, "something": {},
	}
)

// IsCharacterCreationNameSyntax matches the character-creation syntax gate in
// src/interpreter.c: names are 2-20 ASCII letters and cannot be a C fill or
// reserved word. The separate xname and staff-reserved checks remain owned by
// their existing packages.
func IsCharacterCreationNameSyntax(name string) bool {
	if len(name) < 2 || len(name) > 20 {
		return false
	}
	for i := 0; i < len(name); i++ {
		letter := name[i]
		if (letter < 'a' || letter > 'z') && (letter < 'A' || letter > 'Z') {
			return false
		}
	}
	_, reserved := creationNameReserved[strings.ToLower(name)]
	return !reserved
}

func IsValidPlayerName(name string) bool {
	// Check length
	if utf8.RuneCountInString(name) < minPlayerNameLength ||
		utf8.RuneCountInString(name) > maxPlayerNameLength {
		return false
	}

	// Check character set
	if !playerNameRegex.MatchString(name) {
		return false
	}

	return !IsReservedPlayerName(name)
}

// IsReservedPlayerName is the Go-only security overlay approved in DP-1378.
// Entry applies it after the C name gate, independently of character syntax.
func IsReservedPlayerName(name string) bool {
	reservedNames := []string{
		"admin", "system", "root", "server", "null", "undefined",
		"gm", "moderator", "god", "implementor", "imp", "staff", "dev", "bot", "agent",
		"zax0rz",
	}
	for _, reserved := range reservedNames {
		if strings.EqualFold(name, reserved) {
			return true
		}
	}
	return false
}

func SanitizePlayerName(name string) string {
	// Remove invalid characters
	var result []rune
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' {
			result = append(result, r)
		}
	}

	// Trim and limit length
	sanitized := string(result)
	if utf8.RuneCountInString(sanitized) > maxPlayerNameLength {
		sanitized = string([]rune(sanitized)[:maxPlayerNameLength])
	}

	// Ensure minimum length after stripping invalid chars (DP-570)
	if utf8.RuneCountInString(sanitized) < minPlayerNameLength {
		return ""
	}

	return sanitized
}
