package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

func cmdLook(s *Session, args []string) error {
	room, ok := s.manager.world.GetRoom(s.player.GetRoom())
	if !ok {
		return fmt.Errorf("invalid room")
	}

	arg := strings.Join(args, " ")
	arg = strings.TrimSpace(strings.ToLower(arg))

	// "look in <container>" — show container contents
	if strings.HasPrefix(arg, "in ") {
		containerName := strings.TrimSpace(arg[3:])
		return cmdLookIn(s, containerName)
	}

	// "look <direction>" — look through an exit
	if dirName := parseDirection(arg); dirName != "" {
		return cmdLookDirection(s, room, dirName)
	}

	// "look <target>" — look at a mob, player, or item
	if arg != "" {
		return cmdLookAt(s, room, arg)
	}

	// Bare "look" — room view
	// Dark room check — C source: utils.h IS_DARK()
	if s.manager.world.IsRoomDark(room.VNum) && !s.playerCanSeeInDark() {
		s.sendText("It is pitch black...")
		return nil
	}

	// Get other players in room
	players := s.manager.world.GetPlayersInRoom(room.VNum)
	var playerNames []string
	for _, p := range players {
		if p.Name != s.player.Name {
			playerNames = append(playerNames, p.Name)
		}
	}

	// Get mobs in room
	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	var mobDescs []string
	for _, mob := range mobs {
		desc := mob.GetShortDesc()
		if mob.Fighting {
			desc += " is here, fighting " + mob.FightingTarget
		} else {
			desc += " is here."
		}
		mobDescs = append(mobDescs, desc)
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
			Mobs:        mobDescs,
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

// cmdLookIn shows the contents of a container or corpse.
func cmdLookIn(s *Session, containerName string) error {
	if containerName == "" {
		s.sendText("Look in what?")
		return nil
	}

	container := findContainerByName(s, containerName)
	if container == nil {
		s.sendText(fmt.Sprintf("You don't see %s here.", containerName))
		return nil
	}

	if game.IsContainerClosed(container) {
		s.sendText(fmt.Sprintf("%s is closed.", container.GetShortDesc()))
		return nil
	}

	contents := container.GetContents()
	if len(contents) == 0 {
		s.sendText(fmt.Sprintf("%s is empty.", container.GetShortDesc()))
		return nil
	}

	s.sendText(fmt.Sprintf("%s contains:", container.GetShortDesc()))
	for _, item := range contents {
		s.sendText("  " + item.GetShortDesc())
	}
	return nil
}

// cmdLookDirection shows what's visible through an exit.
func cmdLookDirection(s *Session, room *parser.Room, dir string) error {
	exit, ok := room.Exits[dir]
	if !ok {
		s.sendText("Alas, you cannot go that way...")
		return nil
	}

	if exit.Description != "" {
		s.sendText(exit.Description)
	}

	// If the exit is open, show a peek into the next room
	isOpen := true
	if s.manager.doorManager != nil {
		canPass, _ := s.manager.doorManager.CanPass(room.VNum, dir)
		isOpen = canPass
	}
	if isOpen {
		if nextRoom, ok := s.manager.world.GetRoom(exit.ToRoom); ok {
			if !s.manager.world.IsRoomDark(exit.ToRoom) || s.playerCanSeeInDark() {
				s.sendText(fmt.Sprintf("Through the %s you see %s.", dir, nextRoom.Name))
			} else {
				s.sendText(fmt.Sprintf("Through the %s you see nothing but darkness.", dir))
			}
		}
	}

	return nil
}

// cmdLookAt looks at a specific mob, player, or item.
func cmdLookAt(s *Session, room *parser.Room, targetName string) error {
	// Check players
	players := s.manager.world.GetPlayersInRoom(room.VNum)
	for _, p := range players {
		if strings.EqualFold(p.Name, targetName) || strings.Contains(strings.ToLower(p.Name), strings.ToLower(targetName)) {
			desc := p.Name
			if p.Title != "" {
				desc = p.Name + " " + p.Title
			}
			if p.Description != "" {
				desc += "\n" + p.Description
			}
			desc += "\n" + p.Name + " has:"
			for _, item := range p.Equipment.GetEquippedItems() {
				desc += "\n  " + item.GetShortDesc()
			}
			s.sendText(desc)
			return nil
		}
	}

	// Check mobs
	mobs := s.manager.world.GetMobsInRoom(room.VNum)
	for _, mob := range mobs {
		mobName := mob.GetShortDesc()
		if strings.Contains(strings.ToLower(mobName), strings.ToLower(targetName)) {
			longDesc := mob.GetLongDesc()
			if longDesc != "" {
				s.sendText(longDesc)
			}
			return nil
		}
	}

	// Check items in room
	items := s.manager.world.GetItemsInRoom(room.VNum)
	for _, item := range items {
		if strings.Contains(strings.ToLower(item.GetShortDesc()), strings.ToLower(targetName)) ||
			strings.Contains(strings.ToLower(item.GetKeywords()), strings.ToLower(targetName)) {
			if item.GetLongDesc() != "" {
				s.sendText(item.GetLongDesc())
			} else {
				s.sendText(item.GetShortDesc())
			}
			extraDesc := item.GetExtraDesc(targetName)
			if extraDesc != "" {
				s.sendText(extraDesc)
			}
			return nil
		}
	}

	// Check items in inventory
	for _, item := range s.player.Inventory.Items {
		if strings.Contains(strings.ToLower(item.GetShortDesc()), strings.ToLower(targetName)) ||
			strings.Contains(strings.ToLower(item.GetKeywords()), strings.ToLower(targetName)) {
			s.sendText(item.GetShortDesc())
			extraDesc := item.GetExtraDesc(targetName)
			if extraDesc != "" {
				s.sendText(extraDesc)
			}
			return nil
		}
	}

	s.sendText("You don't see that here.")
	return nil
}

// playerCanSeeInDark checks if the player can see in darkness.
func (s *Session) playerCanSeeInDark() bool {
	if s.player.GetLevel() > 30 {
		return true
	}
	// TODO: Check AFF_INFRAVISION affect, light-producing items
	return false
}

// findContainerByName searches inventory then room for a container.
func findContainerByName(s *Session, name string) *game.ObjectInstance {
	for _, item := range s.player.Inventory.Items {
		if item.IsContainer() && isnameMatch(name, item.GetKeywords()) {
			return item
		}
	}
	items := s.manager.world.GetItemsInRoom(s.player.GetRoom())
	for _, item := range items {
		if item.IsContainer() && isnameMatch(name, item.GetKeywords()) {
			return item
		}
	}
	return nil
}

// isnameMatch checks if a name matches a keyword (prefix match on space-separated keywords).
func isnameMatch(name, keywords string) bool {
	name = strings.ToLower(name)
	kwList := strings.Split(strings.ToLower(keywords), " ")
	for _, kw := range kwList {
		if kw != "" && strings.HasPrefix(kw, name) {
			return true
		}
	}
	return false
}

// parseDirection maps a direction name/alias to its canonical form.
func parseDirection(dir string) string {
	switch strings.ToLower(dir) {
	case "n", "north":
		return "north"
	case "s", "south":
		return "south"
	case "e", "east":
		return "east"
	case "w", "west":
		return "west"
	case "u", "up":
		return "up"
	case "d", "down":
		return "down"
	default:
		return ""
	}
}
