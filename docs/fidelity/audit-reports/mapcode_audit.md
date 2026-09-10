# Port Fidelity Audit: Module 34 (`mapcode.c`)

This audit examines the port fidelity between the legacy C source file `src/mapcode.c` and its Go counterparts in `pkg/game/` and `pkg/session/`.

---

## 1. Architectural Mapping & Discrepancies

### C Source File
- **File**: `src/mapcode.c` (227 lines)
- **Functions**: 
  - `map`: Recursive coordinate mapping function. Explores room exits up to 3 cells away, checks for diagonal up/down indicators, and detects overlapping (non-Euclidean) room links.
  - `do_map`: Command endpoint. Decodes grid display positions, maps sector indexes to ANSI-colored sector symbols, and prints the mini-map and key legend.

### Go Port Files
- **Production Session Wrapper**:
  - [map_cmds.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/session/map_cmds.go): Production implementation containing the full recursive `mapRecurse` logic, exact C-matching ANSI color symbols mapping, and `CmdMap` command registration.
- **Obsolete Duplicate Game file**:
  - [mapcode.go](file:///Users/zach/.openclaw/workspace/darkpawns_repo/pkg/game/mapcode.go): A duplicate plain-text (non-colored) mapping package marked as unused (`lint:file-ignore U1000`).

---

## 2. Critical Logic Gaps & Severe Bugs

### 1. C Stack Out-of-Bounds Memory Read Bug & Go Resolution
- **Source Context**: `src/mapcode.c#L135` (C post-loop check), `pkg/session/map_cmds.go#L129` (Go index clamping).
- **Fidelity Bug**: In legacy C, the overlap link indicator check is placed outside the cardinal exploration loop:
  ```c
  for(dir=0;(dir<4);dir++) { ... }
  if((overlap)&&(display[x+offx[dir]][y+offy[dir]] != 0))
    display[x+offx[dir]][y+offy[dir]] = -6;
  ```
  Because the loop ends when `dir == 4`, C executes the check with index `dir = 4`. However, `offx` and `offy` are fixed arrays of size 4 (valid indices `0` to `3`). C performs an **out-of-bounds stack memory read**, reading whatever stack garbage happens to sit next to the offsets in memory.
- **Go Resolution**: In Go, calling `offX[4]` would instantly trigger an out-of-bounds panic. The Go port correctly caught this hazard and clamped the check to the final valid direction index `3` (West):
  ```go
  if overlap != 0 && (*display)[x+offX[3]][y+offY[3]] != 0 {
      (*display)[x+offX[3]][y+offY[3]] = -6
  }
  ```
  This is an excellent safety correction that preserves execution stability in Go.

### 2. Plain-Text vs ANSI-Color Code Duplication (Code Clutter)
- **Source Context**: `pkg/game/mapcode.go` (Unused file), `pkg/session/map_cmds.go` (Active file).
- **Discrepancy**: The Go codebase contains two parallel ports of `mapcode.c`. 
  - `pkg/game/mapcode.go` renders plain-text ASCII (e.g. `sectorIcons = []string{"0", "#", ":"}`) and is not wired to any command registry.
  - `pkg/session/map_cmds.go` contains the actual production command with full ANSI colors (e.g. `sectIcons = []string{"&g0&n", "&m#&n"}`) and is registered under `init()`.
- **Impact**: While the production command is highly faithful and functions perfectly, the duplicate unused file in `pkg/game/mapcode.go` creates code clutter and should be cleaned up.

---

## 3. Go Improvements Over C

### 1. Concurrency Goroutine-Safety
- **Go Enhancement**: Legacy C utilized a shared global buffer array `int display[78][27]` to draw maps, which would trigger severe data races under concurrent execution in Go. The Go port safely converts this array to a locally-allocated slice (`display := make([][]int)`) passed by pointer, allowing concurrent players to use the `map` command safely.

### 2. O(n) String Assembly
- **Go Enhancement**: C’s map output relied on sequential `strcat` calls which have O(n²) time complexity for line drawing. Go leverages `strings.Builder` to perform O(n) linear buffer allocation, speeding up map prints.

### 3. String-Keyed Exits
- **Go Enhancement**: Unlike C's rigid numeric direction indices (0-5), Go maps directions cleanly to descriptive string-keyed exits (`room.Exits["down"]`, `room.Exits["up"]`), greatly improving readability.

---

## 4. Concurrency & Thread Safety

- **Local Slice Allocations**:
  - Because `display` is allocated locally inside `CmdMap` on each command execution and passed down via pointers, it isolates concurrent goroutines from overlapping memory races.
- **Lock-Free World Snapshots**:
  - The map generator reads world room records by pulling a thread-safe snapshot (`s.manager.world.GetSnapshotManager().Snapshot()`), avoiding data races with room modifications while maintaining low execution latency.

---

## 5. Summary of Recommended Fixes

1. **Delete Obsolete `pkg/game/mapcode.go`**:
   Remove the redundant, plain-text copy of the mapcode file inside `pkg/game/mapcode.go` to eliminate developer confusion and maintain codebase cleanliness.
