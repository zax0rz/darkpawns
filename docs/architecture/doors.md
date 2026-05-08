**Last updated:** 2026-05-08

# Door System for Dark Pawns

## Overview

The door system provides functionality for doors between rooms in the Dark Pawns MUD server. Doors can be open, closed, locked, pickproof, bashable, hidden, and more — matching the original CircleMUD door semantics.

## Architecture

### Core Components

1. **Door Struct** (`pkg/game/systems/door.go`)
   - Represents a single door with state and properties
   - Manages door operations (open, close, lock, unlock, pick, bash)
   - Tracks door health for bashing
   - Fields: `Closed`, `Locked`, `Pickproof`, `Bashable`, `Hidden`, `KeyVNum`, `Difficulty`, `Hp`, `MaxHp`, `FromRoom`, `ToRoom`, `Direction`

2. **DoorManager** (`pkg/game/systems/door_manager.go`)
   - Manages all doors in the world
   - Provides door lookup by room and direction
   - Handles door operations with proper synchronization

3. **Door Commands** (`pkg/session/door_cmds.go`)
   - Player commands for interacting with doors
   - Commands: open, close, lock, unlock, pick, bash

### Integration Points

- **Parser Integration**: Doors are created from parsed room exits (`parser.Exit`)
- **World Integration**: DoorManager integrated with the game World
- **Command Integration**: Door commands registered with the command dispatcher via `pkg/session/door_cmds.go`

## Door States and Properties

### Basic States
- **Open**: Players can pass through
- **Closed**: Players cannot pass through (but can open)
- **Locked**: Closed and requires key or picking to open

### Door Properties
| Field | Type | Description |
|-------|------|-------------|
| `Closed` | bool | Door is closed |
| `Locked` | bool | Door is locked |
| `Pickproof` | bool | Cannot be picked (requires key) |
| `Bashable` | bool | Can be bashed down (reduces door HP) |
| `Hidden` | bool | Not visible without detect hidden skill |
| `KeyVNum` | int | VNum of key that unlocks this door (-1 for no key) |
| `Difficulty` | int | Lock difficulty 0–100, higher = harder to pick |
| `Hp` | int | Current door health for bashing |
| `MaxHp` | int | Maximum door health (0 = indestructible) |
| `FromRoom` | int | VNum of room the door originates in |
| `ToRoom` | int | VNum of destination room |
| `Direction` | int | Direction index (N/E/S/W/U/D) |

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
doorManager := systems.NewDoorManager()
```

### 2. Load Doors from World Data
```go
// After parsing world files
doorManager.LoadDoorsFromWorld(parsedWorld)
```

### 3. Integrate with World
The DoorManager is part of the game World struct:
```go
type World struct {
    // ... existing fields ...
    Doors *systems.DoorManager
}
```

### 4. Register Commands
Door commands are registered via `pkg/session/door_cmds.go` with the session command dispatcher.

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

## Testing

Run tests with:
```bash
go test ./pkg/game/systems/... -v
```
