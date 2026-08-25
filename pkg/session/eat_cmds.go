package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// cmdEat implements the eat command.
// Source: src/act.item.c ACMD(do_eat)
func cmdEat(s *Session, args []string) error {
	s.manager.world.ExecEat(s.player, strings.Join(args, " "))
	return nil
}

// cmdTaste implements the taste command — src/act.item.c ACMD(do_eat) SCMD_TASTE.
func cmdTaste(s *Session, args []string) error {
	s.manager.world.ExecTaste(s.player, strings.Join(args, " "))
	return nil
}

// cmdDrink implements the drink command.
// Source: src/act.item.c ACMD(do_drink)
func cmdDrink(s *Session, args []string) error {
	s.manager.world.ExecDrink(s.player, strings.Join(args, " "))
	return nil
}

// cmdSip implements the sip command — src/act.item.c ACMD(do_drink) SCMD_SIP.
func cmdSip(s *Session, args []string) error {
	s.manager.world.ExecSip(s.player, strings.Join(args, " "))
	return nil
}

// cmdQuaff implements the quaff command.
// Source: src/act.other.c (do_use SCMD_QUAFF) + src/spell_parser.c (mag_objectmagic ITEM_POTION)
func cmdQuaff(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("What do you want to quaff?")
		return nil
	}

	// C do_use parses with half_chop (first token) and resolves via
	// get_obj_in_list_vis (keyword prefix, carrying) — not the short desc.
	arg := args[0]
	item := s.manager.world.FindCarriedVis(s.player, arg)
	if item == nil {
		s.Send(fmt.Sprintf("You don't seem to have %s %s.", articleFor(arg), arg))
		return nil
	}

	// Check it's a potion
	if item.GetTypeFlag() != 10 { // ITEM_POTION
		s.Send("You can only quaff potions.")
		return nil
	}

	// Can't quaff while sitting (C: spell_parser.c line check)
	if s.player.GetPosition() == combat.PosSitting {
		s.Send("You can't do this sitting!")
		return nil
	}

	// Quaff message
	s.Send(fmt.Sprintf("You quaff %s.", item.GetShortDesc()))
	if item.Prototype != nil && item.Prototype.ActionDesc != "" {
		broadcastToRoom(s, fmt.Sprintf("%s %s", s.player.Name, item.Prototype.ActionDesc))
	} else {
		broadcastToRoom(s, fmt.Sprintf("%s quaffs %s.", s.player.Name, item.GetShortDesc()))
	}

	// Apply potion effects
	// In C: for i=1; i<4; i++ call_magic(ch, ch, NULL, GET_OBJ_VAL(obj, i), GET_OBJ_VAL(obj, 0), CAST_POTION)
	// Values[0] = spell level, Values[1-3] = spell numbers
	// For now, apply object affects as stat modifiers
	if item.Prototype != nil {
		for _, aff := range item.Prototype.Affects {
			applyAffect(s.player, aff.Location, aff.Modifier, item.GetShortDesc())
		}
	}

	// Remove potion from inventory
	s.player.Inventory.RemoveItem(item)
	s.markDirty(VarInventory)

	return nil
}

// applyAffect applies a stat/HP/mana/move modifier from a potion affect.
// Location values from CircleMUD structs.h APPLY_* constants.
func applyAffect(p *game.Player, location, modifier int, source string) {
	switch location {
	case 1: // APPLY_STR
		p.Stats.Str += modifier
	case 2: // APPLY_DEX
		p.Stats.Dex += modifier
	case 3: // APPLY_INT
		p.Stats.Int += modifier
	case 4: // APPLY_WIS
		p.Stats.Wis += modifier
	case 5: // APPLY_CON
		p.Stats.Con += modifier
	case 12: // APPLY_HIT (HP)
		p.Health += modifier
		if p.Health > p.MaxHealth {
			p.Health = p.MaxHealth
		}
		if p.Health < 0 {
			p.Health = 0
		}
	case 13: // APPLY_MANA
		p.Mana += modifier
		if p.Mana > p.MaxMana {
			p.Mana = p.MaxMana
		}
		if p.Mana < 0 {
			p.Mana = 0
		}
	case 14: // APPLY_MOVE
		p.Move += modifier
		if p.Move > p.MaxMove {
			p.Move = p.MaxMove
		}
		if p.Move < 0 {
			p.Move = 0
		}
	case 17: // APPLY_HITROLL
		p.Hitroll += modifier
	case 18: // APPLY_DAMROLL
		p.Damroll += modifier
	}
}
