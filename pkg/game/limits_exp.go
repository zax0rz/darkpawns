package game

import (
	"fmt"
	"log/slog"
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

// ---------------------------------------------------------------------------
// SetTitle — from limits.c set_title()
// ---------------------------------------------------------------------------
func SetTitle(p *Player, title string) {
	if title == "" {
		class := p.Class
		if class >= 0 && class < len(Titles) {
			title = Titles[class]
		} else {
			title = "the Adventurer"
		}
	}
	if len(title) > MAX_TITLE_LENGTH {
		title = title[:MAX_TITLE_LENGTH]
	}
	p.Title = title
}

// ---------------------------------------------------------------------------
// CheckAutowiz — from limits.c check_autowiz()
// ---------------------------------------------------------------------------
func CheckAutowiz(p *Player) {
	if p == nil || p.Level < LVL_IMMORT {
		return
	}
	// C spawns autowiz external binary. In Go, log and defer to admin system.
	// Source: src/limits.c:268-281
	slog.Info("autowiz triggered", "player", p.Name, "level", p.Level)
}

// ---------------------------------------------------------------------------
// FindExp — from class.c find_exp()
// ---------------------------------------------------------------------------
func FindExp(class int, level int) int {
	var modifier float64

	switch class {
	case ClassMageUser:
		modifier = 0.3
	case ClassCleric:
		modifier = 0.4
	case ClassWarrior:
		modifier = 0.7
	case ClassThief:
		modifier = 0.1
	case ClassMagus, ClassMystic:
		modifier = 1.5
	case ClassAvatar:
		modifier = 1.6
	case ClassAssassin:
		modifier = 1.2
	case ClassPaladin, ClassRanger:
		modifier = 1.9
	case ClassNinja, ClassPsionic:
		modifier = 0.6
	default:
		modifier = 1.0
	}

	switch {
	case level <= 0:
		return 1
	case level == 1:
		return 1500
	case level == 2:
		return 3000
	case level == 3:
		return 6000
	case level == 4:
		return 11000
	case level == 5:
		return 21000
	case level == 6:
		return 42000
	case level == 7:
		return 80000
	case level == 8:
		return 155000
	case level == 9:
		return 300000
	case level == 10:
		return 450000
	case level == 11:
		return 650000
	case level == 12:
		return 870000
	default:
		return 900000 + ((level-13)*level*20000) + (level*level*1000) + int(modifier*10000*float64(level))
	}
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
		// Per-level XP cap: limits single-kill XP to level * 1000.
		// Prevents low-level characters from gaining disproportionate XP
		// from high-level kills, while still allowing meaningful gains at
		// higher levels. This scales with level rather than being a flat cap.
		perLevelCap := p.Level * 1000
		if perLevelCap < 1000 {
			perLevelCap = 1000
		}
		if gain > maxExpGain {
			gain = maxExpGain
		}
		if gain > perLevelCap {
			gain = perLevelCap
		}

		maxExp := FindExp(p.Class, p.Level+1) - p.Exp
		if gain > maxExp-1 {
			gain = maxExp - 1
			if gain < 1 {
				gain = 1
			}
		}

		p.Exp += gain

		if p.Level < LVL_IMPL-1 && p.Exp >= ExpNeededForLevel(p) {
			// AFF_FLESH_ALTER handling — adjust hit/damroll before/after level-up
			// C: flesh_alter_from() removes bonuses, advance_level(), flesh_alter_to() restores
			// Source: src/new_cmds.c:1751-1769, src/limits.c:305-311
			const affFleshAlterBit = 16 // AFF_FLESH_ALTER from structs.h:326
			hasFleshAlter := p.Affects&(1<<affFleshAlterBit) != 0
			if hasFleshAlter {
				// flesh_alter_from: temporarily remove flesh alter bonuses
				p.mu.Lock()
				p.Hitroll -= (p.Level/3) + 1
				p.Damroll -= (p.Level/2) + 1
				p.mu.Unlock()
			}
			p.Level++
			p.AdvanceLevel()
			if hasFleshAlter {
				// flesh_alter_to: restore flesh alter bonuses at new level
				p.mu.Lock()
				p.Hitroll += (p.Level/3) + 1
				p.Damroll += (p.Level/2) + 1
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
	if p == nil {
		return
	}

	p.Exp += gain
	if p.Exp < 0 {
		p.Exp = 0
	}

	if p.IsNPC() {
		return
	}

	numLevels := 0
	const affFleshAlterBit = 16 // AFF_FLESH_ALTER from structs.h:326
	for p.Level < LVL_IMPL && p.Exp >= ExpNeededForLevel(p) {
		hasFleshAlter := p.Affects&(1<<affFleshAlterBit) != 0
		if hasFleshAlter {
			p.mu.Lock()
			p.Hitroll -= (p.Level/3) + 1
			p.Damroll -= (p.Level/2) + 1
			p.mu.Unlock()
		}
		p.Level++
		numLevels++
		p.AdvanceLevel()
		if hasFleshAlter {
			p.mu.Lock()
			p.Hitroll += (p.Level/3) + 1
			p.Damroll += (p.Level/2) + 1
			p.mu.Unlock()
		}
	}

	if numLevels > 0 {
		if numLevels == 1 {
			sendToChar(p, "You rise a level!\r\n")
		} else {
			sendToChar(p, fmt.Sprintf("You rise %d levels!\r\n", numLevels))
		}
		CheckAutowiz(p)
	}
}

// ---------------------------------------------------------------------------
// CheckIdling — from limits.c check_idling() (lines 419-441)
// Tracks idle time, pulls idle players to void, disconnects after extended idle.
