package game

import (
	"log/slog"
	"math/rand"
)

// conAppType represents the con_app[] table from constants.c:1124-1150
// Original: struct con_app_type { sh_int hitp; sh_int shock; }
type conAppType struct {
	Hitp  int // HP bonus per level
	Shock int // Shock value (not used in advance_level)
}

// conApp is the con_app[] table from constants.c:1124-1150
// Index is CON score (0-25)
var conApp = []conAppType{
	{-4, 20}, // con = 0
	{-3, 25}, // con = 1
	{-2, 30},
	{-2, 35},
	{-1, 40},
	{-1, 45}, // con = 5
	{-1, 50},
	{0, 55},
	{0, 60},
	{0, 65},
	{0, 70}, // con = 10
	{0, 75},
	{0, 80},
	{0, 85},
	{0, 88},
	{1, 90}, // con = 15
	{2, 95},
	{2, 97},
	{3, 99}, // con = 18
	{3, 99},
	{4, 99}, // con = 20
	{5, 99},
	{5, 99},
	{5, 99},
	{6, 99},
	{6, 99}, // con = 25
}

// wisAppType represents the wis_app[] table from constants.c:1152-1178
// Original: struct wis_app_type { sh_int bonus; };
type wisAppType struct {
	Bonus int // Practice bonus
}

// wisApp is the wis_app[] table from constants.c:1152-1178
// Index is WIS score (0-25)
var wisApp = []wisAppType{
	{0}, // wis = 0
	{0}, // wis = 1
	{0},
	{0},
	{0},
	{0}, // wis = 5
	{0},
	{0},
	{0},
	{0},
	{0}, // wis = 10
	{0},
	{2},
	{2},
	{3},
	{3}, // wis = 15
	{3},
	{4},
	{5}, // wis = 18
	{6},
	{6}, // wis = 20
	{6},
	{6},
	{7},
	{7},
	{7}, // wis = 25
}

// AdvanceLevel implements advance_level() from class.c:600-720
// Calculates HP/mana/move gains when a player levels up.
// Called from do_start() at level 1, so even level 1 chars get more than 10 HP.
func (p *Player) AdvanceLevel() {
	p.mu.Lock()
	defer p.mu.Unlock()

	addHP := 0
	addMana := 0
	addMove := 0

	// Base HP gain from constitution
	con := p.Stats.Con
	if con < 0 {
		con = 0
	}
	if con >= len(conApp) {
		con = len(conApp) - 1
	}
	addHP = conApp[con].Hitp

	// Class-specific gains
	switch p.Class {
	case ClassMageUser, ClassMagus:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addHP += rand.Intn(5) + 4                          // number(4,8)
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMana = rand.Intn(3*p.Level-p.Level+1) + p.Level // number(GET_LEVEL(ch), (int) (3 * GET_LEVEL(ch)))
		if addMana > 10 {
			addMana = 10
		}
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMove = rand.Intn(3) + 1 // number(1, 3)
		// Practices: MAX(2, wis_app[GET_WIS(ch)].bonus)
		wis := p.Stats.Wis
		if wis < 0 {
			wis = 0
		}
		if wis >= len(wisApp) {
			wis = len(wisApp) - 1
		}
		practices := wisApp[wis].Bonus
		if practices < 2 {
			practices = 2
		}
		p.Practices += practices

	case ClassCleric, ClassAvatar:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addHP += rand.Intn(5) + 5                          // number(5, 9)
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMana = rand.Intn(3*p.Level-p.Level+1) + p.Level // number(GET_LEVEL(ch), (int) (3 * GET_LEVEL(ch)))
		if addMana > 10 {
			addMana = 10
		}
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMove = rand.Intn(3) + 1 // number(1, 3)
		// Practices: MAX(2, wis_app[GET_WIS(ch)].bonus)
		wis := p.Stats.Wis
		if wis < 0 {
			wis = 0
		}
		if wis >= len(wisApp) {
			wis = len(wisApp) - 1
		}
		practices := wisApp[wis].Bonus
		if practices < 2 {
			practices = 2
		}
		p.Practices += practices

	case ClassAssassin:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addHP += rand.Intn(7) + 8                          // number(8, 14)
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMana = rand.Intn(2*p.Level-p.Level+1) + p.Level // number(GET_LEVEL(ch), (int)(2 * GET_LEVEL(ch)))
		if addMana > 5 {
			addMana = 5
		}
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMove = rand.Intn(4) + 1 // number(1, 4)
		// Practices: MIN(2, MAX(1, wis_app[GET_WIS(ch)].bonus))
		wis := p.Stats.Wis
		if wis < 0 {
			wis = 0
		}
		if wis >= len(wisApp) {
			wis = len(wisApp) - 1
		}
		practices := wisApp[wis].Bonus
		if practices < 1 {
			practices = 1
		}
		if practices > 2 {
			practices = 2
		}
		p.Practices += practices

	case ClassThief:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addHP += rand.Intn(7) + 7  // number(7, 13)
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMove = rand.Intn(4) + 1 // number(1, 4)
		// Practices: MIN(2, MAX(1, wis_app[GET_WIS(ch)].bonus))
		wis := p.Stats.Wis
		if wis < 0 {
			wis = 0
		}
		if wis >= len(wisApp) {
			wis = len(wisApp) - 1
		}
		practices := wisApp[wis].Bonus
		if practices < 1 {
			practices = 1
		}
		if practices > 2 {
			practices = 2
		}
		p.Practices += practices

	case ClassPaladin:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMana = rand.Intn(2*p.Level-p.Level+1) + p.Level // number(GET_LEVEL(ch), (int)(2 * GET_LEVEL(ch)))
		if addMana > 5 {
			addMana = 5
		}
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addHP += rand.Intn(5) + 12 // number(12, 16)
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMove = rand.Intn(4) + 1 // number(1, 4)
		// Practices: MIN(2, MAX(1, wis_app[GET_WIS(ch)].bonus))
		wis := p.Stats.Wis
		if wis < 0 {
			wis = 0
		}
		if wis >= len(wisApp) {
			wis = len(wisApp) - 1
		}
		practices := wisApp[wis].Bonus
		if practices < 1 {
			practices = 1
		}
		if practices > 2 {
			practices = 2
		}
		p.Practices += practices

	case ClassRanger:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addHP += rand.Intn(4) + 13 // number(13, 16)
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMove = rand.Intn(3) + 2 // number(2, 4)
		// Practices: MIN(2, MAX(1, wis_app[GET_WIS(ch)].bonus))
		wis := p.Stats.Wis
		if wis < 0 {
			wis = 0
		}
		if wis >= len(wisApp) {
			wis = len(wisApp) - 1
		}
		practices := wisApp[wis].Bonus
		if practices < 1 {
			practices = 1
		}
		if practices > 2 {
			practices = 2
		}
		p.Practices += practices

	case ClassWarrior:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addHP += rand.Intn(4) + 11 // number(11, 14)
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMove = rand.Intn(4) + 1 // number(1, 4)
		// Practices: MIN(2, MAX(1, wis_app[GET_WIS(ch)].bonus))
		wis := p.Stats.Wis
		if wis < 0 {
			wis = 0
		}
		if wis >= len(wisApp) {
			wis = len(wisApp) - 1
		}
		practices := wisApp[wis].Bonus
		if practices < 1 {
			practices = 1
		}
		if practices > 2 {
			practices = 2
		}
		p.Practices += practices

	case ClassNinja:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addHP += rand.Intn(6) + 8                          // number(8, 13)
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMana = rand.Intn(2*p.Level-p.Level+1) + p.Level // number(GET_LEVEL(ch), (int)(2 * GET_LEVEL(ch)))
		if addMana > 10 {
			addMana = 10
		}
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMove = rand.Intn(4) + 1 // number(1, 4)
		// Practices: MIN(2, MAX(1, wis_app[GET_WIS(ch)].bonus))
		wis := p.Stats.Wis
		if wis < 0 {
			wis = 0
		}
		if wis >= len(wisApp) {
			wis = len(wisApp) - 1
		}
		practices := wisApp[wis].Bonus
		if practices < 1 {
			practices = 1
		}
		if practices > 2 {
			practices = 2
		}
		p.Practices += practices

	case ClassPsionic, ClassMystic:
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addHP += rand.Intn(5) + 4 // number(4,8) for psionic, (5,9) for mystic
		if p.Class == ClassMystic {
			// #nosec G404 — game RNG, not cryptographic
// #nosec G404
			addHP = rand.Intn(5) + 5 // number(5, 9)
		}
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMana = rand.Intn(2*p.Level-p.Level+1) + p.Level // number(GET_LEVEL(ch), (int)(2 * GET_LEVEL(ch)))
		if addMana > 10 {
			addMana = 10
		}
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		addMove = rand.Intn(4) + 1 // number(1, 4)
		// Practices: MAX(2, wis_app[GET_WIS(ch)].bonus)
		wis := p.Stats.Wis
		if wis < 0 {
			wis = 0
		}
		if wis >= len(wisApp) {
			wis = len(wisApp) - 1
		}
		practices := wisApp[wis].Bonus
		if practices < 2 {
			practices = 2
		}
		p.Practices += practices
	}

	// Apply gains with minimum of 1
	if addHP < 1 {
		addHP = 1
	}
	if addMove < 1 {
		addMove = 1
	}

	p.MaxHealth += addHP
	if p.Level > 1 {
		p.MaxMana += addMana
	}
	p.MaxMove += addMove

	// Heal to new max, including move points
	p.Health = p.MaxHealth
	p.Mana = p.MaxMana
	p.Move = p.MaxMove

	// Immortal perks
	if p.Level >= LVL_IMMORT {
		for i := 0; i < 3; i++ {
			p.SetCondition(i, -1)
		}
		p.HolyLight = true
	}

	// Save after leveling up
	if err := SavePlayer(p); err != nil {
		slog.Error("Failed to save player after leveling up", "name", p.Name, "error", err)
	}

	slog.Info("advanced to level", "name", p.Name, "level", p.Level)
}

