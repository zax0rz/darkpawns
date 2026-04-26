package combat

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// CombatPairKey uniquely identifies a combat pair by both participants.
type CombatPairKey struct {
	Attacker string
	Target   string
}

// CombatPair represents two entities fighting each other
type CombatPair struct {
	Attacker       Combatant
	Defender       Combatant
	Started        time.Time
	LastAttackType int // Track what type of attack killed the victim (spell number, skill number, or TYPE_ constant)
}

// CombatEngine manages all active combat in the game
type CombatEngine struct {
	mu sync.RWMutex

	// Active combat pairs
	combatPairs map[CombatPairKey]*CombatPair // key: (attacker, target)

	// Combat ticker
	ticker   *time.Ticker
	stopChan chan struct{}
	stopped atomic.Bool

	// Message broadcaster function (set by game)
	BroadcastFunc func(roomVNum int, message string, exclude string)

	// DeathFunc handles corpse creation and respawn (set by game layer)
	// Called after death messages are sent.
	DeathFunc func(victim, killer Combatant, attackType int)

	// ScriptFightFunc fires the "fight" trigger on a mob after each combat round.
	// Set by the game layer. Called with (mobName, targetName, roomVNum).
	// Source: mobact.c — mobs use scripts during combat
	ScriptFightFunc func(mobName string, targetName string, roomVNum int)

	// DamageFunc is called after damage is applied to a combatant each round.
	// Set by the session manager to propagate health changes to agent sessions.
	// victimName is the name of the character who took damage.
	DamageFunc func(victimName string)
}

// NewCombatEngine creates a new combat engine
func NewCombatEngine() *CombatEngine {
	return &CombatEngine{
		combatPairs: make(map[CombatPairKey]*CombatPair),
		stopChan:    make(chan struct{}),
	}
}

// SetBroadcastFunc sets the function used to broadcast messages to rooms
func (ce *CombatEngine) SetBroadcastFunc(fn func(roomVNum int, message string, exclude string)) {
	ce.BroadcastFunc = fn
}

// Start begins the combat tick loop
func (ce *CombatEngine) Start() {
	ce.ticker = time.NewTicker(2 * time.Second) // Combat round every 2 seconds

	go func() {
		for {
			select {
			case <-ce.ticker.C:
				ce.PerformRound()
			case <-ce.stopChan:
				ce.ticker.Stop()
				return
			}
		}
	}()
}

// PositionedMob is an interface for entities that can be knocked down and need position recovery.
type PositionedMob interface {
	GetName() string
	GetStatus() string
	SetStatus(string)
	GetFighting() string
}

// StartMobPositionRecovery starts a goroutine that periodically checks mob positions.
// Mobs that are sitting/resting/sleeping and not in combat are stood back up.
// getMobs returns all mobs that should be checked for position recovery.
// Separate from the combat ticker so position recovery runs at its own cadence.
func (ce *CombatEngine) StartMobPositionRecovery(getMobs func() []PositionedMob) {
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mobs := getMobs()
				for _, mob := range mobs {
					status := mob.GetStatus()
					fighting := mob.GetFighting()
					if fighting != "" {
						continue
					}
					if status != "sleeping" && status != "resting" && status != "sitting" {
						continue
					}
					mob.SetStatus("standing")
				}
			case <-ce.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// Stop halts the combat engine
func (ce *CombatEngine) Stop() {
	if ce.stopped.CompareAndSwap(false, true) {
		close(ce.stopChan)
	}
}

// StartCombat initiates combat between two combatants
func (ce *CombatEngine) StartCombat(attacker, defender Combatant) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	attackerName := attacker.GetName()
	defenderName := defender.GetName()

	// Build composite key
	key := CombatPairKey{
		Attacker: attackerName,
		Target:   defenderName,
	}

	// Check if already fighting
	if _, exists := ce.combatPairs[key]; exists {
		return fmt.Errorf("%s is already fighting", attackerName)
	}

	// Also check same attacker attacking different target (prevent silent overwrite)
	for k := range ce.combatPairs {
		if k.Attacker == attackerName {
			return fmt.Errorf("%s is already fighting", attackerName)
		}
	}

	// Set fighting state
	attacker.SetFighting(defenderName)
	defender.SetFighting(attackerName)

	// Start combat
	ce.combatPairs[key] = &CombatPair{
		Attacker: attacker,
		Defender: defender,
		Started:  time.Now(),
	}

	return nil
}

// StopCombat ends combat for a character
func (ce *CombatEngine) StopCombat(charName string) {
	ce.mu.Lock()
	defer ce.mu.Unlock()

	// Find and stop combat — iterate all pairs to find matches by attacker or defender
	for key, pair := range ce.combatPairs {
		if key.Attacker == charName || pair.Defender.GetName() == charName {
			pair.Attacker.StopFighting()
			if pair.Defender.GetFighting() == key.Attacker {
				pair.Defender.StopFighting()
			}
			delete(ce.combatPairs, key)
		}
	}
}

// IsFighting checks if a character is in combat
func (ce *CombatEngine) IsFighting(charName string) bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	// Check if attacking or being attacked — must iterate with composite keys
	for key, pair := range ce.combatPairs {
		if key.Attacker == charName || pair.Defender.GetName() == charName {
			return true
		}
	}

	return false
}

// PerformRound executes one round of combat for all active fighters
func (ce *CombatEngine) PerformRound() {
	ce.mu.Lock()

	// Snapshot pairs under write lock to prevent TOCTOU races
	pairs := make([]*CombatPair, 0, len(ce.combatPairs))
	for _, pair := range ce.combatPairs {
		pairs = append(pairs, pair)
	}

	ce.mu.Unlock()

	// Process each combat pair
	for _, pair := range pairs {
		ce.processCombatPair(pair)
	}
}

// processCombatPair handles a single combat exchange
func (ce *CombatEngine) processCombatPair(pair *CombatPair) {
	attacker := pair.Attacker
	defender := pair.Defender

	// Check if both combatants are still valid and alive
	if attacker.GetHP() <= 0 || defender.GetHP() <= 0 {
		ce.StopCombat(attacker.GetName())
		return
	}

	// Check if they're in the same room
	if attacker.GetRoom() != defender.GetRoom() {
		ce.StopCombat(attacker.GetName())
		return
	}

	// Calculate number of attacks for attacker
	numAttacks := GetAttacksPerRound(attacker, false, false)

	// Perform attacks
	for i := 0; i < numAttacks; i++ {
		// Check if defender is still alive
		if defender.GetHP() <= 0 {
			break
		}

		// Check hit
		if !CalculateHitChance(attacker, defender, HitModifiers{}) {
			ce.sendMissMessage(attacker, defender)
			continue
		}

		// Calculate damage
		weaponDamage := attacker.GetDamageRoll()
		damage := CalculateDamage(attacker, defender, weaponDamage, AttackNormal)

		// Apply damage
		defender.TakeDamage(damage)
		if ce.DamageFunc != nil {
			ce.DamageFunc(defender.GetName())
		}

		// Send combat messages
		ce.sendHitMessage(attacker, defender, damage)

		// Check for death
		if defender.GetHP() <= 0 {
			ce.handleDeath(defender, attacker)
			ce.StopCombat(attacker.GetName())
			break
		}
	}

	// Fire fight trigger on mob attacker after combat round
	// Source: mobact.c — mob_activity() calls mob scripts after violence
	if attacker.IsNPC() && ce.ScriptFightFunc != nil && defender.GetHP() > 0 {
		ce.ScriptFightFunc(attacker.GetName(), defender.GetName(), attacker.GetRoom())
	}
}

// sendHitMessage sends hit messages to combatants and room
func (ce *CombatEngine) sendHitMessage(attacker, defender Combatant, damage int) {
	attackerName := attacker.GetName()
	defenderName := defender.GetName()
	roomVNum := attacker.GetRoom()

	// Message to attacker
	attacker.SendMessage(fmt.Sprintf("You hit %s for %d damage!", defenderName, damage))

	// Message to defender
	defender.SendMessage(fmt.Sprintf("%s hits you for %d damage!", attackerName, damage))

	// Message to room
	if ce.BroadcastFunc != nil {
		ce.BroadcastFunc(roomVNum,
			fmt.Sprintf("%s hits %s!", attackerName, defenderName),
			attackerName)
	}
}

// sendMissMessage sends miss messages
func (ce *CombatEngine) sendMissMessage(attacker, defender Combatant) {
	attackerName := attacker.GetName()
	defenderName := defender.GetName()
	roomVNum := attacker.GetRoom()

	attacker.SendMessage(fmt.Sprintf("You miss %s!", defenderName))
	defender.SendMessage(fmt.Sprintf("%s misses you!", attackerName))

	if ce.BroadcastFunc != nil {
		ce.BroadcastFunc(roomVNum,
			fmt.Sprintf("%s misses %s!", attackerName, defenderName),
			attackerName)
	}
}

// handleDeath processes character death.
//
// Faithful to Dark Pawns die()/raw_kill() in fight.c:
//   - Player: lose EXP/3, create corpse with inventory+equipment+gold, send to room 8004
//   - Mob: create corpse, remove from world
//   - Corpse creation is delegated via DeathFunc callback (set by game layer)
func (ce *CombatEngine) handleDeath(victim, killer Combatant) {
	victimName := victim.GetName()
	killerName := killer.GetName()
	roomVNum := victim.GetRoom()

	// Death message to victim
	victim.SendMessage("You have been KILLED!\r\n")

	// Death message to room
	if ce.BroadcastFunc != nil {
		ce.BroadcastFunc(roomVNum,
			fmt.Sprintf("%s has been killed by %s!", victimName, killerName),
			"")
	}

	// Delegate to game layer for corpse creation + respawn
	if ce.DeathFunc != nil {
		// Get attack type from combat pair if available
		attackType := -1 // TYPE_UNDEFINED
		ce.mu.RLock()
		for key, pair := range ce.combatPairs {
			if key.Attacker == killerName {
				attackType = pair.LastAttackType
				break
			}
		}
		ce.mu.RUnlock()
		ce.DeathFunc(victim, killer, attackType)
	}
}

// GetCombatTarget returns who a character is fighting
func (ce *CombatEngine) GetCombatTarget(charName string) (Combatant, bool) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	// Check if attacking someone or being attacked — must iterate with composite keys
	for key, pair := range ce.combatPairs {
		if key.Attacker == charName {
			return pair.Defender, true
		}
		if pair.Defender.GetName() == charName {
			return pair.Attacker, true
		}
	}

	return nil, false
}

// GetCombatStatus returns combat status for a character
func (ce *CombatEngine) GetCombatStatus(charName string) string {
	ce.mu.RLock()
	defer ce.mu.RUnlock()

	for key, pair := range ce.combatPairs {
		if key.Attacker == charName {
			return fmt.Sprintf("You are fighting %s", pair.Defender.GetName())
		}
		if pair.Defender.GetName() == charName {
			return fmt.Sprintf("You are being attacked by %s", key.Attacker)
		}
	}

	return "You are not in combat"
}
