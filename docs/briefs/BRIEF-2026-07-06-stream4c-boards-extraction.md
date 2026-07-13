# BRIEF — Stream 4c: Boards Leaf Extraction

**Linear:** DP-953 (F14 — pkg/game flat god package, first leaf)
**Effort:** M (estimated 2-3 PRs)
**Agent:** Kimi
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — F14 / C-03

## Goal

Extract the bulletin board system from `pkg/game/boards.go` (618 lines) into `pkg/boards/`, breaking the `pkg/game` → `pkg/boards` circular dependency via interfaces. This is the first leaf extraction in the god-package cleanup — keep it small and prove the pattern works before doing socials/houses/clans.

## Scope

### What moves to `pkg/boards/`

1. **`pkg/game/boards.go`** (618 lines) → `pkg/boards/boards.go`
   - Types: `BoardMsgInfo`, `BoardInfo`, `BoardSystem`
   - Constants: `NumBoards`, `MaxBoardMessages`, `MaxMessageLength`, `BoardMagic`
   - Methods: `InitBoards`, `WriteMessage`, `ShowBoard`, `DisplayMsg`, `RemoveMsg`, `AppendBoardLine`, `FinalizeBoardWrite`
   - `safeInt32()` utility (also used by `mail.go` — duplicate a copy into `pkg/boards/` or `pkg/common/`)

2. **`pkg/game/boards_test.go`** (30 lines) → `pkg/boards/boards_test.go`

### What does NOT move (stays in `pkg/game/`)

- `note_write.go` (110 lines) — note-writing state machine for ITEM_NOTE objects. This is a separate subsystem that happens to share the PLR_WRITING intercept in session_login.go. Extracting it alongside boards adds complexity without benefit. Leave it for a separate pass.
- `World.FindBoard()` — stays as a thin method on World that delegates to the new package.
- `World.GetOrInitBoards()` — stays on World, calls into `pkg/boards.InitBoards()`.
- `ObjSpecAssign` map entries for board VNMs — stays in `pkg/game/spec_assign.go`.
- `genBoard` spec proc — stays in `pkg/game/` because it must conform to `SpecFunc` signature `(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool`. It delegates to `BoardSystem` methods.

### What does NOT move and needs interface injection

The new `pkg/boards/` package must NOT import `pkg/game/` (that would be circular). Break dependencies with interfaces:

#### 1. `BoardWorld` interface (replaces `*World` on BoardSystem)

`BoardSystem` currently holds `*World` for:
- `ch.GetRoomVNum()` — FindBoard needs room items
- `w.GetItemsInRoom(roomVNum)` — scan room for board objects
- `actToRoom(w, roomVNum, format, excludeName)` — room echo on RemoveMsg

Define in `pkg/boards/`:
```go
type BoardWorld interface {
    RoomEcho(roomVNum int, message string, excludeName string)
}
```

`FindBoard` stays on `*World` — it scans room items (using `GetItemsInRoom` which returns `[]*ObjectInstance`) and resolves the board index. `BoardSystem` methods take `boardType int` directly. The `BoardWorld` interface is minimal: just room echo for RemoveMsg.

#### 2. `BoardPlayer` interface (replaces `*Player` on all board methods)

Board methods use from `*Player`:
- `ch.Level` — write/remove level checks
- `ch.Name` / `ch.GetName()` — author attribution
- `ch.SendMessage(msg)` — output
- `ch.GetRoomVNum()` — FindBoard
- `ch.WriteMagic` — editor token (set by genBoard, read by session_login.go)

Define in `pkg/boards/`:
```go
type BoardPlayer interface {
    Level() int
    GetName() string
    SendMessage(msg string)
    GetRoomVNum() int
}
```

The `*Player` struct in `pkg/game/` already satisfies this.

#### 3. Logging

Replace `BasicMudLogf(...)` calls with `slog.Error(...)` / `slog.Info(...)` directly — no need to import game's logging helper.

#### 4. Small helpers

- `isAbbrev(arg, name)` — used once in DisplayMsg. Either inline the logic (it's 3 lines) or copy.
- `atoi(s)` — used in DisplayMsg/RemoveMsg. Use `strconv.Atoi` directly.
- `sendToChar(ch, msg)` — used only in note_write.go (not moving). No boards.go usage.

## Files

### New files
| File | Purpose |
|---|---|
| `pkg/boards/boards.go` | BoardSystem, BoardInfo, BoardMsgInfo, constants, InitBoards |
| `pkg/boards/interfaces.go` | BoardWorld, BoardPlayer, BoardItem interfaces |
| `pkg/boards/boards_test.go` | Existing test, adapted for interfaces |

### Modified files
| File | Change |
|---|---|
| `pkg/game/boards.go` | DELETE (618 lines → 0) |
| `pkg/game/boards_test.go` | DELETE (30 lines → 0) |
| `pkg/game/world.go` | `Boards *BoardSystem` → `Boards *boards.BoardSystem`. Add adapter methods for `BoardWorld` if needed. |
| `pkg/game/boards.go` (new thin file) | Re-export: `genBoard` spec proc, `FindBoard`, `GetOrInitBoards` as thin wrappers delegating to `pkg/boards`. Or move genBoard to spec_assign.go. |
| `pkg/session/session_login.go` | Import `pkg/boards` directly for `FinalizeBoardWrite`, `AppendBoardLine`. Remove `s.manager.world.Boards` indirection — or keep it if World still holds the reference. |
| `cmd/server/main.go` | Import `pkg/boards`. `GetOrInitBoards()` may need a thin wrapper. |

### NOT modified
| File | Reason |
|---|---|
| `pkg/game/note_write.go` | Out of scope — separate subsystem |
| `pkg/game/mail.go` | `safeInt32` shared — copy to `pkg/boards` or `pkg/common` |
| `pkg/game/spec_assign.go` | Board VNum mappings stay here; gen_board stays here |
| `pkg/session/commands.go` | Spec dispatch is generic — no board-specific code |
| `pkg/game/player.go` | `WriteMagic int` field stays on Player |

## C Source Reference

- `src/boards.c` (552 lines) — original C implementation
- `src/boards.h` (77 lines) — struct definitions, constants
- `src/spec_assign.c` lines 520-556 — 12 board VNum assignments

## PR Strategy

### PR 1: Create `pkg/boards/` with interfaces + BoardSystem

1. Create `pkg/boards/interfaces.go` with `BoardWorld`, `BoardPlayer`, `BoardItem` interfaces
2. Create `pkg/boards/boards.go` — copy `boards.go`, replace `*World` with `BoardWorld`, `*Player` with `BoardPlayer`, `BasicMudLogf` with `slog`, `isAbbrev`/`atoi` with stdlib or inline
3. Create `pkg/boards/boards_test.go` — adapt existing test with mock implementations of interfaces
4. Do NOT delete `pkg/game/boards.go` yet — both exist in parallel
5. Verify: `go build ./... && go vet ./...` pass (pkg/game/boards.go still exists, pkg/boards/boards.go is new)

### PR 2: Wire `pkg/boards` into game, delete old `pkg/game/boards.go`

1. Add `BoardWorld` adapter methods on `*World` (if not automatically satisfied)
2. Change `World.Boards *BoardSystem` → `Boards *boards.BoardSystem`
3. Move `FindBoard` to use `pkg/boards` directly
4. Move `GetOrInitBoards` to delegate to `pkg/boards.InitBoards()`
5. Update `session_login.go` WriteMagic intercept to use `pkg/boards` types
6. Update `cmd/server/main.go` init
7. Delete `pkg/game/boards.go` and `pkg/game/boards_test.go`
8. Keep `genBoard` in game — it calls `w.Boards.ShowBoard(...)` etc. which now uses `pkg/boards.BoardSystem`
9. Move `safeInt32` copy to `pkg/boards` (mail.go keeps its own)
10. Verify: `go build ./... && go vet ./... && go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s && gofumpt -l . && golangci-lint run ./...`

## Build Gate

```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

All five must pass. No exceptions.

## Regression Tests

Existing: `TestDisplayMsgBoardArgumentDoesNotLeakReadLock` — must continue to pass.

Add new tests in PR 1:
- `TestBoardSystem_InitAndWrite` — init, write message, verify stored
- `TestBoardSystem_ShowBoard_Empty` — show with no messages
- `TestBoardSystem_RemoveMsg_LevelCheck` — low-level player cannot remove high-level message
- `TestBoardSystem_FullBoard_RejectsOverflow` — 61st message rejected
- `TestBoardSystem_DisplayMsg_InvalidNumber` — returns false for non-numeric / out-of-range
- All use mock `BoardPlayer` and mock `BoardWorld`

## Constraints

1. **No circular imports.** `pkg/boards` must NOT import `pkg/game`. Only interfaces.
2. **No behavioral changes.** This is pure refactoring. Board read/write/remove/look must work identically before and after.
3. **Keep `genBoard` in pkg/game/.** It must conform to the `SpecFunc` signature. It delegates to `BoardSystem` methods — no code duplication.
4. **Keep `note_write.go` in pkg/game/.** Out of scope for this extraction.
5. **`Player.WriteMagic` stays on Player.** The session_login.go intercept reads it directly. Don't move it.
6. **C fidelity.** Board persistence format (binary board files) must not change. The `safeInt32` / `int32Bytes` encoding is part of the save format contract.

## What Could Go Wrong

1. **Interface satisfaction** — `*World.GetItemsInRoom()` returns `[]*ObjectInstance`, not `[]BoardItem`. Go does not allow returning a concrete slice where an interface slice is expected. **Solution: keep `FindBoard` on `*World`** — it already iterates items and resolves the board index. `BoardSystem` methods take `boardType int` and never need to scan items. Remove `GetItemsInRoom` from `BoardWorld` — it's not needed.

2. **`actToRoom`** — boards.go calls `actToRoom(bs.world, roomVNum, fmt.Sprintf(...), ch.GetName())`. This needs to go through the `BoardWorld` interface. Add `RoomEcho(roomVNum int, message string, excludeName string)` to the interface. Implement on World by calling the existing `actToRoom`.

3. **Test double setup** — the existing test uses `NewPlayer()` from game package. After extraction, tests in `pkg/boards/` can't import `pkg/game`. Must create mock `BoardPlayer` in the test file. This is the whole point — boards become independently testable.
