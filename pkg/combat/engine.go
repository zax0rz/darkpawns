package combat

import (
	"fmt"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/game"
)

// CombatPair represents two characters fighting each other
type CombatPair struct {
	Attacker *game.Player
	Defender *game.Player
	Started  time.Time
}

// CombatEngine manages all active combat in the game
type CombatEngine struct {
	mu sync.RWMutex
	
	// Active combat pairs
	combatPairs map[string]*CombatPair // key: attacker name
	
	// Mob combat tracking
	mobCombat map[string]string // mob name -> target name
	
	// Combat ticker
	ticker   *time.Ticker
	stopChan chan struct{}
}

// NewCombatEngine creates a new combat engine
func NewCombatEngine() *CombatEngine {
	return &CombatEngine{
		combatPairs: make(map[string]*CombatPair),
		mobCombat:   make(map[string]string),
		stopChan:    make(chan struct{}),
	}
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

// Stop halts the combat engine
func (ce *CombatEngine) Stop() {
	close(ce.stopChan)
}

// StartCombat initiates combat between two players
func (ce *CombatEngine) StartCombat(attacker, defender *game.Player) error {
	ce.mu.Lock()
	defer ce.mu.Unlock()
	
	// Check if already fighting
	if _, exists := ce.combatPairs[attacker.Name]; exists {
		return fmt.Errorf("%s is already fighting", attacker.Name)
	}
	
	// Start combat
	ce.combatPairs[attacker.Name] = &CombatPair{
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
	
	// Remove from combat pairs
	delete(ce.combatPairs, charName)
	
	// Also check if anyone is fighting this character
	for attacker, pair := range ce.combatPairs {
		if pair.Defender.Name == charName {
			delete(ce.combatPairs, attacker)
		}
	}
	
	// Remove from mob combat
	delete(ce.mobCombat, charName)
}

// IsFighting checks if a character is in combat
func (ce *CombatEngine) IsFighting(charName string) bool {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	
	// Check if attacking
	if _, exists := ce.combatPairs[charName]; exists {
		return true
	}
	
	// Check if being attacked
	for _, pair := range ce.combatPairs {
		if pair.Defender.Name == charName {
			return true
		}
	}
	
	// Check mob combat
	if _, exists := ce.mobCombat[charName]; exists {
		return true
	}
	
	return false
}

// PerformRound executes one round of combat for all active fighters
func (ce *CombatEngine) PerformRound() {
	ce.mu.RLock()
	
	// Get all combat pairs
	pairs := make([]*CombatPair, 0, len(ce.combatPairs))
	for _, pair := range ce.combatPairs {
		pairs = append(pairs, pair)
	}
	
	ce.mu.RUnlock()
	
	// Process each combat pair
	for _, pair := range pairs {
		ce.processCombatPair(pair)
	}
}

// processCombatPair handles a single combat exchange
func (ce *CombatEngine) processCombatPair(pair *CombatPair) {
	// Check if both combatants are still valid
	if pair.Attacker == nil || pair.Defender == nil {
		ce.StopCombat(pair.Attacker.Name)
		return
	}
	
	// For now, use simple damage calculation
	// In a full implementation, this would use the formulas package
	damage := 5 + time.Now().UnixNano()%10 // Random 5-14 damage
	
	// Apply damage
	pair.Defender.Health -= damage
	
	// Send combat messages
	ce.sendCombatMessage(pair.Attacker, pair.Defender, damage)
	
	// Check for death
	if pair.Defender.Health <= 0 {
		ce.handleDeath(pair.Defender, pair.Attacker)
		ce.StopCombat(pair.Attacker.Name)
	}
}

// sendCombatMessage sends combat messages to the room
func (ce *CombatEngine) sendCombatMessage(attacker, defender *game.Player, damage int) {
	// This would send messages to the room
	// For now, just log
	fmt.Printf("[COMBAT] %s hits %s for %d damage\n", 
		attacker.Name, defender.Name, damage)
	
	// In a real implementation, this would broadcast to the room:
	// - "%s hits %s for %d damage!"
	// - "%s is hit for %d damage!"
}

// handleDeath processes character death
func (ce *CombatEngine) handleDeath(victim, killer *game.Player) {
	fmt.Printf("[DEATH] %s has been killed by %s\n", victim.Name, killer.Name)
	
	// Reset victim's health
	victim.Health = victim.MaxHealth
	
	// In a full implementation:
	// - Send death messages to room
	// - Handle experience gain
	// - Handle corpse/loot
	// - Move to respawn location
}

// GetCombatTarget returns who a character is fighting
func (ce *CombatEngine) GetCombatTarget(charName string) (*game.Player, bool) {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	
	// Check if attacking someone
	if pair, exists := ce.combatPairs[charName]; exists {
		return pair.Defender, true
	}
	
	// Check if being attacked
	for attacker, pair := range ce.combatPairs {
		if pair.Defender.Name == charName {
			// Return the attacker
			return pair.Attacker, true
		}
	}
	
	return nil, false
}

// GetCombatStatus returns combat status for a character
func (ce *CombatEngine) GetCombatStatus(charName string) string {
	ce.mu.RLock()
	defer ce.mu.RUnlock()
	
	if pair, exists := ce.combatPairs[charName]; exists {
		return fmt.Sprintf("You are fighting %s", pair.Defender.Name)
	}
	
	for attacker, pair := range ce.combatPairs {
		if pair.Defender.Name == charName {
			return fmt.Sprintf("You are being attacked by %s", attacker)
		}
	}
	
	return "You are not in combat"
}