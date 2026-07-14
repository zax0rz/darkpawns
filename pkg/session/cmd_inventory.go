package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

func cmdQuit(s *Session) error {
	if s.player.IsFighting() {
		s.sendText("No way!  You are fighting!")
		return nil
	}

	room := s.player.GetRoom()
	if s.manager.world.RoomHasFlag(room, "death") {
		s.sendText("You cannot quit from this room!")
		return nil
	}

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
	s.Close()

	return nil
}

// cmdInventory shows the player's inventory.
// Delegates to the game-layer DoInventory for C-faithful formatting.
func cmdInventory(s *Session, args []string) error {
	s.manager.world.DoInventory(s.player)
	return nil
}

// cmdEquipment shows the player's equipped items.
// Delegates to the game-layer DoEquipment for C-faithful formatting.
func cmdEquipment(s *Session, args []string) error {
	s.manager.world.DoEquipment(s.player)
	return nil
}

// cmdWear equips an item from inventory.
func cmdWear(s *Session, args []string) error {
	s.manager.world.DoWear(s.player, strings.Join(args, " "))
	s.markDirty(VarInventory, VarEquipment)
	return nil
}

// cmdRemove unequips an item.
func cmdRemove(s *Session, args []string) error {
	s.manager.world.DoRemove(s.player, strings.Join(args, " "))
	s.markDirty(VarInventory, VarEquipment)
	return nil
}

// cmdWield equips a weapon.
func cmdWield(s *Session, args []string) error {
	s.manager.world.DoWield(s.player, strings.Join(args, " "))
	s.markDirty(VarInventory, VarEquipment)
	return nil
}

// cmdHold holds an item.
func cmdHold(s *Session, args []string) error {
	s.manager.world.DoGrab(s.player, strings.Join(args, " "))
	s.markDirty(VarInventory, VarEquipment)
	return nil
}

// cmdGet picks up an item from the room, container, or corpse.
// Delegates to game-layer doGet which handles:
//
//	get <item>, get all, get all.<item>, get <item> <container>
func cmdGet(s *Session, args []string) error {
	arg := strings.Join(args, " ")
	s.manager.world.DoGet(s.player, arg)
	s.markDirty(VarInventory, VarRoomItems)
	return nil
}

// cmdGive gives an item or gold to another character.
// Delegates to game-layer doGive which handles:
//
//	give <item> <player>, give <N> coins <player>, give <N> <player>
func cmdGive(s *Session, args []string) error {
	arg := strings.Join(args, " ")
	s.manager.world.DoGive(s.player, arg)
	s.markDirty(VarInventory)
	return nil
}

// cmdPut puts an item into a container.
// Delegates to game-layer doPut which handles:
//
//	put <item> <container>, put all <container>, put all.<item> <container>
func cmdPut(s *Session, args []string) error {
	arg := strings.Join(args, " ")
	s.manager.world.DoPut(s.player, arg)
	s.markDirty(VarInventory, VarRoomItems)
	return nil
}

// cmdDrop drops an item from inventory.
func cmdDrop(s *Session, args []string) error {
	s.manager.world.DoDrop(s.player, strings.Join(args, " "))
	s.markDirty(VarInventory, VarRoomItems)
	return nil
}

// cmdJunk destroys carried item(s) or gold for a small experience reward.
// Delegates to game-layer DoJunk which handles:
//
//	junk <item>, junk all.<item>, junk <N> coins
func cmdJunk(s *Session, args []string) error {
	arg := strings.Join(args, " ")
	s.manager.world.DoJunk(s.player, arg)
	s.markDirty(VarInventory, VarRoomItems)
	return nil
}

// cmdDonate donates carried item(s) or gold to the donation room.
// Delegates to game-layer DoDonate which handles:
//
//	donate <item>, donate all.<item>, donate <N> coins
func cmdDonate(s *Session, args []string) error {
	arg := strings.Join(args, " ")
	s.manager.world.DoDonate(s.player, arg)
	s.markDirty(VarInventory, VarRoomItems)
	return nil
}

// cmdFollow sets the player to follow another player.
// Source: act.movement.c do_follow() lines 883–951
