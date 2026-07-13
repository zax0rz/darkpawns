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
		s.Send("Recite what?")
		return nil
	}

	fullInput := strings.Join(args, " ")

	// Parse item name and optional target
	var itemName, targetName string
	parts := strings.SplitN(fullInput, " ", 2)
	itemName = parts[0]
	if len(parts) > 1 {
		targetName = strings.TrimSpace(parts[1])
	}

	// Find scroll in inventory
	item, found := s.player.Inventory.FindItem(itemName)
	if !found {
		s.Send("You don't have that item.")
		return nil
	}

	// Check it's a scroll — flexible check:
	// CircleMUD ITEM_SCROLL = 12, but also accept 2 or 11 as fallback
	// Also accept any item that has spell values in Values[0]
	typeFlag := item.GetTypeFlag()
	if typeFlag != 2 && typeFlag != 11 && typeFlag != 12 {
		// Still allow if it has spell values that look valid
		if item.Prototype == nil || item.Prototype.Values[0] <= 0 || len(item.Prototype.Values) < 2 {
			s.Send("You can't recite that.")
			return nil
		}
	}

	if item.Prototype == nil || len(item.Prototype.Values) < 2 {
		s.Send("Nothing magical happens.")
		return nil
	}

	// Extract spell data from prototype values
	// Values[0] = spell level, Values[1]/[2]/[3] = spell numbers (1-3 spells on scroll)
	spellLevel := item.Prototype.Values[0]
	spellNumbers := []int{item.Prototype.Values[1]}
	if len(item.Prototype.Values) >= 3 && item.Prototype.Values[2] > 0 {
		spellNumbers = append(spellNumbers, item.Prototype.Values[2])
	}
	if len(item.Prototype.Values) >= 4 && item.Prototype.Values[3] > 0 {
		spellNumbers = append(spellNumbers, item.Prototype.Values[3])
	}

	// Determine target — default to self.
	var target interface{} = s.player

	if targetName != "" {
		// Resolve via the canonical in-room resolver (DP-907) so `recite scroll
		// X` agrees with consider/kick/... on what "X" is.
		if tgt, found := s.manager.world.ResolveCharInRoom(s.player, targetName); found {
			switch {
			case tgt.Player != nil:
				target = tgt.Player
			case tgt.Mob != nil:
				target = tgt.Mob
			}
		}
	}

	// Room message
	if target != s.player {
		// Pointing at someone — notify target if it's a player
		broadcastToRoom(s, fmt.Sprintf("$n reads %s and points at you.", item.GetShortDesc()))
	}
	broadcastToRoom(s, fmt.Sprintf("$n reads %s.", item.GetShortDesc()))

	// Player message
	s.Send(fmt.Sprintf("You read %s.", item.GetShortDesc()))

	// Remove scroll from inventory
	s.player.Inventory.RemoveItem(item)
	s.markDirty(VarInventory)

	// Cast each spell on the scroll
	for _, spellNum := range spellNumbers {
		if spellNum <= 0 {
			continue
		}
		spells.Cast(s.player, target, spellNum, spellLevel, s.manager.world)
	}

	return nil
}

// objectivePronoun returns the objective pronoun for a player sex value.
// Source: C SEX_* constants — 0=male, 1=female, 2=neutral.
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
				targetNameDisp = m.Prototype.ShortDesc
			}
			s.Send(fmt.Sprintf("Your %s flares up with a blinding glow that surges toward %s!", item.GetShortDesc(), targetNameDisp))
			broadcastToRoom(s, fmt.Sprintf("%s's %s flares up with a blinding glow that surges toward %s!", actorName, item.GetShortDesc(), targetNameDisp))
		}
		spells.Cast(s.player, target, spellNum, spellLevel, s.manager.world)
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
	cmdRegistry.Register("recite", wrapArgs(cmdRecite), "Read a scroll.", 0, 0)
	cmdRegistry.Register("zap", wrapArgs(cmdZap), "Zap with a wand or staff.", 0, 0)
}
