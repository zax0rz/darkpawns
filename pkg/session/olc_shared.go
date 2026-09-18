package session

import (
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// Shared OLC helpers for the descriptor-owned CircleMUD editors (medit, and
// later oedit/sedit/zedit). These mirror the helpers the redit branch carries
// locally in pkg/session/redit.go (atoiC, firstByte, isASCIIDigit,
// reditAuthorized, reditZoneForVNum); when that branch merges, its local
// copies should be deleted in favor of these shared definitions.

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

// olcAuthorized mirrors do_olc's zone permission gate (src/olc.c): builders
// at LVL_SET_BUILD (LVL_GOD+1) may edit any zone; lower builders are confined
// to their assigned OLC zone.
func olcAuthorized(s *Session, zoneNumber int) bool {
	return getEffectiveLevel(s) >= game.LVL_GOD+1 || s.olcZone == zoneNumber
}

// olcZoneForVNum finds the zone whose [number*100, top] range contains vnum,
// mirroring C's real_zone. It returns false when no zone covers the vnum.
func olcZoneForVNum(world *game.World, vnum int) (*parser.Zone, bool) {
	for _, zone := range world.GetAllZones() {
		if vnum >= zone.Number*100 && vnum <= zone.TopRoom {
			return zone, true
		}
	}
	return nil, false
}

// olcDuplicateName returns the name of the player currently editing vnum with
// the given editor, or "" when nobody is. This mirrors do_olc's
// descriptor-list scan ("That %s is currently being edited by %s."). The
// accessor reports the vnum under edit and whether an edit is active.
func olcDuplicateName(manager *Manager, vnum int, editingVNum func(*Session) (int, bool)) string {
	manager.mu.RLock()
	sessions := make([]*Session, 0, len(manager.sessions))
	for _, candidate := range manager.sessions {
		sessions = append(sessions, candidate)
	}
	manager.mu.RUnlock()

	for _, candidate := range sessions {
		candidate.textEditMu.Lock()
		edited, active := editingVNum(candidate)
		name := candidate.playerName
		candidate.textEditMu.Unlock()
		if active && edited == vnum {
			return name
		}
	}
	return ""
}
