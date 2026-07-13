# Brief: Test Coverage Expansion

**Date:** 2026-05-27
**Requested by:** The Architect
**Execute via:** Claude Code or Gemini (Antigravity)

## Goal

Expand test coverage for the Dark Pawns codebase. The linting baseline (commit `fb86252`) caught static issues. Tests catch runtime regressions. You need both.

## Current State

| Package | Coverage | Test Files | Priority |
|---------|----------|------------|----------|
| `pkg/game` | 6.8% | 14 | **CRITICAL** — core game logic, 60+ source files |
| `pkg/spells` | 8.4% | 3 | **CRITICAL** — spell system, affects, damage |
| `pkg/dreaming` | 11.8% | 1 | HIGH — memory consolidation, agent integration |
| `pkg/session` | 14.7% | 9 | **CRITICAL** — player session, commands, display |
| `pkg/moderation` | 15.9% | 1 | HIGH — chat moderation, profanity |
| `pkg/scripting` | 19.5% | 2 | HIGH — Lua engine, behavioral scripts |
| `pkg/validation` | 20.8% | 1 | MEDIUM — input validation |
| `pkg/privacy` | 33.7% | 1 | MEDIUM — privacy filter |
| `pkg/agentcli` | 34.3% | 4 | MEDIUM — AI agent CLI |
| `pkg/admin` | 47.5% | 4 | MEDIUM — admin API |
| `pkg/engine` | 47.6% | 2 | MEDIUM — affect system, game loop |
| `pkg/game/systems` | 49.4% | 3 | MEDIUM — shop manager, door manager |
| `pkg/auth` | 56.5% | 1 | LOW — JWT auth |
| `pkg/grapevine` | 59.9% | 1 | LOW — external integration |
| `pkg/combat` | 64.5% | 4 | LOW — combat engine |
| `pkg/events` | 67.2% | 2 | LOW — event bus |
| `pkg/parser` | 75.4% | 5 | LOW — world file parser |
| `pkg/agent` | 0.0% | 0 | NEW — memory hooks |
| `pkg/audit` | 0.0% | 0 | NEW — audit logging |
| `pkg/command` | 0.0% | 0 | NEW — command registry |
| `pkg/common` | 0.0% | 0 | NEW — shared interfaces |
| `pkg/db` | 0.0% | 0 | NEW — decision log, narrative memory |
| `pkg/optimization` | 0.0% | 0 | NEW — caching, object pools |
| `pkg/secrets` | 0.0% | 0 | NEW — secrets management |
| `pkg/storage` | 0.0% | 0 | NEW — SQLite storage |
| `pkg/telnet` | 0.0% | 0 | NEW — telnet listener, GMCP |

## Scope

**Focus on the CRITICAL and HIGH packages first.** These are the ones where a regression would break the game for players.

### Tier 1: Core Game Logic (biggest bang)

**`pkg/game` — 6.8% coverage, 60+ source files**

This is the heart of Dark Pawns. 14 test files exist but they only cover a fraction. Priority areas to add tests for:

- `save.go` — save/load player data (serialization round-trip)
- `death.go` — death handling, corpse creation, XP penalty
- `movement.go` — room transitions, sector checks, door states
- `object.go` — object manipulation, wear/remove, container logic
- `skills.go` — skill checks, cooldowns, learning
- `clans.go` — clan membership, clan bank
- `mail.go` — in-game mail system
- `weather.go` — weather/time system

**How to test these:** These functions are deeply coupled to game state. You'll need test helpers that create minimal `World`, `Player`, `Room`, `Object`, and `Mob` instances. Check existing test files in `pkg/game/` for patterns — `death_test.go`, `movement_test.go`, and `object_movement_test.go` show how the test infrastructure works.

**`pkg/spells` — 8.4% coverage**

- `spells.go` — spell casting, effect application, saving throws
- `affect_spells.go` — affect management (add/remove/expire)
- `damage_spells.go` — direct damage spells

**How to test:** Spells operate on `Character` and `Room` objects. Create test fixtures with minimal game state. Mock the random number generator if needed (check if there's a `dice` package or similar).

### Tier 2: Session & Commands

**`pkg/session` — 14.7% coverage**

This is the player-facing layer. 9 test files exist but coverage is thin. Priority:

- `cmd_info.go` — the `info` command (most-used by players)
- `cmd_look.go` — the `look` command (second most-used)
- `commands.go` — command dispatch
- `session_login.go` — login flow
- `session_player.go` — player state management

**How to test:** Session tests need a `Session` object with a mock `net.Conn`. Check `cmd_look_test.go` for the pattern — it creates a `mockWorld` and wires up sessions.

### Tier 3: New Packages (0% coverage)

These need tests from scratch. Lower priority but important for completeness:

- `pkg/db` — `decision_log.go` and `narrative_memory.go` (data layer)
- `pkg/command` — command registry (lookup, registration)
- `pkg/telnet` — telnet protocol handling (GMCP, negotiation)

**How to test:** These are more isolated. `pkg/db` needs a SQLite test database. `pkg/command` is pure logic. `pkg/telnet` needs mock connections.

## Test Style Guidelines

Follow the existing patterns in the codebase:

1. **Table-driven tests** — use `[]struct{ name, input, want }` pattern
2. **Subtests** — `t.Run(name, func(t *testing.T) { ... })` for each case
3. **Test helpers** — extract common setup into helper functions (e.g., `createTestPlayer()`, `createTestRoom()`)
4. **No mocking frameworks** — use manual mocks / interfaces
5. **Test file naming** — `foo_test.go` in the same package (white-box testing)
6. **Build tags** — use `//go:build integration` for tests that need database/network

## Execution Strategy

**If using Claude Code:** Break into 3 briefs by tier. Each brief covers one tier. Run them sequentially.

**If using Gemini/Antigravity:** Can potentially run tiers in parallel since they touch different packages. But verify no cross-package dependencies break.

### Brief 1 (Tier 1): Core Game Logic
Focus: `pkg/game` and `pkg/spells`
Target: Get `pkg/game` from 6.8% → 30%+, `pkg/spells` from 8.4% → 25%+

### Brief 2 (Tier 2): Session & Commands
Focus: `pkg/session`
Target: Get `pkg/session` from 14.7% → 30%+

### Brief 3 (Tier 3): New Packages
Focus: `pkg/db`, `pkg/command`, `pkg/telnet`, `pkg/optimization`
Target: Get each from 0% → 15%+ (basic happy-path coverage)

## Pre-Flight

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
go build ./... && go vet ./... && go test ./... && golangci-lint run ./...
```

All four MUST pass before starting.

## Post-Flight

After each brief:

```bash
go test -cover ./pkg/<target>/...
go test -cover ./...
golangci-lint run ./...
```

All three must pass. Report the coverage delta (before → after) for the targeted packages.

## Commit Format

```
test: add coverage for <package> (<before>% → <after>%)
```

One commit per brief. Reference the tier in the commit message.

## What This Does NOT Do

- Does not change game behavior
- Does not refactor production code
- Does not touch infrastructure
- Does not add integration/E2E tests (those are a separate effort)
- Does not add test coverage for `cmd/`, `admin-ui/`, `examples/`, `benchmarks/`

This is purely about expanding the safety net for the packages that matter most.
