// Package world provides door management for Dark Pawns.
package systems

import (
	"fmt"
	"strings"
)

// Door represents a door between rooms with various states and properties.
// Based on original MUD door flags: closed, locked, pickproof, bashable, hidden, etc.
type Door struct {
	// Basic state
	Closed     bool // Door is closed (can't pass through)
	Locked     bool // Door is locked (requires key or picking)
	Pickproof  bool // Door cannot be picked
	Bashable   bool // Door can be bashed down
	Hidden     bool // Door is hidden (not visible without detect hidden)
	
	// Door properties
	KeyVNum    int  // VNum of key that unlocks this door (-1 for no key)
	Difficulty int  // Lock difficulty (0-100, higher = harder to pick)
	Hp         int  // Door hit points for bashing (0 = destroyed)
	MaxHp      int  // Maximum door hit points
	
	// Connection info
	FromRoom  int    // Source room VNum
	ToRoom    int    // Destination room VNum  
	Direction string // Direction (north, south, east, west, up, down)
}

// NewDoor creates a new door from an exit definition.
func NewDoor(fromRoom, toRoom int, direction string, doorState, keyVNum int) *Door {
	d := &Door{
		FromRoom:   fromRoom,
		ToRoom:     toRoom,
		Direction:  direction,
		KeyVNum:    keyVNum,
		Difficulty: 50, // Default difficulty
		Hp:         100,
		MaxHp:      100,
	}
	
	// Set initial state based on doorState (0=open, 1=closed, 2=locked)
	switch doorState {
	case 0:
		d.Closed = false
		d.Locked = false
	case 1:
		d.Closed = true
		d.Locked = false
	case 2:
		d.Closed = true
		d.Locked = true
	default:
		// Default to open
		d.Closed = false
		d.Locked = false
	}
	
	return d
}

// IsPassable returns true if a player can pass through this door.
func (d *Door) IsPassable() bool {
	return !d.Closed
}

// CanSee returns true if the door is visible (not hidden).
func (d *Door) CanSee() bool {
	return !d.Hidden
}

// Open attempts to open the door.
// Returns true if successful, false otherwise with a reason.
func (d *Door) Open() (bool, string) {
	if !d.Closed {
		return false, "It's already open."
	}
	
	if d.Locked {
		return false, "It's locked."
	}
	
	d.Closed = false
	return true, "You open the door."
}

// Close attempts to close the door.
func (d *Door) Close() (bool, string) {
	if d.Closed {
		return false, "It's already closed."
	}
	
	d.Closed = true
	return true, "You close the door."
}

// Lock attempts to lock the door with a key.
// keyVNum is the VNum of the key being used.
func (d *Door) Lock(keyVNum int) (bool, string) {
	if d.Locked {
		return false, "It's already locked."
	}
	
	if !d.Closed {
		return false, "You must close it first."
	}
	
	if d.KeyVNum != keyVNum && d.KeyVNum != -1 {
		return false, "You don't have the right key."
	}
	
	d.Locked = true
	return true, "You lock the door."
}

// Unlock attempts to unlock the door with a key.
func (d *Door) Unlock(keyVNum int) (bool, string) {
	if !d.Locked {
		return false, "It's already unlocked."
	}
	
	if d.KeyVNum != keyVNum && d.KeyVNum != -1 {
		return false, "You don't have the right key."
	}
	
	d.Locked = false
	return true, "You unlock the door."
}

// Pick attempts to pick the door lock.
// skill is the player's picking skill (0-100).
func (d *Door) Pick(skill int) (bool, string) {
	if !d.Locked {
		return false, "It's not locked."
	}
	
	if d.Pickproof {
		return false, "This lock is too complex to pick."
	}
	
	// Simple skill check: skill must be >= difficulty
	if skill < d.Difficulty {
		return false, "You fail to pick the lock."
	}
	
	d.Locked = false
	return true, "You pick the lock."
}

// Bash attempts to bash the door down.
// strength is the player's strength or bash skill.
func (d *Door) Bash(strength int) (bool, string) {
	if !d.Closed {
		return false, "It's already open."
	}
	
	if !d.Bashable {
		return false, "This door is too sturdy to bash."
	}
	
	// Simple bashing: reduce HP based on strength
	damage := strength / 10
	if damage < 1 {
		damage = 1
	}
	
	d.Hp -= damage
	
	if d.Hp <= 0 {
		// Door is destroyed
		d.Closed = false
		d.Locked = false
		d.Hp = 0
		return true, "You bash the door down!"
	}
	
	return false, fmt.Sprintf("You bash the door. It looks damaged.")
}

// GetStatus returns a string describing the door's state.
func (d *Door) GetStatus() string {
	if !d.CanSee() {
		return "hidden"
	}
	
	if d.Closed {
		if d.Locked {
			return "closed and locked"
		}
		return "closed"
	}
	return "open"
}

// GetDescription returns a descriptive string for the door.
func (d *Door) GetDescription() string {
	parts := []string{}
	
	if d.Hidden {
		parts = append(parts, "hidden")
	}
	
	if d.Closed {
		parts = append(parts, "closed")
		if d.Locked {
			parts = append(parts, "locked")
		}
		if d.Pickproof {
			parts = append(parts, "pickproof")
		}
		if d.Bashable {
			parts = append(parts, "bashable")
		}
	} else {
		parts = append(parts, "open")
	}
	
	return strings.Join(parts, ", ")
}

// Reset resets the door to its default state.
func (d *Door) Reset() {
	// Reset to closed/locked based on original state
	// For now, just reset HP
	if d.Hp <= 0 {
		d.Hp = d.MaxHp
	}
}