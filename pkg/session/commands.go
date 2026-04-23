package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/combat"
	"github.com/zax0rz/darkpawns/pkg/command"
	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/game/systems"
)

// cmdRegistry is the global command registry, initialized on first use.
var cmdRegistry = command.NewRegistry()

// commandSession wraps a *Session to satisfy common.CommandSession.
// It adapts GetPlayer() *game.Player to GetPlayer() interface{}.
type commandSession struct {
	*Session
}

func (cs *commandSession) GetPlayer() interface{} {
	return cs.Session.GetPlayer()
}

// init registers all built-in commands at package initialization.
func init() {
	// Movement
	cmdRegistry.Register("north", wrapMove("north"), "Move north.", 0, 0, "n")
	cmdRegistry.Register("east", wrapMove("east"), "Move east.", 0, 0, "e")
	cmdRegistry.Register("south", wrapMove("south"), "Move south.", 0, 0, "s")
	cmdRegistry.Register("west", wrapMove("west"), "Move west.", 0, 0, "w")
	cmdRegistry.Register("up", wrapMove("up"), "Move up.", 0, 0, "u")
	cmdRegistry.Register("down", wrapMove("down"), "Move down.", 0, 0, "d")

	// Look
	cmdRegistry.Register("look", wrapArgs(cmdLook), "Look around the room.", 0, 0, "l")

	// Communication
	cmdRegistry.Register("say", wrapArgs(cmdSay), "Say something to the room.", 0, 0)
	cmdRegistry.Register("tell", wrapArgs(cmdTell), "Send a private message to a player.", 0, 0)
	cmdRegistry.Register("emote", wrapArgs(cmdEmote), "Perform a roleplay action.", 0, 0, "me")
	cmdRegistry.Register("shout", wrapArgs(cmdShout), "Shout to everyone in your zone.", 0, 0)
	cmdRegistry.Register("gtell", wrapArgs(cmdGtell), "Send a message to your group.", 0, 0, "gsay")

	// Combat
	cmdRegistry.Register("hit", wrapArgs(cmdHit), "Attack a target.", 0, 0, "attack", "kill")
	cmdRegistry.Register("flee", wrapNoArgs(cmdFlee), "Attempt to flee from combat.", 0, 0)

	// Position / Movement
	cmdRegistry.Register("stand", wrapNoArgs(cmdStand), "Stand up.", 0, 0)
	cmdRegistry.Register("sit", wrapNoArgs(cmdSit), "Sit down.", 0, 0)
	cmdRegistry.Register("rest", wrapNoArgs(cmdRest), "Rest.", 0, 0)
	cmdRegistry.Register("sleep", wrapNoArgs(cmdSleep), "Go to sleep.", 0, 0)
	cmdRegistry.Register("wake", wrapArgs(cmdWake), "Wake up or wake someone else.", 0, 0)

	// Items
	cmdRegistry.Register("inventory", wrapArgs(cmdInventory), "Show your inventory.", 0, 0, "i", "inv")
	cmdRegistry.Register("equipment", wrapArgs(cmdEquipment), "Show your equipped items.", 0, 0, "eq")
	cmdRegistry.Register("wear", wrapArgs(cmdWear), "Wear an item from your inventory.", 0, 0)
	cmdRegistry.Register("remove", wrapArgs(cmdRemove), "Remove an equipped item.", 0, 0)
	cmdRegistry.Register("wield", wrapArgs(cmdWield), "Wield a weapon.", 0, 0)
	cmdRegistry.Register("hold", wrapArgs(cmdHold), "Hold an item.", 0, 0)
	cmdRegistry.Register("get", wrapArgs(cmdGet), "Pick up an item from the room.", 0, 0, "take")
	cmdRegistry.Register("drop", wrapArgs(cmdDrop), "Drop an item from your inventory.", 0, 0)
	cmdRegistry.Register("eat", wrapArgs(cmdEat), "Eat some food.", 0, 0)
	cmdRegistry.Register("drink", wrapArgs(cmdDrink), "Drink from a container.", 0, 0)
	cmdRegistry.Register("quaff", wrapArgs(cmdQuaff), "Quaff a potion.", 0, 0, "q")

	// Info
	cmdRegistry.Register("score", wrapNoArgs(cmdScore), "Show your character stats.", 0, 0, "sc")
	cmdRegistry.Register("who", wrapNoArgs(cmdWho), "List all online players.", 0, 0)
	cmdRegistry.Register("where", wrapNoArgs(cmdWhere), "Show player locations.", 0, 0)
	cmdRegistry.Register("help", wrapArgs(cmdHelp), "Show available commands or help for a topic.", 0, 0)

	// Group
	cmdRegistry.Register("follow", wrapArgs(cmdFollow), "Follow another player.", 0, 0)
	cmdRegistry.Register("group", wrapArgs(cmdGroup), "Manage your group.", 0, 0, "party")
	cmdRegistry.Register("ungroup", wrapArgs(cmdUngroup), "Disband or leave a group.", 0, 0, "disband")

	// Skills (delegated to pkg/command)
	cmdRegistry.Register("skills", wrapSkill(command.CmdSkills), "Show your learned skills.", 0, 0, "sk")
	cmdRegistry.Register("practice", wrapSkill(command.CmdPractice), "Practice a skill.", 0, 0)
	cmdRegistry.Register("learn", wrapSkill(command.CmdLearn), "Learn a new skill.", 0, 0)
	cmdRegistry.Register("listskills", wrapSkill(command.CmdListSkills), "List available skills.", 0, 0, "skills")

	// Shop
	cmdRegistry.Register("list", wrapArgs(cmdList), "List items for sale at a shop.", 0, 0)
	cmdRegistry.Register("buy", wrapArgs(cmdBuy), "Buy an item from a shop.", 0, 0)
	cmdRegistry.Register("sell", wrapArgs(cmdSell), "Sell an item to a shop.", 0, 0)
	cmdRegistry.Register("forget", wrapSkill(command.CmdForget), "Forget a skill.", 0, 0)
	cmdRegistry.Register("confirm", wrapSkill(command.CmdConfirmForget), "Confirm forgetting a skill.", 0, 0, "confirm forget")
	cmdRegistry.Register("use", wrapSkill(command.CmdUseSkill), "Use a skill.", 0, 0)
	cmdRegistry.Register("skillinfo", wrapSkill(command.CmdSkillInfo), "Show info about a skill.", 0, 0, "sinfo")

	// Combat skills (delegated to pkg/command)
	cmdRegistry.Register("backstab", wrapSkill(command.CmdBackstab), "Backstab a target with a piercing weapon.", 0, combat.PosStanding, "bs")
	cmdRegistry.Register("bash", wrapSkill(command.CmdBash), "Bash a target, potentially stunning them.", 0, combat.PosFighting)
	cmdRegistry.Register("kick", wrapSkill(command.CmdKick), "Kick a target for damage.", 0, combat.PosFighting)
	cmdRegistry.Register("trip", wrapSkill(command.CmdTrip), "Trip a target, knocking them down.", 0, combat.PosFighting)
	cmdRegistry.Register("rescue", wrapSkill(command.CmdRescue), "Rescue someone from combat.", 0, combat.PosStanding)
	cmdRegistry.Register("sneak", wrapSkill(command.CmdSneak), "Attempt to move silently.", 0, combat.PosStanding)
	cmdRegistry.Register("hide", wrapSkill(command.CmdHide), "Attempt to hide in the shadows.", 0, combat.PosResting)
	cmdRegistry.Register("steal", wrapSkill(command.CmdSteal), "Steal from a target.", 0, combat.PosStanding)
	cmdRegistry.Register("pick", wrapSkill(command.CmdPickLock), "Pick a lock on a door.", 0, combat.PosStanding, "pick lock")

	// Admin / debug
	cmdRegistry.Register("summon", wrapArgs(cmdSummon), "Summon a player to your room.", 0, 0)

	// Doors
	cmdRegistry.Register("open", wrapArgs(cmdOpen), "Open a door in a direction: open <north|south|east|west|up|down>", 0, 0)
	cmdRegistry.Register("close", wrapArgs(cmdClose), "Close a door in a direction: close <north|south|east|west|up|down>", 0, 0)
	cmdRegistry.Register("lock", wrapArgs(cmdLock), "Lock a door with your key: lock <north|south|east|west|up|down>", 0, 0)
	cmdRegistry.Register("unlock", wrapArgs(cmdUnlock), "Unlock a door with your key: unlock <north|south|east|west|up|down>", 0, 0)
	cmdRegistry.Register("pick", wrapArgs(cmdPick), "Pick a door lock: pick <north|south|east|west|up|down>", 0, 0)

	// Wizard commands
	cmdRegistry.Register("goto", wrapArgs(cmdGoto), "Teleport to a room by VNum.", LVL_IMMORT, 0)
	cmdRegistry.Register("at", wrapArgs(cmdAt), "Execute a command at another room.", LVL_IMMORT, 0)
	cmdRegistry.Register("load", wrapArgs(cmdLoad), "Load a mob or object by VNum.", LVL_IMMORT, 0)
	cmdRegistry.Register("purge", wrapArgs(cmdPurge), "Remove all mobs/items from a room.", LVL_GOD, 0)
	cmdRegistry.Register("teleport", wrapArgs(cmdTeleport), "Teleport another player to a room.", LVL_GOD, 0)
	cmdRegistry.Register("heal", wrapArgs(cmdHeal), "Fully heal a target.", LVL_IMMORT, 0)
	cmdRegistry.Register("restore", wrapArgs(cmdRestore), "Restore all stats of a target.", LVL_IMMORT, 0)
	cmdRegistry.Register("set", wrapArgs(cmdSet), "Set character fields.", LVL_IMMORT, 0)
	cmdRegistry.Register("switch", wrapArgs(cmdSwitch), "Enter another character's body.", LVL_IMMORT, 0)
	cmdRegistry.Register("return", wrapArgs(cmdReturn), "Return from switched body.", LVL_IMMORT, 0)
	cmdRegistry.Register("invis", wrapArgs(cmdInvis), "Become invisible to players.", LVL_IMMORT, 0)
	cmdRegistry.Register("vis", wrapArgs(cmdVis), "Become visible again.", LVL_IMMORT, 0)
	cmdRegistry.Register("gecho", wrapArgs(cmdGecho), "Echo a message to all players.", LVL_GOD, 0)
	cmdRegistry.Register("echo", wrapArgs(cmdEcho), "Echo a message to the room.", LVL_IMMORT, 0)
	cmdRegistry.Register("send", wrapArgs(cmdSend), "Send a message to another character.", LVL_GOD, 0)
	cmdRegistry.Register("force", wrapArgs(cmdForce), "Force a command on another character.", LVL_GRGOD, 0)
	cmdRegistry.Register("shutdown", wrapArgs(cmdShutdown), "Shutdown the server.", LVL_GRGOD, 0)
	cmdRegistry.Register("snoop", wrapArgs(cmdSnoop), "Spy on a player's input.", LVL_GOD, 0)
	cmdRegistry.Register("advance", wrapArgs(cmdAdvance), "Advance a player's level.", LVL_GRGOD, 0)
	cmdRegistry.Register("reload", wrapArgs(cmdReload), "Reload world data.", LVL_GOD, 0)

	// Informative
	cmdRegistry.Register("consider", wrapArgs(cmdConsider), "Compare yourself to a target.", 0, 0, "con")
	cmdRegistry.Register("examine", wrapArgs(cmdExamine), "Examine something in detail.", 0, 0, "exa")
	cmdRegistry.Register("time", wrapArgs(cmdTime), "Show the current time.", 0, 0)
	cmdRegistry.Register("weather", wrapArgs(cmdWeather), "Show the current weather.", 0, 0)
	cmdRegistry.Register("affects", wrapArgs(cmdAffects), "Show active affects.", 0, 0)
	cmdRegistry.Register("autoexit", wrapArgs(cmdAutoExit), "Toggle auto-exit display.", 0, 0)
	cmdRegistry.Register("title", wrapArgs(cmdTitle), "Set your title.", 0, 0)
	cmdRegistry.Register("describe", wrapArgs(cmdDescribe), "Set your description.", 0, 0, "desc")
	cmdRegistry.Register("spells", wrapArgs(cmdSpells), "List known spells.", 0, 0)

	// Quit
	cmdRegistry.Register("quit", wrapNoArgs(cmdQuit), "Quit the game.", 0, 0)
}

// wrapArgs adapts a func(*Session, []string) error to command.Handler.
func wrapArgs(fn func(*Session, []string) error) command.Handler {
	return func(s common.CommandSession, args []string) error {
		return fn(s.(*commandSession).Session, args)
	}
}

// wrapNoArgs adapts a func(*Session) error to command.Handler.
func wrapNoArgs(fn func(*Session) error) command.Handler {
	return func(s common.CommandSession, args []string) error {
		return fn(s.(*commandSession).Session)
	}
}

// wrapMove adapts cmdMove to the registry handler signature.
func wrapMove(direction string) command.Handler {
	return func(s common.CommandSession, args []string) error {
		return cmdMove(s.(*commandSession).Session, direction)
	}
}

// wrapSkill adapts a skill command (which uses command.SessionInterface) to command.Handler.
func wrapSkill(fn func(command.SessionInterface, []string) error) command.Handler {
	return func(s common.CommandSession, args []string) error {
		return fn(s.(*commandSession).Session, args)
	}
}

// ExecuteCommand processes a game command.
func ExecuteCommand(s *Session, cmdStr string, args []string) error {
	cmd := strings.ToLower(cmdStr)

	// Check for mob scripts with oncmd trigger before processing
	// Based on the original MUD's script handling
	if s.player != nil && s.player.GetRoomVNum() > 0 {
		// Get mobs in the room
		mobs := s.manager.world.GetMobsInRoom(s.player.GetRoomVNum())
		fullCommand := cmdStr
		if len(args) > 0 {
			fullCommand = cmdStr + " " + strings.Join(args, " ")
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
					slog.Error("error running oncmd script", "mob_vnum", mob.GetVNum(), "error", err)
				}
				if handled {
					// Script handled the command, don't process further
					return nil
				}
			}
		}
	}

	entry, ok := cmdRegistry.Lookup(cmd)
	if !ok {
		s.sendText(fmt.Sprintf("Unknown command: %s", cmdStr))
		return nil
	}
	return entry.Handler(&commandSession{Session: s}, args)
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
			Doors:       getDoorInfo(s.manager.doorManager, room.VNum, room.Exits),
			Players:     playerNames,
			Items:       itemDescs,
		},
	}

	msg, err := json.Marshal(ServerMessage{
		Type: MsgState,
		Data: state,
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return nil
	}
	s.send <- msg
	return nil
}

// cmdMove moves the player in a direction.
// Also drags followers into the new room.
// Source: act.movement.c do_follow() — followers move when leader moves
func cmdMove(s *Session, direction string) error {
	oldRoom := s.player.GetRoom()

	// Check if a door blocks the exit
	dm := s.manager.doorManager
	if dm != nil {
		canPass, msg := dm.CanPass(oldRoom, direction)
		if !canPass {
			s.sendText(msg)
			return nil
		}
	}

	// Collect followers in this room before moving (cannot query after move holds lock)
	followers := s.manager.world.GetFollowersInRoom(s.player.Name, oldRoom)

	newRoom, err := s.manager.world.MovePlayer(s.player, direction)
	if err != nil {
		s.sendText(fmt.Sprintf("You can't go %s.", direction))
		return nil
	}

	// Notify old room
	leaveMsg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "leave",
			From: s.player.Name,
			Text: fmt.Sprintf("%s leaves %s.", s.player.Name, direction),
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return nil
	}
	s.manager.BroadcastToRoom(oldRoom, leaveMsg, s.player.Name)

	// Notify new room
	enterMsg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "enter",
			From: s.player.Name,
			Text: fmt.Sprintf("%s has arrived.", s.player.Name),
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return nil
	}
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

	// Drag followers into the new room — act.movement.c follower movement
	for _, follower := range followers {
		followerOldRoom := follower.GetRoom()
		if _, ferr := s.manager.world.MovePlayer(follower, direction); ferr == nil {
			follower.SendMessage(fmt.Sprintf("You follow %s %s.\r\n", s.player.Name, direction))
			// Notify follower's old room
			fleaveMsg, err := json.Marshal(ServerMessage{
				Type: MsgEvent,
				Data: EventData{
					Type: "leave",
					From: follower.Name,
					Text: fmt.Sprintf("%s leaves %s.", follower.Name, direction),
				},
			})
			if err != nil {
				slog.Error("json.Marshal error", "error", err)
				continue
			}
			s.manager.BroadcastToRoom(followerOldRoom, fleaveMsg, follower.Name)
			// Notify new room of follower arrival
			fenterMsg, err := json.Marshal(ServerMessage{
				Type: MsgEvent,
				Data: EventData{
					Type: "enter",
					From: follower.Name,
					Text: fmt.Sprintf("%s has arrived.", follower.Name),
				},
			})
			if err != nil {
				slog.Error("json.Marshal error", "error", err)
				continue
			}
			s.manager.BroadcastToRoom(newRoom.VNum, fenterMsg, follower.Name)
			// Send look to follower's session
			if fSess, ok := s.manager.GetSession(follower.Name); ok {
				cmdLook(fSess, nil)
				fSess.markDirty(VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems)
			}
		}
	}

	// Mark room vars dirty for agents after movement
	s.markDirty(VarRoomVnum, VarRoomName, VarRoomExits, VarRoomMobs, VarRoomItems)

	// Send new room state to player
	return cmdLook(s, nil)
}

// cmdSay sends a message to the room.

// cmdQuit handles player logout.
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
	s.manager.world.AddItemToRoom(item, roomVNum)

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
func cmdFollow(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Whom do you wish to follow?")
		return nil
	}

	targetName := args[0]

	// follow self = stop following (act.movement.c line 912–917)
	if strings.EqualFold(targetName, s.player.Name) {
		if s.player.Following == "" {
			s.sendText("You are already following yourself.")
			return nil
		}
		oldLeader := s.player.Following
		s.player.Following = ""
		s.player.InGroup = false // REMOVE_BIT AFF_GROUP — act.movement.c line 926
		s.sendText(fmt.Sprintf("You stop following %s.", oldLeader))
		if leader, ok := s.manager.world.GetPlayer(oldLeader); ok {
			leader.SendMessage(fmt.Sprintf("%s stops following you.\r\n", s.player.Name))
		}
		return nil
	}

	// Find target — get_char_room_vis (act.movement.c line 895)
	target, ok := s.manager.world.GetPlayer(targetName)
	if !ok {
		s.sendText("There is no one by that name here.")
		return nil
	}
	if target.GetRoom() != s.player.GetRoom() {
		s.sendText("They are not here.")
		return nil
	}

	// Already following? (act.movement.c line 904)
	if s.player.Following == target.Name {
		s.sendText(fmt.Sprintf("You are already following %s.", target.Name))
		return nil
	}

	// Stop following previous leader (act.movement.c line 924–925 stop_follower)
	if s.player.Following != "" {
		oldLeader := s.player.Following
		if leader, ok := s.manager.world.GetPlayer(oldLeader); ok {
			leader.SendMessage(fmt.Sprintf("%s stops following you.\r\n", s.player.Name))
		}
	}

	// REMOVE_BIT AFF_GROUP — act.movement.c line 926 (leaving old group when changing leader)
	s.player.Following = target.Name
	s.player.InGroup = false

	// add_follower() — act.movement.c line 948
	s.sendText(fmt.Sprintf("You now follow %s.", target.Name))
	target.SendMessage(fmt.Sprintf("%s now follows you.\r\n", s.player.Name))
	return nil
}

// cmdGroup adds/removes players from a group, or prints group status.
// Source: act.other.c do_group() lines 685–740 and perform_group() lines 624–635
func cmdGroup(s *Session, args []string) error {
	// No args: print group — act.other.c do_group() line 693
	if len(args) == 0 {
		return printGroup(s)
	}

	// Must have no master to enroll others — act.other.c line 699
	if s.player.Following != "" {
		s.sendText("You cannot enroll group members without being head of a group.")
		return nil
	}

	targetName := strings.Join(args, " ")

	// "group all" — act.other.c lines 706–717
	if strings.EqualFold(targetName, "all") {
		s.player.InGroup = true
		found := 0
		for _, f := range s.manager.world.GetFollowersInRoom(s.player.Name, s.player.GetRoom()) {
			if !f.InGroup {
				f.InGroup = true
				s.sendText(fmt.Sprintf("%s is now a member of your group.", f.Name))
				f.SendMessage(fmt.Sprintf("You are now a member of %s's group.\r\n", s.player.Name))
				found++
			}
		}
		if found == 0 {
			s.sendText("Everyone following you here is already in your group.")
		}
		return nil
	}

	// Single target — act.other.c lines 719–738
	target, ok := s.manager.world.GetPlayer(targetName)
	if !ok {
		s.sendText("There is no one by that name here.")
		return nil
	}

	// Target must be following us — act.other.c line 721: vict->master != ch
	// Agent exception: agents auto-follow and auto-accept the invite.
	if target.Following != s.player.Name {
		targetSess, hasSess := s.manager.GetSession(target.Name)
		if hasSess && targetSess.isAgent {
			// Agent auto-follow — mirrors BRENDA accepting an invite
			target.Following = s.player.Name
			target.InGroup = false
			target.SendMessage(fmt.Sprintf("You start following %s.\r\n", s.player.Name))
			s.sendText(fmt.Sprintf("%s starts following you.", target.Name))
		} else {
			s.sendText(fmt.Sprintf("%s must follow you to enter your group.", target.Name))
			return nil
		}
	}

	// Toggle membership — perform_group() / kick-out path (act.other.c lines 726–738)
	if !target.InGroup {
		// perform_group(): SET_BIT AFF_GROUP
		target.InGroup = true
		s.player.InGroup = true // leader is also in the group
		if target.Name != s.player.Name {
			s.sendText(fmt.Sprintf("%s is now a member of your group.", target.Name))
		}
		target.SendMessage(fmt.Sprintf("You are now a member of %s's group.\r\n", s.player.Name))
	} else {
		// Kick out — REMOVE_BIT AFF_GROUP (act.other.c line 737)
		target.InGroup = false
		s.sendText(fmt.Sprintf("%s is no longer a member of your group.", target.Name))
		target.SendMessage(fmt.Sprintf("You have been kicked out of %s's group!\r\n", s.player.Name))
	}
	return nil
}

// printGroup displays the current group composition.
// Source: act.other.c print_group() lines 638–681
func printGroup(s *Session) error {
	if !s.player.InGroup {
		s.sendText("But you are not the member of a group!")
		return nil
	}

	leaderName := s.player.Name
	if s.player.Following != "" {
		leaderName = s.player.Following
	}

	leader, ok := s.manager.world.GetPlayer(leaderName)
	if !ok {
		s.sendText("Your group leader is not online.")
		return nil
	}

	var sb strings.Builder
	sb.WriteString("Your group consists of:\r\n")
	if leader.InGroup {
		sb.WriteString(fmt.Sprintf("     [%3dH %3dM] [%2d] %s (Head of group)\r\n",
			leader.Health, leader.Mana, leader.Level, leader.Name))
	}
	for _, m := range s.manager.world.GetGroupMembers(leaderName) {
		if m.Name == leaderName {
			continue // already printed above
		}
		sb.WriteString(fmt.Sprintf("     [%3dH %3dM] [%2d] %s\r\n",
			m.Health, m.Mana, m.Level, m.Name))
	}
	s.sendText(sb.String())
	return nil
}

// cmdUngroup removes a player from the group or disbands the entire group.
// Source: act.other.c do_ungroup() lines 744–794
func cmdUngroup(s *Session, args []string) error {
	// No args: disband if leader — act.other.c lines 752–770
	if len(args) == 0 {
		if s.player.Following != "" || !s.player.InGroup {
			s.sendText("But you lead no group!")
			return nil
		}
		disbandMsg := fmt.Sprintf("%s has disbanded the group.\r\n", s.player.Name)
		for _, m := range s.manager.world.GetGroupMembers(s.player.Name) {
			if m.Name == s.player.Name {
				continue
			}
			m.InGroup = false
			m.Following = "" // stop_follower — act.other.c line 764
			m.SendMessage(disbandMsg)
		}
		s.player.InGroup = false
		s.sendText("You disband the group.")
		return nil
	}

	// Remove specific member — act.other.c lines 772–793
	targetName := strings.Join(args, " ")
	target, ok := s.manager.world.GetPlayer(targetName)
	if !ok {
		s.sendText("There is no such person!")
		return nil
	}
	if target.Following != s.player.Name {
		s.sendText("That person is not following you!")
		return nil
	}
	if !target.InGroup {
		s.sendText("That person isn't in your group.")
		return nil
	}

	target.InGroup = false
	target.Following = "" // stop_follower — act.other.c line 793
	s.sendText(fmt.Sprintf("%s is no longer a member of your group.", target.Name))
	target.SendMessage(fmt.Sprintf("You have been kicked out of %s's group!\r\n", s.player.Name))
	return nil
}

// cmdGtell sends a message to all group members.
// Source: act.comm.c do_gsay() lines 824–870 (registered as "gtell" in interpreter.c line 484)
func cmdGtell(s *Session, args []string) error {
	if !s.player.InGroup {
		s.sendText("But you are not the member of a group!")
		return nil
	}
	if len(args) == 0 {
		s.sendText("Yes, but WHAT do you want to group-say?")
		return nil
	}

	text := strings.Join(args, " ")
	broadcastMsg := fmt.Sprintf("%s tells the group, '%s'\r\n", s.player.Name, text)

	// Find leader — act.comm.c do_gsay() line 838–841
	leaderName := s.player.Name
	if s.player.Following != "" {
		leaderName = s.player.Following
	}

	// Send to leader if not self (act.comm.c lines 846–851)
	if leaderName != s.player.Name {
		if leader, ok := s.manager.world.GetPlayer(leaderName); ok && leader.InGroup {
			leader.SendMessage(broadcastMsg)
		}
	}

	// Send to all group followers excluding self (act.comm.c lines 852–858)
	for _, f := range s.manager.world.GetFollowers(leaderName) {
		if f.InGroup && f.Name != s.player.Name {
			f.SendMessage(broadcastMsg)
		}
	}

	// Confirm to sender — act.comm.c line 862–865
	s.sendText(fmt.Sprintf("You tell the group, '%s'", text))
	return nil
}

// sendText sends a simple text message to the player.
func (s *Session) sendText(text string) {
	msg, err := json.Marshal(ServerMessage{
		Type: MsgText,
		Data: TextData{Text: text},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	select {
	case s.send <- msg:
	default:
	}
}

// cmdScore shows the player's stats.
// Source: act.informative.c do_score() lines 1168-1451
func cmdScore(s *Session) error {
	p := s.player
	if p == nil {
		return nil
	}
	s.Send(fmt.Sprintf("Name: %s  Level: %d  XP: %d/%d", p.Name, p.Level, p.Exp, 1000))
	s.Send(fmt.Sprintf("HP: %d/%d  Mana: %d/%d  Move: %d/%d", p.Health, p.MaxHealth, p.Mana, p.MaxMana, p.Move, p.MaxMove))
	s.Send(fmt.Sprintf("STR:%d  INT:%d  WIS:%d  DEX:%d  CON:%d  CHA:%d", p.Stats.Str, p.Stats.Int, p.Stats.Wis, p.Stats.Dex, p.Stats.Con, p.Stats.Cha))
	s.Send(fmt.Sprintf("AC:%d  Hitroll:%d  Damroll:%d  Align:%d  Gold:%d", p.AC, p.Hitroll, p.Damroll, p.Alignment, p.Gold))
	return nil
}
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

// cmdEmote broadcasts a roleplay action to the room.
// Source: act.comm.c do_emote() — "$n laughs." style

// cmdShout broadcasts a message to all players in the same zone.
// Source: act.comm.c do_gen_comm() SCMD_SHOUT lines 1286-1289
// Original: zone-scoped; receivers must be POS_RESTING or higher.

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

// cmdSummon pulls a named player into your current room. Debug/admin convenience.
func cmdSummon(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Summon who?")
		return nil
	}
	targetName := strings.ToLower(args[0])
	s.manager.mu.RLock()
	defer s.manager.mu.RUnlock()
	for _, sess := range s.manager.sessions {
		if sess.player == nil {
			continue
		}
		if strings.ToLower(sess.player.Name) == targetName {
			old := sess.player.RoomVNum
			sess.player.RoomVNum = s.player.RoomVNum
			s.sendText(fmt.Sprintf("%s materializes before you.", sess.player.Name))
			sess.sendText(fmt.Sprintf("You are summoned by %s.", s.player.Name))
			_ = old
			return nil
		}
	}
	s.sendText("No one by that name online.")
	return nil
}

// cmdHelp provides a basic help stub.
// Full implementation deferred to a later phase.
func cmdHelp(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Available commands: look, north/south/east/west/up/down, say, hit, flee, " +
			"inventory, equipment, wear, remove, wield, hold, get, drop, " +
			"score, who, tell, emote, shout, where, quit, " +
			"open, close, lock, unlock, pick\n" +
			"Type 'help <topic>' for more info (stub — full help coming later).")
		return nil
	}
	s.sendText(fmt.Sprintf("No help available for '%s' yet.", strings.Join(args, " ")))
	return nil
}

// directions maps abbreviated direction names to full names.
var directions = map[string]string{
	"north": "north", "n": "north",
	"east":  "east",  "e": "east",
	"south": "south", "s": "south",
	"west":  "west",  "w": "west",
	"up":    "up",    "u": "up",
	"down":  "down",  "d": "down",
}

// resolveDirection returns the full direction name or empty string if invalid.
func resolveDirection(input string) string {
	if dir, ok := directions[input]; ok {
		return dir
	}
	return ""
}

// doorBroadcast sends a door-related message to all players in the same room, excluding the actor.
func doorBroadcast(s *Session, message string) {
	if s.player == nil {
		return
	}
	roomVNum := s.player.GetRoom()
	msg, err := json.Marshal(ServerMessage{
		Type: MsgEvent,
		Data: EventData{
			Type: "door",
			Text: message,
		},
	})
	if err != nil {
		slog.Error("json.Marshal error", "error", err)
		return
	}
	s.manager.BroadcastToRoom(roomVNum, msg, s.player.Name)
}

// playerHasKey checks if the player has an item with the given VNum in their inventory.
func playerHasKey(s *Session, keyVNum int) bool {
	if s.player == nil {
		return false
	}
	inv := s.player.Inventory
	if inv == nil {
		return false
	}
	for _, item := range inv.Items {
		if item.VNum == keyVNum {
			return true
		}
	}
	return false
}

// getDoorManager returns the DoorManager from the world.
func getDoorManager(s *Session) *systems.DoorManager {
	if s.manager == nil {
		return nil
	}
	return s.manager.doorManager
}

// CmdOpen handles 'open <direction>' — open a closed door.
// Source: act.door.c do_open()
func cmdOpen(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Open what?")
		return nil
	}

	dir := resolveDirection(strings.ToLower(args[0]))
	if dir == "" {
		s.sendText("Open what?  Try north, south, east, west, up, or down.")
		return nil
	}

	dm := getDoorManager(s)
	if dm == nil {
		s.sendText("You can't do that right now.")
		return nil
	}

	roomVNum := s.player.GetRoomVNum()
	success, msg := dm.OpenDoor(roomVNum, dir)
	if !success {
		s.sendText(msg)
		return nil
	}

	s.sendText(msg)
	doorBroadcast(s, fmt.Sprintf("%s opens the %s door.", s.player.Name, dir))
	return nil
}

// CmdClose handles 'close <direction>' — close an open door.
// Source: act.door.c do_close()
func cmdClose(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Close what?")
		return nil
	}

	dir := resolveDirection(strings.ToLower(args[0]))
	if dir == "" {
		s.sendText("Close what?  Try north, south, east, west, up, or down.")
		return nil
	}

	dm := getDoorManager(s)
	if dm == nil {
		s.sendText("You can't do that right now.")
		return nil
	}

	roomVNum := s.player.GetRoomVNum()
	success, msg := dm.CloseDoor(roomVNum, dir)
	if !success {
		s.sendText(msg)
		return nil
	}

	s.sendText(msg)
	doorBroadcast(s, fmt.Sprintf("%s closes the %s door.", s.player.Name, dir))
	return nil
}

// CmdLock handles 'lock <direction>' — lock a closed door with the correct key.
// Source: act.door.c do_lock()
func cmdLock(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Lock what?")
		return nil
	}

	dir := resolveDirection(strings.ToLower(args[0]))
	if dir == "" {
		s.sendText("Lock what?  Try north, south, east, west, up, or down.")
		return nil
	}

	dm := getDoorManager(s)
	if dm == nil {
		s.sendText("You can't do that right now.")
		return nil
	}

	roomVNum := s.player.GetRoomVNum()
	door, ok := dm.GetDoor(roomVNum, dir)
	if !ok {
		s.sendText("There is no door there.")
		return nil
	}

	if door.KeyVNum == -1 {
		s.sendText("This door doesn't require a key.")
		return nil
	}

	if !playerHasKey(s, door.KeyVNum) {
		s.sendText("You don't have the right key.")
		return nil
	}

	success, msg := dm.LockDoor(roomVNum, dir, door.KeyVNum)
	if !success {
		s.sendText(msg)
		return nil
	}

	s.sendText(msg)
	doorBroadcast(s, fmt.Sprintf("%s locks the %s door.", s.player.Name, dir))
	return nil
}

// CmdUnlock handles 'unlock <direction>' — unlock a locked door with the correct key.
// Source: act.door.c do_unlock()
func cmdUnlock(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Unlock what?")
		return nil
	}

	dir := resolveDirection(strings.ToLower(args[0]))
	if dir == "" {
		s.sendText("Unlock what?  Try north, south, east, west, up, or down.")
		return nil
	}

	dm := getDoorManager(s)
	if dm == nil {
		s.sendText("You can't do that right now.")
		return nil
	}

	roomVNum := s.player.GetRoomVNum()
	door, ok := dm.GetDoor(roomVNum, dir)
	if !ok {
		s.sendText("There is no door there.")
		return nil
	}

	if door.KeyVNum == -1 {
		s.sendText("This door doesn't require a key.")
		return nil
	}

	if !playerHasKey(s, door.KeyVNum) {
		s.sendText("You don't have the right key.")
		return nil
	}

	success, msg := dm.UnlockDoor(roomVNum, dir, door.KeyVNum)
	if !success {
		s.sendText(msg)
		return nil
	}

	s.sendText(msg)
	doorBroadcast(s, fmt.Sprintf("%s unlocks the %s door.", s.player.Name, dir))
	return nil
}

// CmdPick handles 'pick <direction>' — pick a locked door's lock.
// Uses the player's Dexterity and thief/assassin level to determine success.
// Source: act.door.c do_pick()
func cmdPick(s *Session, args []string) error {
	if len(args) == 0 {
		s.sendText("Pick what?")
		return nil
	}

	dir := resolveDirection(strings.ToLower(args[0]))
	if dir == "" {
		s.sendText("Pick what?  Try north, south, east, west, up, or down.")
		return nil
	}

	dm := getDoorManager(s)
	if dm == nil {
		s.sendText("You can't do that right now.")
		return nil
	}

	roomVNum := s.player.GetRoomVNum()

	// Calculate pick skill based on class and level
	// Thieves (class 2) and Assassins (class 6) get bonus
	// Base skill = level * 5 + DEX bonus
	classBonus := 0
	if s.player.Class == 2 || s.player.Class == 6 {
		classBonus = 20 // thieves/assassins are naturally better
	}
	skill := s.player.Level*5 + s.player.Stats.Dex + classBonus
	if skill > 95 {
		skill = 95 // cap at 95% — there's always a chance of failure
	}

	success, msg := dm.PickDoor(roomVNum, dir, skill)
	if !success {
		s.sendText(msg)
		return nil
	}

	s.sendText(msg)
	doorBroadcast(s, fmt.Sprintf("%s picks the lock on the %s door.", s.player.Name, dir))
	return nil
}
