package session

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func cmdLook(s *Session, args []string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}
	return s.sendObservation(s.manager.world.DoLook(s.player, "look", strings.Join(args, " ")), "")
}

// cmdMovementLook is the one allowed observation/movement cross-cut: C room
// entry honors BRIEF, while an explicit look always ignores it.
func cmdMovementLook(s *Session) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}
	return s.sendObservation(s.manager.world.DoLookRoom(s.player, false), "")
}

func cmdRead(s *Session, args []string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}
	return s.sendObservation(s.manager.world.DoLook(s.player, "read", strings.Join(args, " ")), "")
}

func cmdExamine(s *Session, args []string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}
	return s.sendObservation(s.manager.world.DoExamine(s.player, strings.Join(args, " ")), "")
}

func cmdExits(s *Session, _ []string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}
	return s.sendObservation(s.manager.world.DoExits(s.player), "")
}

func cmdDiagnose(s *Session, args []string) error {
	if s.player == nil {
		return fmt.Errorf("not logged in")
	}
	return s.sendObservation(s.manager.world.DoDiagnose(s.player, strings.Join(args, " ")), "")
}

// sendObservation is the thin dual renderer: game-owned messages travel
// through act(), while RoomView is translated to the unchanged WebSocket
// StateData/RoomState schema. Non-room observations only emit text.
func (s *Session) sendObservation(result game.ObservationResult, token string) error {
	s.manager.world.RenderObservationMessages(result)
	if result.Room == nil {
		return nil
	}

	state := StateData{
		Player: observationPlayerState(s.player),
		Room:   observationRoomState(result.Room),
		Token:  token,
	}
	message, err := json.Marshal(ServerMessage{Type: MsgState, Data: state})
	if err != nil {
		return fmt.Errorf("marshal observation state: %w", err)
	}
	s.send <- message
	return nil
}

func observationPlayerState(player *game.Player) PlayerState {
	if player == nil {
		return PlayerState{}
	}
	return PlayerState{
		Name:      player.Name,
		Health:    player.Health,
		MaxHealth: player.MaxHealth,
		Mana:      player.Mana,
		MaxMana:   player.MaxMana,
		Move:      player.Move,
		MaxMove:   player.MaxMove,
		Gold:      player.Gold,
		Level:     player.Level,
		Class:     game.ClassNames[player.Class],
		Race:      game.RaceNames[player.Race],
		Str:       player.Stats.Str,
		Int:       player.Stats.Int,
		Wis:       player.Stats.Wis,
		Dex:       player.Stats.Dex,
		Con:       player.Stats.Con,
		Cha:       player.Stats.Cha,
	}
}

func observationRoomState(view *game.RoomView) RoomState {
	if view == nil {
		return RoomState{}
	}
	doors := make([]DoorInfo, 0, len(view.Doors))
	for _, door := range view.Doors {
		doors = append(doors, DoorInfo{
			Direction: door.Direction,
			Closed:    door.Closed,
			Locked:    door.Locked,
		})
	}
	return RoomState{
		VNum:        view.VNum,
		Name:        view.Name,
		Description: view.Description,
		Exits:       append([]string(nil), view.Exits...),
		Doors:       doors,
		Players:     append([]string(nil), view.Players...),
		Mobs:        append([]string(nil), view.Mobs...),
		Items:       append([]string(nil), view.Items...),
	}
}

func (s *Session) sendRoomObservation(roomVNum int, ignoreBrief bool, token string) {
	result := s.manager.world.DoLookRoomAt(s.player, roomVNum, ignoreBrief)
	if err := s.sendObservation(result, token); err != nil {
		slog.Error("send room observation", "player", s.playerName, "room", roomVNum, "error", err)
	}
}
