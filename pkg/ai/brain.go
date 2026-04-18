// Package ai implements mob AI behaviors.
package ai

import (
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// Brain manages a mob's AI state.
type Brain struct {
	// Current behavior
	Behavior Behavior
	
	// Target player (if any)
	Target string
	
	// Last time AI was updated
	LastUpdate time.Time
	
	// AI tick interval
	TickInterval time.Duration
}

// NewBrain creates a new brain with default behavior based on mob flags.
func NewBrain(mob Mob) *Brain {
	brain := &Brain{
		LastUpdate:   time.Now(),
		TickInterval: 10 * time.Second,
	}
	
	// Determine initial behavior based on mob flags
	if mob.HasFlag(MOB_SENTINEL) {
		brain.Behavior = &SentinelBehavior{}
	} else if mob.HasFlag(MOB_AGGRESSIVE) {
		brain.Behavior = &AggressiveBehavior{}
	} else {
		brain.Behavior = &WanderingBehavior{}
	}
	
	return brain
}

// Update runs the mob's AI logic.
func (b *Brain) Update(mob Mob, world *game.World) error {
	now := time.Now()
	
	// Check if it's time for an AI tick
	if now.Sub(b.LastUpdate) < b.TickInterval {
		return nil
	}
	
	// Update the behavior
	if b.Behavior != nil {
		if err := b.Behavior.Act(mob, world); err != nil {
			return err
		}
	}
	
	b.LastUpdate = now
	return nil
}

// SetBehavior changes the mob's current behavior.
func (b *Brain) SetBehavior(behavior Behavior) {
	b.Behavior = behavior
}

// ClearTarget clears the mob's current target.
func (b *Brain) ClearTarget() {
	b.Target = ""
}

// SetTarget sets the mob's target.
func (b *Brain) SetTarget(target string) {
	b.Target = target
}