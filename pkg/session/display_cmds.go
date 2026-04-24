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
)

// Infobar state constants — from structs.h
const (
	InfobarOff = 0
	InfobarOn  = 1
)

// Info update bitmask constants — INFO_* from act.display.c
const (
	InfoMana = 1 << iota
	InfoMove
	InfoHit
	InfoExp
	InfoGold
)

// VT100 escape sequences — from vt100.h
const (
	vtHomeClr  = "\033[H\033[J"                 // VT_HOMECLR
	vtMarSet   = "\033[%d;%dr"                  // VT_MARGSET
	vtCurSp    = "\033[%d;%dH"                  // VT_CURSPOS
	vtCurSave  = "\033[s"                       // VT_CURSAVE
	vtCurRest  = "\033[u"                       // VT_CURREST
	vtNorm     = "\033[0m"                      // CCNRM — reset
	vtGreen    = "\033[32m"                     // CCGRN
	vtYellow   = "\033[33m"                     // CCYEL
	vtRed      = "\033[31m"                     // CCRED
	vtBlue     = "\033[34m"                     // CCBLU
	vtMagenta  = "\033[35m"                     // CCMAG
)

// infobarSeparator draws the separator line in the infobar.
func infobarSeparator(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+", ch.screenSize-4, 1)
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
	return fmt.Sprintf(vtCurSp+"Move Pts: ", ch.screenSize-3, 48)
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

	return fmt.Sprintf(vtCurSp+"%s%d%s(%s%d%s)", ch.screenSize-3, 60,
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
// Infobar helper: clear functions (write spaces to clear the region)
// ---------------------------------------------------------------------------

func infobarClearHit(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"          ", ch.screenSize-3, 10)
}

func infobarClearMana(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"          ", ch.screenSize-3, 36)
}

func infobarClearMove(ch *infobarState) string {
	return fmt.Sprintf(vtCurSp+"          ", ch.screenSize-3, 60)
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
	return fmt.Sprintf(vtCurSp+"           ", ch.screenSize-1, 7)
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
	// Estimate exp needed — simplified: 1000 * level per level
	expNeeded := 1000 * p.Level
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

	size, err := strconv.Atoi(args[0])
	if err != nil {
		s.Send("Usage: lines <number>\r\n")
		return nil
	}

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

	if is.level < 50 { // LVL_IMMORT
		output += infobarLevelStr(is)
		output += infobarNeededExpPointsStr(is)
	}

	output += infobarGoldStr(is)

	// Draw values
	output += infobarHitPoints(is)
	output += infobarManaPoints(is)
	output += infobarMovePoints(is)
	output += infobarExpPoints(is)

	if is.level < 50 { // LVL_IMMORT
		output += infobarNeededExpPoints(is)
		output += infobarLevel(is)
	}

	output += infobarGold(is)

	// Cursor to top-left
	output += fmt.Sprintf(vtCurSp, 0, 0)

	s.Send(output)
}

// cmdInfoBarOff — InfoBarOff from act.display.c
func cmdInfoBarOff(s *Session) {
	output := ""
	// Reset margin to full screen
	output += fmt.Sprintf(vtMarSet, 0, s.screenSize-1)
	// Clear screen
	output += vtHomeClr

	s.Send(output)
}

// cmdInfoBarUpdate — InfoBarUpdate from act.display.c
// update is a bitmask of InfoMana | InfoMove | InfoHit | InfoExp | InfoGold
func cmdInfoBarUpdate(s *Session, update int) {
	p := s.player
	if p == nil {
		return
	}

	if s.screenSize <= 0 {
		return
	}

	is := &infobarState{
		screenSize:  s.screenSize,
		lastHit:     p.Health,
		lastMaxHit:  p.MaxHealth,
		lastMana:    p.Mana,
		lastMaxMana: p.MaxMana,
		lastMove:    p.Move,
		lastMaxMove: p.MaxMove,
		lastExp:     p.Exp,
		lastGold:    p.Gold,
		level:       p.Level,
	}
	is.nextLevel = is.level + 1
	is.expNeededForLevel = 1000 * is.level

	output := ""

	if update&InfoMana != 0 {
		output += vtCurSave
		output += infobarClearMana(is)
		output += infobarManaPoints(is)
		output += vtCurRest
	}

	if update&InfoMove != 0 {
		output += vtCurSave
		output += infobarClearMove(is)
		output += infobarMovePoints(is)
		output += vtCurRest
	}

	if update&InfoHit != 0 {
		output += vtCurSave
		output += infobarClearHit(is)
		output += infobarHitPoints(is)
		output += vtCurRest
	}

	if update&InfoExp != 0 {
		output += vtCurSave
		output += infobarClearExpPoints(is)
		output += infobarExpPoints(is)
		if is.level < 50 { // LVL_IMMORT
			output += infobarClearLevel(is)
			output += infobarLevel(is)
			output += infobarClearNeededExpPoints(is)
			output += infobarNeededExpPoints(is)
		}
		output += vtCurRest
	}

	if update&InfoGold != 0 {
		output += vtCurSave
		output += infobarClearGold(is)
		output += infobarGold(is)
		output += vtCurRest
	}

	if output != "" {
		s.Send(output)
	}
}
