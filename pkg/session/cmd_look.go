package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
)

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

// cmdMove moves the player in a direction.
// Also drags followers into the new room.
// Source: act.movement.c do_follow() — followers move when leader moves
