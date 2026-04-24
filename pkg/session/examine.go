package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// itemCheck returns true if the player can reasonably reach or get an item
// from the room (weight capacity permitting, etc.). Simple version — just
// checks that the item exists in the room and the player isn't dead/incap.
func itemCheck(s *Session, item *game.ObjectInstance) bool {
	if s.player.Position < combat.PosResting {
		return false
	}
	// If the player has a weight-tracking inventory, check capacity here.
	// For now — if it's in the room and they're conscious, it's reachable.
	return true
}

// cmdExamine implements the examine command from act.informative.c.
// Shows detailed information about a target (item, mob, or player) in the room.
func cmdExamine(s *Session, args []string) error {
	if len(args) == 0 {
		s.Send("Examine what?")
		return nil
	}

	targetName := strings.ToLower(strings.Join(args, " "))
	roomVNum := s.player.RoomVNum

	// 1. Check items in the room
	items := s.manager.world.GetItemsInRoom(roomVNum)
	for _, item := range items {
		if !matchesTarget(item, targetName) {
			continue
		}
		examineItem(s, item)
		return nil
	}

	// Also check items in player's inventory
	if invItem, found := s.player.Inventory.FindItem(targetName); found {
		examineItem(s, invItem)
		return nil
	}

	// 2. Check mobs in the room
	mobs := s.manager.world.GetMobsInRoom(roomVNum)
	for _, mob := range mobs {
		if !strings.Contains(strings.ToLower(mob.GetShortDesc()), targetName) &&
			!strings.Contains(strings.ToLower(mob.GetName()), targetName) {
			continue
		}
		examineMob(s, mob)
		return nil
	}

	// 3. Check players in the room
	players := s.manager.world.GetPlayersInRoom(roomVNum)
	for _, p := range players {
		if strings.ToLower(p.Name) != targetName {
			continue
		}
		examinePlayer(s, p)
		return nil
	}

	s.Send("You don't see that here.")
	return nil
}

// matchesTarget checks if an item's keywords or short description match the target name.
func matchesTarget(item *game.ObjectInstance, targetName string) bool {
	kw := strings.ToLower(item.GetKeywords())
	if strings.Contains(kw, targetName) {
		return true
	}
	sd := strings.ToLower(item.GetShortDesc())
	return strings.Contains(sd, targetName)
}

// examineItem prints detailed information about a room/inventory item.
func examineItem(s *Session, item *game.ObjectInstance) {
	// Long description
	if desc := item.GetLongDesc(); desc != "" {
		s.Send(desc)
	} else {
		s.Send(fmt.Sprintf("You see %s.", item.GetShortDesc()))
	}

	// Extra descriptions
	for _, ed := range item.GetExtraDescs() {
		if ed.Description != "" {
			s.Send(ed.Description)
		}
	}

	// Item keywords
	s.Send(fmt.Sprintf("Keywords: %s", item.GetKeywords()))

	// Item type
	itemTypeName := itemTypeString(item.GetTypeFlag())
	weight := item.GetWeight()
	s.Send(fmt.Sprintf("Type: %s  Weight: %d", itemTypeName, weight))

	// Wear location info
	if item.IsWearable() {
		wearSlots := getWearLocationString(item)
		if wearSlots != "" {
			s.Send(fmt.Sprintf("Can be worn: %s", wearSlots))
		}
	}

	// Item stats
	switch item.GetTypeFlag() {
	case 5: // ITEM_WEAPON
		s.Send(fmt.Sprintf("Damage: %dd%d", item.Prototype.Values[1], item.Prototype.Values[2]))
		if item.Prototype.Values[3] != 0 {
			s.Send(fmt.Sprintf("Type: weapon type %d", item.Prototype.Values[3]))
		}
	case 9: // ITEM_ARMOR
		acBonus := item.Prototype.Values[0]
		s.Send(fmt.Sprintf("Armor Class: %d", acBonus))
	}

	// Affects
	if affects := item.GetAffects(); len(affects) > 0 {
		for _, aff := range affects {
			s.Send(fmt.Sprintf("Affects %s by %d.", affectName(aff.Location), aff.Modifier))
		}
	}
}

// examineMob shows details about a mob in the room.
func examineMob(s *Session, mob *game.MobInstance) {
	desc := mob.GetLongDesc()
	if desc == "" {
		desc = fmt.Sprintf("You see %s.", mob.GetShortDesc())
	}
	s.Send(desc)

	level := mob.GetLevel()
	// Generic level descriptions for NPCs — no exact numbers
	levelDesc := describeLevel(level)
	s.Send(fmt.Sprintf("This is %s, %s.", mob.GetShortDesc(), levelDesc))

	// Show health status
	hp := mob.GetHP()
	maxHP := mob.GetMaxHP()
	if hp < maxHP/4 {
		s.Send(fmt.Sprintf("%s looks severely wounded.", mob.GetShortDesc()))
	} else if hp < maxHP/2 {
		s.Send(fmt.Sprintf("%s looks wounded.", mob.GetShortDesc()))
	} else if hp < maxHP {
		s.Send(fmt.Sprintf("%s looks slightly wounded.", mob.GetShortDesc()))
	}

	// Show equipment if carrying anything notable
	if len(mob.Equipment) > 0 {
		s.Send(fmt.Sprintf("%s is carrying:", mob.GetShortDesc()))
		for _, eq := range mob.Equipment {
			if eq != nil {
				s.Send(fmt.Sprintf("  %s", eq.GetShortDesc()))
			}
		}
	}
}

// examinePlayer shows details about a player in the room.
func examinePlayer(s *Session, p *game.Player) {
	desc := p.Description
	if desc == "" {
		desc = fmt.Sprintf("You see %s.", p.Name)
	}
	s.Send(desc)

	className := game.ClassNames[p.Class]
	s.Send(fmt.Sprintf("This is %s, level %d %s.", p.Name, p.Level, className))

	// Show equipment
	equipped := p.Equipment.GetEquippedItems()
	if len(equipped) > 0 {
		s.Send(fmt.Sprintf("%s is wearing:", p.Name))
		for slot, item := range equipped {
			s.Send(fmt.Sprintf("  <%s> %s", slot.String(), item.GetShortDesc()))
		}
	}
}

// getWearLocationString returns a human-readable string of wear locations for an item.
func getWearLocationString(item *game.ObjectInstance) string {
	if item.Prototype == nil {
		return ""
	}
	var slots []string
	wf := item.Prototype.WearFlags
	if len(wf) < 1 {
		return ""
	}

	// Check primary wear flags (first word)
	primary := wf[0]
	if primary&(1<<1) != 0 {
		slots = append(slots, "finger")
	}
	if primary&(1<<2) != 0 {
		slots = append(slots, "neck")
	}
	if primary&(1<<3) != 0 {
		slots = append(slots, "body")
	}
	if primary&(1<<4) != 0 {
		slots = append(slots, "head")
	}
	if primary&(1<<5) != 0 {
		slots = append(slots, "legs")
	}
	if primary&(1<<6) != 0 {
		slots = append(slots, "feet")
	}
	if primary&(1<<7) != 0 {
		slots = append(slots, "hands")
	}
	if primary&(1<<8) != 0 {
		slots = append(slots, "arms")
	}
	if primary&(1<<9) != 0 {
		slots = append(slots, "shield")
	}
	if primary&(1<<10) != 0 {
		slots = append(slots, "about")
	}
	if primary&(1<<11) != 0 {
		slots = append(slots, "waist")
	}
	if primary&(1<<12) != 0 {
		slots = append(slots, "wrist")
	}
	if primary&(1<<13) != 0 {
		slots = append(slots, "wield")
	}
	if primary&(1<<14) != 0 {
		slots = append(slots, "hold")
	}

	return strings.Join(slots, ", ")
}

// itemTypeString returns a display name for an item type flag.
func itemTypeString(typeFlag int) string {
	switch typeFlag {
	case 0:
		return "undefined"
	case 1:
		return "container"
	case 2:
		return "drink container"
	case 3:
		return "light"
	case 4:
		return "key"
	case 5:
		return "weapon"
	case 6:
		return "money"
	case 7:
		return "treasure"
	case 8:
		return "potion"
	case 9:
		return "armor"
	case 10:
		return "food"
	case 11:
		return "pill"
	case 12:
		return "scroll"
	case 13:
		return "wand"
	case 14:
		return "staff"
	case 15:
		return "boat"
	case 16:
		return "furniture"
	case 17:
		return "trash"
	case 18:
		return "gem"
	case 19:
		return "jewelry"
	case 20:
		return "drum"
	case 21:
		return "missile"
	case 22:
		return "map"
	case 23:
		return "clock"
	case 24:
		return "lever"
	case 25:
		return "book"
	case 26:
		return "spellbook"
	case 27:
		return "amulet"
	case 28:
		return "ring"
	case 29:
		return "bottle"
	case 30:
		return "instrument"
	case 31:
		return "quiver"
	case 32:
		return "note"
	case 33:
		return "lockpick"
	case 34:
		return "portal"
	case 35:
		return "corpse"
	case 36:
		return "runestone"
	case 37:
		return "enchantment"
	case 38:
		return "component"
	case 39:
		return "trap"
	default:
		return fmt.Sprintf("type %d", typeFlag)
	}
}

// describeLevel returns a generic level description for NPCs.
func describeLevel(level int) string {
	switch {
	case level < 5:
		return "a novice"
	case level < 10:
		return "an apprentice"
	case level < 15:
		return "an expert"
	case level < 20:
		return "a veteran"
	case level < 25:
		return "a master"
	case level < 30:
		return "a grandmaster"
	case level < 35:
		return "a legend"
	default:
		return "a myth"
	}
}

// affectName returns a human-readable name for an affect location.
func affectName(location int) string {
	switch location {
	case 1:
		return "strength"
	case 2:
		return "dexterity"
	case 3:
		return "constitution"
	case 4:
		return "intelligence"
	case 5:
		return "wisdom"
	case 6:
		return "charisma"
	case 7:
		return "level"
	case 8:
		return "age"
	case 12:
		return "hit points"
	case 13:
		return "mana"
	case 14:
		return "move"
	case 15:
		return "sex"
	case 17:
		return "armor class"
	case 18:
		return "hit roll"
	case 19:
		return "damage roll"
	case 20:
		return "saving spell"
	case 21:
		return "saving rod"
	case 22:
		return "saving para"
	case 23:
		return "saving breath"
	default:
		return fmt.Sprintf("affect %d", location)
	}
}
