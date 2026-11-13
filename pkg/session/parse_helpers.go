package session

import (
	"strings"

	"github.com/zax0rz/darkpawns/pkg/common"
)

// parseSignedIntPrefix mirrors C atoi(): trim surrounding whitespace, accept
// an optional sign, consume leading decimal digits, and ignore a suffix.
// C's atoi never fails, so the bool reports only whether any digits were
// present; overflow wraps through (int)strtol and keeps the wrapped value —
// common.AtoiC carries that, which matters for dig/dc/advance where a huge
// token lands on a C refusal arm keyed to the wrapped sign.
func parseSignedIntPrefix(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	index := 0
	if value[0] == '+' || value[0] == '-' {
		index++
	}
	start := index
	for index < len(value) && value[index] >= '0' && value[index] <= '9' {
		index++
	}
	if index == start {
		return 0, false
	}
	return common.AtoiC(value), true
}
