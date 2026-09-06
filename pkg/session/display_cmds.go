// Package session — display commands ported from act.display.c
//
// Infobar display commands for the Dark Pawns MUD.
//
// The infobar is a VT100-based stat display drawn at the bottom of the
// terminal, showing hit points, mana, move, experience, level, and gold.
// It uses VT100 scroll-region margins and cursor save/restore sequences.
package session

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// Infobar state constants — from structs.h
const (
	InfobarOff = 0
	InfobarOn  = 1
)

// Info update bitmask constants — INFO_* from act.display.c
const (
	InfoHit = 1 << iota
	InfoMana
	InfoMove
	InfoExp
	InfoGold
)

// VT100 escape sequences — from vt100.h
const (
	vtHomeClr = "\033[2J\033[0;0H" // VT_HOMECLR
	vtMarSet  = "\033[%d;%dr"      // VT_MARGSET
	vtCurSp   = "\033[%d;%dH"      // VT_CURSPOS
	vtCurSave = "\0337"            // VT_CURSAVE
	vtCurRest = "\0338"            // VT_CURREST
	vtNorm    = "\033[0m"          // CCNRM — reset
	vtGreen   = "\033[32m"         // CCGRN
	vtYellow  = "\033[33m"         // CCYEL
	vtRed     = "\033[31m"         // CCRED
	vtBlue    = "\033[34m"         // CCBLU
	vtMagenta = "\033[35m"         // CCMAG
)

// infobarSeparator draws the separator line in the infobar.
func infobarSeparator(ch *infobarState) string {
	// C's overlapping sprintf in IB_Seperator produces this five-cell
	// string on the oracle's libc; preserve the observed player-facing bytes.
	return fmt.Sprintf(vtCurSp+"+-----+-----+-----+-----+-----+", ch.screenSize-4, 1)
}

// infobarHitPointsStr draws the "Hit Pts:" label.
func infobarHitPointsStr(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"Hit Pts: ", ch.screenSize-3, 1)
}

// infobarHitPoints draws the actual hit points value with color.
func infobarHitPoints(ch *infobarState) string {
	count := ch.lastHit
	maxcount := ch.lastMaxHit
	percent := float64(count) / float64(maxcount)

	var colorOpen, colorClose string
	if percent >= 0.95 {
		colorOpen = vtGreen
	} else if percent >= 0.33 {
		colorOpen = vtYellow
	} else {
		colorOpen = vtRed
	}
	colorClose = vtNorm

	return fmt.Sprintf(vtCurSp+"%s%d%s(%s%d%s)", ch.screenSize-3, 10,
		colorOpen, count, colorClose, vtGreen, maxcount, vtNorm)
}

// infobarManaPointsStr draws the "Mana Pts:" label.
func infobarManaPointsStr(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"Mana Pts: ", ch.screenSize-3, 26)
}

// infobarManaPoints draws the mana points value with color.
func infobarManaPoints(ch *infobarState) string {
	count := ch.lastMana
	maxcount := ch.lastMaxMana
	percent := float64(count) / float64(maxcount)

	var colorOpen string
	if percent >= 0.95 {
		colorOpen = vtGreen
	} else if percent >= 0.33 {
		colorOpen = vtYellow
	} else {
		colorOpen = vtRed
	}

	return fmt.Sprintf(vtCurSp+"%s%d%s(%s%d%s)", ch.screenSize-3, 36,
		colorOpen, count, vtNorm, vtGreen, maxcount, vtNorm)
}

// infobarMovePointsStr draws the "Move Pts:" label.
func infobarMovePointsStr(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"Move Pts: ", ch.screenSize-3, 53)
}

// infobarMovePoints draws the move points value with color.
func infobarMovePoints(ch *infobarState) string {
	count := ch.lastMove
	maxcount := ch.lastMaxMove
	percent := float64(count) / float64(maxcount)

	var colorOpen string
	if percent >= 0.95 {
		colorOpen = vtGreen
	} else if percent >= 0.33 {
		colorOpen = vtYellow
	} else {
		colorOpen = vtRed
	}

	return fmt.Sprintf(vtCurSp+"%s%d%s(%s%d%s)", ch.screenSize-3, 63,
		colorOpen, count, vtNorm, vtGreen, maxcount, vtNorm)
}

// infobarExpPointsStr draws the "Exp:" label.
func infobarExpPointsStr(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"Exp: ", ch.screenSize-2, 1)
}

// infobarExpPoints draws the experience points value.
func infobarExpPoints(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"%s%d%s", ch.screenSize-2, 6,
		vtBlue, ch.lastExp, vtNorm)
}

// infobarNeededExpPointsStr draws the "Needed for Level " label.
func infobarNeededExpPointsStr(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"Needed for Level ", ch.screenSize-2, 26)
}

// infobarNeededExpPoints draws the needed experience value.
func infobarNeededExpPoints(ch *infobarState) string {
	neededExp := ch.expNeededForLevel - ch.lastExp
	return fmt.Sprintf(vtCurSp+"%d", ch.screenSize-2, 47, neededExp)
}

// infobarLevelStr draws the ": " separator after "Needed for Level X".
func infobarLevelStr(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+": ", ch.screenSize-2, 45)
}

// infobarLevel draws the next level number.
func infobarLevel(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"%2d", ch.screenSize-2, 43, ch.nextLevel)
}

// infobarGoldStr draws the "Gold:" label.
func infobarGoldStr(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"Gold: ", ch.screenSize-1, 1)
}

// infobarGold draws the gold value.
func infobarGold(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"%s%d%s", ch.screenSize-1, 7,
		vtMagenta, ch.lastGold, vtNorm)
}

// ---------------------------------------------------------------------------
// infobarState — VT100 state for building infobar output
// ---------------------------------------------------------------------------

type infobarState struct {
	screenSize        int
	lastHit           int
	lastMaxHit        int
	lastMana          int
	lastMaxMana       int
	lastMove          int
	lastMaxMove       int
	lastExp           int
	lastGold          int
	expNeededForLevel int
	nextLevel         int
	level             int
}

func newInfobarState(s *Session) *infobarState {
	p := s.player
	// C's exp_needed_for_level(ch) passes the current level to find_exp.
	expNeeded := game.FindExp(p.Class, p.Level)
	nextLvl := p.Level + 1

	return &infobarState{
		screenSize:        s.screenSize,
		lastHit:           p.Health,
		lastMaxHit:        p.MaxHealth,
		lastMana:          p.Mana,
		lastMaxMana:       p.MaxMana,
		lastMove:          p.Move,
		lastMaxMove:       p.MaxMove,
		lastExp:           p.Exp,
		lastGold:          p.Gold,
		expNeededForLevel: expNeeded,
		nextLevel:         nextLvl,
		level:             p.Level,
	}
}

func (s *Session) rememberInfobarValues() {
	if s.player == nil {
		return
	}
	s.infobarLastHit = s.player.Health
	s.infobarLastMaxHit = s.player.MaxHealth
	s.infobarLastMana = s.player.Mana
	s.infobarLastMaxMana = s.player.MaxMana
	s.infobarLastMove = s.player.Move
	s.infobarLastMaxMove = s.player.MaxMove
	s.infobarLastExp = s.player.Exp
	s.infobarLastGold = s.player.Gold
}

func infobarClearHitPoints(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"          ", ch.screenSize-3, 10)
}

func infobarClearManaPoints(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"          ", ch.screenSize-3, 36)
}

func infobarClearMovePoints(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"          ", ch.screenSize-3, 63)
}

func infobarClearExpPoints(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"        ", ch.screenSize-2, 6)
}

func infobarClearNeededExpPoints(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"        ", ch.screenSize-2, 47)
}

func infobarClearLevel(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"  ", ch.screenSize-2, 43)
}

func infobarClearGold(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"        ", ch.screenSize-1, 7)
}

// cmdInfoBarUpdate mirrors comm.c:1158-1193 and act.display.c:226-285.
// The prompt cycle detects changed values, repaints each changed field in C's
// bit order, then records the current values for the next cycle.
func cmdInfoBarUpdate(s *Session) {
	if s.player == nil || s.screenSize <= 0 || s.infobarMode != InfobarOn {
		return
	}

	p := s.player
	update := 0
	if p.Move != s.infobarLastMove || p.MaxMove != s.infobarLastMaxMove {
		update |= InfoMove
	}
	if p.Mana != s.infobarLastMana || p.MaxMana != s.infobarLastMaxMana {
		update |= InfoMana
	}
	if p.Health != s.infobarLastHit || p.MaxHealth != s.infobarLastMaxHit {
		update |= InfoHit
	}
	if p.Gold != s.infobarLastGold {
		update |= InfoGold
	}
	if p.Exp != s.infobarLastExp {
		update |= InfoExp
	}
	if update == 0 {
		return
	}

	is := newInfobarState(s)
	output := ""
	if update&InfoMana != 0 {
		output += vtCurSave + infobarClearManaPoints(is) + infobarManaPoints(is) + vtCurRest
	}
	if update&InfoMove != 0 {
		output += vtCurSave + infobarClearMovePoints(is) + infobarMovePoints(is) + vtCurRest
	}
	if update&InfoHit != 0 {
		output += vtCurSave + infobarClearHitPoints(is) + infobarHitPoints(is) + vtCurRest
	}
	if update&InfoExp != 0 {
		output += vtCurSave + infobarClearExpPoints(is) + infobarExpPoints(is)
		if is.level < game.LVL_IMMORT {
			output += infobarClearLevel(is) + infobarLevel(is)
			output += infobarClearNeededExpPoints(is) + infobarNeededExpPoints(is)
		}
		output += vtCurRest
	}
	if update&InfoGold != 0 {
		output += vtCurSave + infobarClearGold(is) + infobarGold(is) + vtCurRest
	}

	s.sendRawEvent(output)
	s.rememberInfobarValues()
}

// ---------------------------------------------------------------------------
// Command handlers
// ---------------------------------------------------------------------------

// cmdLines implements do_lines from act.display.c
// Syntax: lines [number]
func cmdLines(s *Session, args []string) error {
	if len(args) == 0 || args[0] == "" {
		s.Send(fmt.Sprintf("Your current screen size is %d.\r\n", s.screenSize))
		return nil
	}

	size := parseLinesSize(args[0])

	if size > 50 {
		s.Send("Screen size is limited to 50 lines.\r\n")
		return nil
	}
	if size < 7 {
		s.Send("Screen size must be at least 7 lines.\r\n")
		return nil
	}

	s.screenSize = size

	// Redraw if infobar is on
	if s.infobarMode == InfobarOn {
		cmdInfoBarOn(s)
	}

	s.Send(fmt.Sprintf("Your new lines count is %d.\r\n", size))
	return nil
}

// parseLinesSize mirrors the C atoi call in do_lines. C accepts an optional
// sign and leading decimal digits, returns zero when no digits are present,
// and ignores a trailing suffix.
func parseLinesSize(value string) int {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	index := 0
	if value[index] == '+' || value[index] == '-' {
		index++
	}
	start := index
	for index < len(value) && value[index] >= '0' && value[index] <= '9' {
		index++
	}
	if index == start {
		return 0
	}

	parsed, err := strconv.Atoi(value[:index])
	if err != nil {
		return 0
	}
	return parsed
}

// cmdInfoBar implements do_infobar from act.display.c
// Syntax: infobar [on|off]
func cmdInfoBar(s *Session, args []string) error {
	p := s.player
	if p == nil {
		return nil
	}

	if len(args) == 0 || args[0] == "" {
		switch s.infobarMode {
		case InfobarOff:
			s.Send("Your infobar is off.\r\n")
		case InfobarOn:
			s.Send("Your infobar is on.\r\n")
		default:
			s.Send("You had an unknown infobar setting.\r\n")
			s.Send("It is being set to OFF.\r\n")
			s.infobarMode = InfobarOff
		}
		return nil
	}

	switch strings.ToLower(args[0]) {
	case "off":
		if s.infobarMode == InfobarOn {
			s.infobarMode = InfobarOff
			cmdInfoBarOff(s)
			s.Send("Your infobar is now set to off.\r\n")
		} else {
			s.Send("Your infobar is already off.\r\n")
		}
	case "on":
		if s.infobarMode == InfobarOff {
			if s.screenSize == 0 {
				s.screenSize = 25
			}
			s.infobarMode = InfobarOn
			cmdInfoBarOn(s)
			s.Send("Your infobar is now set to on.\r\n")
		} else {
			s.Send("Your infobar is already on.\r\n")
		}
	default:
		s.Send("Usage:  infobar < on | off >\r\n")
	}

	return nil
}

// cmdInfoBarOn — InfoBarOn from act.display.c
func cmdInfoBarOn(s *Session) {
	p := s.player
	if p == nil {
		return
	}

	is := newInfobarState(s)
	output := ""

	// Clear screen
	output += vtHomeClr

	// Set scroll margin
	output += fmt.Sprintf(vtMarSet, 0, is.screenSize-5)

	// Draw labels and separators
	output += infobarSeparator(is)
	output += infobarHitPointsStr(is)
	output += infobarManaPointsStr(is)
	output += infobarMovePointsStr(is)
	output += infobarExpPointsStr(is)

	if is.level < game.LVL_IMMORT {
		output += infobarLevelStr(is)
		output += infobarNeededExpPointsStr(is)
	}

	output += infobarGoldStr(is)

	// Draw values
	output += infobarHitPoints(is)
	output += infobarMovePoints(is)
	output += infobarManaPoints(is)
	output += infobarExpPoints(is)

	if is.level < game.LVL_IMMORT {
		output += infobarNeededExpPoints(is)
		output += infobarLevel(is)
	}

	output += infobarGold(is)
	s.rememberInfobarValues()

	// Cursor to top-left
	output += fmt.Sprintf(vtCurSp, 0, 0)

	s.sendRawEvent(output)
}

// cmdInfoBarOff — InfoBarOff from act.display.c
func cmdInfoBarOff(s *Session) {
	output := ""
	// Reset margin to full screen
	output += fmt.Sprintf(vtMarSet, 0, s.screenSize-1)
	// Clear screen
	output += vtHomeClr

	s.sendRawEvent(output)
}
