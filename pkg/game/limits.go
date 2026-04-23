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

// EatFood attempts to eat an item. Returns true if successful.
// Item must be ITEM_FOOD (type 19). Value[0] = hours of fullness.
func EatFood(p *Player, item *ObjectInstance) error {
	if item == nil || item.Prototype == nil {
		return fmt.Errorf("You can't eat that!")
	}
	if item.Prototype.TypeFlag != 19 { // ITEM_FOOD
		return fmt.Errorf("You can't eat that!")
	}

	// Value[0] = hours of fullness (how much hunger it restores)
	restore := item.Prototype.Values[0]
	if restore <= 0 {
		restore = 1
	}

	p.mu.Lock()
	p.Hunger += restore
	if p.Hunger > 24 {
		p.Hunger = 24
	}
	p.mu.Unlock()

	p.SendMessage(fmt.Sprintf("You eat %s.\r\n", item.GetShortDesc()))
	return nil
}

// DrinkLiquid attempts to drink from an item. Returns true if successful.
// Item must be ITEM_DRINKCON (type 17). Value[0] = liquid total, Value[1] = liquid left, Value[2] = liquid type.
func DrinkLiquid(p *Player, item *ObjectInstance) error {
	if item == nil || item.Prototype == nil {
		return fmt.Errorf("You can't drink from that!")
	}
	if item.Prototype.TypeFlag != 17 { // ITEM_DRINKCON
		return fmt.Errorf("You can't drink from that!")
	}

	// Value[1] = drinks left
	drinksLeft := item.Prototype.Values[1]
	if drinksLeft <= 0 {
		return fmt.Errorf("It's empty.")
	}

	// Each drink restores some thirst
	p.mu.Lock()
	p.Thirst += 4 // ~4 drinks to go from 0 to full
	if p.Thirst > 24 {
		p.Thirst = 24
	}
	// TODO: Decrement drinks left on the object instance when mutable state is added
	p.mu.Unlock()

	p.SendMessage(fmt.Sprintf("You drink %s.\r\n", item.GetShortDesc()))
	return nil
}
