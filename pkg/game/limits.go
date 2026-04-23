package game

import (
	"fmt"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// Condition constants — from structs.h
const (
	CondDrunk  = 0
	CondFull   = 1
	CondThirst = 2
)

// Position constants from combat package (for reference)
// POS_DEAD=0, POS_MORTALLY=1, POS_INCAP=2, POS_STUNNED=3,
// POS_SLEEPING=4, POS_RESTING=5, POS_SITTING=6, POS_FIGHTING=7, POS_STANDING=8

// ---------------------------------------------------------------------------
// GainCondition — from limits.c gain_condition()
// ---------------------------------------------------------------------------
// Tracks hunger/thirst/drunk for players. Values range -1 (gone) to 24 (full).
// In the original, clamped to 0-48. We clamp to 0-24 for player-facing range.
// Called every point_update tick.
func GainCondition(p *Player, condition int, value int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var current *int
	switch condition {
	case CondFull:
		current = &p.Hunger
	case CondThirst:
		current = &p.Thirst
	case CondDrunk:
		current = &p.Drunk
	default:
		return
	}

	if *current == -1 {
		return // No change
	}

	wasIntoxicated := p.Drunk > 0

	*current += value
	if *current < 0 {
		*current = 0
	}
	if *current > 48 {
		*current = 48
	}

	// Messages only when crossing thresholds
	if *current > 1 {
		return
	}

	if *current > 0 {
		switch condition {
		case CondFull:
			p.sendLocked("Your stomach growls with hunger.\r\n")
		case CondThirst:
			p.sendLocked("You feel a bit parched.\r\n")
		case CondDrunk:
			if wasIntoxicated {
				p.sendLocked("Your head starts to clear.\r\n")
			}
		}
	} else {
		switch condition {
		case CondFull:
			p.sendLocked("You are hungry.\r\n")
		case CondThirst:
			p.sendLocked("You are thirsty.\r\n")
		case CondDrunk:
			if wasIntoxicated {
				p.sendLocked("You are now sober.\r\n")
			}
		}
	}
}

// sendLocked sends a message without acquiring the lock (caller must hold it).
func (p *Player) sendLocked(msg string) {
	select {
	case p.Send <- []byte(msg):
	default:
	}
}

// ---------------------------------------------------------------------------
// ManaGain — from limits.c mana_gain()
// ---------------------------------------------------------------------------
// Calculates mana regeneration per tick. Dark Pawns uses flat base values
// with position/class modifiers, not percentage-of-max like stock CircleMUD.
func ManaGain(p *Player) int {
	if p == nil {
		return 0
	}

	p.mu.RLock()
	class := p.Class
	pos := p.Position
	p.mu.RUnlock()

	gain := 14

	// Position calculations
	switch pos {
	case combat.PosSleeping:
		gain <<= 1 // x2
	case combat.PosResting:
		gain += gain >> 1 // +50%
	case combat.PosSitting:
		gain += gain >> 2 // +25%
	}

	// Class calculations
	switch class {
	case ClassMageUser, ClassCleric:
		gain <<= 1 // x2
	case ClassMagus, ClassAvatar:
		gain <<= 1 // x2
	case ClassPsionic, ClassNinja:
		gain += gain >> 2 // +25%
	case ClassMystic:
		gain <<= 1 // x2
	}

	// TODO: Equipment mana regen bonuses (APPLY_MANA_REGEN) — Phase 3
	// TODO: Poison/flaming/cutthroat affect checks — Phase 3
	// TODO: Hunger/thirst reduction — Phase 3
	// TODO: Regen room bonus — Phase 3

	if gain < 0 {
		gain = 0
	}
	return gain
}

// ---------------------------------------------------------------------------
// HitGain — from limits.c hit_gain()
// ---------------------------------------------------------------------------
func HitGain(p *Player) int {
	if p == nil {
		return 0
	}

	p.mu.RLock()
	class := p.Class
	pos := p.Position
	p.mu.RUnlock()

	gain := 20

	// Position calculations
	switch pos {
	case combat.PosSleeping:
		gain += gain >> 1 // +50%
		// TODO: Equipment hit regen bonuses while sleeping — Phase 3
	case combat.PosResting:
		gain += gain >> 2 // +25%
	case combat.PosSitting:
		gain += gain >> 3 // +12.5%
	}

	// Class calculations
	if class == ClassMageUser || class == ClassCleric {
		gain >>= 1 // Half for casters
	}

	// TODO: KK_JIN skill bonus when not fighting — Phase 3
	// TODO: Poison/flaming/cutthroat affect checks — Phase 3
	// TODO: Hunger/thirst reduction — Phase 3
	// TODO: Regen room bonus — Phase 3

	if gain < 0 {
		gain = 0
	}
	return gain
}

// ---------------------------------------------------------------------------
// MoveGain — from limits.c move_gain()
// ---------------------------------------------------------------------------
func MoveGain(p *Player) int {
	if p == nil {
		return 0
	}

	p.mu.RLock()
	pos := p.Position
	p.mu.RUnlock()

	gain := 20

	// Position calculations
	switch pos {
	case combat.PosSleeping:
		// TODO: Equipment move regen bonuses while sleeping — Phase 3
		gain += gain >> 1 // +50%
	case combat.PosResting:
		gain += gain >> 2 // +25%
	case combat.PosSitting:
		gain += gain >> 3 // +12.5%
	}

	// TODO: KK_ZHEN skill bonus when not fighting — Phase 3
	// TODO: Poison/flaming/cutthroat affect checks — Phase 3
	// TODO: Hunger/thirst reduction — Phase 3
	// TODO: Regen room bonus — Phase 3

	if gain < 0 {
		gain = 0
	}
	return gain
}

// ---------------------------------------------------------------------------
// PointUpdate — from limits.c point_update()
// ---------------------------------------------------------------------------
// Main tick function called every ~30 seconds. Iterates all players,
// applies condition decay, and regenerates HMV.
func PointUpdate(world *World) {
	if world == nil {
		return
	}

	players := world.GetAllPlayers()

	for _, p := range players {
		if p == nil {
			continue
		}

		// Condition decay — only if not "inactive" (chat mode)
		// TODO: Check PRF_INACTIVE flag when preferences are implemented
		GainCondition(p, CondFull, -1)
		GainCondition(p, CondDrunk, -1)
		GainCondition(p, CondThirst, -1)

		// Regeneration only if position >= POS_STUNNED
		p.mu.RLock()
		pos := p.Position
		hp := p.Health
		maxHP := p.MaxHealth
		mana := p.Mana
		maxMana := p.MaxMana
		move := p.Move
		maxMove := p.MaxMove
		p.mu.RUnlock()

		if pos >= combat.PosStunned {
			// HP regen
			if hp < maxHP {
				gain := HitGain(p)
				newHP := hp + gain
				if newHP > maxHP {
					newHP = maxHP
				}
				p.mu.Lock()
				p.Health = newHP
				p.mu.Unlock()
			}

			// Mana regen
			if mana < maxMana {
				gain := ManaGain(p)
				newMana := mana + gain
				if newMana > maxMana {
					newMana = maxMana
				}
				p.mu.Lock()
				p.Mana = newMana
				p.mu.Unlock()
			}

			// Move regen (always regen move, even at max)
			gain := MoveGain(p)
			newMove := move + gain
			if newMove > maxMove {
				newMove = maxMove
			}
			p.mu.Lock()
			p.Move = newMove
			p.mu.Unlock()

			// TODO: Poison damage — Phase 3 (damage(i, i, 10, SPELL_POISON))
			// TODO: Cutthroat damage — Phase 3 (damage(i, i, 13, SKILL_CUTTHROAT))

			// Update position if HP dropped below thresholds
			p.mu.RLock()
			hp = p.Health
			p.mu.RUnlock()
			updatePosFromHP(p, hp)
		} else if pos == combat.PosIncap {
			// Incapacitated: 1 damage per tick
			p.TakeDamage(1)
			p.mu.RLock()
			hp := p.Health
			p.mu.RUnlock()
			updatePosFromHP(p, hp)
		} else if pos == combat.PosMortally {
			// Mortally wounded: 2 damage per tick
			p.TakeDamage(2)
			p.mu.RLock()
			hp := p.Health
			p.mu.RUnlock()
			updatePosFromHP(p, hp)
		}

		// Hunger/thirst damage when at 0
		p.mu.RLock()
		hunger := p.Hunger
		thirst := p.Thirst
		hp = p.Health
		p.mu.RUnlock()

		if hunger <= 0 || thirst <= 0 {
			if hp > 0 {
				p.TakeDamage(1)
				if hunger <= 0 {
					p.SendMessage("You are STARVING!\r\n")
				}
				if thirst <= 0 {
					p.SendMessage("You are DYING OF THIRST!\r\n")
				}
				p.mu.RLock()
				hp = p.Health
				p.mu.RUnlock()
				updatePosFromHP(p, hp)
			}
		}
	}
}

// updatePosFromHP updates player position based on HP, mirroring update_pos() from fight.c.
func updatePosFromHP(p *Player, hp int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if hp > 0 && p.Position > combat.PosStunned {
		return
	}
	if hp > 0 {
		p.Position = combat.PosStanding
		return
	}
	if hp <= -11 {
		p.Position = combat.PosDead
	} else if hp <= -6 {
		p.Position = combat.PosMortally
	} else if hp <= -3 {
		p.Position = combat.PosIncap
	} else {
		p.Position = combat.PosStunned
	}
}

// ---------------------------------------------------------------------------
// GetAllPlayers returns a snapshot of all online players.
// ---------------------------------------------------------------------------
func (w *World) GetAllPlayers() []*Player {
	w.mu.RLock()
	defer w.mu.RUnlock()

	players := make([]*Player, 0, len(w.players))
	for _, p := range w.players {
		players = append(players, p)
	}
	return players
}

// ---------------------------------------------------------------------------
// Eat / Drink helpers
// ---------------------------------------------------------------------------

// EatFood consumes a food item from inventory, applying condition changes.
// Ported from src/act.item.c ACMD(do_eat).
// Item must be ITEM_FOOD (type 19). Values[0] = hours of fullness.
// Returns the amount of fullness restored, or an error.
func EatFood(p *Player, item *ObjectInstance) (int, error) {
	if item == nil || item.Prototype == nil {
		return 0, fmt.Errorf("You can't eat that!")
	}
	if item.Prototype.TypeFlag != 19 { // ITEM_FOOD
		return 0, fmt.Errorf("You can't eat that!")
	}

	// Values[0] = hours of fullness restored
	amount := item.Prototype.Values[0]
	if amount <= 0 {
		amount = 1
	}

	GainCondition(p, CondFull, amount)

	p.mu.RLock()
	isFull := p.Hunger >= 20
	p.mu.RUnlock()

	_ = isFull // caller handles fullness messages

	return amount, nil
}

// DrinkLiquid consumes liquid from a drink container.
// Ported from src/act.item.c ACMD(do_drink).
// Returns the amount consumed and liquid index, or error.
func DrinkLiquid(p *Player, item *ObjectInstance) (amount int, liqIndex int, err error) {
	if item == nil || item.Prototype == nil {
		return 0, 0, fmt.Errorf("You can't drink from that!")
	}
	if item.Prototype.TypeFlag != 17 && item.Prototype.TypeFlag != 23 { // ITEM_DRINKCON or ITEM_FOUNTAIN
		return 0, 0, fmt.Errorf("You can't drink from that!")
	}

	// Values[1] = drinks left
	if item.Prototype.Values[1] <= 0 {
		return 0, 0, fmt.Errorf("It's empty.")
	}

	// Values[2] = liquid type
	liqIndex = item.Prototype.Values[2]

	// Calculate amount to drink
	drunkThirst := 0
	if liqIndex >= 0 && liqIndex < len(Liquids) {
		drunkThirst = Liquids[liqIndex].DrunkAffect
	}

	if drunkThirst > 0 {
		// Drink enough to fill thirst
		p.mu.RLock()
		curThirst := p.Thirst
		p.mu.RUnlock()
		amount = (25 - curThirst) / drunkThirst
		if amount < 1 {
			amount = 1
		}
	} else {
		amount = 4 // reasonable default
	}

	// Cap to available liquid
	if amount > item.Prototype.Values[1] {
		amount = item.Prototype.Values[1]
	}

	// Apply condition changes from drink_aff
	if liqIndex >= 0 && liqIndex < len(Liquids) {
		liq := Liquids[liqIndex]
		drunkVal := (liq.DrunkAffect * amount) / 4
		fullVal := (liq.FullAffect * amount) / 4
		thirstVal := (liq.ThirstAffect * amount) / 4

		GainCondition(p, CondDrunk, drunkVal)
		GainCondition(p, CondFull, fullVal)
		GainCondition(p, CondThirst, thirstVal)
	}

	return amount, liqIndex, nil
}

// FillContainer fills a drink container from a source (fountain or another container).
// Ported from src/act.item.c ACMD(do_pour) SCMD_FILL.
func FillContainer(toObj, fromObj *ObjectInstance) error {
	if toObj == nil || fromObj == nil || toObj.Prototype == nil || fromObj.Prototype == nil {
		return fmt.Errorf("You can't fill that!")
	}
	if toObj.Prototype.TypeFlag != 17 { // ITEM_DRINKCON
		return fmt.Errorf("You can't fill that!")
	}
	if fromObj.Prototype.TypeFlag != 23 { // ITEM_FOUNTAIN
		return fmt.Errorf("You can't fill something from that!")
	}

	// Check source has liquid
	if fromObj.Prototype.Values[1] <= 0 {
		return fmt.Errorf("The %s is empty.", fromObj.GetShortDesc())
	}

	fromLiq := fromObj.Prototype.Values[2]

	// Check destination doesn't have a different liquid
	if toObj.Prototype.Values[1] > 0 && toObj.Prototype.Values[2] != fromLiq {
		return fmt.Errorf("There is already another liquid in it!")
	}

	// Check destination has room
	if toObj.Prototype.Values[1] >= toObj.Prototype.Values[0] {
		return fmt.Errorf("There is no room for more.")
	}

	// Set liquid type on destination
	toObj.Prototype.Values[2] = fromLiq

	// Calculate amount to transfer
	space := toObj.Prototype.Values[0] - toObj.Prototype.Values[1]
	available := fromObj.Prototype.Values[1]

	transfer := space
	if transfer > available {
		transfer = available
	}

	toObj.Prototype.Values[1] += transfer
	fromObj.Prototype.Values[1] -= transfer

	// If source emptied, reset
	if fromObj.Prototype.Values[1] <= 0 {
		fromObj.Prototype.Values[1] = 0
		fromObj.Prototype.Values[2] = 0
		fromObj.Prototype.Values[3] = 0
	}

	// Transfer poison flag
	toObj.Prototype.Values[3] = boolToInt(toObj.Prototype.Values[3] == 1 || fromObj.Prototype.Values[3] == 1)

	return nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
