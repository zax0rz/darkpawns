# Affect System for Dark Pawns

## Overview

The affect system manages temporary effects (buffs, debuffs, and status effects) on characters, mobs, and objects in Dark Pawns. It handles application, removal, duration tracking, stacking rules, and periodic effects.

## Core Components

### 1. Affect (`affect.go`)
The `Affect` struct represents a single temporary effect:

```go
type Affect struct {
    ID          string      // Unique identifier
    Type        AffectType  // Type of affect (strength, poison, haste, etc.)
    Duration    int         // Duration in ticks (0 = permanent)
    Magnitude   int         // Effect magnitude (positive for buffs, negative for debuffs)
    Flags       uint64      // Bitmask of affect flags
    Source      string      // Source of affect (spell, item, etc.)
    AppliedAt   time.Time   // When applied
    ExpiresAt   time.Time   // When expires
    StackID     string      // ID for stacking purposes
    MaxStacks   int         // Maximum number of stacks
}
```

### 2. Affect Types
The system supports three categories of affects:

- **Stat Modifiers**: Strength, Dexterity, Intelligence, Wisdom, Constitution, Charisma
- **Combat Modifiers**: Hit Roll, Damage Roll, Armor Class, THAC0, HP, Mana, Movement
- **Status Effects**: Blind, Invisible, Poison, Haste, Slow, Regeneration, etc.

### 3. Affect Manager (`affect_manager.go`)
The `AffectManager` handles all affect operations:

- Apply/remove affects to entities
- Track affect durations
- Handle stacking rules
- Apply periodic effects (poison damage, regeneration)
- Send notifications to entities

### 4. Tick System (`affect_tick.go`)
The `TickManager` provides global tick processing:

- Processes all affects at regular intervals
- Handles affect expiration
- Can be configured with custom tick intervals

## Usage Examples

### Applying an Affect

```go
// Create affect manager
am := engine.NewAffectManager()

// Create a player or mob (must implement Affectable interface)
player := &game.Player{Name: "TestPlayer", ID: 1}

// Register entity with affect manager
am.RegisterEntity(player)

// Create a strength buff affect (duration: 10 ticks, +5 strength)
strengthAffect := engine.NewAffect(engine.AffectStrength, 10, 5, "strength potion")

// Apply the affect
am.ApplyAffect(player, strengthAffect)
```

### Applying a Status Effect

```go
// Create a poison affect (duration: 5 ticks, magnitude 3)
poisonAffect := engine.NewAffect(engine.AffectPoison, 5, 3, "poison dart")

// Apply poison
am.ApplyAffect(player, poisonAffect)

// Poison will automatically damage the player each tick
```

### Using the Tick System

```go
// Create affect tick system
ats := engine.NewAffectTickSystem()

// Start automatic tick processing (default: 1 tick per second)
ats.Start()

// Apply affects as needed
hasteAffect := engine.NewAffect(engine.AffectHaste, 30, 0, "haste spell")
ats.ApplyAffect(player, hasteAffect)

// Manual tick (useful for testing)
ats.ManualTick()

// Stop tick system when done
ats.Stop()
```

### Checking and Removing Affects

```go
// Check if entity has a specific affect type
if ats.HasAffect(player, engine.AffectPoison) {
    fmt.Println("Player is poisoned!")
}

// Get all affects on an entity
affects := ats.GetAffects(player)
for _, aff := range affects {
    fmt.Printf("Affect: %v, Duration: %d\n", aff.Type, aff.Duration)
}

// Remove a specific affect
ats.RemoveAffect(player, affectID)

// Remove all affects of a type
removed := am.RemoveAffectsByType(player, engine.AffectPoison)

// Remove all affects
count := ats.RemoveAllAffects(player)
```

## Stacking Rules

The affect system supports flexible stacking rules:

1. **No Stacking** (default): New affect replaces existing affect of same type
2. **Limited Stacking**: Up to `MaxStacks` affects of same `StackID` can coexist
3. **Infinite Stacking**: Unlimited stacks (not recommended for balance)

Example with stacking:

```go
// Create poison affect that stacks up to 3 times
poisonAffect := engine.NewAffect(engine.AffectPoison, 10, 2, "weak poison")
poisonAffect.StackID = "poison"
poisonAffect.MaxStacks = 3

// First application
am.ApplyAffect(player, poisonAffect) // Poison damage: 2/tick

// Second application (stacks)
am.ApplyAffect(player, poisonAffect) // Poison damage: 4/tick

// Third application (stacks)
am.ApplyAffect(player, poisonAffect) // Poison damage: 6/tick

// Fourth application replaces oldest stack
am.ApplyAffect(player, poisonAffect) // Still 6/tick (replaced oldest)
```

## Periodic Effects

Some affects have periodic effects that trigger each tick:

- **Poison**: Damages HP each tick
- **Regeneration**: Heals HP each tick
- **Haste/Slow**: Modify attack speed (handled elsewhere)

Periodic effects are automatically applied during the tick cycle.

## Integration with Existing Systems

### Player Integration
To make `Player` affectable, implement the `Affectable` interface:

```go
func (p *Player) GetStrength() int {
    return p.Stats.Str
}

func (p *Player) SetStrength(v int) {
    p.Stats.Str = v
}

// ... implement other required methods
```

### Mob Integration
Similar to players, `MobInstance` should implement `Affectable`:

```go
func (m *MobInstance) GetStrength() int {
    if m.Prototype != nil {
        return m.Prototype.Strength
    }
    return 10
}

func (m *MobInstance) SetStrength(v int) {
    // Store modified strength
    m.ModifiedStrength = v
}
```

### Combat Integration
Affects should modify combat calculations:

```go
// In combat formulas, check for affects
func CalculateDamage(attacker, defender Affectable) int {
    baseDamage := rollDice(attacker.GetDamageRoll())
    
    // Apply strength bonus if attacker has strength buff
    if attacker.HasAffect(AffectStrength) {
        strengthBonus := attacker.GetStrength() - 10 // Example calculation
        baseDamage += strengthBonus
    }
    
    // Apply damage reduction if defender has protection
    if defender.HasAffect(AffectProtectionEvil) {
        baseDamage /= 2
    }
    
    return baseDamage
}
```

## Database Persistence

Affects can be saved and loaded with character/mob data:

```go
// Serialize affects to JSON
func (am *AffectManager) SaveAffects(entity Affectable) ([]byte, error) {
    affects := am.GetAffects(entity)
    return json.Marshal(affects)
}

// Deserialize affects from JSON
func (am *AffectManager) LoadAffects(entity Affectable, data []byte) error {
    var affects []*Affect
    if err := json.Unmarshal(data, &affects); err != nil {
        return err
    }
    
    // Clear existing affects
    am.RemoveAllAffects(entity)
    
    // Apply loaded affects
    for _, aff := range affects {
        am.ApplyAffect(entity, aff)
    }
    
    return nil
}
```

## Testing

Run the test suite:

```bash
cd /home/zach/.openclaw/workspace/darkpawns_repo
go test ./pkg/engine/... -v
```

Tests cover:
- Affect creation and ticking
- Affect application and removal
- Stacking rules
- Periodic effects
- Tick manager functionality

## Performance Considerations

1. **Tick Frequency**: Default is 1 tick per second. Adjust based on game needs.
2. **Affect Count**: Large numbers of affects per entity may impact performance.
3. **Memory Usage**: Affects are stored in memory. Consider limits for long-duration affects.
4. **Concurrency**: The system uses mutexes for thread-safe operations.

## Future Enhancements

1. **Affect Resistance**: Chance to resist certain affect types
2. **Affect Dispelling**: Magic to remove specific affect types
3. **Affect Immunity**: Entities immune to certain affect types
4. **Visual Effects**: Client-side representation of affects
5. **Affect Combinations**: Synergies between different affects

## See Also

- `pkg/scripting/engine.go` - Lua scripting integration
- `pkg/combat/` - Combat system integration
- `pkg/game/` - Player and mob implementations