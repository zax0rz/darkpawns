package game

import (
	"math/rand"
	"time"

	"github.com/zax0rz/darkpawns/pkg/combat"
)

// CombatEngine interface for AI to initiate combat
type CombatEngine interface {
	StartCombat(attacker, defender combat.Combatant) error
	IsFighting(name string) bool
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

// runMobAI runs AI for a single mob
func (w *World) runMobAI(mob *MobInstance) {
	// Check mob flags from prototype
	if mob.Prototype == nil {
		return
	}

	// Check for sentinel behavior
	isSentinel := false
	isAggressive := false
	for _, flag := range mob.Prototype.ActionFlags {
		if flag == "sentinel" {
			isSentinel = true
		}
		if flag == "aggressive" {
			isAggressive = true
		}
	}

	// Sentinel mobs don't move or attack automatically
	if isSentinel {
		return
	}

	// Don't act if already fighting
	if aiCombatEngine != nil && aiCombatEngine.IsFighting(mob.GetName()) {
		return
	}

	// Check for aggressive mobs
	if isAggressive {
		// Check for players in room
		players := w.GetPlayersInRoom(mob.RoomVNum)
		for _, player := range players {
			// Attack via combat engine!
			if aiCombatEngine != nil {
				aiCombatEngine.StartCombat(mob, player)
			}
			return
		}
	}

	// Wandering behavior (if not sentinel)
	// 25% chance to wander
	if rand.Intn(100) < 25 {
		w.wanderMob(mob)
	}
}

// wanderMob moves a mob to a random adjacent room
func (w *World) wanderMob(mob *MobInstance) {
	room, ok := w.rooms[mob.RoomVNum]
	if !ok {
		return
	}

	// Get available exits
	if len(room.Exits) == 0 {
		return
	}

	// Pick random exit
	directions := make([]string, 0, len(room.Exits))
	for dir := range room.Exits {
		directions = append(directions, dir)
	}

	direction := directions[rand.Intn(len(directions))]
	exit := room.Exits[direction]

	// Check if target room exists
	targetRoom, ok := w.rooms[exit.ToRoom]
	if !ok {
		return
	}

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

// StartAITicker starts the AI tick loop
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
}