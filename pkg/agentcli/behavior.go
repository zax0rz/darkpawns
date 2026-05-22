package agentcli

import (
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
)

// BehaviorPattern is a single pattern-matched rule for the behavior tree.
// When Matcher returns true, Action produces the LLMResponse to execute.
// Patterns are evaluated in descending Priority order; the first match wins.
type BehaviorPattern struct {
	Name     string
	Priority int
	Matcher  func(*GameState) bool
	Action   func(*GameState) *LLMResponse
}

// BehaviorEngine evaluates a set of BehaviorPatterns against the current
// game state. If no pattern matches, it returns nil so the caller can
// fall through to the LLM.
type BehaviorEngine struct {
	patterns []BehaviorPattern
	mu       sync.RWMutex
}

// NewBehaviorEngine creates an engine pre-loaded with the built-in patterns.
func NewBehaviorEngine() *BehaviorEngine {
	be := &BehaviorEngine{}
	be.addBuiltins()
	return be
}

// Evaluate returns the action from the highest-priority matching pattern,
// or nil if no pattern matches.
func (be *BehaviorEngine) Evaluate(state *GameState) *LLMResponse {
	if state == nil {
		return nil
	}

	be.mu.RLock()
	defer be.mu.RUnlock()

	// Patterns are kept sorted by priority descending (descending = higher first).
	for i := range be.patterns {
		p := &be.patterns[i]
		if p.Matcher(state) {
			slog.Debug("behavior match", "pattern", p.Name, "priority", p.Priority)
			return p.Action(state)
		}
	}
	return nil
}

// AddPattern inserts a pattern and re-sorts by priority descending.
func (be *BehaviorEngine) AddPattern(p BehaviorPattern) {
	be.mu.Lock()
	defer be.mu.Unlock()
	be.patterns = append(be.patterns, p)
	be.sort()
}

// RemovePattern removes a pattern by name. Returns true if found and removed.
func (be *BehaviorEngine) RemovePattern(name string) bool {
	be.mu.Lock()
	defer be.mu.Unlock()
	for i := range be.patterns {
		if be.patterns[i].Name == name {
			be.patterns = append(be.patterns[:i], be.patterns[i+1:]...)
			return true
		}
	}
	return false
}

// Patterns returns a snapshot of the current pattern list (for testing).
func (be *BehaviorEngine) Patterns() []BehaviorPattern {
	be.mu.RLock()
	defer be.mu.RUnlock()
	out := make([]BehaviorPattern, len(be.patterns))
	copy(out, be.patterns)
	return out
}

func (be *BehaviorEngine) sort() {
	// Simple insertion sort — patterns are small.
	for i := 1; i < len(be.patterns); i++ {
		key := be.patterns[i]
		j := i - 1
		for j >= 0 && be.patterns[j].Priority < key.Priority {
			be.patterns[j+1] = be.patterns[j]
			j--
		}
		be.patterns[j+1] = key
	}
}

// ---------------------------------------------------------------------------
// Built-in patterns
// ---------------------------------------------------------------------------

func (be *BehaviorEngine) addBuiltins() {
	// Higher priority = evaluated first.
	be.AddPattern(fleeWhenLowHP())
	be.AddPattern(avoidDangerousRooms())
	be.AddPattern(respondToGreeting())
	be.AddPattern(collectNearbyGold())
	be.AddPattern(equipBestWeapon())
	be.AddPattern(autoAttack())
}

// fleeWhenLowHP replaces the old FSMDecision flee path.
// Triggers when HP < 25% of MaxHealth AND in combat.
func fleeWhenLowHP() BehaviorPattern {
	return BehaviorPattern{
		Name:     "flee_when_low_hp",
		Priority: 100,
		Matcher: func(s *GameState) bool {
			if s.Player.MaxHealth <= 0 {
				return false
			}
			hpPct := s.Player.Health * 100 / s.Player.MaxHealth
			return hpPct < 25 && s.Fighting != ""
		},
		Action: func(_ *GameState) *LLMResponse {
			return &LLMResponse{ActionType: "flee"}
		},
	}
}

// avoidDangerousRooms flees when the room name signals obvious danger.
func avoidDangerousRooms() BehaviorPattern {
	return BehaviorPattern{
		Name:     "avoid_dangerous_rooms",
		Priority: 90,
		Matcher: func(s *GameState) bool {
			name := strings.ToLower(s.Room.Name)
			return strings.Contains(name, "dragon") ||
				strings.Contains(name, "lava") ||
				strings.Contains(name, "abyss")
		},
		Action: func(_ *GameState) *LLMResponse {
			return &LLMResponse{ActionType: "flee"}
		},
	}
}

// respondToGreeting replies to a player's hello/hi/greet in recent events.
// The event data is checked for common greetings in the tell/message text.
func respondToGreeting() BehaviorPattern {
	greetings := []string{"hello", "hi ", "hi!", "greet", "hey "}
	return BehaviorPattern{
		Name:     "respond_to_greeting",
		Priority: 50,
		Matcher: func(s *GameState) bool {
			for _, ev := range s.Events {
				if ev.Type != "tell" && ev.Type != "speech" {
					continue
				}
				text := strings.ToLower(ev.Text())
				for _, g := range greetings {
					if strings.Contains(text, g) {
						return true
					}
				}
			}
			return false
		},
		Action: func(_ *GameState) *LLMResponse {
			return &LLMResponse{
				ActionType: "say",
				SayLine:    "Hello!",
			}
		},
	}
}

// collectNearbyGold picks up gold items in the room.
func collectNearbyGold() BehaviorPattern {
	return BehaviorPattern{
		Name:     "collect_nearby_gold",
		Priority: 30,
		Matcher: func(s *GameState) bool {
			for _, item := range s.Room.Items {
				if strings.Contains(strings.ToLower(item.Name), "gold") ||
					strings.Contains(strings.ToLower(item.Name), "coin") {
					return true
				}
			}
			return false
		},
		Action: func(s *GameState) *LLMResponse {
			for _, item := range s.Room.Items {
				if strings.Contains(strings.ToLower(item.Name), "gold") ||
					strings.Contains(strings.ToLower(item.Name), "coin") {
					return &LLMResponse{
						ActionType: "get",
						Args:       []string{item.Name},
					}
				}
			}
			return nil // unreachable
		},
	}
}

// equipBestWeapon wields a weapon from inventory if none is equipped.
func equipBestWeapon() BehaviorPattern {
	return BehaviorPattern{
		Name:     "equip_best_weapon",
		Priority: 20,
		Matcher: func(s *GameState) bool {
			if _, ok := s.Equipment["wield"]; ok {
				return false // already wielding something
			}
			for _, item := range s.Inventory {
				if strings.Contains(strings.ToLower(item.Name), "sword") ||
					strings.Contains(strings.ToLower(item.Name), "axe") ||
					strings.Contains(strings.ToLower(item.Name), "dagger") ||
					strings.Contains(strings.ToLower(item.Name), "mace") ||
					strings.Contains(strings.ToLower(item.Name), "staff") {
					return true
				}
			}
			return false
		},
		Action: func(s *GameState) *LLMResponse {
			for _, item := range s.Inventory {
				if strings.Contains(strings.ToLower(item.Name), "sword") ||
					strings.Contains(strings.ToLower(item.Name), "axe") ||
					strings.Contains(strings.ToLower(item.Name), "dagger") ||
					strings.Contains(strings.ToLower(item.Name), "mace") ||
					strings.Contains(strings.ToLower(item.Name), "staff") {
					return &LLMResponse{
						ActionType: "wield",
						Args:       []string{item.Name},
					}
				}
			}
			return nil // unreachable
		},
	}
}

// autoAttack attacks the nearest hostile mob when in combat.
func autoAttack() BehaviorPattern {
	return BehaviorPattern{
		Name:     "auto_attack",
		Priority: 10,
		Matcher: func(s *GameState) bool {
			if s.Fighting == "" {
				return false
			}
			// In combat — pick the first mob that's fighting us.
			for _, mob := range s.Room.Mobs {
				if mob.Fighting {
					return true
				}
			}
			return false
		},
		Action: func(s *GameState) *LLMResponse {
			for _, mob := range s.Room.Mobs {
				if mob.Fighting {
					target := mob.TargetString
					if target == "" {
						target = mob.Name
					}
					return &LLMResponse{
						ActionType: "hit",
						Args:       []string{target},
					}
				}
			}
			return nil // unreachable
		},
	}
}

// Text returns the raw event text for greeting detection.
// For tell/speech events the server sends a structure with a "message" field.
func (e Event) Text() string {
	// Marshal then scan — handles any Data type uniformly.
	raw, err := json.Marshal(e.Data)
	if err != nil {
		return ""
	}
	s := string(raw)
	// Fast extract: look for "message":"..."
	if idx := strings.Index(s, `"message":`); idx >= 0 {
		start := idx + len(`"message":`)
		for start < len(s) && s[start] == ' ' {
			start++
		}
		if start < len(s) && s[start] == '"' {
			start++
			end := strings.IndexByte(s[start:], '"')
			if end > 0 {
				return s[start : start+end]
			}
		}
	}
	return s
}
