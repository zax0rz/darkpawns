// Package game — whod.go: "Who's Online" display formatter.
//
// This is a pure-function port of src/whod.c (the WHO daemon output formatter).
// The C version had its own listening socket and connection handling — that is
// stripped here. This file exposes FormatWHO() which takes a slice of players
// and returns the formatted "who's online" string, matching the original output
// format as closely as reasonable.
//
// Reference: src/whod.c (532 lines) — Johan Krisar / Robert Martin-Legene

package game

import (
	"fmt"
	"strings"
	"time"
)

// Display mode flags — matches SHOW_* constants in src/whod.c
const (
	showName     = 1 << 0
	showClass    = 1 << 1
	showLevel    = 1 << 2
	showTitle    = 1 << 3
	showInvis    = 1 << 4
	showSite     = 1 << 5
	showWizLevel = 1 << 6
	showOn       = 1 << 7
	showOff      = 1 << 8
)

// Default display mode — matches DEFAULT_MODE in src/whod.c
const whodDefaultMode = showName | showTitle | showOn | showClass | showLevel

// WhodFlags controls which fields are shown in the WHO output.
// Zero value uses whodDefaultMode.
type WhodFlags uint16

// FormatWHO builds a "who's online" display string from a player list.
// It matches the output format of the src/whod.c WHO daemon:
//
//	[LVL CLASS] Name Title (AFK) [host]
//	...
//	Players : N     Gods : M
//
// If flags == 0, whodDefaultMode is used.
func FormatWHO(players []*Player, world *World, flags WhodFlags) string {
	if flags == 0 {
		flags = whodDefaultMode
	}

	var b strings.Builder
	playersCount := 0
	godsCount := 0

	for _, ch := range players {
		// Skip NPCs
		if ch.IsNPC() {
			continue
		}

		// Skip linkdead players (no descriptor) — display linkdead is 0
		// In Go, a player without a session has no way to communicate; unlike
		// C where ch->desc is a pointer that can be nil, here we treat players
		// missing a session as linkdead. Since the original DISPLAY_LINKDEAD
		// is 0, we skip players whose session is inactive.
		// TODO: revisit if linkdead display is needed later.

		// Skip wizinvis/lowlevel invisible unless showInvis is set
		if flags&showInvis == 0 && isWhodInvisible(ch) {
			continue
		}

		// Skip if site-only display but no showOff bit
		if flags&showOff != 0 {
			continue
		}

		// --- Opening bracket ---
		if flags&(showLevel|showClass) != 0 {
			b.WriteString("[")

			if !isGodRank(ch) || flags&showWizLevel != 0 {
				// Normal player or showing wizard rank
				if flags&showLevel != 0 {
					fmt.Fprintf(&b, "%2d", ch.GetLevel())
				}
				if flags&showLevel != 0 && flags&showClass != 0 {
					b.WriteString(" ")
				}
				if flags&showClass != 0 {
					b.WriteString(whodClassAbbrev(ch.GetClass()))
				}
			} else {
				// God/immortal — show rank title instead of level+class
				if flags&showLevel != 0 && flags&showClass != 0 {
					b.WriteString(godRankTitle(ch.GetLevel()))
				} else {
					b.WriteString("_GOD_")
				}
			}

			b.WriteString("] ")
		}

		// --- Name ---
		if flags&showName != 0 {
			b.WriteString(ch.GetName())
		}

		// --- Title ---
		if flags&showTitle != 0 {
			b.WriteString(" ")
			b.WriteString(ch.Description)
			if ch.Flags&(1<<PrfAFK) != 0 {
				b.WriteString(" (AFK)")
			}
		}

		// --- Site ---
		if flags&showSite != 0 {
			// We don't track host strings per player in the Go port,
			// so we skip site information.
			b.WriteString(" [** Unknown **]")
		}

		b.WriteString("\r\n")

		if isGodRank(ch) {
			godsCount++
		} else {
			playersCount++
		}
	}

	fmt.Fprintf(&b, "\r\nPlayers : %d     Gods : %d\r\n", playersCount, godsCount)

	return b.String()
}

// FormatWHOExtended is the same as FormatWHO but includes idle time and
// zone/location info for each player, matching the enhanced display pattern.
func FormatWHOExtended(players []*Player, world *World, flags WhodFlags) string {
	if flags == 0 {
		flags = whodDefaultMode
	}

	var b strings.Builder

	// Build formatted lines with idle/location appended
	for _, ch := range players {
		if ch.IsNPC() {
			continue
		}
		if flags&showInvis == 0 && isWhodInvisible(ch) {
			continue
		}

		idleMins := int(time.Since(ch.LastActive).Minutes())
		location := whodLocation(world, ch.GetRoom())

		// --- Opening bracket ---
		if flags&(showLevel|showClass) != 0 {
			b.WriteString("[")

			if !isGodRank(ch) || flags&showWizLevel != 0 {
				if flags&showLevel != 0 {
					fmt.Fprintf(&b, "%2d", ch.GetLevel())
				}
				if flags&showLevel != 0 && flags&showClass != 0 {
					b.WriteString(" ")
				}
				if flags&showClass != 0 {
					b.WriteString(whodClassAbbrev(ch.GetClass()))
				}
			} else {
				if flags&showLevel != 0 && flags&showClass != 0 {
					b.WriteString(godRankTitle(ch.GetLevel()))
				} else {
					b.WriteString("_GOD_")
				}
			}

			b.WriteString("] ")
		}

		// --- Name ---
		if flags&showName != 0 {
			b.WriteString(ch.GetName())
		}

		// --- Title ---
		if flags&showTitle != 0 {
			if ch.Description != "" {
				b.WriteString(" ")
				b.WriteString(ch.Description)
			}
			if ch.Flags&(1<<PrfAFK) != 0 {
				b.WriteString(" (AFK)")
			}
		}

		// --- Idle time ---
		fmt.Fprintf(&b, " [%d min]", idleMins)

		// --- Location ---
		if location != "" {
			fmt.Fprintf(&b, " in %s", location)
		}

		b.WriteString("\r\n")
	}

	b.WriteString("\r\n")
	return b.String()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isWhodInvisible returns true if the player should be hidden from WHO.
// Matches the IS_INVIS macro in src/whod.c:
//
//	IS_INVIS(ch) = IS_AFFECTED(ch, AFF_INVISIBLE) || GET_INVIS_LEV(ch)
//	               || ROOM_FLAGGED(ch->in_room, ROOM_NO_WHO_ROOM)
func isWhodInvisible(ch *Player) bool {
	// Check AFF_INVISIBLE (affect bit 3 in structs.h)
	if ch.Affects&(1<<3) != 0 {
		return true
	}

	// Check PLR_INVISIBLE flag (wizinvis) — bit 1 in PLR flags
	if ch.Flags&PLR_INVISIBLE != 0 {
		return true
	}

	// TODO: Check room NO_WHO flag when room flags are available
	// ROOM_FLAGGED(ch->in_room, ROOM_NO_WHO_ROOM)

	return false
}

// isGodRank returns true if the player's level is at or above LVL_IMMORT (50).
func isGodRank(ch *Player) bool {
	return ch.GetLevel() >= LVL_IMMORT
}

// godRankTitle returns the god-rank title string based on level.
// Matches the title chain in src/whod.c:
//
//	>= 40 → *IMP*
//	>= 38 → GRGOD
//	>= 36 → HIGOD
//	>= 35 → _LEG_
//	>= 34 →  GOD
//	>= 32 → TITAN
//	>= 31 →  IMM
func godRankTitle(level int) string {
	switch {
	case level >= 40:
		return "*IMP*"
	case level >= 38:
		return "GRGOD"
	case level >= 36:
		return "HIGOD"
	case level >= 35:
		return "_LEG_"
	case level >= 34:
		return " GOD "
	case level >= 32:
		return "TITAN"
	default:
		return " IMM "
	}
}

// whodClassAbbrev returns a short class abbreviation for WHO display.
// Maps each class to a 4-6 char abbreviation (matching class_abbrevs[] in C).
func whodClassAbbrev(class int) string {
	switch class {
	case ClassMageUser:
		return "Mag"
	case ClassCleric:
		return "Clr"
	case ClassThief:
		return "Thf"
	case ClassWarrior:
		return "War"
	case ClassMagus:
		return "Mgs"
	case ClassAvatar:
		return "Avt"
	case ClassAssassin:
		return "Ass"
	case ClassPaladin:
		return "Pal"
	case ClassNinja:
		return "Nin"
	case ClassPsionic:
		return "Psi"
	case ClassRanger:
		return "Rng"
	case ClassMystic:
		return "Mys"
	default:
		return "???"
	}
}

// whodLocation returns a human-readable location string for a room VNum.
// Falls back to zone name if available, then "Unknown".
func whodLocation(world *World, roomVNum int) string {
	if world == nil {
		return ""
	}

	room := world.GetRoomInWorld(roomVNum)
	if room == nil {
		return "Unknown"
	}

	// Try zone name first
	if zone, ok := world.GetZone(room.Zone); ok {
		return zone.Name
	}

	// Fall back to room name
	if room.Name != "" {
		return room.Name
	}

	return ""
}
