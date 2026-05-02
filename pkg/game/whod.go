// Package game — WHOD "who is online" display (ported from whod.c)
// Source: src/whod.c — do_whod(), whod_loop(), old_search_block()
// Port: Wave 13, 2026-04-25
//
// The original whod.c implemented a separate TCP port daemon that external
// clients could connect to and see who was online. In our Go rewrite, the
// "who daemon" concept is replaced by the WebSocket API — any client that
// connects can issue "who". The whod_mode flag system and do_whod() command
// are ported faithfully as an admin display filter, even though the raw TCP
// socket listening part is not ported (superseded by the WS protocol).
package game

import (
	"fmt"
	"strings"
)

// WhodMode bit constants — control what fields appear in the WHOD output.
// Source: whod.c lines 88–96
const (
	WhodShowName     = 1 << 0 // Show player name
	WhodShowClass    = 1 << 1 // Show class abbreviation
	WhodShowLevel    = 1 << 2 // Show numeric level
	WhodShowTitle    = 1 << 3 // Show player title + AFK marker
	WhodShowInvis    = 1 << 4 // Show invisible/wizinvis players
	WhodShowSite     = 1 << 5 // Show connection host
	WhodShowWizLevel = 1 << 6 // Show wizard level (e.g. GOD/TITAN/IMM)
	WhodShowOn       = 1 << 7 // WHOD is turned on
	WhodShowOff      = 1 << 8 // WHOD is turned off
)

// WhodDefaultMode is the default display mode applied at startup.
// Source: whod.c #define DEFAULT_MODE — name|title|on|class|level
const WhodDefaultMode = WhodShowName | WhodShowTitle | WhodShowOn | WhodShowClass | WhodShowLevel

// WhodModeNames maps bit positions to their string tokens.
// Source: whod.c do_whod() static char *modes[] lines 136–148
var WhodModeNames = []string{
	"name",     // bit 0
	"class",    // bit 1
	"level",    // bit 2
	"title",    // bit 3
	"wizinvis", // bit 4
	"site",     // bit 5
	"wizlevel", // bit 6
	"on",       // bit 7
	"off",      // bit 8
}

// WhodMinLevel is the minimum level to be considered a wizard/immortal.
// Source: whod.c #define WIZ_MIN_LEVEL LVL_IMMORT
const WhodMinLevel = LVLImmort // 31

// WhodEntry represents one player's entry in the who display.
type WhodEntry struct {
	Name       string
	Class      int
	Level      int
	Title      string
	AFK        bool
	Host       string
	IsInvisible bool // AFF_INVISIBLE or wizinvis
}

// Whod holds the WHOD mode flags and generates who-list output.
// Corresponds to the static whod_mode variable in whod.c.
type Whod struct {
	Mode int // bitmask of WhodShow* flags
}

// NewWhod creates a Whod instance with the default display mode.
func NewWhod() *Whod {
	return &Whod{Mode: WhodDefaultMode}
}

// WhodSearchBlock searches a modes list for a prefix match.
// Source: whod.c old_search_block() lines 495–527
//
// mode=0: prefix match (argument prefix must match list entry)
// mode=1: exact length match
// Returns (1-based index of match) or -1 if not found.
func WhodSearchBlock(argument string, list []string, mode int) int {
	length := len(argument)
	if length < 1 {
		return 1 // zero-length always matches (from C: found = (length < 1))
	}
	for guess, entry := range list {
		found := false
		if mode == 1 {
			found = length == len(entry)
			if found {
				found = strings.EqualFold(argument, entry[:length])
			}
		} else {
			// Prefix match
			if length <= len(entry) {
				found = strings.EqualFold(argument, entry[:length])
			}
		}
		if found {
			return guess + 1 // 1-based, matching C return value
		}
	}
	return -1
}

// DoWhod processes the in-game "whod" admin command to change WHOD display mode.
// Source: whod.c do_whod() lines 129–235
//
// With no argument: show current mode.
// With argument: toggle the named mode bit.
func (w *Whod) DoWhod(playerName, argument string) string {
	argument = strings.TrimSpace(strings.ToLower(argument))

	if argument == "" {
		// No argument: show current mode
		// Source: whod.c lines 153–158
		active := w.activeModesString()
		return fmt.Sprintf("Current WHOD mode:\n\r------------------\n\r%s\n\r", active)
	}

	// Find the mode by prefix match
	// Source: whod.c lines 160–173 — old_search_block with mode=0 (prefix)
	bit := WhodSearchBlock(argument, WhodModeNames, 0)
	if bit == -1 {
		var sb strings.Builder
		sb.WriteString("That mode does not exist.\n\rAvailable modes are:\n\r")
		for _, name := range WhodModeNames {
			sb.WriteString(name)
			sb.WriteString(" ")
		}
		sb.WriteString("\n\r")
		return sb.String()
	}

	bit-- // Convert to 0-based index (C: bit-- after search returns 1-based)
	bitMask := 1 << bit

	// Handle ON
	// Source: whod.c lines 176–191
	if bitMask == WhodShowOn {
		if w.Mode&WhodShowOn != 0 {
			return "WHOD already turned on.\n\r"
		}
		if w.Mode&WhodShowOff != 0 {
			w.Mode &^= WhodShowOff
			w.Mode |= WhodShowOn
			return "WHOD turned on.\n\r"
		}
		return "WHOD is not turned off.\n\r"
	}

	// Handle OFF
	// Source: whod.c lines 193–211
	if bitMask == WhodShowOff {
		if w.Mode&WhodShowOff != 0 {
			return "WHOD already turned off.\n\r"
		}
		if w.Mode&WhodShowOn != 0 {
			w.Mode &^= WhodShowOn
			w.Mode |= WhodShowOff
			return "WHOD turned off.\n\r"
		}
		return "WHOD is not turned on.\n\r"
	}

	// Toggle other mode bits
	// Source: whod.c lines 212–230
	if w.Mode&bitMask != 0 {
		w.Mode &^= bitMask
		return fmt.Sprintf("%s will not be shown on WHOD.\n\r", WhodModeNames[bit])
	}
	w.Mode |= bitMask
	return fmt.Sprintf("%s will now be shown on WHOD.\n\r", WhodModeNames[bit])
}

// activeModesString returns a space-separated list of currently active mode names.
// Source: whod.c sprintbit() call in do_whod() no-argument path.
func (w *Whod) activeModesString() string {
	var parts []string
	for i, name := range WhodModeNames {
		if w.Mode&(1<<i) != 0 {
			parts = append(parts, name)
		}
	}
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, " ")
}

// BuildWhoList generates the "who" player list string for WHOD output.
// Source: whod.c whod_loop() WHOD_OPEN case lines 301–428
//
// Uses class abbreviations from ClassAbbrevs (matching class_abbrevs[] in C).
// Wizard-level players get special level display unless WhodShowWizLevel is set.
func (w *Whod) BuildWhoList(entries []WhodEntry) string {
	var sb strings.Builder
	players, gods := 0, 0

	for _, entry := range entries {
		// Visibility check — Source: whod.c IS_INVIS(ch) check line 345
		if entry.IsInvisible && (w.Mode&WhodShowInvis == 0) {
			continue
		}

		// Level/class bracket
		// Source: whod.c lines 347–391
		if w.Mode&(WhodShowLevel|WhodShowClass) != 0 {
			sb.WriteString("[")

			isWiz := entry.Level >= WhodMinLevel

			if isWiz && (w.Mode&WhodShowWizLevel == 0) {
				// Show wizard rank label instead of level+class
				// Source: whod.c lines 354–370
				if w.Mode&WhodShowLevel != 0 && w.Mode&WhodShowClass != 0 {
					switch {
					case entry.Level >= 40:
						sb.WriteString("*IMP*")
					case entry.Level >= 38:
						sb.WriteString("GRGOD")
					case entry.Level >= 36:
						sb.WriteString("HIGOD")
					case entry.Level >= 35:
						sb.WriteString("_LEG_")
					case entry.Level >= 34:
						sb.WriteString(" GOD ")
					case entry.Level >= 32:
						sb.WriteString("TITAN")
					default:
						sb.WriteString(" IMM ")
					}
				} else {
					sb.WriteString("_GOD_")
				}
			} else {
				// Normal level+class display
				// Source: whod.c lines 372–390
				if w.Mode&WhodShowLevel != 0 {
					fmt.Fprintf(&sb, "%2d", entry.Level)
				}
				if w.Mode&WhodShowClass != 0 && w.Mode&WhodShowLevel != 0 {
					sb.WriteString(" ")
				}
				if w.Mode&WhodShowClass != 0 {
					abbrev := classAbbrev(entry.Class)
					sb.WriteString(abbrev)
				}
			}
			sb.WriteString("] ")
		}

		// Name — Source: whod.c lines 394–395
		if w.Mode&WhodShowName != 0 {
			sb.WriteString(entry.Name)
		}

		// Title + AFK — Source: whod.c lines 397–402
		if w.Mode&WhodShowTitle != 0 {
			sb.WriteString(" ")
			sb.WriteString(entry.Title)
			if entry.AFK {
				sb.WriteString(" (AFK)")
			}
		}

		// Site — Source: whod.c lines 404–411
		if w.Mode&WhodShowSite != 0 {
			if entry.Host != "" {
				fmt.Fprintf(&sb, " [%s]", entry.Host)
			} else {
				sb.WriteString(" [ ** Unknown ** ]")
			}
		}

		sb.WriteString("\r\n")

		if entry.Level >= WhodMinLevel {
			gods++
		} else {
			players++
		}
	}

	fmt.Fprintf(&sb, "\n\rPlayers : %d     Gods : %d\n\r\n\r", players, gods)
	return sb.String()
}

// classAbbrev returns a 3-character class abbreviation for use in WHOD display.
// Source: whod.c uses extern char *class_abbrevs[] — we use ClassAbbrevs from level.go.
// Fallback to "???" if the class index is out of range.
func classAbbrev(class int) string {
	// ClassAbbrevs is defined in level.go — see class.c class_abbrevs[]
	if class >= 0 && class < len(ClassAbbrevs) {
		return ClassAbbrevs[class]
	}
	return "???"
}

// Go Improvements Over C
// ======================
// 1. TCP SOCKET DAEMON REPLACED: The original whod.c opened a second TCP port and
//    served raw text to telnet connections. The Go port replaces this with a method
//    on World that returns a string — clients get it through the WebSocket API.
//    The raw TCP socket code (init_whod, close_whod, whod_loop) is intentionally
//    not ported — it's superseded by the WS protocol.
//
// 2. GLOBAL STATE ELIMINATED: C kept whod_mode, state, s (socket) as static
//    module-level variables. Go encapsulates them in Whod struct.
//
// 3. old_search_block() GENERALIZED: The C function had fixed-length char** list.
//    Go uses []string with range — no off-by-one risk from '\n' sentinel comparison.
//
// 4. NO STRING BUFFER OVERFLOW: C used fixed char buf[MAX_STRING_LENGTH] with
//    manual strcat() calls. Go uses strings.Builder which grows dynamically.
//
// 5. POTENTIAL MODERNIZATION (do not implement now):
//    - Expose BuildWhoList() via a JSON API endpoint for web clients.
//    - Make WhodDefaultMode configurable from a server config file.
//    - Add filtering by zone/area (C had no such feature).
