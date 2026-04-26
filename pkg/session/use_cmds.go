// Package session provides command handlers and WebSocket-based player sessions.
package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/engine"
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

	// Determine target
	var target interface{} = s.player // default: self

	if targetName != "" {
		// Try to find target — check players first, then mobs
		room, ok := s.manager.world.GetRoom(s.player.GetRoom())
		if ok {
			// Check players in room
			players := s.manager.world.GetPlayersInRoom(room.VNum)
			for _, p := range players {
				if strings.Contains(strings.ToLower(p.Name), strings.ToLower(targetName)) {
					target = p
					break
				}
			}

			// Check mobs if no player found
			if target == s.player {
				mobs := s.manager.world.GetMobsInRoom(room.VNum)
				for _, mob := range mobs {
					if strings.Contains(strings.ToLower(mob.GetShortDesc()), strings.ToLower(targetName)) ||
						strings.Contains(strings.ToLower(mob.GetName()), strings.ToLower(targetName)) {
						target = mob
						break
					}
				}
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
	am := engine.NewAffectManager()
	for _, spellNum := range spellNumbers {
		if spellNum <= 0 {
			continue
		}
		spells.Cast(s.player, target, spellNum, spellLevel, nil, am)
	}

	return nil
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

	// Check it's a wand or staff
	typeFlag := item.GetTypeFlag()
	if typeFlag != 4 && typeFlag != 5 { // ITEM_STAFF = 4, ITEM_WAND = 5 in the codebase
		s.Send("You can't zap with that!")
		return nil
	}

	if item.Prototype == nil || len(item.Prototype.Values) < 3 {
		s.Send("It seems to be empty.")
		return nil
	}

	// Check charges — Values[2] = current charges
	charges := item.Prototype.Values[2]
	if charges <= 0 {
		s.Send("It has no charges left.")
		return nil
	}

	// Decrement charges
	item.Prototype.Values[2]--

	// Extract spell data
	// Values[0] = spell level, Values[1] = spell number
	spellLevel := item.Prototype.Values[0]
	spellNum := item.Prototype.Values[1]

	if spellNum <= 0 {
		s.Send("Nothing happens.")
		return nil
	}

	// Find target in room
	var target interface{}
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		s.Send("You are in a strange void.")
		return nil
	}

	// Check players in room
	players := s.manager.world.GetPlayersInRoom(room.VNum)
	for _, p := range players {
		if strings.Contains(strings.ToLower(p.Name), strings.ToLower(targetName)) {
			target = p
			break
		}
	}

	// Check mobs if no player found
	if target == nil {
		mobs := s.manager.world.GetMobsInRoom(room.VNum)
		for _, mob := range mobs {
			if strings.Contains(strings.ToLower(mob.GetShortDesc()), strings.ToLower(targetName)) ||
				strings.Contains(strings.ToLower(mob.GetName()), strings.ToLower(targetName)) {
				target = mob
				break
			}
		}
	}

	if target == nil {
		s.Send("They aren't here.")
		return nil
	}

	// Room broadcast for zap effect
	broadcastToRoom(s, fmt.Sprintf("$n blasts %s with %s.", targetName, item.GetShortDesc()))

	// Player message
	s.Send(fmt.Sprintf("You blast %s with %s.", targetName, item.GetShortDesc()))

	// Cast the spell
	am := engine.NewAffectManager()
	spells.Cast(s.player, target, spellNum, spellLevel, nil, am)

	s.markDirty(VarInventory)

	return nil
}

func init() {
	cmdRegistry.Register("recite", wrapArgs(cmdRecite), "Read a scroll.", 0, 0)
	cmdRegistry.Register("zap", wrapArgs(cmdZap), "Zap with a wand or staff.", 0, 0)
}
