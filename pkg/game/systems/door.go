// Package world provides door management for Dark Pawns.
package systems

import (
	"strings"
	"sync"
)

// Door represents a door between rooms with various states and properties.
// Based on original MUD door flags: closed, locked, pickproof, bashable, hidden, etc.
// Door has its own mutex; callers that obtain a *Door from DoorManager may read
// state through the exported accessor methods, while mutating methods lock
// internally. Direct field access is retained for tests and package helpers that
// already hold the enclosing DoorManager lock or operate on a single goroutine.
type Door struct {
	mu sync.RWMutex

	// Basic state
	Closed    bool // Door is closed (can't pass through)
	Locked    bool // Door is locked (requires key or picking)
	Pickproof bool // Door cannot be picked
	Bashable  bool // Door can be bashed down
	Hidden    bool // Door is hidden (not visible without detect hidden)

	// Door properties
	KeyVNum    int // VNum of key that unlocks this door (-1 for no key)
	Difficulty int // Lock difficulty (0-100, higher = harder to pick)
	Hp         int // Door hit points for bashing (0 = destroyed)
	MaxHp      int // Maximum door hit points

	// Connection info
	FromRoom  int    // Source room VNum
	ToRoom    int    // Destination room VNum
	Direction string // Direction (north, south, east, west, up, down)

	// Initial state for resets
	initialClosed bool
	initialLocked bool
}

// DoorSnapshot is a concurrency-safe, immutable copy of a Door's exported
// state. DoorManager getters return DoorSnapshot instead of *Door so callers
// can't mutate a door's fields outside the Door's own lock; unlike Door, it
// holds no mutex, so it's safe to copy and pass around by value.
type DoorSnapshot struct {
	Closed    bool
	Locked    bool
	Pickproof bool
	Bashable  bool
	Hidden    bool

	KeyVNum    int
	Difficulty int
	Hp         int
	MaxHp      int

	FromRoom  int
	ToRoom    int
	Direction string
}

// IsClosed reports whether the door is closed.
func (s DoorSnapshot) IsClosed() bool { return s.Closed }

// IsLocked reports whether the door is locked.
func (s DoorSnapshot) IsLocked() bool { return s.Locked }

// IsHidden reports whether the door is hidden.
func (s DoorSnapshot) IsHidden() bool { return s.Hidden }

// IsPickproof reports whether the door is pickproof.
func (s DoorSnapshot) IsPickproof() bool { return s.Pickproof }

// IsBashable reports whether the door can be bashed.
func (s DoorSnapshot) IsBashable() bool { return s.Bashable }

// GetHp returns the door's hit points as of the snapshot.
func (s DoorSnapshot) GetHp() int { return s.Hp }

// GetKeyVNum returns the VNum of the key that unlocks this door.
func (s DoorSnapshot) GetKeyVNum() int { return s.KeyVNum }

// GetToRoom returns the destination room VNum.
func (s DoorSnapshot) GetToRoom() int { return s.ToRoom }

// GetDirection returns the door's direction string.
func (s DoorSnapshot) GetDirection() string { return s.Direction }

// IsPassable returns true if a player can pass through this door.
func (s DoorSnapshot) IsPassable() bool { return !s.Closed }

// CanSee returns true if the door is visible (not hidden).
func (s DoorSnapshot) CanSee() bool { return !s.Hidden }

// GetStatus returns a string describing the door's state.
func (s DoorSnapshot) GetStatus() string {
	if s.Hidden {
		return "hidden"
	}
	if s.Closed {
		if s.Locked {
			return "closed and locked"
		}
		return "closed"
	}
	return "open"
}

// GetDescription returns a descriptive string for the door.
func (s DoorSnapshot) GetDescription() string {
	parts := []string{}

	if s.Hidden {
		parts = append(parts, "hidden")
	}

	if s.Closed {
		parts = append(parts, "closed")
		if s.Locked {
			parts = append(parts, "locked")
		}
		if s.Pickproof {
			parts = append(parts, "pickproof")
		}
		if s.Bashable {
			parts = append(parts, "bashable")
		}
	} else {
		parts = append(parts, "open")
	}

	return strings.Join(parts, ", ")
}

// Snapshot returns a concurrency-safe copy of the door's exported state.
func (d *Door) Snapshot() DoorSnapshot {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return DoorSnapshot{
		Closed:     d.Closed,
		Locked:     d.Locked,
		Pickproof:  d.Pickproof,
		Bashable:   d.Bashable,
		Hidden:     d.Hidden,
		KeyVNum:    d.KeyVNum,
		Difficulty: d.Difficulty,
		Hp:         d.Hp,
		MaxHp:      d.MaxHp,
		FromRoom:   d.FromRoom,
		ToRoom:     d.ToRoom,
		Direction:  d.Direction,
	}
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

	d.initialClosed = d.Closed
	d.initialLocked = d.Locked

	return d
}

// IsClosed reports whether the door is closed.
func (d *Door) IsClosed() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Closed
}

// IsLocked reports whether the door is locked.
func (d *Door) IsLocked() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Locked
}

// IsHidden reports whether the door is hidden.
func (d *Door) IsHidden() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Hidden
}

// IsPickproof reports whether the door is pickproof.
func (d *Door) IsPickproof() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Pickproof
}

// IsBashable reports whether the door can be bashed.
func (d *Door) IsBashable() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Bashable
}

// GetHp returns the door's current hit points.
func (d *Door) GetHp() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Hp
}

// GetKeyVNum returns the VNum of the key that unlocks this door.
func (d *Door) GetKeyVNum() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.KeyVNum
}

// GetToRoom returns the destination room VNum.
func (d *Door) GetToRoom() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.ToRoom
}

// GetDirection returns the door's direction string.
func (d *Door) GetDirection() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Direction
}

// IsPassable returns true if a player can pass through this door.
func (d *Door) IsPassable() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return !d.Closed
}

// CanSee returns true if the door is visible (not hidden).
func (d *Door) CanSee() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return !d.Hidden
}

// Open attempts to open the door.
// Returns true if successful, false otherwise with a reason.
func (d *Door) Open() (bool, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
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
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.Closed {
		return false, "It's already closed."
	}

	d.Closed = true
	return true, "You close the door."
}

// Lock attempts to lock the door with a key.
// keyVNum is the VNum of the key being used.
func (d *Door) Lock(keyVNum int) (bool, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
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
	d.mu.Lock()
	defer d.mu.Unlock()
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
	d.mu.Lock()
	defer d.mu.Unlock()
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

// reverseDirection returns the opposite compass direction.
func reverseDirection(dir string) string {
	switch dir {
	case "north":
		return "south"
	case "south":
		return "north"
	case "east":
		return "west"
	case "west":
		return "east"
	case "up":
		return "down"
	case "down":
		return "up"
	case "northeast":
		return "southwest"
	case "southwest":
		return "northeast"
	case "northwest":
		return "southeast"
	case "southeast":
		return "northwest"
	case "in":
		return "out"
	case "out":
		return "in"
	default:
		return ""
	}
}

// Bash attempts to bash the door down.
// strength is the player's strength or bash skill.
func (d *Door) Bash(strength int) (bool, string) {
	d.mu.Lock()
	defer d.mu.Unlock()
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

	return false, "You bash the door. It looks damaged."
}

// GetStatus returns a string describing the door's state.
func (d *Door) GetStatus() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.Hidden {
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
	d.mu.RLock()
	defer d.mu.RUnlock()
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
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Hp = d.MaxHp
	d.Closed = d.initialClosed
	d.Locked = d.initialLocked
}
