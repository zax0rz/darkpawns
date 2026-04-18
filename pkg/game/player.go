package game

import (
	"sync"
	"time"
)

// Player represents an active player in the game.
type Player struct {
	mu sync.RWMutex

	// Identity
	ID   int
	Name string

	// Core stats
	Health    int
	MaxHealth int
	Mana      int
	MaxMana   int
	Level     int
	Exp       int

	// Position
	RoomVNum int // Current room

	// State
	ConnectedAt time.Time
	LastActive  time.Time

	// Communication
	Send chan []byte // Channel for sending messages to player
}

// NewPlayer creates a new player.
func NewPlayer(id int, name string, roomVNum int) *Player {
	now := time.Now()
	return &Player{
		ID:          id,
		Name:        name,
		RoomVNum:    roomVNum,
		Health:      100,
		MaxHealth:   100,
		Mana:        100,
		MaxMana:     100,
		Level:       1,
		Exp:         0,
		ConnectedAt: now,
		LastActive:  now,
		Send:        make(chan []byte, 256),
	}
}

// UpdateActivity marks the player as active.
func (p *Player) UpdateActivity() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.LastActive = time.Now()
}

// SetRoom changes the player's current room.
func (p *Player) SetRoom(vnum int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.RoomVNum = vnum
}

// GetRoom returns the player's current room VNum.
func (p *Player) GetRoom() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.RoomVNum
}