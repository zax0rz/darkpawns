// Package game — WHOD "who is online" display (ported from whod.c)
// Source: src/whod.c — do_whod(), whod_loop(), old_search_block()
// Port: Wave 13, 2026-04-25
//
// The command mode flags are ported. The separate C TCP daemon is not
// implemented; its socket lifecycle and producers remain an explicit frontier.
package game

import (
	"fmt"
	"strings"
	"sync"
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
const WhodMinLevel = LVLImmort

// WhodEntry represents one player's entry in the who display.
type WhodEntry struct {
	Name        string
	Class       int
	Level       int
	Title       string
	AFK         bool
	Host        string
	IsInvisible bool // AFF_INVISIBLE or wizinvis
}

// Whod holds the WHOD mode flags and generates who-list output.
// Corresponds to the static whod_mode variable in whod.c. The listener reads
// it from its own goroutine, so every access goes through the mutex.
type Whod struct {
	mu   sync.Mutex
	mode int // bitmask of WhodShow* flags
}

// NewWhod creates a Whod instance with the default display mode.
func NewWhod() *Whod {
	return &Whod{mode: WhodDefaultMode}
}

// GetMode returns the current whod_mode bitmask.
func (w *Whod) GetMode() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.mode
}

// SetMode replaces the whod_mode bitmask.
func (w *Whod) SetMode(mode int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.mode = mode
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
// Live callers supply output so acknowledgement precedes the producer.
// src/whod.c:39,184-188,202-206,214-219,224-228.
func (w *Whod) DoWhod(playerName, argument string, output func(string)) string {
	// The live caller supplies synchronous output: C acknowledges before LOG.
	ackLog := func(message, payload string) string {
		output(message)
		MudLog(payload, MudlogBrief, LVL_GOD, true)
		return ""
	}
	// src/whod.c:150: half_chop ignores the remaining operands.
	tokens := strings.Fields(argument)
	argument = ""
	if len(tokens) > 0 {
		argument = strings.ToLower(tokens[0])
	}

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

	// C reads and writes the static whod_mode directly; the snapshot keeps
	// each command's decision consistent while the listener may read it.
	mode := w.GetMode()

	// Handle ON
	// Source: whod.c lines 176–191 (REMOVE_BIT/SET_BIT, then ack, then LOG)
	if bitMask == WhodShowOn {
		if mode&WhodShowOn != 0 {
			return "WHOD already turned on.\n\r"
		}
		if mode&WhodShowOff != 0 {
			w.SetMode((mode &^ WhodShowOff) | WhodShowOn)
			return ackLog("WHOD turned on.\n\r", fmt.Sprintf("WHOD turned on by %s.", playerName))
		}
		return "WHOD is not turned off.\n\r"
	}

	// Handle OFF
	// Source: whod.c lines 193–211
	if bitMask == WhodShowOff {
		if mode&WhodShowOff != 0 {
			return "WHOD already turned off.\n\r"
		}
		if mode&WhodShowOn != 0 {
			w.SetMode((mode &^ WhodShowOn) | WhodShowOff)
			return ackLog("WHOD turned off.\n\r", fmt.Sprintf("WHOD turned off by %s.", playerName))
		}
		return "WHOD is not turned on.\n\r"
	}

	// Toggle other mode bits. C acknowledges and LOGs before the bit changes
	// (src/whod.c:212–230), so the mutation follows ackLog.
	if mode&bitMask != 0 {
		message := ackLog(fmt.Sprintf("%s will not be shown on WHOD.\n\r", WhodModeNames[bit]), fmt.Sprintf("%s removed from WHOD by %s.", WhodModeNames[bit], playerName))
		w.SetMode(mode &^ bitMask)
		return message
	}
	message := ackLog(fmt.Sprintf("%s will now be shown on WHOD.\n\r", WhodModeNames[bit]), fmt.Sprintf("%s added to WHOD by %s.", WhodModeNames[bit], playerName))
	w.SetMode(mode | bitMask)
	return message
}

// activeModesString returns a space-separated list of currently active mode names.
// Source: whod.c sprintbit() call in do_whod() no-argument path.
func (w *Whod) activeModesString() string {
	var parts []string
	mode := w.GetMode()
	for i, name := range WhodModeNames {
		if mode&(1<<i) != 0 {
			parts = append(parts, name)
		}
	}
	if len(parts) == 0 {
		return "(none)"
	}
	return strings.Join(parts, " ")
}

// WhodMudName is C's MUDNAME (#define at src/whod.c:27).
const WhodMudName = "Dark Pawns"

// WhodGreeting returns C's SHORT_GREETINGS (src/config.c:221-231) byte for
// byte. C writes it to the accepted socket before the list (src/whod.c:337-339).
func WhodGreeting() string {
	return "\n\r\n\r" +
		"         (_____)           (_)    (_____)\n\r" +
		"   _     /  __ \\           | |    |  __ \\                            _\n\r" +
		"  ;*;   /| |  | | __ _ _ __| | __ | |__) |_ _(_      _)_ __ (___)   ;*;\n\r" +
		"   =    /| |  | |/ _` | '__| |/ / |  ___/ _` \\ \\ /\\ / / '_ \\/ __|    =\n\r" +
		" .***.  /| |__| | (_| | |  |   <  | |  | (_| |\\ V  V /| | | \\__ \\  .***.\n\r" +
		" ~~~~~  /|_____/ \\__,_|_|  |_|\\_\\ |||   \\__,_| \\_/\\_/ |_| |_|___/  ~~~~~\n\r" +
		"        /                         |||\n\r" +
		"                             |||\n\r" +
		"                 `.'\n\r" +
		"\n\r"
}

// WhodHeader is C's list header (src/whod.c:341), written after the greeting.
func WhodHeader() string {
	return "\n\rA list of active players on " + WhodMudName + ":\n\r\n\r"
}

// BuildWhoList renders one WHOD response body: the header, one line per entry
// in the caller's order, then the player/god footer. It ports the WHOD_OPEN
// assembly at src/whod.c:341-425. The caller supplies only entries C would
// have walked (a descriptor, not an NPC) and flags IS_INVIS per entry; the
// invisible filter and every byte below are C's.
func (w *Whod) BuildWhoList(entries []WhodEntry) string {
	var sb strings.Builder
	sb.WriteString(WhodHeader())

	// One consistent snapshot for the whole response: C reads the static
	// whod_mode repeatedly, and the port must not change mid-render.
	mode := w.GetMode()

	players, gods := 0, 0
	for _, entry := range entries {
		// IS_INVIS gate — src/whod.c:345.
		if entry.IsInvisible && mode&WhodShowInvis == 0 {
			continue
		}

		if mode&(WhodShowLevel|WhodShowClass) != 0 {
			sb.WriteString("[")
			if mode&WhodShowWizLevel == 0 && entry.Level >= WhodMinLevel {
				if mode&WhodShowLevel != 0 && mode&WhodShowClass != 0 {
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
				if mode&WhodShowLevel != 0 {
					fmt.Fprintf(&sb, "%2d", entry.Level)
				}
				if mode&WhodShowClass != 0 && mode&WhodShowLevel != 0 {
					sb.WriteString(" ")
				}
				if mode&WhodShowClass != 0 {
					sb.WriteString(classAbbrev(entry.Class))
				}
			}
			sb.WriteString("] ")
		}

		if mode&WhodShowName != 0 {
			sb.WriteString(entry.Name)
		}
		if mode&WhodShowTitle != 0 {
			sb.WriteString(" ")
			sb.WriteString(entry.Title)
			if entry.AFK {
				sb.WriteString(" (AFK)")
			}
		}
		if mode&WhodShowSite != 0 {
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

// classAbbrev returns the two-character class abbreviation C indexes with
// class_abbrevs[GET_CLASS(ch)] (src/class.c:51-65).
func classAbbrev(class int) string {
	if class >= 0 && class < len(ClassAbbrevs) {
		return ClassAbbrevs[class]
	}
	return "???"
}

// WhodInvisible mirrors C's IS_INVIS (src/whod.c:33): AFF_INVISIBLE, a
// wizinvis level, or standing in a ROOM_NO_WHO_ROOM.
func (w *World) WhodInvisible(p *Player) bool {
	if p == nil {
		return false
	}
	if p.IsAffected(affInvisible) || p.GetInvisLevel() != 0 {
		return true
	}
	if room := w.GetRoomInWorld(p.GetRoom()); room != nil && room.HasFlag(RoomNoWho) {
		return true
	}
	return false
}
