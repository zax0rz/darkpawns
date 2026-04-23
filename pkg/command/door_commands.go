// Package command provides door-related commands for Dark Pawns.
package command

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/common"
	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/game/systems"
)

// DoorCommands provides door-related command handlers.
type DoorCommands struct {
	doorManager *systems.DoorManager
	world       *game.World
}

// NewDoorCommands creates a new DoorCommands instance.
func NewDoorCommands(dm *systems.DoorManager, w *game.World) *DoorCommands {
	return &DoorCommands{
		doorManager: dm,
		world:       w,
	}
}

// CmdOpen handles the 'open' command.
func (dc *DoorCommands) CmdOpen(s common.CommandSession, args []string) error {
	if len(args) == 0 {
		s.Send("Open what?")
		return nil
	}

	target := strings.ToLower(strings.Join(args, " "))
	roomVNum := s.GetPlayerRoomVNum()

	// Check if target is a direction
	directions := map[string]string{
		"north": "north", "n": "north",
		"east": "east", "e": "east",
		"south": "south", "s": "south",
		"west": "west", "w": "west",
		"up": "up", "u": "up",
		"down": "down", "d": "down",
	}

	if dir, ok := directions[target]; ok {
		// Try to open door in that direction
		success, msg := dc.doorManager.OpenDoor(roomVNum, dir)
		if !success {
			s.Send(msg)
			return nil
		}

		s.Send(msg)

		// Notify room
		dc.broadcastToRoom(s, fmt.Sprintf("%s opens the %s door.", s.GetPlayerName(), dir))
		return nil
	}

	// Check if target is "door" or specific door
	if strings.Contains(target, "door") {
		// Player needs to specify direction
		s.Send("Open which door? (north, south, east, west, up, down)")
		return nil
	}

	s.Send("You don't see that here.")
	return nil
}

// CmdClose handles the 'close' command.
func (dc *DoorCommands) CmdClose(s common.CommandSession, args []string) error {
	if len(args) == 0 {
		s.Send("Close what?")
		return nil
	}

	target := strings.ToLower(strings.Join(args, " "))
	roomVNum := s.GetPlayerRoomVNum()

	// Check if target is a direction
	directions := map[string]string{
		"north": "north", "n": "north",
		"east": "east", "e": "east",
		"south": "south", "s": "south",
		"west": "west", "w": "west",
		"up": "up", "u": "up",
		"down": "down", "d": "down",
	}

	if dir, ok := directions[target]; ok {
		// Try to close door in that direction
		success, msg := dc.doorManager.CloseDoor(roomVNum, dir)
		if !success {
			s.Send(msg)
			return nil
		}

		s.Send(msg)

		// Notify room
		dc.broadcastToRoom(s, fmt.Sprintf("%s closes the %s door.", s.GetPlayerName(), dir))
		return nil
	}

	// Check if target is "door" or specific door
	if strings.Contains(target, "door") {
		// Player needs to specify direction
		s.Send("Close which door? (north, south, east, west, up, down)")
		return nil
	}

	s.Send("You don't see that here.")
	return nil
}

// CmdLock handles the 'lock' command.
func (dc *DoorCommands) CmdLock(s common.CommandSession, args []string) error {
	if len(args) == 0 {
		s.Send("Lock what?")
		return nil
	}

	target := strings.ToLower(strings.Join(args, " "))
	roomVNum := s.GetPlayerRoomVNum()

	// Check if target is a direction
	directions := map[string]string{
		"north": "north", "n": "north",
		"east": "east", "e": "east",
		"south": "south", "s": "south",
		"west": "west", "w": "west",
		"up": "up", "u": "up",
		"down": "down", "d": "down",
	}

	if dir, ok := directions[target]; ok {
		// Check if player has a key
		door, ok := dc.doorManager.GetDoor(roomVNum, dir)
		if !ok {
			s.Send("There is no door there.")
			return nil
		}

		// Find key in player's inventory
		keyVNum := door.KeyVNum
		if keyVNum == -1 {
			s.Send("This door doesn't require a key.")
			return nil
		}

		// Check if player has the key
		hasKey := dc.playerHasItem(s, keyVNum)
		if !hasKey {
			s.Send("You don't have the key.")
			return nil
		}

		// Try to lock the door
		success, msg := dc.doorManager.LockDoor(roomVNum, dir, keyVNum)
		if !success {
			s.Send(msg)
			return nil
		}

		s.Send(msg)

		// Notify room
		dc.broadcastToRoom(s, fmt.Sprintf("%s locks the %s door.", s.GetPlayerName(), dir))
		return nil
	}

	// Check if target is "door" or specific door
	if strings.Contains(target, "door") {
		// Player needs to specify direction
		s.Send("Lock which door? (north, south, east, west, up, down)")
		return nil
	}

	s.Send("You don't see that here.")
	return nil
}

// CmdUnlock handles the 'unlock' command.
func (dc *DoorCommands) CmdUnlock(s common.CommandSession, args []string) error {
	if len(args) == 0 {
		s.Send("Unlock what?")
		return nil
	}

	target := strings.ToLower(strings.Join(args, " "))
	roomVNum := s.GetPlayerRoomVNum()

	// Check if target is a direction
	directions := map[string]string{
		"north": "north", "n": "north",
		"east": "east", "e": "east",
		"south": "south", "s": "south",
		"west": "west", "w": "west",
		"up": "up", "u": "up",
		"down": "down", "d": "down",
	}

	if dir, ok := directions[target]; ok {
		// Check if player has a key
		door, ok := dc.doorManager.GetDoor(roomVNum, dir)
		if !ok {
			s.Send("There is no door there.")
			return nil
		}

		// Find key in player's inventory
		keyVNum := door.KeyVNum
		if keyVNum == -1 {
			s.Send("This door doesn't require a key.")
			return nil
		}

		// Check if player has the key
		hasKey := dc.playerHasItem(s, keyVNum)
		if !hasKey {
			s.Send("You don't have the key.")
			return nil
		}

		// Try to unlock the door
		success, msg := dc.doorManager.UnlockDoor(roomVNum, dir, keyVNum)
		if !success {
			s.Send(msg)
			return nil
		}

		s.Send(msg)

		// Notify room
		dc.broadcastToRoom(s, fmt.Sprintf("%s unlocks the %s door.", s.GetPlayerName(), dir))
		return nil
	}

	// Check if target is "door" or specific door
	if strings.Contains(target, "door") {
		// Player needs to specify direction
		s.Send("Unlock which door? (north, south, east, west, up, down)")
		return nil
	}

	s.Send("You don't see that here.")
	return nil
}

// CmdPick handles the 'pick' command for picking locks.
func (dc *DoorCommands) CmdPick(s common.CommandSession, args []string) error {
	if len(args) == 0 {
		s.Send("Pick what?")
		return nil
	}

	target := strings.ToLower(strings.Join(args, " "))
	roomVNum := s.GetPlayerRoomVNum()

	// Check if target is a direction
	directions := map[string]string{
		"north": "north", "n": "north",
		"east": "east", "e": "east",
		"south": "south", "s": "south",
		"west": "west", "w": "west",
		"up": "up", "u": "up",
		"down": "down", "d": "down",
	}

	if dir, ok := directions[target]; ok {
		// For now, use a fixed skill value (would come from player stats)
		skill := 50 // Default skill

		// Try to pick the door
		success, msg := dc.doorManager.PickDoor(roomVNum, dir, skill)
		if !success {
			s.Send(msg)
			return nil
		}

		s.Send(msg)

		// Notify room
		dc.broadcastToRoom(s, fmt.Sprintf("%s picks the lock on the %s door.", s.GetPlayerName(), dir))
		return nil
	}

	// Check if target is "lock" or specific door
	if strings.Contains(target, "lock") || strings.Contains(target, "door") {
		// Player needs to specify direction
		s.Send("Pick which lock? (north, south, east, west, up, down)")
		return nil
	}

	s.Send("You don't see that here.")
	return nil
}

// CmdBash handles the 'bash' command for bashing doors.
func (dc *DoorCommands) CmdBash(s common.CommandSession, args []string) error {
	if len(args) == 0 {
		s.Send("Bash what?")
		return nil
	}

	target := strings.ToLower(strings.Join(args, " "))
	roomVNum := s.GetPlayerRoomVNum()

	// Check if target is a direction
	directions := map[string]string{
		"north": "north", "n": "north",
		"east": "east", "e": "east",
		"south": "south", "s": "south",
		"west": "west", "w": "west",
		"up": "up", "u": "up",
		"down": "down", "d": "down",
	}

	if dir, ok := directions[target]; ok {
		// For now, use a fixed strength value (would come from player stats)
		strength := 70 // Default strength

		// Try to bash the door
		success, msg := dc.doorManager.BashDoor(roomVNum, dir, strength)
		if !success {
			s.Send(msg)
			return nil
		}

		s.Send(msg)

		// Notify room
		dc.broadcastToRoom(s, fmt.Sprintf("%s bashes the %s door.", s.GetPlayerName(), dir))
		return nil
	}

	// Check if target is "door"
	if strings.Contains(target, "door") {
		// Player needs to specify direction
		s.Send("Bash which door? (north, south, east, west, up, down)")
		return nil
	}

	s.Send("You don't see that here.")
	return nil
}

// playerHasItem checks if the player has an item with the given VNum.
func (dc *DoorCommands) playerHasItem(s common.CommandSession, vnum int) bool {
	// This is a simplified check - in reality, we'd check player's inventory
	// For now, return false to require actual implementation
	return false
}

// broadcastToRoom broadcasts a message to all players in the room.
func (dc *DoorCommands) broadcastToRoom(s common.CommandSession, message string) {
	// This would use the session manager's broadcast functionality
	// For now, just log it
	fmt.Printf("[ROOM] %s\n", message)
}

// RegisterCommands registers door commands with the session command handler.
// This would need to be integrated with the existing command system.
func (dc *DoorCommands) RegisterCommands() {
	// In a real implementation, this would register the command handlers
	// with the session's command dispatcher
}
