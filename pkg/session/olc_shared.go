package session

import (
	"math"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Shared OLC helpers for the descriptor-owned CircleMUD editors. These
// helpers deliberately contain no descriptor concerns; editor ownership and
// dirty-save coordination live in olc_registry.go.

// atoiC mirrors glibc atoi(s) == (int)strtol(s, NULL, 10): leading whitespace
// skipped, an optional sign, then the longest digit run (trailing garbage is
// ignored; no digits means 0). strtol clamps to LONG_MIN/LONG_MAX on overflow
// and the cast to int keeps the low 32 bits, so huge tokens wrap: an
// overflowing positive token is negative after the int cast.
func atoiC(input string) int {
	input = strings.TrimLeft(input, " \t\r\n\v\f")
	if input == "" {
		return 0
	}
	index := 0
	negative := false
	if input[0] == '+' || input[0] == '-' {
		negative = input[0] == '-'
		index++
	}
	start := index
	var value int64
	overflow := false
	for index < len(input) && input[index] >= '0' && input[index] <= '9' {
		digit := int64(input[index] - '0')
		if value > (math.MaxInt64-digit)/10 {
			overflow = true
		} else {
			value = value*10 + digit
		}
		index++
	}
	if index == start {
		return 0
	}
	if overflow {
		// strtol returns LONG_MAX/LONG_MIN; truncating to 32 bits gives
		// 0x7FFFFFFFFFFFFFFF -> -1 and 0x8000000000000000 -> 0.
		if negative {
			return 0
		}
		return -1
	}
	if negative {
		value = -value
	}
	return int(int32(value))
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
	return olc.Authorized(getEffectiveLevel(s), s.actingOLCZone(), zoneNumber)
}

// olcZoneForVNum adapts the world-owned zone snapshot for the shared parser-
// only lookup, keeping the descriptor-owned editor call sites unchanged.
func olcZoneForVNum(world *game.World, vnum int) (*parser.Zone, bool) {
	return olc.ZoneForVNum(world.GetAllZones(), vnum)
}
