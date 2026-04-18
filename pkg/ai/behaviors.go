// Package ai implements mob AI behaviors.
package ai

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// Mob defines the interface that game.Mob must implement for AI.
type Mob interface {
	HasFlag(flag string) bool
	Attack(player *game.Player, world *game.World) error
	GetRoom() int
	SetRoom(vnum int)
	GetShortDesc() string
}

// Behavior defines the interface for mob AI behaviors.
type Behavior interface {
	// Act performs an action for the mob based on its current state.
	Act(mob Mob, world *game.World) error
	// Name returns the name of the behavior.
	Name() string
}

// AggressiveBehavior attacks players on sight.
type AggressiveBehavior struct{}

func (b *AggressiveBehavior) Name() string {
	return "aggressive"
}

func (b *AggressiveBehavior) Act(mob Mob, world *game.World) error {
	// Check if we already have a target
	if mob.Brain.Target != "" {
		// Check if target is still in the same room
		players := world.GetPlayersInRoom(mob.RoomVNum)
		for _, player := range players {
			if player.Name == mob.Brain.Target {
				// Attack the target
				return mob.Attack(player, world)
			}
		}
		// Target left the room, clear it
		mob.Brain.Target = ""
		return nil
	}

	// Look for players in the same room
	players := world.GetPlayersInRoom(mob.RoomVNum)
	if len(players) == 0 {
		return nil // No players to attack
	}

	// Pick a random player to attack
	target := players[rand.Intn(len(players))]
	mob.Brain.Target = target.Name
	return mob.Attack(target, world)
}

// WanderingBehavior moves randomly between rooms.
type WanderingBehavior struct {
	lastMove time.Time
}

func (b *WanderingBehavior) Name() string {
	return "wandering"
}

func (b *WanderingBehavior) Act(mob Mob, world *game.World) error {
	// Don't move if we're a sentinel
	if mob.HasFlag(MOB_SENTINEL) {
		return nil
	}

	// Don't move too frequently
	now := time.Now()
	if now.Sub(b.lastMove) < 30*time.Second {
		return nil
	}

	// Get current room
	room, ok := world.GetRoom(mob.RoomVNum)
	if !ok {
		return fmt.Errorf("mob in invalid room %d", mob.RoomVNum)
	}

	// Get available exits
	var exits []string
	for dir := range room.Exits {
		exits = append(exits, dir)
	}

	if len(exits) == 0 {
		return nil // No exits to move through
	}

	// Pick a random exit
	direction := exits[rand.Intn(len(exits))]
	exit := room.Exits[direction]

	// Move the mob
	mob.RoomVNum = exit.ToRoom
	b.lastMove = now

	// Notify players in the new room
	players := world.GetPlayersInRoom(mob.RoomVNum)
	for _, player := range players {
		player.Send <- []byte(fmt.Sprintf("%s wanders in from %s.\n", mob.ShortDesc, direction))
	}

	return nil
}

// SentinelBehavior stays in place and does nothing.
type SentinelBehavior struct{}

func (b *SentinelBehavior) Name() string {
	return "sentinel"
}

func (b *SentinelBehavior) Act(mob Mob, world *game.World) error {
	// Sentinels don't do anything on their own
	return nil
}

// Mob flags constants
const (
	MOB_SENTINEL   = "SENTINEL"
	MOB_AGGRESSIVE = "AGGRESSIVE"
	MOB_HUNTER     = "HUNTER"
)