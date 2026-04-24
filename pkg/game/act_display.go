// Package game — display commands ported from src/act.display.c
//
// The infobar is a VT100-based stat display drawn at the bottom of the
// terminal, showing hit points, mana, move, experience, level, and gold.
// It uses VT100 scroll-region margins and cursor save/restore sequences.
//
// The actual terminal-rendering implementation lives in pkg/session/display_cmds.go
// (the session layer owns VT100 output). This file defines the data model
// and constants shared between game and session packages.
//
// Original C functions and their Go equivalents (in pkg/session/display_cmds.go):
//
//   C function                     Go session function
//   ─────────────────────────      ─────────────────────────────
//   do_lines()                     cmdLines()
//   do_infobar()                   cmdInfoBar()
//   InfoBarOn()                    cmdInfoBarOn()
//   InfoBarOff()                   cmdInfoBarOff()
//   InfoBarUpdate()                cmdInfoBarUpdate()
//   IB_Seperator()                 infobarSeparator()
//   IB_HitPointsStr()              infobarHitPointsStr()
//   IB_HitPoints() / IB_W_*       infobarHitPoints()
//   IB_ClearHit()                  infobarClearHit()
//   IB_ManaPointsStr()             infobarManaPointsStr()
//   IB_ManaPoints() / IB_W_*      infobarManaPoints()
//   IB_ClearMana()                 infobarClearMana()
//   IB_MovePointsStr()             infobarMovePointsStr()
//   IB_MovePoints() / IB_W_*      infobarMovePoints()
//   IB_ClearMove()                  infobarClearMove()
//   IB_ExpPointsStr()              infobarExpPointsStr()
//   IB_ExpPoints() / IB_W_*       infobarExpPoints()
//   IB_ClearExpPoints()            infobarClearExpPoints()
//   IB_NeededExpPointsStr()        infobarNeededExpPointsStr()
//   IB_NeededExpPoints() / IB_W_* infobarNeededExpPoints()
//   IB_ClearNeededExpPoints()      infobarClearNeededExpPoints()
//   IB_LevelStr()                  infobarLevelStr()
//   IB_Level() / IB_W_*           infobarLevel()
//   IB_ClearLevel()                infobarClearLevel()
//   IB_GoldStr()                   infobarGoldStr()
//   IB_Gold() / IB_W_*            infobarGold()
//   IB_ClearGold()                 infobarClearGold()
//
// Porting notes:
//   - C macros GET_HIT, GET_MAX_HIT → Player.Health, Player.MaxHealth
//   - C macros GET_MANA, GET_MAX_MANA → Player.Mana, Player.MaxMana
//   - C macros GET_MOVE, GET_MAX_MOVE → Player.Move, Player.MaxMove
//   - C macro GET_EXP → Player.Exp
//   - C macro GET_GOLD → Player.Gold
//   - C macro GET_LEVEL → Player.Level
//   - VT100 sequences moved to constants in session package
//   - write_to_output → s.Send() on the Session
//   - send_to_char → s.Send() on the Session
//   - LVL_IMMORT → 50 (session/wizard_cmds.go)
//   - exp_needed_for_level → estimated as 1000*level (simplified)
//   - "last known" values tracked in infobarState struct
//   - GET_LASTHIT/LASTMAXHIT/LASTMANA/etc → infobarState fields
package game

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
