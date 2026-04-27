# Door System for Dark Pawns

## Overview

The door system provides functionality for doors between rooms in the Dark Pawns MUD server. Doors can be open, closed, locked, pickproof, bashable, hidden, and more - similar to the original MUD door system.

## Architecture

### Core Components

1. **Door Struct** (`pkg/world/door.go`)
   - Represents a single door with state and properties
   - Manages door operations (open, close, lock, unlock, pick, bash)
   - Tracks door health for bashing

2. **DoorManager** (`pkg/world/door_manager.go`)
   - Manages all doors in the world
   - Provides door lookup by room and direction
   - Handles door operations with proper synchronization

3. **Door Commands** (`pkg/command/door_commands.go`)
   - Player commands for interacting with doors
   - Commands: open, close, lock, unlock, pick, bash

### Integration Points

- **Parser Integration**: Doors are created from parsed room exits (`parser.Exit`)
- **World Integration**: DoorManager should be integrated with the game World
- **Command Integration**: Door commands need to be registered with the command dispatcher

## Door States and Properties

### Basic States
- **Open**: Players can pass through
- **Closed**: Players cannot pass through (but can open)
- **Locked**: Closed and requires key or picking to open

### Door Properties
- **Pickproof**: Cannot be picked (requires key)
- **Bashable**: Can be bashed down (reduces door HP)
- **Hidden**: Not visible without detect hidden skill
- **KeyVNum**: VNum of key that unlocks this door (-1 for no key)
- **Difficulty**: Lock difficulty (0-100, higher = harder to pick)
- **HP/MaxHP**: Door health for bashing (0 = destroyed)

## Commands

### Basic Door Commands

```
open <direction>      - Open a door in the specified direction
close <direction>     - Close a door in the specified direction
lock <direction>      - Lock a door (requires key in inventory)
unlock <direction>    - Unlock a door (requires key in inventory)
pick <direction>      - Attempt to pick a locked door
bash <direction>      - Attempt to bash a closed door
```

### Direction Aliases
- north, n
- east, e  
- south, s
- west, w
- up, u
- down, d

## Usage Examples

### Opening and Closing Doors
```
> open north
You open the door.
Room sees: Player opens the north door.

> close north  
You close the door.
Room sees: Player closes the north door.
```

### Locking and Unlocking Doors
```
> lock east
You lock the door. (requires key 500 in inventory)
Room sees: Player locks the east door.

> unlock east
You unlock the door. (requires key 500 in inventory)
Room sees: Player unlocks the east door.
```

### Picking Locks
```
> pick south
You pick the lock. (requires sufficient picking skill)
Room sees: Player picks the lock on the south door.
```

### Bashing Doors
```
> bash west
You bash the door. It looks damaged. (reduces door HP)
Room sees: Player bashes the west door.

> bash west
You bash the door down! (when HP reaches 0)
Room sees: Player bashes the west door.
```

## Integration Guide

### 1. Initialize DoorManager
```go
doorManager := world.NewDoorManager()
```

### 2. Load Doors from World Data
```go
// After parsing world files
doorManager.LoadDoorsFromWorld(parsedWorld)
```

### 3. Integrate with World
The DoorManager should be added to the World struct:
```go
type World struct {
    // ... existing fields ...
    Doors *world.DoorManager
}
```

### 4. Register Commands
```go
doorCommands := command.NewDoorCommands(doorManager, world)
// Register with command dispatcher
```

### 5. Update Movement System
Modify the movement system to check doors:
```go
func (w *World) MovePlayer(p *Player, direction string) (*parser.Room, error) {
    // Check if door exists and is passable
    if canPass, msg := w.Doors.CanPass(p.RoomVNum, direction); !canPass {
        return nil, fmt.Errorf(msg)
    }
    // ... rest of movement logic ...
}
```

## Data Persistence

Door state should be persisted:
- **Static data**: Door properties (key, difficulty, flags) from world files
- **Dynamic state**: Door state (open/closed/locked, HP) should be saved/loaded

## Testing

Run tests with:
```bash
cd /home/zach/.openclaw/workspace/darkpawns_repo
go test ./pkg/world/... -v
```

## Future Enhancements

1. **Door Resets**: Reset doors to default state on zone reset
2. **Skill Integration**: Integrate with player skills (picking, bashing)
3. **Magic Doors**: Doors that require spells to open
4. **Trap Doors**: Doors with traps
5. **Secret Doors**: Better hidden door detection
6. **Door Descriptions**: Custom descriptions for different door types
7. **Two-way Doors**: Ensure door state is synchronized between both sides
8. **Door Sounds**: Sound effects for door operations