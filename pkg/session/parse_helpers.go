package session

import (
	"strconv"
	"strings"
)

// parseSignedIntPrefix mirrors C atoi(): trim surrounding whitespace, accept
// an optional sign, consume leading decimal digits, and ignore a suffix.
func parseSignedIntPrefix(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	index := 0
	sign := 1
	if value[0] == '+' || value[0] == '-' {
		if value[0] == '-' {
			sign = -1
		}
		index++
	}
	start := index
	for index < len(value) && value[index] >= '0' && value[index] <= '9' {
		index++
	}
	if index == start {
		return 0, false
	}

	parsed, err := strconv.Atoi(value[start:index])
	if err != nil {
		return 0, false
	}
	return sign * parsed, true
}
