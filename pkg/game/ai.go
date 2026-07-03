package game

import (
	"context"
	"math/rand/v2"
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
//
// DP-590: this used to hold mob.mu for the whole AI cycle (CRIT-004), but the
// dispatch below runs entirely through self-locking accessors (GetName,
// GetFighting, SetStatus, wanderMob → GetRoom/SetRoom, …). Holding the write
// lock made the first accessor self-deadlock, so the mob AI never ran. The
// accessors provide their own synchronization; we no longer hold mob.mu here.
func (w *World) runMobAI(mob *MobInstance) {
	mob.mu.RLock()
	ready := mob.Prototype != nil
	mob.mu.RUnlock()
	if !ready {
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
	w.MobileActivityForMob(mob)

	// Post-activity: wandering. Prototype is immutable after creation, so it is
	// safe to read its flags without holding mob.mu.
	// MOB_SENTINEL prevents movement only.
	// #nosec G404 — game RNG, not cryptographic
	if !hasMobFlag(mob, "sentinel") && rand.IntN(100) < 25 {
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

	if rand.IntN(19) >= 6 {
		return
	}

	// Get available exits
	if len(room.Exits) == 0 {
		return
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
		if hasMobFlag(mob, "stay_zone") && targetRoom.Zone != room.Zone {
			continue
		}

		// Sector constraints (C: mobact.c:130-138)
		// Skip water rooms for mobs that can't swim (no CAN_SWIM check in Go — skip water for all non-water mobs)
		if targetRoom.Sector == SECT_WATER_SWIM || targetRoom.Sector == SECT_WATER_NOSWIM {
			continue // Conservative: skip water rooms for all wandering mobs
		}
		if targetRoom.Sector == SECT_FLYING {
			continue // Skip flying rooms — mobs can't fly unless flagged
		}

		// Check ROOM_DEATH and ROOM_NOMOB before mob movement
		if roomHasFlag(targetRoom, "death") || roomHasFlag(targetRoom, "no_mob") {
			continue
		}

		validDirections = append(validDirections, dir)
	}

	if len(validDirections) == 0 {
		return
	}

	// #nosec G404 — game RNG, not cryptographic
	direction := validDirections[rand.IntN(len(validDirections))]
	exit := room.Exits[direction]
	targetRoom := snap.Rooms[exit.ToRoom]

	// Move mob. GetRoom/SetRoom lock individually; the caller no longer holds
	// mob.mu (DP-590), so there is no lock to release or re-acquire here.
	oldRoom := mob.GetRoom()
	mob.SetRoom(targetRoom.VNum)

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

	// MobProg entry trigger — fires when mob enters a room
	w.EntryProg(mob, targetRoom.VNum)
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
