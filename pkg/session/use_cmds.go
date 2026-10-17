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
	if item.Prototype == nil || len(item.Prototype.Values) < 2 {
		s.Send("Nothing magical happens.")
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
		// C generic_find resolves a room character first, then carried/room objects.
		if tgt, found := s.manager.world.ResolveCharInRoom(s.player, targetName); found {
			switch {
			case tgt.Player != nil:
				target = tgt.Player
			case tgt.Mob != nil:
				target = tgt.Mob
			}
		} else if obj, found := s.manager.world.ResolveObjectInInventory(s.player, targetName); found {
			target = nil
			objectTarget = obj
		} else if obj, found := s.manager.world.ResolveObjectInRoom(s.player, targetName); found {
			target = nil
			objectTarget = obj
		} else if obj, found := s.manager.world.ResolveObjectInEquipment(s.player, targetName); found {
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

// objectivePronoun returns the objective pronoun for a player sex value.
// Go encoding: 0=male, 1=female, 2=neutral (differs from C SEX_* in structs.h).
func objectivePronoun(sex int) string {
	switch sex {
	case 1: // female
		return "her"
	case 2: // neutral
		return "it"
	default: // male
		return "him"
	}
}

// cmdZap implements the zap command for using wands and staves.
// zap <target>
// Source: src/act.other.c (do_use SCMD_ZAP) + src/spell_parser.c (mag_objectmagic ITEM_WAND)
func cmdZap(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Zap who?")
		return nil
	}

	targetName := strings.Join(args, " ")

	// Check if player has a wand or staff held or wielded
	var item *game.ObjectInstance
	var found bool

	// Check hold slot first, then wield slot
	for _, slot := range []game.EquipmentSlot{game.SlotHold, game.SlotWield} {
		equippedItem, exists := s.player.Equipment.GetItemInSlot(slot)
		if exists {
			item = equippedItem
			found = true
			break
		}
	}

	if !found || item == nil {
		s.Send("You aren't holding that.")
		return nil
	}

	// Check it's a wand or staff.
	typeFlag := item.GetTypeFlag()
	if typeFlag != game.ITEM_WAND && typeFlag != game.ITEM_STAFF {
		s.Send("You can't zap with that!")
		return nil
	}

	if item.Prototype == nil || len(item.Prototype.Values) < 3 {
		s.Send("It seems to be empty.")
		return nil
	}

	// Check charges — Values[2] = current charges. Use GetValue so an
	// instance-level override is respected and never mutate the shared
	// prototype; decrement via SetValue to keep the change instance-local.
	charges := item.GetValue(2)
	if charges <= 0 {
		s.Send("It seems powerless.")
		broadcastToRoom(s, "Nothing seems to happen.")
		return nil
	}

	// Decrement charges instance-safely (DP-1110)
	item.SetValue(2, charges-1)

	// Extract spell data. CircleMUD wand/staff layout:
	// Values[0] = level, Values[3] = spell number.
	spellLevel := item.GetValue(0)
	spellNum := item.GetValue(3)

	if spellNum <= 0 {
		s.Send("Nothing happens.")
		return nil
	}

	// Resolve target via the canonical in-room resolver (DP-907) so `use/zap
	// <item> X` agrees with consider/kick/... on what "X" is.
	var target interface{}
	if tgt, found := s.manager.world.ResolveCharInRoom(s.player, targetName); found {
		switch {
		case tgt.Player != nil:
			target = tgt.Player
		case tgt.Mob != nil:
			target = tgt.Mob
		}
	}
	if target == nil {
		s.Send("They aren't here.")
		return nil
	}

	// Player-facing messages mirror C's mag_objectmagic (src/spell_parser.c).
	// Unlike C's act(), broadcastToRoom does NOT perform $-substitution, so we
	// pre-substitute the actor name and pronouns here (see eat_cmds.go).
	actorName := s.player.Name
	actorPronoun := objectivePronoun(s.player.Sex)
	if typeFlag == game.ITEM_WAND {
		if target == s.player {
			s.Send(fmt.Sprintf("Your %s bathes you in a blinding glow!", item.GetShortDesc()))
			broadcastToRoom(s, fmt.Sprintf("%s's %s bathes %s in a blinding glow!", actorName, item.GetShortDesc(), actorPronoun))
		} else {
			targetNameDisp := targetName
			if p, ok := target.(*game.Player); ok {
				targetNameDisp = p.Name
			} else if m, ok := target.(*game.MobInstance); ok {
				targetNameDisp = m.Proto().ShortDesc
			}
			s.Send(fmt.Sprintf("Your %s flares up with a blinding glow that surges toward %s!", item.GetShortDesc(), targetNameDisp))
			broadcastToRoom(s, fmt.Sprintf("%s's %s flares up with a blinding glow that surges toward %s!", actorName, item.GetShortDesc(), targetNameDisp))
		}
		spellArgument, _ := game.OneArgument(targetName)
		spells.CallMagicWithArgument(s.player, target, nil, spellNum, spellLevel, spells.CastWand, s.manager.world, spellArgument)
	} else { // ITEM_STAFF
		s.Send(fmt.Sprintf("Your %s radiates an ethereal glow that lights the room.", item.GetShortDesc()))
		broadcastToRoom(s, fmt.Sprintf("%s's %s sparks blindingly, bathing you in its glow.", actorName, item.GetShortDesc()))

		room := s.player.GetRoomVNum()
		for _, p := range s.manager.world.GetPlayersInRoom(room) {
			if p != nil && p != s.player {
				spells.Cast(s.player, p, spellNum, spellLevel, s.manager.world)
			}
		}
		for _, m := range s.manager.world.GetMobsInRoom(room) {
			if m != nil {
				spells.Cast(s.player, m, spellNum, spellLevel, s.manager.world)
			}
		}
	}

	s.markDirty(VarInventory)

	return nil
}

func init() {
	registerCommand("recite", wrapArgs(cmdRecite), "Read a scroll.")
}
