package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdQuit(s *Session) error {
	room := s.player.GetRoom()

	// Notify room
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "leave",
			From: s.player.Name,
			Text: fmt.Sprintf("%s has left the game.", s.player.Name),
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return nil
	}
	s.manager.BroadcastToRoom(room, msg, s.player.Name)

	// Remove from world and close connection
	s.manager.world.RemovePlayer(s.player.Name)
	s.manager.Unregister(s.player.Name)
	_ = s.conn.Close()

	return nil
}

// cmdInventory shows the player's inventory.
func cmdInventory(s *Session, args []string) error {
	// Get item count first
	count := s.player.Inventory.GetItemCount()
	if count == 0 {
		s.sendText("You are carrying nothing.")
		return nil
	}

	// Get all items
	items := s.player.Inventory.FindItems("") // Empty string returns all items
	var itemDescs []string
	for _, item := range items {
		itemDescs = append(itemDescs, item.GetShortDesc())
	}

	msg := fmt.Sprintf("You are carrying:\n%s", strings.Join(itemDescs, "\n"))
	s.sendText(msg)
	return nil
}

// cmdEquipment shows the player's equipped items.
func cmdEquipment(s *Session, args []string) error {
	equipped := s.player.Equipment.GetEquippedItems()
	if len(equipped) == 0 {
		s.sendText("You are not wearing anything.")
		return nil
	}

	var items []string
	for slot, item := range equipped {
		items = append(items, fmt.Sprintf("%-10s: %s", slot.String(), item.GetShortDesc()))
	}

	msg := fmt.Sprintf("You are wearing:\n%s", strings.Join(items, "\n"))
	s.sendText(msg)
	return nil
}

// cmdWear equips an item from inventory.
func cmdWear(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Wear what?")
		return nil
	}

	itemName := strings.Join(args, " ")

	// Find item in inventory
	item, found := s.player.Inventory.FindItem(itemName)
	if !found {
		s.sendText(fmt.Sprintf("You don't have '%s'.", itemName))
		return nil
	}

	// Try to equip the item
	if err := s.player.Equipment.Equip(item, s.player.Inventory); err != nil {
		s.sendText(fmt.Sprintf("You can't wear that: %v", err))
		return nil
	}

	// Remove from inventory (equip should have moved it)
	s.player.Inventory.RemoveItem(item)
	s.sendText(fmt.Sprintf("You wear %s.", item.GetShortDesc()))
	s.markDirty(VarInventory, VarEquipment)

	// Broadcast to room
	broadcastEquipmentChange(s, "wear", item)
	return nil
}

// cmdRemove unequips an item.
func cmdRemove(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Remove what?")
		return nil
	}

	itemName := strings.Join(args, " ")

	// Find the item in equipment
	var itemToRemove *game.ObjectInstance
	var slotToRemove game.EquipmentSlot
	equipped := s.player.Equipment.GetEquippedItems()

	for slot, item := range equipped {
		if strings.Contains(strings.ToLower(item.GetKeywords()), strings.ToLower(itemName)) ||
			strings.Contains(strings.ToLower(item.GetShortDesc()), strings.ToLower(itemName)) {
			itemToRemove = item
			slotToRemove = slot
			break
		}
	}

	if itemToRemove == nil {
		s.sendText(fmt.Sprintf("You're not wearing '%s'.", itemName))
		return nil
	}

	if err := s.player.Equipment.Unequip(slotToRemove, s.player.Inventory); err != nil {
		s.sendText(fmt.Sprintf("You can't remove that: %v", err))
		return nil
	}

	s.sendText(fmt.Sprintf("You remove %s.", itemToRemove.GetShortDesc()))
	s.markDirty(VarInventory, VarEquipment)
	broadcastEquipmentChange(s, "remove", itemToRemove)
	return nil
}

// cmdWield equips a weapon.
func cmdWield(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Wield what?")
		return nil
	}

	itemName := strings.Join(args, " ")

	// Find item in inventory
	item, found := s.player.Inventory.FindItem(itemName)
	if !found {
		s.sendText(fmt.Sprintf("You don't have '%s'.", itemName))
		return nil
	}

	// Check if item is a weapon
	if item.GetTypeFlag() != 5 { // ITEM_WEAPON = 5 from structs.h
		s.sendText("That's not a weapon.")
		return nil
	}

	// Unequip current weapon if any
	if _, ok := s.player.Equipment.GetItemInSlot(game.SlotWield); ok {
		if err := s.player.Equipment.Unequip(game.SlotWield, s.player.Inventory); err != nil {
			s.sendText(fmt.Sprintf("You can't unwield your current weapon: %v", err))
			return nil
		}
	}

	// Equip new weapon
	if err := s.player.Equipment.Equip(item, s.player.Inventory); err != nil {
		s.sendText(fmt.Sprintf("You can't wield that: %v", err))
		return nil
	}

	// Remove from inventory
	s.player.Inventory.RemoveItem(item)
	s.sendText(fmt.Sprintf("You wield %s.", item.GetShortDesc()))
	s.markDirty(VarInventory, VarEquipment)
	broadcastEquipmentChange(s, "wield", item)
	return nil
}

// cmdHold holds an item.
func cmdHold(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Hold what?")
		return nil
	}

	itemName := strings.Join(args, " ")

	// Find item in inventory
	item, found := s.player.Inventory.FindItem(itemName)
	if !found {
		s.sendText(fmt.Sprintf("You don't have '%s'.", itemName))
		return nil
	}

	// Unequip current held item if any
	if _, ok := s.player.Equipment.GetItemInSlot(game.SlotHold); ok {
		if err := s.player.Equipment.Unequip(game.SlotHold, s.player.Inventory); err != nil {
			s.sendText(fmt.Sprintf("You can't unhold your current item: %v", err))
			return nil
		}
	}

	// Try to equip in hold slot
	if err := s.player.Equipment.Equip(item, s.player.Inventory); err != nil {
		s.sendText(fmt.Sprintf("You can't hold that: %v", err))
		return nil
	}

	// Remove from inventory
	s.player.Inventory.RemoveItem(item)
	s.sendText(fmt.Sprintf("You hold %s.", item.GetShortDesc()))
	broadcastEquipmentChange(s, "hold", item)
	return nil
}

// cmdGet picks up an item from the room.
func cmdGet(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Get what?")
		return nil
	}

	itemName := strings.Join(args, " ")
	roomVNum := s.player.GetRoom()

	// Check if inventory is full
	if s.player.Inventory.IsFull() {
		s.sendText("Your inventory is full.")
		return nil
	}

	// Find item in room
	items := s.manager.world.GetItemsInRoom(roomVNum)
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.GetShortDesc()), strings.ToLower(itemName)) {
			// Remove from room
			if !s.manager.world.RemoveItemFromRoom(item, roomVNum) {
				s.sendText("You can't get that.")
				return nil
			}

			// Add to inventory
			if err := s.player.Inventory.AddItem(item); err != nil {
				// Put back in room
				s.manager.world.AddItemToRoom(item, roomVNum) //nolint:staticcheck // TODO: migrate to MoveObjectToRoom
				s.sendText(fmt.Sprintf("Can't pick that up: %v", err))
				return nil
			}

			s.sendText(fmt.Sprintf("You pick up %s.", item.GetShortDesc()))
			s.markDirty(VarInventory, VarRoomItems)

			// Notify room
			msg, err := json.Marshal(ServerMessage{
				Type: MsgEvent,
				Data: EventData{
					Type: "get",
					From: s.player.Name,
					Text: fmt.Sprintf("%s picks up %s.", s.player.Name, item.GetShortDesc()),
				},
			})
			if err != nil {
				slog.Error("json.Marshal error", "error", err)
				return nil
			}
			s.manager.BroadcastToRoom(roomVNum, msg, s.player.Name)
			return nil
		}
	}

	s.sendText("You don't see that here.")
	return nil
}

// cmdDrop drops an item from inventory.
func cmdDrop(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Drop what?")
		return nil
	}

	itemName := strings.Join(args, " ")
	roomVNum := s.player.GetRoom()

	item, found := s.player.Inventory.FindItem(itemName)
	if !found {
		s.sendText(fmt.Sprintf("You don't have '%s'.", itemName))
		return nil
	}

	// Remove from inventory and place in room
	s.player.Inventory.RemoveItem(item)
	item.RoomVNum = roomVNum
	s.manager.world.AddItemToRoom(item, roomVNum) //nolint:staticcheck // TODO: migrate to MoveObjectToRoom

	s.sendText(fmt.Sprintf("You drop %s.", item.GetShortDesc()))
	s.markDirty(VarInventory, VarRoomItems)

	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "drop",
			From: s.player.Name,
			Text: fmt.Sprintf("%s drops %s.", s.player.Name, item.GetShortDesc()),
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return nil
	}
	s.manager.BroadcastToRoom(roomVNum, msg, s.player.Name)

	return nil
}

// broadcastEquipmentChange broadcasts equipment changes to the room.
func broadcastEquipmentChange(s *Session, action string, item *game.ObjectInstance) {
	event := EventData{
		Type: "equipment",
		From: s.player.Name,
		Text: fmt.Sprintf("%s %s %s.", s.player.Name, action, item.GetShortDesc()),
	}

	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: event,
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}

	s.manager.BroadcastToRoom(s.player.GetRoom(), msg, s.player.Name)
}

// cmdFollow sets the player to follow another player.
// Source: act.movement.c do_follow() lines 883–951
