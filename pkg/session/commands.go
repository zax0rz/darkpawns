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
	// Social / info commands
	case "score", "sc":
		return cmdScore(s)
	case "who":
		return cmdWho(s)
	case "tell":
		return cmdTell(s, args)
	case "emote", "me":
		return cmdEmote(s, args)
	case "shout":
		return cmdShout(s, args)
	case "where":
		return cmdWhere(s)
	case "help":
		return cmdHelp(s, args)
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

	// Mark room vars dirty for agents after movement
	s.markDirty(VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems)

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
				s.manager.world.AddItemToRoom(item, roomVNum)
				s.sendText(fmt.Sprintf("Can't pick that up: %v", err))
				return nil
			}

			s.sendText(fmt.Sprintf("You pick up %s.", item.GetShortDesc()))
			s.markDirty(VarInventory, VarRoomItems)

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
	s.markDirty(VarInventory, VarRoomItems)

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

// cmdScore shows the player's stats.
// Source: act.informative.c do_score() lines 1168-1451
func cmdScore(s *Session) error {
	p := s.player
	className := game.ClassNames[p.Class]
	raceName := game.RaceNames[p.Race]

	// Mana label — act.informative.c line 1197
	manaLabel := "Mana"
	if p.Class == game.ClassPsionic || p.Class == game.ClassMystic {
		manaLabel = "Mind/Psi"
	}

	// Alignment description — act.informative.c lines 1203-1228
	align := p.Alignment
	var alignDesc string
	switch {
	case align == 1000:
		alignDesc = "You are the Epitome of Righteousness!"
	case align >= 900:
		alignDesc = "You're so good, you make the angels jealous."
	case align >= 750:
		alignDesc = "You are feeling pretty righteous."
	case align >= 500:
		alignDesc = "You are aligned with the path of right."
	case align >= 350:
		alignDesc = "You are feeling pretty good today."
	case align >= 100:
		alignDesc = "You are a little more good than neutral, but yet still bland."
	case align > -100:
		alignDesc = "You are neutral, how boring."
	case align > -350:
		alignDesc = "You are little more evil than neutral, but not very exciting."
	case align > -500:
		alignDesc = "I actually think you would kill your own mother."
	case align > -750:
		alignDesc = "You are so evil it hurts."
	case align > -900:
		alignDesc = "Charles Manson is in your fan club."
	default:
		alignDesc = "You are the Epitome of Evil!"
	}

	// AC description — act.informative.c lines 1230-1257
	ac := p.AC
	var acDesc string
	switch {
	case ac == 100:
		acDesc = "You are naked, have you no shame?"
	case ac > 70:
		acDesc = "You are lightly clothed."
	case ac > 40:
		acDesc = "You are pretty well clothed."
	case ac > 10:
		acDesc = "You are lightly armored."
	case ac > -10:
		acDesc = "You are well armored."
	case ac > -40:
		acDesc = "You are getting pretty sweaty with all that armor on."
	case ac > -50:
		acDesc = "You are extremely well armored."
	case ac > -75:
		acDesc = "You are decked out in full battle armor."
	case ac > -125:
		acDesc = "You are armored like a wyvern!"
	case ac > -150:
		acDesc = "You are armored like a dragon!"
	case ac > -175:
		acDesc = "You could walk through the gates of Hell in all that armor!"
	default:
		acDesc = "You are armored like a god!"
	}

	// Pack weight — act.informative.c lines 1301-1317 (simplified: track item count)
	var packDesc string
	count := p.Inventory.GetItemCount()
	switch {
	case count == 0:
		packDesc = "Your pack is empty."
	case count <= 3:
		packDesc = "Your pack is light."
	case count <= 6:
		packDesc = "Your pack is fairly heavy."
	case count <= 9:
		packDesc = "Your pack is heavy."
	default:
		packDesc = "Your pack is almost too heavy to lift."
	}

	// Position — act.informative.c lines 1321-1356 (POS_STANDING=8 from structs.h)
	var posDesc string
	if p.Fighting != "" {
		posDesc = fmt.Sprintf("You are fighting %s.", p.Fighting)
	} else {
		posDesc = "You are standing."
	}

	out := fmt.Sprintf(
		"%s\n"+
			"Hit points: %d(%d)  %s points: %d(%d)\n"+
			"%s\n"+
			"%s\n"+
			"Experience: %d points\n"+
			"Coins carried: %d gold coins\n"+
			"This ranks you as %s %s (level %d).\n"+
			"You are %s %s.\n"+
			"%s\n"+
			"%s",
		p.Name,
		p.Health, p.MaxHealth, manaLabel, p.Mana, p.MaxMana,
		alignDesc,
		acDesc,
		p.Exp,
		p.Gold,
		p.Name, className, p.Level,
		raceName, className,
		packDesc,
		posDesc,
	)
	s.sendText(out)
	return nil
}

// cmdWho lists all online players.
// Source: act.informative.c do_who() lines 1681-1943
func cmdWho(s *Session) error {
	s.manager.mu.RLock()
	sessions := make([]*Session, 0, len(s.manager.sessions))
	for _, sess := range s.manager.sessions {
		sessions = append(sessions, sess)
	}
	s.manager.mu.RUnlock()

	out := "Players\n-------\n"
	count := 0
	for _, sess := range sessions {
		if sess.player == nil {
			continue
		}
		p := sess.player
		className := game.ClassNames[p.Class]
		raceName := game.RaceNames[p.Race]
		// Format: [ LV  Class ] Name Race — act.informative.c line 1874
		tag := "player"
		if sess.isAgent {
			tag = "agent"
		}
		out += fmt.Sprintf("[ %2d  %-8s] %-15s (%s, %s, %s)\n",
			p.Level, className, p.Name, raceName, className, tag)
		count++
	}
	if count == 0 {
		out += "\nNo-one at all!\n"
	} else if count == 1 {
		out += "\nOne character displayed.\n"
	} else {
		out += fmt.Sprintf("\n%d characters displayed.\n", count)
	}
	s.sendText(out)
	return nil
}

// cmdTell sends a private message to another player.
// Source: act.comm.c do_tell() lines 901-931, perform_tell()
func cmdTell(s *Session, args []string) error {
	if len(args) < 2 {
		s.sendText("Who do you wish to tell what??")
		return nil
	}
	targetName := args[0]
	message := strings.Join(args[1:], " ")

	if strings.EqualFold(targetName, s.player.Name) {
		s.sendText("You try to tell yourself something.")
		return nil
	}

	// Find target session — act.comm.c line 909 get_char_vis()
	target, ok := s.manager.GetSession(targetName)
	if !ok || target.player == nil {
		s.sendText("There is no such player online.")
		return nil
	}

	// Deliver to target — act.comm.c perform_tell()
	target.sendText(fmt.Sprintf("%s tells you, '%s'", s.player.Name, message))
	// Confirm to sender
	s.sendText(fmt.Sprintf("You tell %s, '%s'", target.player.Name, message))
	return nil
}

// cmdEmote broadcasts a roleplay action to the room.
// Source: act.comm.c do_emote() — "$n laughs." style
func cmdEmote(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Emote what?")
		return nil
	}
	action := strings.Join(args, " ")
	text := fmt.Sprintf("%s %s", s.player.Name, action)

	s.sendText(text)
	msg, _ := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "emote",
			From: s.player.Name,
			Text: text,
		},
	})
	s.manager.BroadcastToRoom(s.player.GetRoom(), msg, s.player.Name)
	return nil
}

// cmdShout broadcasts a message to all players in the same zone.
// Source: act.comm.c do_gen_comm() SCMD_SHOUT lines 1286-1289
// Original: zone-scoped; receivers must be POS_RESTING or higher.
func cmdShout(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Yes, shout, fine, shout we must, but WHAT???")
		return nil
	}
	message := strings.Join(args, " ")

	// Get the shouter's zone
	senderRoom, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return nil
	}
	senderZone := senderRoom.Zone

	text := fmt.Sprintf("%s shouts, '%s'", s.player.Name, message)
	s.sendText(fmt.Sprintf("You shout, '%s'", message))

	s.manager.mu.RLock()
	targets := make([]*Session, 0)
	for name, sess := range s.manager.sessions {
		if name == s.player.Name || sess.player == nil {
			continue
		}
		// Restrict to same zone — act.comm.c line 1287
		targetRoom, ok := s.manager.world.GetRoom(sess.player.GetRoom())
		if !ok || targetRoom.Zone != senderZone {
			continue
		}
		targets = append(targets, sess)
	}
	s.manager.mu.RUnlock()

	msg, _ := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "shout",
			From: s.player.Name,
			Text: text,
		},
	})
	for _, sess := range targets {
		select {
		case sess.send <- msg:
		default:
		}
	}
	return nil
}

// cmdWhere lists all online players and their locations.
// Source: act.informative.c do_where() lines 2244-2307
func cmdWhere(s *Session) error {
	s.manager.mu.RLock()
	sessions := make([]*Session, 0, len(s.manager.sessions))
	for _, sess := range s.manager.sessions {
		sessions = append(sessions, sess)
	}
	s.manager.mu.RUnlock()

	out := "Players\n-------\n"
	found := false
	for _, sess := range sessions {
		if sess.player == nil {
			continue
		}
		p := sess.player
		room, ok := s.manager.world.GetRoom(p.GetRoom())
		if !ok {
			continue
		}
		// Format mirrors do_where() line 2272: name - [vnum] room name
		out += fmt.Sprintf("%-20s - [%5d] %s\n", p.Name, room.VNum, room.Name)
		found = true
	}
	if !found {
		out += "No-one visible.\n"
	}
	s.sendText(out)
	return nil
}

// cmdHelp provides a basic help stub.
// Full implementation deferred to a later phase.
func cmdHelp(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Available commands: look, north/south/east/west/up/down, say, hit, flee, " +
			"inventory, equipment, wear, remove, wield, hold, get, drop, " +
			"score, who, tell, emote, shout, where, quit\n" +
			"Type 'help <topic>' for more info (stub — full help coming later).")
		return nil
	}
	s.sendText(fmt.Sprintf("No help available for '%s' yet.", strings.Join(args, " ")))
	return nil
}