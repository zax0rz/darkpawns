package game

import (
	"fmt"
)

func updatePosFromHP(p *Player, hp int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if hp > 0 {
		if p.Position > PosStunned {
			return
		}
		p.Position = PosStanding
		return
	}
	if hp <= -11 {
		p.Position = PosDead
	} else if hp <= -6 {
		p.Position = PosMortally
	} else if hp <= -3 {
		p.Position = PosIncap
	} else {
		p.Position = PosStunned
	}
}

// updateMobPosFromHP mirrors updatePosFromHP for NPCs — derives the POS_*
// wounded band from current HP. Used by the point-update bleed loop so mobs
// progress incap → mortally → dead instead of dying instantly at HP 0.
func updateMobPosFromHP(m *MobInstance, hp int) {
	if hp > 0 {
		if m.GetPosition() > PosStunned {
			return
		}
		m.SetPosition(PosStanding)
		return
	}
	if hp <= -11 {
		m.SetPosition(PosDead)
	} else if hp <= -6 {
		m.SetPosition(PosMortally)
	} else if hp <= -3 {
		m.SetPosition(PosIncap)
	} else {
		m.SetPosition(PosStunned)
	}
}

// ---------------------------------------------------------------------------
// SetTitle — from limits.c set_title()
// ---------------------------------------------------------------------------
func SetTitle(p *Player, title string) {
	if len(title) > MAX_TITLE_LENGTH {
		title = title[:MAX_TITLE_LENGTH]
	}
	p.Title = title
}

// ---------------------------------------------------------------------------
// CheckAutowiz — from limits.c check_autowiz()
// ---------------------------------------------------------------------------

// CheckAutowiz is C's check_autowiz (src/limits.c:269-283), called from
// gain_exp_regardless the moment a rise message is sent (:360).
//
// C logs "Initiating autowiz." at CMP / LVL_IMMORT / file FALSE and then shells
// out to ../bin/autowiz, which rewrites wizlist and immlist. The port emits the
// producer and spawns nothing: the reference oracle has no autowiz binary and
// src/util/ has no source for one, so C's system() call fails quietly there
// too. Go's wizlist and immlist are read as static files, so nothing needs
// regenerating.
//
// The gate is C's, and both terms are constant here. use_autowiz is YES
// (src/config.c:273) and the port has no such flag, so YES is written directly
// rather than adding one (R4). mini_mud is set only by C's -m startup option
// (src/comm.c:194-196) and the port has no mini-mud mode, so !mini_mud always
// holds.
func CheckAutowiz(p *Player) {
	if p == nil || p.GetLevel() < LVL_IMMORT {
		return
	}
	MudLog("Initiating autowiz.", MudlogComplete, LVL_IMMORT, false)
}

// ---------------------------------------------------------------------------
// FindExp — from class.c find_exp()
// ---------------------------------------------------------------------------
// findExpClassModifiers mirrors the class modifier table in src/class.c.
// Classes outside this table use C's default modifier of 1.0.
var findExpClassModifiers = map[int]float64{
	ClassMageUser: 0.3,
	ClassCleric:   0.4,
	ClassWarrior:  0.7,
	ClassThief:    0.1,
	ClassMagus:    1.5,
	ClassMystic:   1.5,
	ClassAvatar:   1.6,
	ClassAssassin: 1.2,
	ClassPaladin:  1.9,
	ClassRanger:   1.9,
	ClassNinja:    0.6,
	ClassPsionic:  0.6,
}

// findExpFixedLevels contains C's hardcoded values for levels 0 through 12.
var findExpFixedLevels = [...]int{
	1,
	1500,
	3000,
	6000,
	11000,
	21000,
	42000,
	80000,
	155000,
	300000,
	450000,
	650000,
	870000,
}

func FindExp(class int, level int) int {
	modifier := 1.0
	if classModifier, ok := findExpClassModifiers[class]; ok {
		modifier = classModifier
	}

	if level <= 0 {
		return findExpFixedLevels[0]
	}
	if level < len(findExpFixedLevels) {
		return findExpFixedLevels[level]
	}
	return 900000 + ((level - 13) * level * 20000) + (level * level * 1000) + int(modifier*10000*float64(level))
}

// ---------------------------------------------------------------------------
// ExpNeededForLevel — from class.c exp_needed_for_level()
// ---------------------------------------------------------------------------
func ExpNeededForLevel(p *Player) int {
	return FindExp(p.Class, p.Level)
}

// ---------------------------------------------------------------------------
// GainExp — from limits.c gain_exp()
// ---------------------------------------------------------------------------
func (w *World) GainExp(p *Player, gain int) {
	if p == nil {
		return
	}

	if p.IsNPC() {
		p.Exp += gain
		return
	}

	if p.Level < 1 || p.Level >= LVL_IMMORT {
		return
	}

	if gain > 0 {
		if gain > maxExpGain {
			gain = maxExpGain
		}

		maxExp := FindExp(p.Class, p.Level+1) - p.Exp
		if gain > maxExp-1 {
			gain = maxExp - 1
			if gain < 1 {
				gain = 1
			}
		}

		p.Exp += gain

		if p.Level < LVL_IMMORT-1 && p.Exp >= ExpNeededForLevel(p) {
			// AFF_FLESH_ALTER handling — adjust hit/damroll before/after level-up
			// C: flesh_alter_from() removes bonuses, advance_level(), flesh_alter_to() restores
			// Source: src/new_cmds.c:1751-1769, src/limits.c:305-311
			// use canonical affFleshAlter from affects_constants.go
			hasFleshAlter := p.Affects&(1<<affFleshAlter) != 0
			if hasFleshAlter {
				// flesh_alter_from: temporarily remove flesh alter bonuses
				p.mu.Lock()
				p.Hitroll -= (p.Level / 3) + 1
				p.Damroll -= (p.Level / 2) + 1
				p.mu.Unlock()
			}
			p.Level++
			p.AdvanceLevel()
			if hasFleshAlter {
				// flesh_alter_to: restore flesh alter bonuses at new level
				p.mu.Lock()
				p.Hitroll += (p.Level / 3) + 1
				p.Damroll += (p.Level / 2) + 1
				p.mu.Unlock()
			}
			sendToChar(p, fmt.Sprintf("You advance to level %d!\r\n", p.Level))
		}
	} else if gain < 0 {
		if gain < -maxExpLoss {
			gain = -maxExpLoss
		}
		p.Exp += gain
		if p.Exp < 0 {
			p.Exp = 0
		}
	}
}

// ---------------------------------------------------------------------------
// GainExpRegardless — from limits.c gain_exp_regardless()
// ---------------------------------------------------------------------------
func (w *World) GainExpRegardless(p *Player, gain int) {
	w.gainExpRegardless(p, gain, true)
}

// GainExpRegardlessSilent applies the same state transitions as
// GainExpRegardless but leaves announcement framing to the caller. This is
// used by do_advance, whose C loop emits one contiguous stream of
// "You rise a level!" messages across repeated gain_exp_regardless calls.
func (w *World) GainExpRegardlessSilent(p *Player, gain int) int {
	return w.gainExpRegardless(p, gain, false)
}

func (w *World) gainExpRegardless(p *Player, gain int, announce bool) int {
	if p == nil {
		return 0
	}

	p.Exp += gain
	if p.Exp < 0 {
		p.Exp = 0
	}

	if p.IsNPC() {
		return 0
	}

	numLevels := 0
	// use canonical affFleshAlter from affects_constants.go
	for p.Level < LVL_IMPL && p.Exp >= ExpNeededForLevel(p) {
		hasFleshAlter := p.Affects&(1<<affFleshAlter) != 0
		if hasFleshAlter {
			p.mu.Lock()
			p.Hitroll -= (p.Level / 3) + 1
			p.Damroll -= (p.Level / 2) + 1
			p.mu.Unlock()
		}
		p.Level++
		numLevels++
		p.AdvanceLevel()
		if hasFleshAlter {
			p.mu.Lock()
			p.Hitroll += (p.Level / 3) + 1
			p.Damroll += (p.Level / 2) + 1
			p.mu.Unlock()
		}
	}

	if numLevels > 0 {
		if announce {
			if numLevels == 1 {
				sendToChar(p, "You rise a level!\r\n")
			} else {
				sendToChar(p, fmt.Sprintf("You rise %d levels!\r\n", numLevels))
			}
		}
		// src/limits.c:357-360: check_autowiz runs inside gain_exp_regardless
		// whenever a level was gained, alongside the rise message and not gated
		// on who prints it. do_advance's Silent call is C's per-level
		// gain_exp_regardless (src/act.wizard.c:1569-1571), so it must emit the
		// producer too, once per level gained at or above LVL_IMMORT.
		CheckAutowiz(p)
	}
	return numLevels
}

// ---------------------------------------------------------------------------
// CheckIdling — from limits.c check_idling() (lines 419-454)
// Tracks idle time, pulls idle players to void, disconnects after extended idle.
