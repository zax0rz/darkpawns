// Package ai implements mob AI behaviors.
package ai

// Behavior defines the interface for mob AI behaviors.
type Behavior interface {
	// Name returns the name of the behavior.
	Name() string
}

// AggressiveBehavior attacks players on sight.
type AggressiveBehavior struct{}

func (b *AggressiveBehavior) Name() string {
	return "aggressive"
}

// WanderingBehavior moves randomly between rooms.
type WanderingBehavior struct{}

func (b *WanderingBehavior) Name() string {
	return "wandering"
}

// SentinelBehavior stays in place.
type SentinelBehavior struct{}

func (b *SentinelBehavior) Name() string {
	return "sentinel"
}
