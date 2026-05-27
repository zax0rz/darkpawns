package agentcli

import (
	"encoding/json"
	"fmt"
	"sync"
)

// CreationConfig holds the choices for character creation.
type CreationConfig struct {
	Color      string `json:"color"`       // "ansi" or "normal"
	Sex        string `json:"sex"`         // "male" or "female"
	Race       string `json:"race"`        // "human", "elf", "dwarf", "kender", "minotaur", "rakshasa", "ssaur"
	Class      string `json:"class"`       // "mage", "cleric", "thief", "warrior"
	Hometown   string `json:"hometown"`    // varies by class
	RerollMin  int    `json:"reroll_min"`  // min total stats to accept (0 = accept first roll)
	MaxRerolls int    `json:"max_rerolls"` // max reroll attempts (default 3)
}

// CreationState tracks which step of the creation wizard we're in.
type CreationState int

const (
	CreationIdle CreationState = iota
	CreationColor
	CreationSex
	CreationRace
	CreationClass
	CreationHometown
	CreationStats
	CreationComplete
)

// charCreateMessage represents the data payload of a char_create message.
type charCreateMessage struct {
	Prompt  string         `json:"prompt"`
	Options []string       `json:"options"`
	Stats   map[string]int `json:"stats,omitempty"`
}

// CharacterCreator drives the server's character creation wizard.
type CharacterCreator struct {
	cfg         CreationConfig
	state       CreationState
	bestTotal   int
	rerollCount int
	mu          sync.Mutex
}

// NewCharacterCreator creates a new character creator with the given config.
func NewCharacterCreator(cfg CreationConfig) *CharacterCreator {
	maxRerolls := cfg.MaxRerolls
	if maxRerolls == 0 {
		maxRerolls = 3
	}
	cfg.MaxRerolls = maxRerolls
	return &CharacterCreator{
		cfg:   cfg,
		state: CreationIdle,
	}
}

// HandleCharCreate processes a char_create message from the server.
// Returns the response to send back, or nil if creation is complete.
func (cc *CharacterCreator) HandleCharCreate(data json.RawMessage) (*map[string]any, error) {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if cc.state == CreationComplete {
		return nil, nil
	}

	var msg charCreateMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("parse char_create: %w", err)
	}

	switch cc.state {
	case CreationIdle:
		cc.state = CreationColor
		return cc.respondColor(msg)
	case CreationColor:
		cc.state = CreationSex
		return cc.respondSex(msg)
	case CreationSex:
		cc.state = CreationRace
		return cc.respondRace(msg)
	case CreationRace:
		cc.state = CreationClass
		return cc.respondClass(msg)
	case CreationClass:
		cc.state = CreationHometown
		return cc.respondHometown(msg)
	case CreationHometown:
		cc.state = CreationStats
		return cc.respondStats(msg)
	case CreationStats:
		return cc.respondStats(msg)
	default:
		return nil, fmt.Errorf("unexpected creation state %d", cc.state)
	}
}

// IsDone returns true if creation is complete.
func (cc *CharacterCreator) IsDone() bool {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	return cc.state == CreationComplete
}

func (cc *CharacterCreator) respondColor(msg charCreateMessage) (*map[string]any, error) {
	choice := cc.cfg.Color
	if choice == "" {
		choice = "ansi"
	}
	return &map[string]any{"choice": choice}, nil
}

func (cc *CharacterCreator) respondSex(msg charCreateMessage) (*map[string]any, error) {
	choice := cc.cfg.Sex
	if choice == "" {
		choice = "male"
	}
	return &map[string]any{"choice": choice}, nil
}

func (cc *CharacterCreator) respondRace(msg charCreateMessage) (*map[string]any, error) {
	choice := cc.cfg.Race
	if choice == "" {
		choice = "human"
	}
	return &map[string]any{"choice": choice}, nil
}

func (cc *CharacterCreator) respondClass(msg charCreateMessage) (*map[string]any, error) {
	choice := cc.cfg.Class
	if choice == "" {
		choice = "warrior"
	}
	return &map[string]any{"choice": choice}, nil
}

func (cc *CharacterCreator) respondHometown(msg charCreateMessage) (*map[string]any, error) {
	choice := cc.cfg.Hometown
	if choice == "" && len(msg.Options) > 0 {
		choice = msg.Options[0]
	}
	return &map[string]any{"choice": choice}, nil
}

func (cc *CharacterCreator) respondStats(msg charCreateMessage) (*map[string]any, error) {
	total := 0
	for _, v := range msg.Stats {
		total += v
	}

	// Accept first roll if no reroll minimum set
	if cc.cfg.RerollMin == 0 {
		cc.state = CreationComplete
		return &map[string]any{"choice": "accept"}, nil
	}

	// Track best roll
	if total > cc.bestTotal || cc.bestTotal == 0 {
		cc.bestTotal = total
	}

	// Accept if meets minimum
	if total >= cc.cfg.RerollMin {
		cc.state = CreationComplete
		return &map[string]any{"choice": "accept"}, nil
	}

	// Reroll if we haven't hit the limit
	cc.rerollCount++
	if cc.rerollCount >= cc.cfg.MaxRerolls {
		// Out of rerolls, accept whatever we have
		cc.state = CreationComplete
		return &map[string]any{"choice": "accept"}, nil
	}

	return &map[string]any{"choice": "reroll"}, nil
}
