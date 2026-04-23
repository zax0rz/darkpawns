package validation

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	playerNameRegex     = regexp.MustCompile(`^[a-zA-Z0-9_\-\. ]+$`)
	maxPlayerNameLength = 32
	minPlayerNameLength = 2
)

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

	// Check for reserved names
	reservedNames := []string{"admin", "system", "root", "server", "null", "undefined"}
	lowerName := strings.ToLower(name)
	for _, reserved := range reservedNames {
		if lowerName == reserved {
			return false
		}
	}

	return true
}

func SanitizePlayerName(name string) string {
	// Remove invalid characters
	runes := []rune(name)
	var result []rune
	for _, r := range runes {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' || r == ' ' {
			result = append(result, r)
		}
	}

	// Trim and limit length
	sanitized := string(result)
	if utf8.RuneCountInString(sanitized) > maxPlayerNameLength {
		sanitized = string([]rune(sanitized)[:maxPlayerNameLength])
	}

	return strings.TrimSpace(sanitized)
}
