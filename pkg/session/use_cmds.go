// Package session provides command handlers and WebSocket-based player sessions.
package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/spells"
)

// cmdRecite implements the recite command for reading scrolls.
// recite <item> [target]
// Source: src/act.other.c (do_use SCMD_RECITE) + src/spell_parser.c (mag_objectmagic ITEM_SCROLL)
func cmdRecite(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("What do you want to recite?")
		return nil
	}

	fullInput := strings.Join(args, " ")

	// C do_use (SCMD_RECITE) parses with half_chop: arg = first token (the
	// scroll), buf = the rest (the target). Resolve via get_obj_in_list_vis
	// (keyword prefix, carrying), not the short description.
	var itemName, targetName string
	parts := strings.SplitN(fullInput, " ", 2)
	itemName = parts[0]
	if len(parts) > 1 {
		targetName = strings.TrimSpace(parts[1])
	}

	// C do_use (SCMD_RECITE) resolves WEAR_HOLD first (act.other.c:897-910),
	// then the carrying list.
	item := s.manager.world.HeldItemVis(s.player, itemName)
	fromHold := item != nil
	if item == nil {
		item = s.manager.world.FindCarriedVis(s.player, itemName)
	}
	if item == nil {
		s.Send(fmt.Sprintf("You don't seem to have %s %s.", articleFor(itemName), itemName))
		return nil
	}

	// C do_use accepts only ITEM_SCROLL here (act.other.c:961-963); the
	// refusal is sent with no terminator (act.other.c:959), so it rides the
	// literal path (PF-063 cluster 4).
	typeFlag := item.GetTypeFlag()
	if typeFlag != game.ITEM_SCROLL {
		s.sendLiteralText("You can only recite scrolls.")
		return nil
	}

	if item.VNum == 8052 && s.player.IsFighting() {
		// C spell_parser.c:661-664 refuses the recall scroll before the
		// sitting gate and before the scroll is consumed.
		s.Send("You can't concentrate enough!.")
		return nil
	}
	if s.player.GetPosition() == game.PosSitting {
		// C spell_parser.c:666-669 leaves the scroll untouched when sitting.
		s.Send("You can't do this sitting!!")
		return nil
	}

	// C uses object-instance values 0..3 for level and up to three spells.
	spellLevel := item.GetValue(0)
	spellNumbers := []int{item.GetValue(1), item.GetValue(2), item.GetValue(3)}
	for len(spellNumbers) > 1 && spellNumbers[len(spellNumbers)-1] <= 0 {
		spellNumbers = spellNumbers[:len(spellNumbers)-1]
	}

	// Determine target — default to self.
	var target interface{} = s.player
	var objectTarget interface{}

	if targetName != "" {
		// C generic_find (src/spell_parser.c:566) resolves a room character first,
		// then equipment, inventory, and room in that fixed order
		// (src/handler.c:1537-1556; DP-1518). The shared GenericFindObject owns
		// the object order so recite agrees with use (R5c).
		if tgt, found := s.manager.world.ResolveCharInRoom(s.player, targetName); found {
			switch {
			case tgt.Player != nil:
				target = tgt.Player
			case tgt.Mob != nil:
				target = tgt.Mob
			}
		} else if obj, _ := s.manager.world.GenericFindObject(s.player, targetName); obj != nil {
			target = nil
			objectTarget = obj
		} else {
			game.Act(nil, false, s.player, nil, item, nil,
				"There is nothing to here to affect with $p.", "", game.ToChar)
			return nil
		}
	}

	// Match C's mag_objectmagic messages. Act owns object substitution and
	// recipient-specific visibility for both the actor and room audiences.
	game.Act(nil, true, s.player, nil, item, nil,
		"You recite $p which dissolves.", "", game.ToChar)
	if item.Prototype != nil && item.Prototype.ActionDesc != "" {
		game.Act(s.manager.world, false, s.player, nil, item, nil,
			item.Prototype.ActionDesc, "", game.ToRoom)
	} else {
		game.Act(s.manager.world, false, s.player, nil, item, nil,
			"$n recites $p.", "", game.ToRoom)
	}

	// C mag_objectmagic stalls the reader for one combat round before the
	// spells resolve (spell_parser.c:683).
	s.player.SetWaitState(1) // C: WAIT_STATE(ch, PULSE_VIOLENCE)

	// Remove scroll (C extract_obj after the cast loop; a held item leaves
	// the equipment slot on the way out).
	if fromHold {
		s.player.Equipment.UnequipItem(item, s.player.Inventory)
		s.markDirty(VarEquipment)
	}
	s.player.Inventory.RemoveItem(item)
	// C's mag_objectmagic consumes the item with extract_obj
	// (spell_parser.c:697): a carried item leaves through obj_from_char and
	// sets PLR_CRASH (handler.c:596-598, 1016-1017); a held one leaves
	// through unequip_char, which does not.
	if !fromHold {
		s.player.MarkCrashNeeded()
	}
	s.markDirty(VarInventory)

	// Cast each spell on the scroll
	spellArgument, _ := game.OneArgument(targetName)
	for _, spellNum := range spellNumbers {
		if spellNum <= 0 {
			break
		}
		if !spells.CallMagicWithArgument(s.player, target, objectTarget, spellNum, spellLevel, spells.CastScroll, s.manager.world, spellArgument) {
			break
		}
	}

	return nil
}

func init() {
	registerCommand("recite", wrapArgs(cmdRecite), "Read a scroll.")
	registerActorCommand("recite", func(ctx game.CharacterCommandContext, commandName string, args []string) game.CharacterCommandResult {
		ctx = withCharacterCommandArgs(ctx, args)
		return game.CharacterItemMagic(ctx, commandName)
	})
}
