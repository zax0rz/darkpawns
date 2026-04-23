package engine

// Example integration with Dark Pawns game systems
// This file shows how to integrate the affect system with existing game code

/*
// Example: Integrating with Player struct from pkg/game/player.go

import (
	"sync"
	"github.com/zax0rz/darkpawns/pkg/game"
)

// PlayerAffectable wraps a Player to implement the Affectable interface
type PlayerAffectable struct {
	player *game.Player
	mu     sync.RWMutex
	
	// Modified stats from affects
	modifiedStrength   int
	modifiedDexterity  int
	modifiedIntelligence int
	modifiedWisdom     int
	modifiedConstitution int
	modifiedCharisma   int
	
	modifiedHitRoll    int
	modifiedDamageRoll int
	modifiedArmorClass int
	modifiedTHAC0      int
	
	modifiedHP         int
	modifiedMaxHP      int
	modifiedMana       int
	modifiedMaxMana    int
	modifiedMovement   int
	
	statusFlags        uint64
	affects            []*Affect
}

func NewPlayerAffectable(player *game.Player) *PlayerAffectable {
	return &PlayerAffectable{
		player: player,
		// Initialize modified stats from player's base stats
		modifiedStrength:   player.Stats.Str,
		modifiedDexterity:  player.Stats.Dex,
		modifiedIntelligence: player.Stats.Int,
		modifiedWisdom:     player.Stats.Wis,
		modifiedConstitution: player.Stats.Con,
		modifiedCharisma:   player.Stats.Cha,
		modifiedHitRoll:    player.Hitroll,
		modifiedDamageRoll: player.Damroll,
		modifiedArmorClass: player.AC,
		modifiedTHAC0:      player.THAC0,
		modifiedHP:         player.Health,
		modifiedMaxHP:      player.MaxHealth,
		modifiedMana:       player.Mana,
		modifiedMaxMana:    player.MaxMana,
		modifiedMovement:   100, // Default movement
		statusFlags:        0,
		affects:            make([]*Affect, 0),
	}
}

// Implement Affectable interface
func (pa *PlayerAffectable) GetAffects() []*Affect {
	pa.mu.RLock()
	defer pa.mu.RUnlock()
	return pa.affects
}

func (pa *PlayerAffectable) SetAffects(affects []*Affect) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	pa.affects = affects
}

func (pa *PlayerAffectable) GetName() string {
	return pa.player.Name
}

func (pa *PlayerAffectable) GetID() int {
	return pa.player.ID
}

// Stat getters/setters
func (pa *PlayerAffectable) GetStrength() int {
	pa.mu.RLock()
	defer pa.mu.RUnlock()
	return pa.modifiedStrength
}

func (pa *PlayerAffectable) SetStrength(v int) {
	pa.mu.Lock()
	defer pa.mu.Unlock()
	pa.modifiedStrength = v
}

// ... implement other stat getters/setters similarly

func (pa *PlayerAffectable) SendMessage(msg string) {
	// Send message to player's connection
	select {
	case pa.player.Send <- []byte(msg + "\n"):
		// Message sent
	default:
		// Channel full, message dropped
	}
}

// Example: Integrating with combat system
func ExampleCombatWithAffects() {
	// Create affect system
	ats := NewAffectTickSystem()
	
	// Create players
	player1 := &game.Player{Name: "Warrior", ID: 1}
	player2 := &game.Player{Name: "Mage", ID: 2}
	
	// Wrap players with affectable interface
	affectable1 := NewPlayerAffectable(player1)
	affectable2 := NewPlayerAffectable(player2)
	
	// Apply haste to warrior
	hasteAffect := NewAffect(AffectHaste, 30, 0, "haste spell")
	ats.ApplyAffect(affectable1, hasteAffect)
	
	// Apply poison to mage
	poisonAffect := NewAffect(AffectPoison, 10, 3, "poison dart")
	ats.ApplyAffect(affectable2, poisonAffect)
	
	// Start tick system
	ats.Start()
	
	// In combat loop:
	// 1. Check for haste/slow affects to modify attack speed
	// 2. Check for strength/dexterity affects to modify hit/damage
	// 3. Check for poison/regeneration for periodic effects
	// 4. Check for status affects (blind, stunned, etc.)
	
	// Stop when done
	ats.Stop()
}

// Example: Spell casting integration
func ExampleSpellCasting() {
	ats := NewAffectTickSystem()
	
	// Cast strength spell
	func CastStrengthSpell(caster, target Affectable) {
		// Calculate spell duration based on caster level
		duration := 10 // Base duration
		magnitude := 5 // Base magnitude
		
		// Apply affect
		strengthAffect := NewAffect(AffectStrength, duration, magnitude, "strength spell")
		ats.ApplyAffect(target, strengthAffect)
		
		// Send messages
		caster.SendMessage("You cast strength on " + target.GetName())
		target.SendMessage("You feel stronger!")
	}
	
	// Cast cure poison
	func CastCurePoison(caster, target Affectable) {
		// Remove poison affects
		removed := ats.AffectManager.RemoveAffectsByType(target, AffectPoison)
		
		if removed > 0 {
			caster.SendMessage("You cure " + target.GetName() + "'s poison")
			target.SendMessage("The poison leaves your body")
		} else {
			caster.SendMessage(target.GetName() + " is not poisoned")
		}
	}
}

// Example: Item with permanent affect
func ExampleMagicItem() {
	// Create a magic sword that gives +2 strength when wielded
	type MagicSword struct {
		strengthAffect *Affect
		owner          Affectable
	}
	
	func (sword *MagicSword) OnWield(owner Affectable, ats *AffectTickSystem) {
		sword.owner = owner
		sword.strengthAffect = NewAffect(AffectStrength, 0, 2, "Magic Sword")
		ats.ApplyAffect(owner, sword.strengthAffect)
	}
	
	func (sword *MagicSword) OnRemove(ats *AffectTickSystem) {
		if sword.owner != nil && sword.strengthAffect != nil {
			ats.RemoveAffect(sword.owner, sword.strengthAffect.ID)
			sword.owner = nil
		}
	}
}
*/