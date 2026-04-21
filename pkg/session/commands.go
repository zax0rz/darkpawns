package session

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// ExecuteCommand processes a game command.
func ExecuteCommand(s *Session, command string, args []string) error {
	cmd := strings.ToLower(command)

	// Check for mob scripts with oncmd trigger before processing
	// Based on the original MUD's script handling
	if s.player != nil && s.player.GetRoomVNum() > 0 {
		// Get mobs in the room
		mobs := s.manager.world.GetMobsInRoom(s.player.GetRoomVNum())
		fullCommand := command
		if len(args) > 0 {
			fullCommand = command + " " + strings.Join(args, " ")
		}
		
		// Check each mob for oncmd script
		for _, mob := range mobs {
			if mob.HasScript("oncmd") {
				// Create script context
				ctx := mob.CreateScriptContext(s.player, nil, fullCommand)
				// Run the script
				handled, err := mob.RunScript("oncmd", ctx)
				if err != nil {
					// Log error but continue
					log.Printf("Error running oncmd script for mob %v: %v", mob.GetVNum(), err)
				}
				if handled {
					// Script handled the command, don't process further
					return nil
				}
			}
		}
	}

	switch cmd {
	case "look", "l":
		return cmdLook(s, args)
	case "north", "n":
		return cmdMove(s, "north")
	case "east", "e":
		return cmdMove(s, "east")
	case "south", "s":
		return cmdMove(s, "south")
	case "west", "w":
		return cmdMove(s, "west")
	case "up", "u":
		return cmdMove(s, "up")
	case "down", "d":
		return cmdMove(s, "down")
	case "say":
		return cmdSay(s, args)
	case "hit", "attack", "kill":
		return cmdHit(s, args)
	case "flee":
		return cmdFlee(s)
	case "quit":
		return cmdQuit(s)
	// Item commands
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
		s.sendText(fmt.Sprintf("Unknown command: %s", command))
		return nil
	}
}

// cmdLook shows the current room.
func cmdLook(s *Session, args []string) error {
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	// Get other players in room
	players := s.manager.world.GetPlayersInRoom(room.VNum)
	var playerNames []string
	for _, p := range players {
		if p.Name != s.player.Name {
			playerNames = append(playerNames, p.Name)
		}
	}

	// Get items in room
	items := s.manager.world.GetItemsInRoom(room.VNum)
	var itemDescs []string
	for _, item := range items {
		itemDescs = append(itemDescs, item.GetLongDesc())
	}

	state := StateData{
		Player: PlayerState{
			Name:      s.player.Name,
			Health:    s.player.Health,
			MaxHealth: s.player.MaxHealth,
			Level:     s.player.Level,
		},
		Room: RoomState{
			VNum:        room.VNum,
			Name:        room.Name,
			Description: room.Description,
			Exits:       getExitNames(room.Exits),
			Players:     playerNames,
			Items:       itemDescs,
		},
	}

	msg, _ := json.Marshal(ServerMessage{
		Type: MsgState,
		Data: state,
	})
	s.send <- msg
	return nil
}

// cmdMove moves the player in a direction.
func cmdMove(s *Session, direction string) error {
	oldRoom := s.player.GetRoom()

	newRoom, err := s.manager.world.MovePlayer(s.player, direction)
	if err != nil {
		s.sendText(fmt.Sprintf("You can't go %s.", direction))
		return nil
	}

	// Notify old room
	leaveMsg, _ := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "leave",
			From: s.player.Name,
			Text: fmt.Sprintf("%s leaves %s.", s.player.Name, direction),
		},
	})
	s.manager.BroadcastToRoom(oldRoom, leaveMsg, s.player.Name)

	// Notify new room
	enterMsg, _ := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "enter",
			From: s.player.Name,
			Text: fmt.Sprintf("%s has arrived.", s.player.Name),
		},
	})
	s.manager.BroadcastToRoom(newRoom.VNum, enterMsg, s.player.Name)

	// Check for mobs with greet scripts
	mobs := s.manager.world.GetMobsInRoom(newRoom.VNum)
	for _, mob := range mobs {
		if mob.HasScript("greet") {
			ctx := mob.CreateScriptContext(s.player, nil, "")
			mob.RunScript("greet", ctx)
		}
	}

	// Check for aggressive mobs in new room
	if s.manager.world.OnPlayerEnterRoom(s.player, newRoom.VNum, s.manager.combatEngine) {
		// Combat was initiated, notify player
		s.sendText("You are attacked!")
	}

	// Send new room state to player
	return cmdLook(s, nil)
}

// cmdSay sends a message to the room.
func cmdSay(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Say what?")
		return nil
	}

	text := strings.Join(args, " ")

	// Confirm to sender
	s.sendText(fmt.Sprintf("You say, \"%s\"", text))

	// Broadcast to room
	msg, _ := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "say",
			From: s.player.Name,
			Text: fmt.Sprintf("%s says, \"%s\"", s.player.Name, text),
		},
	})
	s.manager.BroadcastToRoom(s.player.GetRoom(), msg, s.player.Name)

	return nil
}

// cmdQuit handles player logout.
func cmdQuit(s *Session) error {
	room := s.player.GetRoom()

	// Notify room
	msg, _ := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "leave",
			From: s.player.Name,
			Text: fmt.Sprintf("%s has left the game.", s.player.Name),
		},
	})
	s.manager.BroadcastToRoom(room, msg, s.player.Name)

	// Remove from world and close connection
	s.manager.world.RemovePlayer(s.player.Name)
	s.manager.Unregister(s.player.Name)
	s.conn.Close()

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
				s.manager.world.AddItemToRoom(item, roomVNum)
				s.sendText(fmt.Sprintf("Can't pick that up: %v", err))
				return nil
			}

			s.sendText(fmt.Sprintf("You pick up %s.", item.GetShortDesc()))
			
			// Notify room
			msg, _ := json.Marshal(ServerMessage{
				Type: MsgEvent,
				Data: EventData{
					Type: "get",
					From: s.player.Name,
					Text: fmt.Sprintf("%s picks up %s.", s.player.Name, item.GetShortDesc()),
				},
			})
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
	s.manager.world.AddItemToRoom(item, roomVNum)

	s.sendText(fmt.Sprintf("You drop %s.", item.GetShortDesc()))

	msg, _ := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "drop",
			From: s.player.Name,
			Text: fmt.Sprintf("%s drops %s.", s.player.Name, item.GetShortDesc()),
		},
	})
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

	msg, _ := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: event,
	})

	s.manager.BroadcastToRoom(s.player.GetRoom(), msg, s.player.Name)
}

// sendText sends a simple text message to the player.
func (s *Session) sendText(text string) {
	msg, _ := json.Marshal(ServerMessage{
		Type: MsgText,
		Data: TextData{Text: text},
	})
	select {
	case s.send <- msg:
	default:
	}
}