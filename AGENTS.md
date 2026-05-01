# Dark Pawns — AI Agent Instructions

## Build & Verify

```bash
go build ./...          # Full build — must pass before committing
go vet ./...            # Static analysis — must pass before committing
go test ./...           # All tests — must pass before committing
go test ./pkg/game/...  # Game package tests specifically
golangci-lint run ./... # Full lint (uses .golangci.yml)
```

**NEVER commit without running all four.** Subagents that self-report passing builds have lied before.

## Project Overview

Dark Pawns is a Go MUD server, ported from C (DikuMUD/Merc 2.2 lineage). ~73K lines of Go, ~66 C files remaining for reference only. The Go port is COMPLETE — do not re-port C files.

### Architecture

- `cmd/server/` — MUD server entrypoint
- `pkg/game/` — Core game logic (combat, skills, objects, rooms, mobs, world state)
- `pkg/session/` — Player session handling, commands, wizard commands
- `pkg/parser/` — Zone/obj/mob/world file parsing (Diku format)
- `pkg/combat/` — Combat formulas and damage calculation
- `pkg/spells/` — Spell system (saving throws, damage, affect spells)
- `pkg/telnet/` — Telnet protocol handling
- `pkg/db/` — SQLite persistence, narrative memory for AI agents
- `pkg/agent/` — AI agent hooks (BRENDA agent integration)
- `pkg/session/memory_hooks.go` — Go→Python memory system bridge

### Key Conventions

- **ObjectLocation system**: Objects track location via tagged union `ObjectLocation` in `location.go`. Use `LocInventoryPlayer()`, `LocEquipped()`, `LocNowhere()` constructors. Never manipulate `Location` fields directly.
- **Runtime state**: `ObjectRuntimeState` and `MobRuntimeState` replace `CustomData map[string]interface{}`. Use typed fields. `CustomData` is the escape hatch for truly dynamic keys.
- **Equipment**: `Equipment` struct with typed `EquipmentSlot` constants. Use `Equip()`/`Unequip()`/`UnequipItem()` — never manipulate `Slots` map directly.
- **Error handling**: Player-facing operations (item transfer, equipment, shop) MUST check error returns. Use `slog.Error()` on rollback failures. Never use `#nosec G104` to suppress error checks in player-facing code.
- **`fmt.Fprintf` to `io.Writer`**: These are user-facing output (MUD text). Do NOT convert to `slog` — they stay as formatted writes.

### What NOT to Do

- Do not re-port C files in `src/`
- Do not convert `fmt.Fprintf(player.Writer, ...)` to slog
- Do not remove `CustomData` field (it's the escape hatch)
- Do not change save file format
- Do not use `#nosec G104` on new code — handle the error

### #nosec Annotations

The codebase has ~600 `#nosec` annotations. Most are:
- **G404 (weak random)** — ~394 intentional `math/rand` for dice rolls, combat, MUD mechanics. Leave these alone.
- **G104 (errcheck)** — ~135 remaining. These are the cleanup targets. Handle the error or log it.
- **G115 (integer overflow)** — ~26. Add explicit casts or range checks.
- **G304/G306 (file path/perms)** — ~24. Use `filepath.Clean()` and explicit permissions.
- Other G-codes — small counts, mostly intentional.

## Commit Messages

Use conventional commits: `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`

## Testing

Tests live alongside source files as `*_test.go`. Run `go test ./...` after any change.
The game package has object movement tests that validate the ObjectLocation system — keep those green.
