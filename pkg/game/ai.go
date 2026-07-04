package game

import (
	"context"
	"log/slog"
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
	//
	// Wandering happens inside MobileActivityForMob (mobileActivityForMob →
	// wanderMob), exactly once per tick — matching C mobile_activity, which
	// runs the movement block once per mob. DP-908 removed a redundant second
	// wanderMob call here plus a non-C 25% gate (rand.IntN(100) < 25) that had
	// no equivalent in mobact.c and dropped the effective wander rate to ~8%.
	w.MobileActivityForMob(mob)
}

// wanderMob moves a mob to a random adjacent room, faithful to C mobact.c's
// movement block (src/mobact.c:119-143). Caller must NOT hold mob.mu — the
// accessors lock individually (DP-590).
//
// C semantics reproduced here (DP-908):
//   - door = number(0, 18): a SINGLE random draw in [0,18].
//   - door < NUM_OF_DIRS (6): the same draw both gates movement (~6/19 ≈ 31.6%
//     chance the door is even a real cardinal direction) AND picks the exit.
//     Previously Go matched the *probability* but drew the direction separately,
//     which changes the per-direction distribution. We now reuse the gate draw
//     as the direction index.
//   - CAN_GO(ch, door): the exit exists in that cardinal direction.
//   - ROOM_DEATH / ROOM_NOMOB: skip.
//   - MOB_STAY_ZONE: skip exits that leave the mob's home zone.
//   - Sector swim/fly checks (Go has no CAN_SWIM/IS_FLYING, so water/flying
//     rooms are skipped for all mobs — pre-existing conservative behavior).
//
// Note: SENTINEL is checked by the caller (mobileActivityForMob), not here.
func (w *World) wanderMob(mob *MobInstance) {
	snap := w.snapshots.Snapshot()
	room, ok := snap.Rooms[mob.GetRoom()] // getter — mutex-protected
	if !ok {
		return
	}

	// Single random draw, exactly like C's door = number(0, 18).
	// #nosec G404 — game RNG, not cryptographic
	door := rand.IntN(19) // [0, 18]

	// door < NUM_OF_DIRS (len(dirs) == 6): if the draw isn't a real direction,
	// the mob doesn't move this tick. This is both the gate and the direction.
	if door >= len(dirs) {
		return
	}

	direction := dirs[door]
	exit, ok := room.Exits[direction] // CAN_GO(ch, door)
	if !ok {
		return
	}
	targetRoom, ok := snap.Rooms[exit.ToRoom]
	if !ok {
		return
	}

	// MOB_STAY_ZONE: skip exits that lead to a different zone (mobact.c:127-128).
	if hasMobFlag(mob, "stay_zone") && targetRoom.Zone != room.Zone {
		return
	}

	// Sector constraints (C: mobact.c:130-138). Go has no CAN_SWIM/IS_FLYING,
	// so skip water/flying rooms for all wandering mobs (pre-existing behavior).
	if targetRoom.Sector == SECT_WATER_SWIM || targetRoom.Sector == SECT_WATER_NOSWIM {
		return
	}
	if targetRoom.Sector == SECT_FLYING {
		return
	}

	// ROOM_DEATH and ROOM_NOMOB (mobact.c:125-126).
	if roomHasFlag(targetRoom, "death") || roomHasFlag(targetRoom, "no_mob") {
		return
	}

	// Move mob. GetRoom/SetRoom lock individually; the caller no longer holds
	// mob.mu (DP-590), so there is no lock to release or re-acquire here.
	oldRoom := mob.GetRoom()
	mob.SetRoom(targetRoom.VNum)

	slog.Debug("mob wandered",
		"mob_vnum", mob.GetVNum(),
		"name", mob.GetName(),
		"from", oldRoom,
		"to", targetRoom.VNum,
		"dir", direction,
	)

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
