package session

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/commands"
)

// ExecuteCommand processes a game command.
func ExecuteCommand(s *Session, command string, args []string) error {
	cmd := strings.ToLower(command)

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
		return commands.ExecuteItemCommand(s, cmd, args)
	case "equipment", "eq":
		return commands.ExecuteItemCommand(s, cmd, args)
	case "wear":
		return commands.ExecuteItemCommand(s, cmd, args)
	case "remove":
		return commands.ExecuteItemCommand(s, cmd, args)
	case "wield":
		return commands.ExecuteItemCommand(s, cmd, args)
	case "hold":
		return commands.ExecuteItemCommand(s, cmd, args)
	case "get", "take":
		return commands.ExecuteItemCommand(s, cmd, args)
	case "drop":
		return commands.ExecuteItemCommand(s, cmd, args)
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