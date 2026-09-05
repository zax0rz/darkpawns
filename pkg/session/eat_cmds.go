package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/spells"
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

	// C do_use resolves the target from WEAR_HOLD first (act.other.c:897-910):
	// a held item whose keyword list matches the argument wins over anything
	// carried; only then does the carrying-list lookup run.
	arg := args[0]
	item := s.manager.world.HeldItemVis(s.player, arg)
	fromHold := item != nil
	if item == nil {
		// C do_use parses with half_chop (first token) and resolves via
		// get_obj_in_list_vis (keyword prefix, carrying) — not the short desc.
		item = s.manager.world.FindCarriedVis(s.player, arg)
	}
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

	// C mag_objectmagic stalls the drinker for one combat round before the
	// spells resolve (spell_parser.c:710).
	s.player.SetWaitState(1) // C: WAIT_STATE(ch, PULSE_VIOLENCE)

	// Fire the potion's spells. C mag_objectmagic ITEM_POTION casts each of
	// GET_OBJ_VAL(obj, 1..3) via call_magic(ch, ch, NULL, val, GET_OBJ_VAL(obj, 0),
	// CAST_POTION), breaking as soon as one returns 0 (spell_parser.c:711-715).
	// call_magic returns false for a zero/negative or unknown spellnum, so the
	// break also handles the -1/0 sentinel spell slots on single-spell potions.
	level := item.GetValue(0)
	for i := 1; i < 4; i++ {
		if !spells.CallMagic(s.player, s.player, nil, item.GetValue(i), level, spells.CastPotion, s.manager.world) {
			break
		}
	}

	// Remove the potion (C extract_obj after the cast loop — a held item
	// leaves the equipment slot on the way out).
	if fromHold {
		s.player.Equipment.UnequipItem(item, s.player.Inventory)
		s.markDirty(VarEquipment)
	}
	s.player.Inventory.RemoveItem(item)
	s.markDirty(VarInventory)

	return nil
}
