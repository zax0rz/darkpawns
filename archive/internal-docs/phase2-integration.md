---
tags: [active]
---
# Phase 2 Integration Notes

## What's Done
- Combat formulas (pkg/combat/formulas.go)
- Combat engine structure (pkg/combat/engine.go) - needs interface updates
- Inventory/equipment system
- Mob spawning from zones
- AI brain and behaviors structure
- World has AI ticker (10s) and mob tracking

## What's Missing

### 1. Combat Engine Interface
The combat engine uses `*game.Player` but needs to work with both players and mobs.

**Solution:** Create a Combatant interface:
```go
type Combatant interface {
    GetName() string
    GetRoom() int
    GetHP() int
    GetMaxHP() int
    GetLevel() int
    TakeDamage(int)
    IsNPC() bool
}
```

### 2. Combat Commands
- `hit` command exists in switch but `cmdHit` not implemented
- `flee` command exists in switch but `cmdFlee` not implemented

### 3. Combat Engine Integration
- Manager needs combatEngine field
- Combat engine needs to be started in main.go
- Combat messages need to be broadcast to room

### 4. AI Integration
- AI behaviors reference `mob.Brain` but Brain is not exported properly
- Aggressive mobs need to attack on player entry
- Wandering mobs need to move between rooms

### 5. Death/Respawn
- Death handling in combat engine is placeholder
- Need corpse creation
- Need respawn logic

## Testing Checklist
- [ ] `hit goblin` starts combat
- [ ] Combat rounds every 2 seconds
- [ ] Damage calculated correctly
- [ ] `flee` escapes combat
- [ ] Death when HP <= 0
- [ ] Aggressive mob attacks on entry
- [ ] Wandering mob moves between rooms
- [ ] Equipment affects combat stats