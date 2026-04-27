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

// combatEngine is stored for AI to use
var aiCombatEngine CombatEngine

// SetAICombatEngine sets the combat engine for AI to use
func SetAICombatEngine(ce CombatEngine) {
	aiCombatEngine = ce
}

// AITick runs AI for all active mobs
func (w *World) AITick() {
	w.mu.RLock()
	mobs := make([]*MobInstance, 0, len(w.activeMobs))
	for _, mob := range w.activeMobs {
		mobs = append(mobs, mob)
	}
	w.mu.RUnlock()

	for _, mob := range mobs {
		w.runMobAI(mob)
	}
}

// runMobAI runs AI for a single mob.
// This is the integration layer that delegates to the faithful
// mobact.go:MobileActivity() and then handles wandering.
func (w *World) runMobAI(mob *MobInstance) {
	if mob.Prototype == nil {
		return
	}

	// Don't act if already fighting
	if aiCombatEngine != nil && aiCombatEngine.IsFighting(mob.GetName()) {
		return
	}

	// Delegate to the faithful mobact.c port
	w.MobileActivity()

	// Post-activity: wandering (handled separately in mobact.c movement section
	// and ai.go wanderMob). The original C wandering is inside mobile_activity(),
	// but the existing Dark Pawns Go architecture handles wandering in this
	// integration layer. We keep wanderMob() here as well since ai.go already
	// has the infrastructure.
	//
	// Parse sentinel flag
	isSentinel := false
	if mob.Prototype != nil {
		for _, flag := range mob.Prototype.ActionFlags {
			if flag == "sentinel" {
				isSentinel = true
				break
			}
		}
	}

	// Wandering behavior — MOB_SENTINEL prevents movement only
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	if !isSentinel && rand.Intn(100) < 25 {
		w.wanderMob(mob)
	}
}

// wanderMob moves a mob to a random adjacent room
func (w *World) wanderMob(mob *MobInstance) {
	snap := w.snapshots.Snapshot()
	room, ok := snap.Rooms[mob.RoomVNum]
	if !ok {
		return
	}

	// Get available exits
	if len(room.Exits) == 0 {
		return
	}

	// Check if mob has MOB_STAY_ZONE flag
	hasStayZone := false
	if mob.Prototype != nil {
		for _, flag := range mob.Prototype.ActionFlags {
			if flag == "stay_zone" {
				hasStayZone = true
				break
			}
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
		// Source: mobact.c - before moving a mob to a room, checks !ROOM_DEATH and !ROOM_NOMOB
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
// #nosec G404
	direction := validDirections[rand.Intn(len(validDirections))]
	exit := room.Exits[direction]
	targetRoom := snap.Rooms[exit.ToRoom]

	// Move mob
	oldRoom := mob.RoomVNum
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

