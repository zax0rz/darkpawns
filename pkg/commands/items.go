package commands

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/session"
)

// ExecuteItemCommand processes item-related commands.
func ExecuteItemCommand(s *session.Session, command string, args []string) error {
	cmd := strings.ToLower(command)

	switch cmd {
	case "inventory", "i", "inv":
		return cmdInventory(s, args)
	case "equipment", "eq":
		return cmdEquipment(s, args)
	case "wear":
		return cmdWear(s, args)
	case "remove":
		return cmdRemove(s, args)
	case "wield":
		return cmdWield(s, args)
	case "hold":
		return cmdHold(s, args)
	case "get", "take":
		return cmdGet(s, args)
	case "drop":
		return cmdDrop(s, args)
	default:
		s.SendText(fmt.Sprintf("Unknown item command: %s", command))
		return nil
	}
}

// cmdInventory shows the player's inventory.
func cmdInventory(s *session.Session, args []string) error {
	// Get item count first
	count := s.Player.Inventory.GetItemCount()
	if count == 0 {
		s.SendText("You are carrying nothing.")
		return nil
	}

	// Get all items
	items := s.Player.Inventory.FindItems("") // Empty string returns all items
	var itemDescs []string
	for _, item := range items {
		itemDescs = append(itemDescs, item.ShortDesc)
	}

	msg := fmt.Sprintf("You are carrying:\n%s", strings.Join(itemDescs, "\n"))
	s.SendText(msg)
	return nil
}

// cmdEquipment shows the player's equipped items.
func cmdEquipment(s *session.Session, args []string) error {
	equipped := s.Player.Equipment.GetEquippedItems()
	if len(equipped) == 0 {
		s.SendText("You are not wearing anything.")
		return nil
	}

	var items []string
	for slot, item := range equipped {
		items = append(items, fmt.Sprintf("%-10s: %s", slot.String(), item.ShortDesc))
	}

	msg := fmt.Sprintf("You are wearing:\n%s", strings.Join(items, "\n"))
	s.SendText(msg)
	return nil
}

// cmdWear equips an item from inventory.
func cmdWear(s *session.Session, args []string) error {
	if len(args) == 0 {
		s.SendText("Wear what?")
		return nil
	}

	itemName := strings.Join(args, " ")
	
	// Find item in inventory
	item, found := s.Player.Inventory.FindItem(itemName)
	if !found {
		s.SendText(fmt.Sprintf("You don't have '%s'.", itemName))
		return nil
	}

	// Try to equip the item
	if err := s.Player.Equipment.Equip(item, s.Player.Inventory); err != nil {
		s.SendText(fmt.Sprintf("You can't wear that: %v", err))
		return nil
	}

	// Remove from inventory (equip should have moved it)
	s.Player.Inventory.RemoveItem(item)
	s.SendText(fmt.Sprintf("You wear %s.", item.ShortDesc))

	// Broadcast to room
	broadcastEquipmentChange(s, "wear", item)
	return nil
}

// cmdRemove unequips an item.
func cmdRemove(s *session.Session, args []string) error {
	if len(args) == 0 {
		s.SendText("Remove what?")
		return nil
	}

	itemName := strings.Join(args, " ")
	
	// Find the item in equipment
	var itemToRemove *parser.Obj
	var slotToRemove game.EquipmentSlot
	equipped := s.Player.Equipment.GetEquippedItems()
	
	for slot, item := range equipped {
		// Check if item matches by name
		if strings.Contains(strings.ToLower(item.Keywords), strings.ToLower(itemName)) ||
		   strings.Contains(strings.ToLower(item.ShortDesc), strings.ToLower(itemName)) {
			itemToRemove = item
			slotToRemove = slot
			break
		}
	}

	if itemToRemove == nil {
		s.SendText(fmt.Sprintf("You're not wearing '%s'.", itemName))
		return nil
	}

	// Try to unequip
	if err := s.Player.Equipment.Unequip(slotToRemove, s.Player.Inventory); err != nil {
		s.SendText(fmt.Sprintf("You can't remove that: %v", err))
		return nil
	}

	s.SendText(fmt.Sprintf("You remove %s.", itemToRemove.ShortDesc))

	// Broadcast to room
	broadcastEquipmentChange(s, "remove", itemToRemove)
	return nil
}

// cmdWield equips a weapon.
func cmdWield(s *session.Session, args []string) error {
	if len(args) == 0 {
		s.SendText("Wield what?")
		return nil
	}

	itemName := strings.Join(args, " ")
	s.Player.Inventory.mu.Lock()
	s.Player.Equipment.mu.Lock()
	defer s.Player.Inventory.mu.Unlock()
	defer s.Player.Equipment.mu.Unlock()

	item, found := s.Player.Inventory.FindItem(itemName)
	if !found {
		s.SendText(fmt.Sprintf("You don't have '%s'.", itemName))
		return nil
	}

	// Check if item is a weapon
	if item.TypeFlag != 1 { // Type 1 is weapon in CircleMUD
		s.SendText("That's not a weapon.")
		return nil
	}

	// Unequip current weapon if any
	if currentWeapon, ok := s.Player.Equipment.GetItemInSlot(game.SlotWield); ok {
		if err := s.Player.Equipment.Unequip(game.SlotWield, s.Player.Inventory); err != nil {
			s.SendText(fmt.Sprintf("You can't unwield your current weapon: %v", err))
			return nil
		}
	}

	// Equip new weapon
	if err := s.Player.Equipment.Equip(item, s.Player.Inventory); err != nil {
		s.SendText(fmt.Sprintf("You can't wield that: %v", err))
		return nil
	}

	// Remove from inventory
	s.Player.Inventory.RemoveItem(item)
	s.SendText(fmt.Sprintf("You wield %s.", item.ShortDesc))

	// Broadcast to room
	broadcastEquipmentChange(s, "wield", item)
	return nil
}

// cmdHold holds an item.
func cmdHold(s *session.Session, args []string) error {
	if len(args) == 0 {
		s.SendText("Hold what?")
		return nil
	}

	itemName := strings.Join(args, " ")
	s.Player.Inventory.mu.Lock()
	s.Player.Equipment.mu.Lock()
	defer s.Player.Inventory.mu.Unlock()
	defer s.Player.Equipment.mu.Unlock()

	item, found := s.Player.Inventory.FindItem(itemName)
	if !found {
		s.SendText(fmt.Sprintf("You don't have '%s'.", itemName))
		return nil
	}

	// Check if item can be held
	wearFlags := s.Player.Equipment.getWearFlags(item)
	canHold := false
	for _, slot := range wearFlags {
		if slot == game.SlotHold {
			canHold = true
			break
		}
	}

	if !canHold {
		s.SendText("You can't hold that.")
		return nil
	}

	// Unequip current held item if any
	if currentItem, ok := s.Player.Equipment.GetItemInSlot(game.SlotHold); ok {
		if err := s.Player.Equipment.Unequip(game.SlotHold, s.Player.Inventory); err != nil {
			s.SendText(fmt.Sprintf("You can't unhold your current item: %v", err))
			return nil
		}
	}

	// Equip new item
	if err := s.Player.Equipment.Equip(item, s.Player.Inventory); err != nil {
		s.SendText(fmt.Sprintf("You can't hold that: %v", err))
		return nil
	}

	// Remove from inventory
	s.Player.Inventory.RemoveItem(item)
	s.SendText(fmt.Sprintf("You hold %s.", item.ShortDesc))

	// Broadcast to room
	broadcastEquipmentChange(s, "hold", item)
	return nil
}

// cmdGet picks up an item from the room.
func cmdGet(s *session.Session, args []string) error {
	if len(args) == 0 {
		s.SendText("Get what?")
		return nil
	}

	itemName := strings.Join(args, " ")
	
	// Get current room
	room, ok := s.Manager.World.GetRoom(s.Player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// In a full implementation, we'd check room contents
	// For now, we'll create a dummy item for testing
	s.Player.Inventory.mu.Lock()
	defer s.Player.Inventory.mu.Unlock()

	// Check if inventory is full
	if s.Player.Inventory.IsFull() {
		s.SendText("Your inventory is full.")
		return nil
	}

	// Create a test item (in real implementation, get from room)
	testItem := &parser.Obj{
		VNum:      1,
		Keywords:  "sword",
		ShortDesc: "a sharp sword",
		LongDesc:  "A sharp sword lies here.",
		TypeFlag:  1, // Weapon
		Weight:    5,
	}

	if err := s.Player.Inventory.AddItem(testItem); err != nil {
		s.SendText(fmt.Sprintf("Can't pick that up: %v", err))
		return nil
	}

	s.SendText(fmt.Sprintf("You pick up %s.", testItem.ShortDesc))
	return nil
}

// cmdDrop drops an item from inventory.
func cmdDrop(s *session.Session, args []string) error {
	if len(args) == 0 {
		s.SendText("Drop what?")
		return nil
	}

	itemName := strings.Join(args, " ")
	s.Player.Inventory.mu.Lock()
	defer s.Player.Inventory.mu.Unlock()

	item, found := s.Player.Inventory.FindItem(itemName)
	if !found {
		s.SendText(fmt.Sprintf("You don't have '%s'.", itemName))
		return nil
	}

	// Remove from inventory
	s.Player.Inventory.RemoveItem(item)
	s.SendText(fmt.Sprintf("You drop %s.", item.ShortDesc))
	return nil
}

// broadcastEquipmentChange broadcasts equipment changes to the room.
func broadcastEquipmentChange(s *session.Session, action string, item *parser.Obj) {
	event := session.EventData{
		Type: "equipment",
		From: s.Player.Name,
		Text: fmt.Sprintf("%s %s %s.", s.Player.Name, action, item.ShortDesc),
	}

	msg, _ := json.Marshal(session.ServerMessage{
		Type: session.MsgEvent,
		Data: event,
	})

	s.Manager.BroadcastToRoom(s.Player.GetRoom(), msg, s.Player.Name)
}