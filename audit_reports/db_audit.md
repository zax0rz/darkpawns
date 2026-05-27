# Port Fidelity Audit: Module 22 (`db.c`)

This audit examines the port fidelity between the legacy C source file `src/db.c` (database bootstrapping, world loading, and zone resets) and its Go implementations in `pkg/game/`, `pkg/session/`, and `pkg/parser/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/db.c` (3,220 lines)
- **Core Functions**:
  - `boot_db`, `boot_world`, `boot_world_files`, `index_boot` (drives zone, room, mob, obj, shop parsing)
  - `reset_zone`, `is_empty` (zone reset core logic)
  - `build_player_index`, `load_char`, `save_char` (player profile indexing & loading)
  - `file_to_string`, `file_to_string_alloc` (text file loaders)
  - `load_banned`, `Read_Invalid_List` (blacklist loading)

### Go Port Files
- `pkg/parser/` (Parses zones `.zon`, rooms `.wld`, mobs `.mob`, and objects `.obj` in native Diku format)
- `pkg/game/world.go` (Central memory holder of world entities: rooms, mobs, objects, players, and events queue)
- `pkg/game/world_zone.go` (Handles zone listings, and manual zone reset delegates)
- `pkg/game/spawner.go` (Executes zone reset commands, handles mob limits, rare items, and periodic timers)
- `pkg/game/zone_dispatcher.go` (Concurrent per-zone worker stubs; disabled on live server)
- `pkg/validation/validation.go` (Hardcodes name blocklists, replacing disk-based banned names files)
- `cmd/server/main.go` (Background routine coordinator that parses world and triggers initial zone load)

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. Go Struct Value-Copy Bug (Doors Never Reset)
- **Source Context**: `pkg/game/spawner.go#L424-L441` (`ExecuteZoneReset` case `"D"`)
- **Fidelity Bug**: In Go, a map of structs returns a value copy when indexed. The spawner attempts to reset exit door states (open/closed/locked) by doing:
  ```go
  ext, ok := room.Exits[roomDirNames[cmd.Arg2]]
  ...
  ext.DoorState = cmd.Arg3 // Modifies a LOCAL copy of parser.Exit struct
  lastCmd = 1
  ```
  Because `room.Exits` is a `map[string]parser.Exit` (holding value types), modifying `ext.DoorState` only updates the temporary local variable. The code **fails to write the modified exit back to the map** (`room.Exits[dir] = ext`), so the change is silently discarded.
- **Impact**: Doors never close or lock during zone resets! Once players open or unlock a door, it stays open permanently until a full server restart, completely breaking locked doors, secret passages, and dungeon boundaries.

### 2. Global First-Spawn Container Bug (P-Command Container Collision)
- **Source Context**: `pkg/game/spawner.go#L393-L403` (`ExecuteZoneReset` case `"P"`), `pkg/game/spawner.go#L578-L584` (`findObjectInstance`)
- **Fidelity Bug**: The zone `"P"` command places an object inside a specified container. In C, the item is placed inside the *last loaded object* by the spawner.
  In Go, the spawner resolves the container by querying:
  ```go
  container := s.findObjectInstance(cmd.Arg3)
  ```
  `findObjectInstance` searches the global spawned tracker and simply returns the **first ever spawned instance** of that container VNum:
  ```go
  func (s *Spawner) findObjectInstance(objVNum int) *ObjectInstance {
      if instances, ok := s.objInstances[objVNum]; ok && len(instances) > 0 {
          return instances[0]
      }
      return nil
  }
  ```
- **Impact**: If a zone spawns 10 separate chest instances of VNum 500 across different rooms, the spawner will **put all 10 spawned treasures inside the first chest spawned in the MUD**, leaving the other 9 chests completely empty.

### 3. Dysfunctional Reset Timers (Hyper-Aggressive Zone Resets)
- **Source Context**: `pkg/game/spawner.go#L693-L709` (`resetEmptyZones`), `pkg/game/world_zone.go#L157-L163` (`StartPeriodicResets`)
- **Fidelity Bug**: In legacy C, `zone_point_update` increments each zone's `age` (in minutes). A zone is only reset when `age >= lifespan` (and dependent on its `reset_mode`).
  In Go, `StartPeriodicResets` fires `resetEmptyZones()` every **60 seconds**, which loops through all zones and triggers resets:
  ```go
  for _, zone := range zones {
      if s.zoneHasPlayers(zone.Number) {
          continue
      }
      if err := s.ExecuteZoneReset(zone); err != nil { ... }
  }
  ```
  This loop **never checks `zone.Lifespan` or `zone.ResetMode`**! 
- **Impact**: Every zone that has no active players inside will reset **every 60 seconds**! If a player exits a zone for one minute and returns, all mobs and items are instantly respawned, allowing infinite grinding. Additionally, zones configured with `ResetMode = 0` (never reset) will reset anyway. Conversely, zones in `ResetMode = 2` (always reset even with players) will **never reset** if players are present.

### 4. Bypassed Zone Dispatcher (Inactive Concurrent Engine)
- **Source Context**: `pkg/game/zone_dispatcher.go`
- **Fidelity Bug**: The Go codebase features a sophisticated, concurrent `ZoneDispatcher` designed to run isolated goroutines per zone to handle resets and tick times safely. However, **`StartZoneDispatcher()` is never called anywhere during server startup** (`cmd/server/main.go` only starts the serial `StartPeriodicResets`).
- **Impact**: The concurrent zone reset system is dead, inactive code. The MUD runs entirely on the single-threaded serial spawner.

### 5. Standard Informative Text Commands Missing
- **Fidelity Bug**: Legacy C loaded multiple informative text files on boot (`credits`, `news`, `info`, `wizlist`, `immlist`, `policies`, `handbook`, `background`, `future`) and served them via respective commands. In Go, these commands and file loaders are **completely missing** from the session and game layers.
- **Name Blacklist Ignored**: C's name validation loaded a custom blocklist from `lib/text/names`. The Go port ignores this file and checks a hardcoded 12-item list in `validation.go`.

---

## 3. Go Improvements Over C

### 1. Robust SQLite Database Integration
- **Fidelity Improvement**: Legacy C managed character profiles by writing to a fragile binary pfile structure (`char_file_u`) on disk. This was notorious for structure alignment mismatches, file locks, and corruption. Go replaces this with standard SQL queries to a SQLite database backend, vastly improving data integrity and query capability.

### 2. High-Fidelity Diku Parser
- **Fidelity Improvement**: Go parses the legacy MUD raw Diku assets (`.wld`, `.mob`, `.obj`, `.zon`) using standard, robust string scanner routines in `pkg/parser/`. This avoids legacy C's unsafe character indexing and scanner overflows (`fscanf`).

---

## 4. Summary of Recommended Fixes

1. **Fix Door State Map Writing in Spawner**:
   In `pkg/game/spawner.go` case `"D"`, write the modified copy back into the room exits map:
   ```go
   ext.DoorState = cmd.Arg3
   room.Exits[roomDirNames[cmd.Arg2]] = ext // Restore to map!
   ```
2. **Track the Last Spawned Object for P-Command Container Placement**:
   Refactor `ExecuteZoneReset` to track a `lastObj *ObjectInstance` pointer. When case `"O"` or `"G"` successfully spawns an object, set `lastObj = obj`. When case `"P"` executes, place the item inside `lastObj` instead of querying the global first-spawn VNum instance.
3. **Restore Zone Age and Reset Mode Logic**:
   - Add an `Age` integer field to `parser.Zone` to keep track of elapsed minutes.
   - Refactor `resetEmptyZones()` to increment `zone.Age` each minute, and only execute `ExecuteZoneReset` if `zone.Age >= zone.Lifespan` and `zone.ResetMode` parameters are satisfied.
4. **Wire in missing text commands**:
   Add file readers and command handlers to the session registry to serve `news`, `credits`, `info`, `wizlist`, `immlist`, and `background` files from `lib/text/` as originally designed.
