package olc

import "github.com/zax0rz/darkpawns/pkg/parser"

// unconfinedLevel is LVL_GOD+1 in the game level ladder. Levels 31 through 34
// are builders and remain confined to their assigned OLC zone.
const unconfinedLevel = 35

// Authorized applies the single OLC zone admission rule. Gods at level 35 or
// above may edit any zone; everyone else must match the assigned OLC zone.
func Authorized(level, olcZone, zoneNumber int) bool {
	return level >= unconfinedLevel || olcZone == zoneNumber
}

// ZoneForVNum finds the zone whose [number*100, top] range contains vnum,
// mirroring C's real_zone. It returns false when no zone covers the vnum.
func ZoneForVNum(zones []*parser.Zone, vnum int) (*parser.Zone, bool) {
	for _, zone := range zones {
		if vnum >= zone.Number*100 && vnum <= zone.TopRoom {
			return zone, true
		}
	}
	return nil, false
}
