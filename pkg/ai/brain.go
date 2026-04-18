// Package ai implements mob AI behaviors.
package ai

import "time"

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

// NewBrain creates a new brain with default behavior.
func NewBrain(mob interface{}) *Brain {
	return &Brain{
		Behavior:     &WanderingBehavior{},
		LastUpdate:   time.Now(),
		TickInterval: 10 * time.Second,
	}
}

// Update runs the mob's AI logic.
func (b *Brain) Update(mob interface{}, world interface{}) error {
	// Stub implementation
	return nil
}

// SetTarget sets the AI target.
func (b *Brain) SetTarget(target string) {
	b.Target = target
}