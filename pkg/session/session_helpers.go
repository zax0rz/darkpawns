// Package session manages WebSocket connections and player sessions.
package session

import (
	"github.com/zax0rz/darkpawns/pkg/parser"
	"github.com/zax0rz/darkpawns/pkg/game/systems"
)

func getExitNames(exits map[string]parser.Exit) []string {
	var names []string
	for dir := range exits {
		names = append(names, dir)
	}
	return names
}

func getDoorInfo(dm *systems.DoorManager, roomVNum int, exits map[string]parser.Exit) []DoorInfo {
	if dm == nil {
		return nil
	}
	var doors []DoorInfo
	for dir := range exits {
		door, ok := dm.GetDoor(roomVNum, dir)
		if !ok {
			continue
		}
		if !door.CanSee() {
			continue
		}
		doors = append(doors, DoorInfo{
			Direction: dir,
			Closed:    door.Closed,
			Locked:    door.Locked,
		})
	}
	if len(doors) == 0 {
		return nil
	}
	return doors
}

// GetPlayer returns the player associated with this session
