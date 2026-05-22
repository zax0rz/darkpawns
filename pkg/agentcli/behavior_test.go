package agentcli

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func makeState() *GameState {
	s := &GameState{}
	s.Player.Health = 100
	s.Player.MaxHealth = 100
	s.Room.Name = "The Great Hall"
	s.Room.Exits = []string{"north", "south"}
	s.Equipment = map[string]Item{}
	return s
}

func makeCombatState() *GameState {
	s := makeState()
	s.Fighting = "a goblin"
	s.Room.Mobs = []Mob{
		{Name: "goblin", TargetString: "goblin", Fighting: true},
	}
	return s
}

func makeLowHPState() *GameState {
	s := makeCombatState()
	s.Player.Health = 20
	s.Player.MaxHealth = 100 // 20%
	return s
}

// ---------------------------------------------------------------------------
// Engine basics
// ---------------------------------------------------------------------------

func TestEngineReturnsNilWhenNoMatch(t *testing.T) {
	be := NewBehaviorEngine()
	s := makeState()
	// No combat, safe room, no gold — nothing should match.
	action := be.Evaluate(s)
	if action != nil {
		t.Fatalf("expected nil action, got %+v", action)
	}
}

func TestEngineReturnsFirstMatchingByPriority(t *testing.T) {
	be := &BehaviorEngine{}
	be.AddPattern(BehaviorPattern{
		Name:     "low_priority",
		Priority: 1,
		Matcher:  func(_ *GameState) bool { return true },
		Action:   func(_ *GameState) *LLMResponse { return &LLMResponse{ActionType: "low"} },
	})
	be.AddPattern(BehaviorPattern{
		Name:     "high_priority",
		Priority: 100,
		Matcher:  func(_ *GameState) bool { return true },
		Action:   func(_ *GameState) *LLMResponse { return &LLMResponse{ActionType: "high"} },
	})

	s := makeState()
	action := be.Evaluate(s)
	if action == nil {
		t.Fatal("expected a match")
	}
	if action.ActionType != "high" {
		t.Fatalf("expected high priority match, got %q", action.ActionType)
	}
}

func TestEngineNilState(t *testing.T) {
	be := NewBehaviorEngine()
	action := be.Evaluate(nil)
	if action != nil {
		t.Fatalf("expected nil for nil state, got %+v", action)
	}
}

// ---------------------------------------------------------------------------
// Built-in pattern tests
// ---------------------------------------------------------------------------

func TestFleeWhenLowHP_Triggers(t *testing.T) {
	be := NewBehaviorEngine()
	s := makeLowHPState()
	action := be.Evaluate(s)
	if action == nil {
		t.Fatal("expected flee action at low HP")
	}
	if action.ActionType != "flee" {
		t.Fatalf("expected flee, got %q", action.ActionType)
	}
}

func TestFleeWhenLowHP_NoTrigger_AboveThreshold(t *testing.T) {
	be := NewBehaviorEngine()
	s := makeCombatState() // 100/100 HP
	action := be.Evaluate(s)
	if action != nil && action.ActionType == "flee" {
		t.Fatal("should not flee at full HP")
	}
}

func TestFleeWhenLowHP_NoTrigger_NotInCombat(t *testing.T) {
	be := NewBehaviorEngine()
	s := makeState()
	s.Player.Health = 10 // 10%
	s.Player.MaxHealth = 100
	action := be.Evaluate(s)
	if action != nil && action.ActionType == "flee" {
		t.Fatal("should not flee when not in combat")
	}
}

func TestAutoAttack_TriggersWhenInCombat(t *testing.T) {
	be := NewBehaviorEngine()
	s := makeCombatState()
	s.Player.Health = 80 // well above flee threshold
	action := be.Evaluate(s)
	if action == nil {
		t.Fatal("expected auto_attack action")
	}
	if action.ActionType != "hit" {
		t.Fatalf("expected hit, got %q", action.ActionType)
	}
	if len(action.Args) == 0 || action.Args[0] != "goblin" {
		t.Fatalf("expected target 'goblin', got %v", action.Args)
	}
}

func TestAutoAttack_NoTriggerOutOfCombat(t *testing.T) {
	be := NewBehaviorEngine()
	s := makeState()
	action := be.Evaluate(s)
	// Should be nil — no combat, safe room, nothing to pick up
	if action != nil {
		t.Fatalf("expected nil out of combat, got %+v", action)
	}
}

func TestAvoidDangerousRooms_Triggers(t *testing.T) {
	be := NewBehaviorEngine()
	s := makeState()
	s.Room.Name = "The Dragon's Lair"
	action := be.Evaluate(s)
	if action == nil {
		t.Fatal("expected flee from dangerous room")
	}
	if action.ActionType != "flee" {
		t.Fatalf("expected flee, got %q", action.ActionType)
	}
}

func TestCollectNearbyGold_Triggers(t *testing.T) {
	be := NewBehaviorEngine()
	s := makeState()
	s.Room.Items = []Item{{Name: "a pile of gold coins"}}
	action := be.Evaluate(s)
	if action == nil {
		t.Fatal("expected get gold action")
	}
	if action.ActionType != "get" {
		t.Fatalf("expected get, got %q", action.ActionType)
	}
}

func TestEquipBestWeapon_Triggers(t *testing.T) {
	be := NewBehaviorEngine()
	s := makeState()
	s.Inventory = []Item{{Name: "a rusty sword"}}
	action := be.Evaluate(s)
	if action == nil {
		t.Fatal("expected wield action")
	}
	if action.ActionType != "wield" {
		t.Fatalf("expected wield, got %q", action.ActionType)
	}
}

func TestEquipBestWeapon_NoTriggerWhenEquipped(t *testing.T) {
	be := NewBehaviorEngine()
	s := makeState()
	s.Inventory = []Item{{Name: "a rusty sword"}}
	s.Equipment["wield"] = Item{Name: "an old dagger"}
	action := be.Evaluate(s)
	if action != nil && action.ActionType == "wield" {
		t.Fatal("should not wield when already wielding")
	}
}

// ---------------------------------------------------------------------------
// Add/Remove patterns
// ---------------------------------------------------------------------------

func TestRemovePattern(t *testing.T) {
	be := NewBehaviorEngine()
	if !be.RemovePattern("auto_attack") {
		t.Fatal("expected to remove auto_attack")
	}
	// Should still have other patterns
	patterns := be.Patterns()
	for _, p := range patterns {
		if p.Name == "auto_attack" {
			t.Fatal("auto_attack should have been removed")
		}
	}
}

func TestRemoveNonexistentPattern(t *testing.T) {
	be := NewBehaviorEngine()
	if be.RemovePattern("nonexistent") {
		t.Fatal("should return false for nonexistent pattern")
	}
}

func TestPatternsSortedByPriority(t *testing.T) {
	be := NewBehaviorEngine()
	patterns := be.Patterns()
	for i := 1; i < len(patterns); i++ {
		if patterns[i].Priority > patterns[i-1].Priority {
			t.Fatalf("patterns not sorted: [%d]=%d > [%d]=%d",
				i, patterns[i].Priority, i-1, patterns[i-1].Priority)
		}
	}
}

func TestAddPatternSorting(t *testing.T) {
	be := &BehaviorEngine{}
	be.AddPattern(BehaviorPattern{Name: "c", Priority: 5})
	be.AddPattern(BehaviorPattern{Name: "a", Priority: 100})
	be.AddPattern(BehaviorPattern{Name: "b", Priority: 10})

	patterns := be.Patterns()
	if patterns[0].Name != "a" || patterns[1].Name != "b" || patterns[2].Name != "c" {
		t.Fatalf("unexpected order: %v", []string{patterns[0].Name, patterns[1].Name, patterns[2].Name})
	}
}

func TestPriorityOverride(t *testing.T) {
	be := &BehaviorEngine{}
	// Add a high-priority custom pattern that always matches
	be.AddPattern(BehaviorPattern{
		Name:     "emergency_override",
		Priority: 200, // above flee_when_low_hp (100)
		Matcher:  func(_ *GameState) bool { return true },
		Action:   func(_ *GameState) *LLMResponse { return &LLMResponse{ActionType: "panic"} },
	})

	s := makeLowHPState()
	action := be.Evaluate(s)
	if action == nil || action.ActionType != "panic" {
		t.Fatalf("expected panic override, got %+v", action)
	}
}
