---
tags: [active]
---
# Phase 2.3: Mob Spawning from Zone Files

## Overview
This phase implements mob and object spawning from zone reset commands (.zon files). The system parses zone commands and spawns mobs/objects in the world with proper tracking and limits.

## Files Created/Modified

### New Files
1. **pkg/game/spawner.go** - Main spawner system
   - `Spawner` struct: Tracks spawned mobs/objects
   - `ExecuteZoneReset()`: Executes all zone commands
   - `CanSpawn()`: Checks maxInWorld limits
   - `SpawnMob()`/`SpawnObject()`: Creates instances
   - Periodic reset system (every 15 minutes)

2. **pkg/game/mob.go** - Mob instance management
   - `MobInstance` struct: Runtime mob state
   - Implements `ai.Mob` interface for AI integration
   - HP management, inventory, equipment
   - Combat methods (Attack, TakeDamage, etc.)

3. **pkg/game/object.go** - Object instance management
   - `ObjectInstance` struct: Runtime object state
   - Location tracking (room, carrier, container)
   - Container functionality
   - Equipment state

### Modified Files
1. **pkg/game/world.go** - Integration
   - Added `spawner` field
   - Added `GetMobPrototype()`, `GetObjPrototype()`, `GetZone()` methods
   - Added `StartZoneResets()` and `StartPeriodicResets()`
   - Updated to use `MobInstance` instead of hypothetical `Mob` type

## Key Features

### Zone Command Support
- **M**: Load mobile into room (with maxInWorld limit)
- **O**: Load object into room (with maxInWorld limit)
- **G**: Give object to last loaded mob
- **E**: Equip object on last loaded mob
- **P**: Put object in container
- **D**: Door state changes
- **R**: Remove obj/mob from room

### Limits and Tracking
- `maxInWorld` limits enforced via `CanSpawn()`
- Spawner tracks all spawned instances by vnum and room
- Prevents spawning beyond configured limits

### Integration with Existing Systems
- Uses existing parser types (`parser.Zone`, `parser.Mob`, `parser.Obj`)
- Implements `ai.Mob` interface for AI compatibility
- Works with existing world room/mob/obj/zone indexing

### Periodic Resets
- Zone resets execute on server start
- Periodic checks every 15 minutes for empty zones
- Configurable interval via `StartPeriodicResets()`

## Usage Example

```go
// Create world from parsed data
world, err := game.NewWorld(parsed)

// Start zone resets (spawns mobs/objects from zone files)
err = world.StartZoneResets()

// Start periodic resets (every 15 minutes)
world.StartPeriodicResets(15 * time.Minute)
```

## Testing
A test file `test_spawner.go` is included demonstrating:
- Creating a minimal world with test data
- Executing zone resets
- Starting periodic resets

## Notes
- The spawner integrates with the existing AI tick system
- Mob instances have AI brains for behavior (aggressive, wandering, sentinel)
- Object instances support containers and equipment
- Zone reset mode (0=never, 1=if empty, 2=always) is parsed but basic implementation always resets