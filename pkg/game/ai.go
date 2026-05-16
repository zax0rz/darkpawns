package game

import (
	"context"
	"math/rand"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// CombatEngine interface for AI to initiate combat
type CombatEngine interface {
	StartCombat(attacker, defender combat.Combatant) error
	IsFighting(name string) bool
	GetCombatTarget(charName string) (combat.Combatant, bool)
}

// AIBehavior defines mob AI behavior
type AIBehavior int

const (
	AIWandering AIBehavior = iota
	AIAggressive
	AISentinel
)

// CRIT-006: aiCombatEngine moved to World.combatEngine.
// SetAICombatEngine is replaced by World.SetCombatEngine.

// AITick runs AI for all active mobs.
// CRIT-004: uses atomic IsAlive() check — no lock needed for the pre-filter.
func (w *World) AITick() {
	w.mu.RLock()
	mobs := make([]*MobInstance, 0, len(w.activeMobs))
	for _, mob := range w.activeMobs {
		mobs = append(mobs, mob)
	}
	w.mu.RUnlock()

	for _, mob := range mobs {
		// CRIT-004: atomic alive check — skip dead mobs without acquiring m.mu
		if !mob.IsAlive() {
			continue
		}
		w.runMobAI(mob)
	}
}

// runMobAI runs AI for a single mob.
// CRIT-004: holds mob.mu for the entire AI cycle to prevent races between
// AI tick, combat state changes, and player interactions.
//
// MED-010: wanderMob uses direct field access (mob.RoomVNum) instead of
// getter methods that would deadlock (they acquire m.mu.RLock/Lock).
func (w *World) runMobAI(mob *MobInstance) {
	mob.mu.Lock()
	defer mob.mu.Unlock()

	if mob.Prototype == nil {
		return
	}

	// Don't act if already fighting
	if w.combatEngine != nil && w.combatEngine.IsFighting(mob.GetName()) {
		return
	}

	// MED-009: call per-mob activity instead of full MobileActivity()
	// This fixes the O(N²) bug where runMobAI called MobileActivity()
	// which re-iterated ALL mobs — making every mob get processed N times
	// per tick. MobileActivityForMob processes a single mob.
	// Caller must hold mob.mu — see MobileActivityForMob contract.
	w.MobileActivityForMob(mob)

	// Post-activity: wandering
	// Parse sentinel flag — direct field access, mob.mu is held
	isSentinel := false
	for _, flag := range mob.Prototype.ActionFlags {
		if flag == "sentinel" {
			isSentinel = true
			break
		}
	}

	// Wandering behavior — MOB_SENTINEL prevents movement only
	// #nosec G404 — game RNG, not cryptographic
	if !isSentinel && rand.Intn(100) < 25 {
		w.wanderMob(mob)
	}
}

// wanderMob moves a mob to a random adjacent room.
// Caller must hold mob.mu. Uses direct field access to avoid deadlock.
// MED-010: snapshot-based room reads, direct mob field writes.
func (w *World) wanderMob(mob *MobInstance) {
	snap := w.snapshots.Snapshot()
	room, ok := snap.Rooms[mob.GetRoom()] // getter — mutex-protected
	if !ok {
		return
	}

	// Get available exits
	if len(room.Exits) == 0 {
		return
	}

	// Check if mob has MOB_STAY_ZONE flag
	hasStayZone := false
	for _, flag := range mob.Prototype.ActionFlags {
		if flag == "stay_zone" {
			hasStayZone = true
			break
		}
	}

	// Pick random exit, filtering by zone if MOB_STAY_ZONE
	var validDirections []string
	for dir, exit := range room.Exits {
		// Check if target room exists
		targetRoom, ok := snap.Rooms[exit.ToRoom]
		if !ok {
			continue
		}

		// MOB_STAY_ZONE: skip exits that lead to a different zone
		// Source: mobact.c:127
		if hasStayZone && targetRoom.Zone != room.Zone {
			continue
		}

		// Check ROOM_DEATH and ROOM_NOMOB before mob movement
		hasDeath := false
		hasNoMob := false
		for _, flag := range targetRoom.Flags {
			if flag == "death" {
				hasDeath = true
			}
			if flag == "nomob" {
				hasNoMob = true
			}
		}
		if hasDeath || hasNoMob {
			continue
		}

		validDirections = append(validDirections, dir)
	}

	if len(validDirections) == 0 {
		return
	}

	// #nosec G404 — game RNG, not cryptographic
	direction := validDirections[rand.Intn(len(validDirections))]
	exit := room.Exits[direction]
	targetRoom := snap.Rooms[exit.ToRoom]

	// Move mob — direct field write, mob.mu is held
	oldRoom := mob.GetRoom()
	mob.SetRoom(targetRoom.VNum)

	// Release mob lock during player notifications (I/O can block)
	mob.mu.Unlock()

	// Notify players in old room
	oldPlayers := w.GetPlayersInRoom(oldRoom)
	for _, p := range oldPlayers {
		p.SendMessage(mob.GetShortDesc() + " leaves " + direction + ".\n")
	}

	// Notify players in new room
	newPlayers := w.GetPlayersInRoom(targetRoom.VNum)
	for _, p := range newPlayers {
		p.SendMessage(mob.GetShortDesc() + " has arrived.\n")
	}

	// Re-acquire lock — defer in runMobAI will fire with this held
	mob.mu.Lock()
}

// StartAITicker starts the AI tick loop and event processing loop.
// The event loop runs at 10 pulses per second (100ms), matching the
// original C code: OPT_USEC = 100000, PASSES_PER_SEC = 10.
// Source: comm.c game_loop() — heartbeat(++pulse) calls event_process().
func (w *World) StartAITicker() {
	w.aiticker = time.NewTicker(10 * time.Second)
	go func() {
		for {
			select {
			case <-w.aiticker.C:
				w.AITick()
			case <-w.done:
				w.aiticker.Stop()
				return
			}
		}
	}()

	// Start event processing loop
	// Source: events.c event_process() — called once per pulse in heartbeat()
	if w.EventQueue != nil {
		ctx := context.Background()
		w.EventQueue.Start(ctx)
	}
}

// StartPointUpdateTicker starts the regen/hunger/thirst tick loop.
// Source: limits.c point_update() — called every ~75 pulses in stock CircleMUD.
// Dark Pawns uses a faster tick (30 seconds).
func (w *World) StartPointUpdateTicker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				w.PointUpdate()
			case <-w.done:
				ticker.Stop()
				return
			}
		}
	}()
}
