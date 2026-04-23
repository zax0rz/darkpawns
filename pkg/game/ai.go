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
	if mob.Prototype == nil {
		return
	}

	// Parse mob flags once
	isSentinel := false
	isAggressive := false
	isAggrEvil := false
	isAggrGood := false
	isAggrNeutral := false
	isWimpy := false
	isMemory := false
	isHelper := false
	isScavenger := false
	for _, flag := range mob.Prototype.ActionFlags {
		switch flag {
		case "sentinel":
			isSentinel = true
		case "aggressive":
			isAggressive = true
		case "aggr_evil":
			isAggrEvil = true
		case "aggr_good":
			isAggrGood = true
		case "aggr_neutral":
			isAggrNeutral = true
		case "wimpy":
			isWimpy = true
		case "memory":
			isMemory = true
		case "helper":
			isHelper = true
		case "scavenger":
			isScavenger = true
		}
	}

	// Don't act if already fighting
	if aiCombatEngine != nil && aiCombatEngine.IsFighting(mob.GetName()) {
		return
	}

	players := w.GetPlayersInRoom(mob.RoomVNum)

	// MOB_MEMORY: hunt players this mob remembers attacking it
	// Source: mobact.c:262-285
	if isMemory && len(mob.Memory) > 0 {
		for _, player := range players {
			for _, name := range mob.Memory {
				if name == player.GetName() {
					player.SendMessage("'Hey!  You're the fiend that attacked me!!!', exclaims " + mob.GetShortDesc() + ".\r\n")
					if aiCombatEngine != nil {
						aiCombatEngine.StartCombat(mob, player)
					}
					return
				}
			}
		}
	}

	// Aggression checks
	// MOB_SENTINEL only prevents movement, NOT aggression — mobact.c:110-132
	hasAlignAggr := isAggrEvil || isAggrGood || isAggrNeutral
	if isAggressive || hasAlignAggr {
		for _, player := range players {
			// MOB_WIMPY: skip awake players — mobact.c:209
			if isWimpy {
				continue // all players considered awake for now
			}
			// Alignment-based aggression — mobact.c:217-225
			// IS_GOOD: alignment >= 350, IS_EVIL: <= -350 (utils.h:454-455)
			if hasAlignAggr {
				align := player.GetAlignment()
				isGood := align >= 350
				isEvil := align <= -350
				isNeutral := !isGood && !isEvil
				if !((isAggrEvil && isEvil) || (isAggrGood && isGood) || (isAggrNeutral && isNeutral)) {
					continue
				}
			}
			if aiCombatEngine != nil {
				aiCombatEngine.StartCombat(mob, player)
			}
			return
		}
	}

	// MOB_HELPER: assist other fighting mobs against players
	// Source: mobact.c:286-302
	if isHelper {
		w.mu.RLock()
		for _, otherMob := range w.activeMobs {
			if otherMob == mob || otherMob.RoomVNum != mob.RoomVNum {
				continue
			}
			if otherMob.Fighting && otherMob.Target == nil {
				// otherMob is fighting a player — join in
				for _, player := range players {
					player.SendMessage(mob.GetShortDesc() + " jumps to the aid of " + otherMob.GetShortDesc() + "!\r\n")
					if aiCombatEngine != nil {
						aiCombatEngine.StartCombat(mob, player)
					}
					w.mu.RUnlock()
					return
				}
			}
		}
		w.mu.RUnlock()
	}

	// Call sound scripts (ambient pulse)
	// Based on original ambient pulse handling
	if mob.HasScript("sound") && rand.Intn(100) < 10 { // 10% chance per tick
		ctx := mob.CreateScriptContext(nil, nil, "")
		mob.RunScript("sound", ctx)
	}

	// MOB_SCAVENGER: pick up highest-value item in room — mobact.c:103-115
	// Only triggers 1 in 10 times (number(0,10) == 0 in original)
	if isScavenger && rand.Intn(11) == 0 {
		items := w.GetItemsInRoom(mob.RoomVNum)
		var bestItem *ObjectInstance
		bestCost := 0
		for _, item := range items {
			if item.GetCost() > bestCost {
				bestCost = item.GetCost()
				bestItem = item
			}
		}
		if bestItem != nil {
			w.RemoveItemFromRoom(bestItem, mob.RoomVNum)
			mob.Inventory = append(mob.Inventory, bestItem)
			for _, player := range players {
				player.SendMessage(mob.GetShortDesc() + " picks up " + bestItem.GetShortDesc() + ".\r\n")
			}
		}
	}

	// Wandering behavior — MOB_SENTINEL prevents movement only
	if !isSentinel && rand.Intn(100) < 25 {
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
		targetRoom, ok := w.rooms[exit.ToRoom]
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

	direction := validDirections[rand.Intn(len(validDirections))]
	exit := room.Exits[direction]
	targetRoom := w.rooms[exit.ToRoom]

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
