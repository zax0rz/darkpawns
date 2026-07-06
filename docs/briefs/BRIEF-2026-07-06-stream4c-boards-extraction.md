# Brief: Stream 4c — Board Leaf Extraction from pkg/game (F14)

## Context

DP-953 from the Fable Audit 2026-07-05 (originally C-03 from April review).
`pkg/game/` is a flat god package (130 non-test files, 43K lines). The fix is
incremental leaf extraction — start with the lowest-fan-in subsystem. Boards
wins at 646 lines, 3 external callers, shallowest dependencies.

**Linear:** DP-953 (F14)
**Branch:** `fix/stream4c-boards-extraction`
**Agent:** Kimi

## Problem

All board code lives in `pkg/game/boards.go` (617 lines) + `boards_test.go`
(29 lines). The `BoardSystem` struct is well-encapsulated (own mutex, own file
I/O, own lifecycle) but can't be tested or reasoned about independently
because it's inside the god package.

## Current State

### Board Code (`pkg/game/boards.go`)
- `BoardSystem` struct with `sync.RWMutex`, message arrays, file I/O
- Methods: `InitBoards`, `ShowBoard`, `DisplayMsg`, `RemoveMsg`,
  `WriteMessage`, `AppendBoardLine`, `FinalizeBoardWrite`
- `World` methods: `FindBoard`, `GetOrInitBoards`
- Spec proc: `genBoard` registered in `init()`
- 12 board definitions with VNum-to-type mappings

### Fan-In (3 files within pkg/game/)
| File | Reference |
|---|---|
| `world.go:92-93` | `Boards *BoardSystem` field |
| `spec_assign.go:331-350` | 12 `gen_board` string registrations |
| `player.go:41-43` | `WriteMagic int` field (board editor state) |

### Fan-In (2 files outside pkg/game/)
| File | Reference |
|---|---|
| `cmd/server/main.go:214-215` | `GetOrInitBoards(...)` + `SetWorld(...)` |
| `pkg/session/session_login.go:382,387` | `FinalizeBoardWrite(...)` + `AppendBoardLine(...)` |

### Dependencies from boards.go
- `*Player` — parameter (uses `ch.Name`, `ch.Level`, `ch.WriteMagic`, methods)
- `*World` — receiver for `FindBoard`, `GetOrInitBoards`; field for `BoardSystem.world`
- `actToRoom()` — free function in game package
- `BasicMudLogf()` — free function in game package
- `isAbbrev()` — free function in game package
- `RegisterSpec()` — free function in game package

## Extraction Approach

Use the **subdirectory pattern** — create `pkg/game/boards/` with
`package boards`. This is NOT the `pkg/game/systems/` pattern (which imports
`pkg/game` as a consumer). Instead, `pkg/game/` will import `pkg/game/boards`
as a leaf subpackage.

### Step 1: Create `pkg/game/boards/boards.go`

Move `boards.go` content here with `package boards`. The `BoardSystem` struct
stays as-is but its methods no longer have access to `*World` or `*Player`
directly.

### Step 2: Break type dependencies

Replace direct struct access with interfaces or injection:

| Current | Replacement |
|---|---|
| `ch.Name` (direct field) | Pass as parameter or add `GetName()` method |
| `ch.Level` (direct field) | `ch.GetLevel()` (already exists on Player) |
| `ch.WriteMagic` (direct field) | Inject via setter or pass as parameter |
| `ch.SendMessage(...)` | Accept `io.Writer` or message callback |
| `w.GetItemsInRoom(...)` | Inject via `FindBoard` callback |
| `actToRoom(...)` | Inject as `ActRoom func(...)` on BoardSystem |
| `BasicMudLogf(...)` | Inject as `LogFunc func(...)` on BoardSystem |
| `RegisterSpec(...)` | Remove from `init()` — register from `spec_assign.go` instead |

The key insight: `BoardSystem` already has a `SetWorld(w *World)` method.
Extend this to inject all dependencies:
```go
type BoardSystemDeps struct {
    ActRoom      func(roomVNum int, msg string, exclude string)
    Log          func(format string, args ...interface{})
    IsAbbrev     func(arg, name string) bool
    Atoi         func(s string) int
}
```

### Step 3: Update callers

- `world.go` — change `Boards *BoardSystem` to `Boards *boards.BoardSystem`,
  update initialization
- `spec_assign.go` — change `gen_board` registration to reference
  `boards.GenBoard`
- `player.go` — `WriteMagic` stays on Player (it's board editor state, owned
  by the player). Pass it as a parameter to `AppendBoardLine`/`FinalizeBoardWrite`
  instead of accessing it directly
- `cmd/server/main.go` — update initialization calls
- `pkg/session/session_login.go` — update board editor calls

### Step 4: Move `gen_board` spec proc

The `gen_board` spec proc is registered in `init()` inside `boards.go`. Move
the registration to `spec_assign.go` (where all other specs are registered)
calling into `boards.GenBoard()`.

## Files

- **`pkg/game/boards.go` → `pkg/game/boards/boards.go`** (MOVE + refactor)
- **`pkg/game/boards_test.go` → `pkg/game/boards/boards_test.go`** (MOVE)
- **`pkg/game/world.go`** — update Boards field type (MODIFY)
- **`pkg/game/spec_assign.go`** — update gen_board registration (MODIFY)
- **`pkg/game/player.go`** — WriteMagic field stays, callers pass as param (MODIFY)
- **`cmd/server/main.go`** — update init calls (MODIFY)
- **`pkg/session/session_login.go`** — update board editor calls (MODIFY)

## Important Notes

- Do NOT use the `systems/` pattern (downstream import). `pkg/game/boards/` is
  an upstream subpackage that `pkg/game/` imports.
- `WriteMagic` on Player is board-specific state. It could be moved to a map
  on BoardSystem keyed by player name, but that's a bigger change. For now,
  keep it on Player and pass it as a parameter.
- The 12 board definitions (VNum-to-type mappings) can live in the new
  `boards` package or stay in `spec_assign.go`. Keeping them in the boards
  package is cleaner since they're board data, not spec dispatch data.
- This validates the extraction pattern. If it works cleanly, the same approach
  applies to socials, houses, clans in future sprints.

## Regression Test

Existing `boards_test.go` moves with the code and must still pass. Add:
```go
func TestBoardSystemExtractionImports(t *testing.T) {
    // Verify pkg/game can import pkg/game/boards without circular imports
    // This is a compile-time test — if it compiles, it passes.
}
```

## Build Gate

```bash
go build ./... && go vet ./... && go test ./... && gofumpt -l . | grep -v vendor
```

Watch for circular import errors — `pkg/game/boards/` must NOT import
`pkg/game/`. If it needs game types, use interfaces.

## Commit

```
refactor: extract boards to pkg/game/boards/ — first leaf extraction from god package (DP-953)
```
