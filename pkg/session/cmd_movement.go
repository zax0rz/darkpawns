package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

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
// #nosec G104
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
// #nosec G104
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
