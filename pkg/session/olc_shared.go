package session

import (
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Shared OLC helpers for the descriptor-owned CircleMUD editors. These
// helpers deliberately contain no descriptor concerns; editor ownership and
// dirty-save coordination live in olc_registry.go.

// atoiC mirrors C's atoi: leading whitespace skipped, optional sign, then the
// longest digit run. Trailing garbage is ignored; no digits means 0.
func atoiC(input string) int {
	input = strings.TrimLeft(input, " \t\r\n\v\f")
	if input == "" {
		return 0
	}
	end := 0
	if input[0] == '+' || input[0] == '-' {
		end = 1
	}
	start := end
	for end < len(input) && input[end] >= '0' && input[end] <= '9' {
		end++
	}
	if end == start {
		return 0
	}
	number, err := strconv.Atoi(input[:end])
	if err != nil {
		return 0
	}
	return number
}

func firstByte(input string) byte {
	if input == "" {
		return 0
	}
	return input[0]
}

func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

// olcAuthorized keeps the descriptor-owned editor call sites on their existing
// shape while the rule itself lives in pkg/olc for all frontends.
func olcAuthorized(s *Session, zoneNumber int) bool {
	return olc.Authorized(getEffectiveLevel(s), s.olcZone, zoneNumber)
}

// olcZoneForVNum adapts the world-owned zone snapshot for the shared parser-
// only lookup, keeping the descriptor-owned editor call sites unchanged.
func olcZoneForVNum(world *game.World, vnum int) (*parser.Zone, bool) {
	return olc.ZoneForVNum(world.GetAllZones(), vnum)
}
