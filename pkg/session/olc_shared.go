package session

import (
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/olc"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Shared OLC helpers for the descriptor-owned CircleMUD editors. These
// helpers deliberately contain no descriptor concerns; editor ownership and
// dirty-save coordination live in olc_registry.go.

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
