//nolint:unused // Game logic port — not yet wired to command registry.
package game

import (
	"fmt"
		"github.com/zax0rz/darkpawns/pkg/combat"
)

// findRemortClass — ported from C find_remort_class()
// Maps the player's current class to a circle of remort classes.
// Returns a slice of class ints to choose from.
func findRemortClass(ch *Player) int {
	var chosen int
	switch ch.GetClass() {
	case combat.ClassCleric:
		chosen = number(1, 4)
	case combat.ClassThief:
		chosen = number(1, 4) + 4
	case combat.ClassWarrior:
		chosen = number(1, 4) + 8
	case combat.ClassMagus:
		chosen = number(1, 2)
		if chosen == 1 {
			chosen = number(1, 4) + 0 // start at 0 (Mage)
		} else {
			chosen = number(1, 4) + 4 // start at 4 (Magus)
		}
	case combat.ClassAvatar:
		chosen = number(1, 2)
		if chosen == 1 {
			chosen = number(1, 4) + 0 // Mage side
		} else {
			chosen = number(1, 4) + 8 // Warrior side
		}
	case combat.ClassAssassin:
		chosen = number(1, 2)
		if chosen == 1 {
			chosen = number(1, 4) + 4 // Thief side
		} else {
			chosen = number(1, 4) + 8 // Warrior side
		}
	case combat.ClassPaladin:
		chosen = number(1, 3)
		switch chosen {
		case 1:
			chosen = number(1, 4) + 0 // Mage
		case 2:
			chosen = number(1, 4) + 4 // Thief
		default:
			chosen = number(1, 4) + 8 // Warrior
		}
	case combat.ClassNinja:
		chosen = number(1, 2)
		switch chosen {
		case 1:
			chosen = number(1, 4) + 0 // Mage
		default:
			chosen = number(1, 4) + 4 // Thief
		}
	case combat.ClassPsionic:
		chosen = number(1, 3)
		switch chosen {
		case 1:
			chosen = number(1, 4) + 0 // Mage
		case 2:
			chosen = number(1, 4) + 4 // Thief
		default:
			chosen = number(1, 4) + 8 // Warrior
		}
	case combat.ClassRanger:
		chosen = number(1, 2)
		switch chosen {
		case 1:
			chosen = number(1, 4) + 4 // Thief
		default:
			chosen = number(1, 4) + 8 // Warrior
		}
	case combat.ClassMystic:
		chosen = number(1, 2)
		if chosen == 1 {
			chosen = number(1, 4) + 0 // Mage
		} else {
			chosen = number(1, 4) + 8 // Warrior
		}
	default: // Mage
		chosen = number(1, 4) + 0
	}
	return chosen
}

// doFirstRemortAdjust — ported from C do_first_remort_adjust()
// Sets the player's experience based on new level.
func doFirstRemortAdjust(w *World, ch *Player) {
	ch.Level -= 10
	if ch.Level < 1 {
		ch.Level = 1
	}
	// Advance level recalculates stats
	advanceLevel(ch, ch.Level)
}

// doSecondRemortAdjust — ported from C do_second_remort_adjust()
// Handles remort level/skill/exp adjustments on second+ remort.
func doSecondRemortAdjust(w *World, ch *Player) {
	ch.Level -= 10
	if ch.Level < 1 {
		ch.Level = 1
	}
	// In C: amount_needed = 90 * level * level * level / played / 2
	// We just reset exp to 0 and advance
	setExp(ch, 0)
	advanceLevel(ch, ch.Level)
}

// advanceLevel — ported from C advance_level()
// Advances player to new level, setting HP/Mana/Move and spells.
func advanceLevel(ch *Player, level int) {
	// Class-based HP per level
	var hpGain int
	switch ch.GetClass() {
	case combat.ClassMage, combat.ClassMagus, combat.ClassPsionic:
		hpGain = 6
	case combat.ClassThief, combat.ClassNinja, combat.ClassAssassin, combat.ClassMystic:
		hpGain = 8
	case combat.ClassCleric, combat.ClassAvatar:
		hpGain = 9
	case combat.ClassWarrior, combat.ClassPaladin, combat.ClassRanger:
		hpGain = 10
	default:
		hpGain = 8
	}
	// Mana per level
	var manaGain int
	switch ch.GetClass() {
	case combat.ClassMage, combat.ClassMagus:
		manaGain = 10
	case combat.ClassPsionic:
		manaGain = 9
	case combat.ClassCleric, combat.ClassAvatar, combat.ClassMystic:
		manaGain = 8
	case combat.ClassThief, combat.ClassNinja, combat.ClassAssassin:
		manaGain = 7
	case combat.ClassWarrior, combat.ClassPaladin, combat.ClassRanger:
		manaGain = 5
	default:
		manaGain = 7
	}
	// Move per level
	moveGain := 2

	for lvl := 1; lvl <= level; lvl++ {
		ch.MaxHealth += hpGain
		ch.MaxMana += manaGain
		ch.MaxMove += moveGain

		// Bonus HP for remort
		ch.MaxHealth += 2

		// Set current to max
		if lvl == level {
			ch.Health = ch.MaxHealth
			ch.Mana = ch.MaxMana
			ch.Move = ch.MaxMove
		}
	}
}

// setExp sets the player's experience points.
func setExp(ch *Player, exp int) {
	ch.Exp = exp
}

// number is an alias for randRange for C compatibility.
func number(from, to int) int {
	return randRange(from, to)
}

// IsOwner — ported from C is_owner()
// Checks if ch is the owner or a guest of the house at room_vnum.
func (w *World) IsOwner(ch *Player, roomVNum int) bool {
	if ch.IsNPC() {
		return false
	}
	i := findHouse(w.HouseControl, roomVNum)
	if i < 0 {
		return false
	}
	h := w.HouseControl[i]
	if int64(ch.GetID()) == h.Owner {
		return true
	}
	for j := 0; j < h.NumOfGuests; j++ {
		if int64(ch.GetID()) == h.Guests[j] {
			return true
		}
	}
	return false
}

// kenderSteal — ported from C kender_steal()
// Kender NPC attempts to steal gold from a player.
func (w *World) kenderSteal(ch *Player, victim *Player) bool {
	// In C this checks victim.isPlayer; in Go we check !IsNPC
	if victim.IsNPC() {
		return false
	}
	if victim.GetLevel() >= LVL_IMMORT {
		return false
	}
	if victim.GetPosition() <= combat.PosSleeping {
		return false
	}
	if victim.GetPosition() == combat.PosDead {
		return false
	}

	stealSkill := 100
	dexBonus := 10
	chance := stealSkill/2 + dexBonus + 10

	if number(1, 100) < chance {
		// Success! Take some gold
		amt := number(10, 25)
		if victim.GetGold() <= 2*amt {
			amt = victim.GetGold() / 5
		}
		if amt < 0 {
			amt = 0
		}
		ch.SetGold(ch.GetGold() + amt)
		victim.SetGold(victim.GetGold() - amt)

		stealRoll := number(1, 100)
		if stealRoll < 10 {
			// Clean getaway
			return true
		} else if stealRoll < 35 {
			// Caught after stealing
			victim.SendMessage(fmt.Sprintf("You catch %s's hand coming out of your coin purse.\r\n", ch.GetName()))
			w.roomMessage(victim.GetRoom(), fmt.Sprintf("%s catches %s's hand leaving their coin purse a little lighter.", victim.GetName(), ch.GetName()))
			return true
		} else if stealRoll < 75 {
			// Caught before stealing
			victim.SendMessage(fmt.Sprintf("You catch %s's hand going into your coin purse.\r\n", ch.GetName()))
			w.roomMessage(victim.GetRoom(), fmt.Sprintf("%s catches %s's hand entering their coin purse.", victim.GetName(), ch.GetName()))
			return true
		} else {
			// Got nothing, didn't get caught
			return true
		}
	}
	return false
}
