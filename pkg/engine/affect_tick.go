package engine

import (
	"sync"
	"time"
)

// TickManager manages the global tick system for affects
type TickManager struct {
	mu            sync.RWMutex
	affectManager *AffectManager
	tickInterval  time.Duration
	ticker        *time.Ticker
	done          chan struct{}
	running       bool
}

// NewTickManager creates a new tick manager
func NewTickManager(affectManager *AffectManager) *TickManager {
	return &TickManager{
		affectManager: affectManager,
		tickInterval:  time.Second, // Default: 1 tick per second
		done:          make(chan struct{}),
		running:       false,
	}
}

// Start begins the tick processing loop
func (tm *TickManager) Start() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.running {
		return // Already running
	}

	tm.ticker = time.NewTicker(tm.tickInterval)
	tm.running = true

	go tm.tickLoop()
}

// Stop halts the tick processing loop
func (tm *TickManager) Stop() {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if !tm.running {
		return // Not running
	}

	tm.ticker.Stop()
	close(tm.done)
	tm.running = false
}

// SetTickInterval changes the tick interval, stopping the old goroutine
// and starting a new one with the updated interval.
func (tm *TickManager) SetTickInterval(interval time.Duration) {
	tm.mu.Lock()
	tm.tickInterval = interval

	if tm.running {
		tm.ticker.Stop()
		close(tm.done)
		tm.done = make(chan struct{})
		tm.ticker = time.NewTicker(tm.tickInterval)
		go tm.tickLoop()
	}
	tm.mu.Unlock()
}

// GetTickInterval returns the current tick interval
func (tm *TickManager) GetTickInterval() time.Duration {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.tickInterval
}

// IsRunning returns whether the tick manager is running
func (tm *TickManager) IsRunning() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.running
}

// tickLoop is the main tick processing loop
func (tm *TickManager) tickLoop() {
	for {
		select {
		case <-tm.ticker.C:
			tm.processTick()
		case <-tm.done:
			return
		}
	}
}

// processTick processes a single tick for all affects
func (tm *TickManager) processTick() {
	// Process all affects
	tm.affectManager.Tick()

	// Additional tick processing could go here
	// For example: update world time, process weather, etc.
}

// ManualTick manually triggers a tick (useful for testing)
func (tm *TickManager) ManualTick() {
	tm.processTick()
}

// Global tick manager instance
var (
	globalTickManager *TickManager
	tickManagerOnce   sync.Once
)

// GetGlobalTickManager returns the global tick manager instance
func GetGlobalTickManager(affectManager *AffectManager) *TickManager {
	tickManagerOnce.Do(func() {
		globalTickManager = NewTickManager(affectManager)
	})
	return globalTickManager
}

// StartGlobalTickManager starts the global tick manager
func StartGlobalTickManager(affectManager *AffectManager) {
	tm := GetGlobalTickManager(affectManager)
	tm.Start()
}

// StopGlobalTickManager stops the global tick manager
func StopGlobalTickManager() {
	if globalTickManager != nil {
		globalTickManager.Stop()
	}
}

// AffectTickSystem combines affect management with tick processing
type AffectTickSystem struct {
	AffectManager *AffectManager
	TickManager   *TickManager
}

// NewAffectTickSystem creates a new combined affect and tick system
func NewAffectTickSystem() *AffectTickSystem {
	affectManager := NewAffectManager()
	tickManager := NewTickManager(affectManager)

	return &AffectTickSystem{
		AffectManager: affectManager,
		TickManager:   tickManager,
	}
}

// Start begins the affect tick system
func (ats *AffectTickSystem) Start() {
	ats.TickManager.Start()
}

// Stop halts the affect tick system
func (ats *AffectTickSystem) Stop() {
	ats.TickManager.Stop()
}

// ApplyAffect is a convenience method to apply an affect
func (ats *AffectTickSystem) ApplyAffect(entity Affectable, affect *Affect) bool {
	return ats.AffectManager.ApplyAffect(entity, affect)
}

// RemoveAffect is a convenience method to remove an affect
func (ats *AffectTickSystem) RemoveAffect(entity Affectable, affectID string) bool {
	return ats.AffectManager.RemoveAffect(entity, affectID)
}

// GetAffects is a convenience method to get all affects on an entity
func (ats *AffectTickSystem) GetAffects(entity Affectable) []*Affect {
	return ats.AffectManager.GetAffects(entity)
}

// HasAffect is a convenience method to check for an affect type
func (ats *AffectTickSystem) HasAffect(entity Affectable, affectType AffectType) bool {
	return ats.AffectManager.HasAffect(entity, affectType)
}

// RemoveAllAffects is a convenience method to remove all affects
func (ats *AffectTickSystem) RemoveAllAffects(entity Affectable) int {
	return ats.AffectManager.RemoveAllAffects(entity)
}

// ManualTick manually triggers a tick
func (ats *AffectTickSystem) ManualTick() {
	ats.TickManager.ManualTick()
}
