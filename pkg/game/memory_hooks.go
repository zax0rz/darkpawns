// memory_hooks.go — callback hooks for narrative memory.
//
// Architecture decision (RESEARCH-LOG.md, 2026-04-21, Gap 1):
//   World fires events; Manager (which has DB and knows which sessions are agents)
//   handles persistence. Game layer stays pure — no direct DB access from World.
//
// How to use:
//   1. Manager calls world.SetMobKillHook(func(killer, victim *MobKillEvent){...})
//   2. On mob death, handleMobDeath calls w.onMobKill(event)
//   3. Manager checks if killer is an agent, writes narrative memory if so.

package game

import "sync"

// MobKillEvent carries the facts of a mob death.
type MobKillEvent struct {
	KillerName  string // player/agent character name (empty if no single killer)
	KillerIsNPC bool   // true if killer was a mob (rare, but possible)
	VictimName  string
	VictimVNum  int
	RoomVNum    int
	RoomName    string
}

// PlayerDeathEvent carries the facts of a player/agent death.
type PlayerDeathEvent struct {
	VictimName  string
	KillerName  string // mob or player that landed the killing blow
	KillerIsNPC bool
	RoomVNum    int
	RoomName    string
	IsCombat    bool
}

// hookState holds registered callbacks. Protected by a mutex because
// hooks are registered at startup but could theoretically be called
// from concurrent goroutines.
type hookState struct {
	mu            sync.RWMutex
	onMobKill     func(*MobKillEvent)
	onPlayerDeath func(*PlayerDeathEvent)
}

var hooks hookState

// SetMobKillHook registers a callback fired every time a mob is killed.
// Called by Manager on startup.
func (w *World) SetMobKillHook(fn func(*MobKillEvent)) {
	hooks.mu.Lock()
	defer hooks.mu.Unlock()
	hooks.onMobKill = fn
}

// SetPlayerDeathHook registers a callback fired every time a player dies.
// Called by Manager on startup.
func (w *World) SetPlayerDeathHook(fn func(*PlayerDeathEvent)) {
	hooks.mu.Lock()
	defer hooks.mu.Unlock()
	hooks.onPlayerDeath = fn
}

// fireMobKill invokes the registered hook (if any) in a separate goroutine.
// Fire-and-forget: game loop never waits on memory writes.
func fireMobKill(evt *MobKillEvent) {
	hooks.mu.RLock()
	fn := hooks.onMobKill
	hooks.mu.RUnlock()
	if fn != nil {
		go fn(evt)
	}
}

// firePlayerDeath invokes the registered hook (if any) in a separate goroutine.
func firePlayerDeath(evt *PlayerDeathEvent) {
	hooks.mu.RLock()
	fn := hooks.onPlayerDeath
	hooks.mu.RUnlock()
	if fn != nil {
		go fn(evt)
	}
}
