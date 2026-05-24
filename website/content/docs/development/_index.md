---
title: "Development"
description: "Contribute to and extend the Dark Pawns project"
date: 2026-04-22
draft: false
section: "docs"
---

# Development Guide

This page covers the project's core philosophy, testing requirements, and the concurrency model governing all shared state. Read this before touching `pkg/game/`, `pkg/session/`, or `pkg/combat/`.

---

## The Prime Directive

**Behavioral fidelity to the original 1997–2010 Dark Pawns C codebase takes precedence over idiomatic Go.**

This means:
- Combat formulas, THAC0 tables, damage multipliers, and experience penalties must match `fight.c` exactly.
- Command output strings (room descriptions, combat messages) must match original output character-for-character where possible.
- Pronoun substitution (`$n`, `$N`, `$e`, `$E`, `$s`, `$S`, `$m`, `$M`) follows ROM `act()` conventions.
- Area files (`.wld`, `.mob`, `.obj`, `.zon`, `.shp`) are loaded without modification.

When in doubt, the original C behavior is the spec. The Go implementation is a port, not a redesign.

---

## Testing Verification Checklist

Before submitting any change, verify the following:

```bash
# 1. Compile — zero warnings, zero errors
go build ./...

# 2. Vet — catches common correctness bugs
go vet ./...

# 3. Tests — must pass with no race conditions
go test -race ./...

# 4. Lint — golangci-lint with project config
golangci-lint run ./...
```

All four must pass clean. The test suite includes:
- `pkg/session/websocket_e2e_test.go` — full login → command → state-update end-to-end
- `pkg/session/session_login_test.go` — auth flow, character creation state machine
- `pkg/combat/` — damage formula correctness, THAC0 verification
- `pkg/game/` — world parser, mob instance, zone resets
- `pkg/scripting/` — Lua sandbox, API surface

---

## Lock Ordering Hierarchy

This project uses a strict top-down lock acquisition order to prevent deadlocks. **394 lock acquisitions across 57 functions** depend on this ordering being correct.

Audited `2026-05-07` by BRENDA69. No violations found at time of audit.

### Acquire locks from top (1) to bottom (14) only:

| # | Lock | Protects |
|---|---|---|
| 1 | `World.mu` | Top-level game state: rooms, mobs, objects |
| 2 | `World.gossipMu` | Gossip channel history |
| 3 | `World.weatherMu` | Weather state |
| 4 | `World.mailWriteMu` | Mail persistence |
| 5 | `Clan.mu` | Clan membership, ranks |
| 6 | `Player.mu` | Player stats, gold, exp, position |
| 7 | `Equipment.mu` | Equipped item slots |
| 8 | `Inventory.mu` | Carried item list |
| 9 | `MobInstance.mu` | Mob state, HP, position |
| 10 | `Spawner.mu` | Zone reset scheduling |
| 11 | `BoardState.mu` | Bulletin board messages |
| 12 | `Shop.mu` | Shop inventory, pricing |
| 13 | `ZoneDispatcher.mu` | Zone command routing |
| 14 | `logWriterMu` | Log file writes (functionally independent) |

### Locks outside the pkg/game hierarchy (always outermost-first):

| Lock | Package | Protects |
|---|---|---|
| `Manager.mu` | `pkg/session` | Active sessions map |
| `CombatEngine.mu` | `pkg/combat` | Combat pairs map (2s tick) |
| `scripting.Engine.mu` | `pkg/scripting` | Lua VM (serialized, one script at a time) |

### Rules

- **Never** hold a lower-numbered lock while acquiring a higher-numbered one.
- **Same-level locks** (e.g., two `Player.mu` on different players): always acquire in consistent order by `Name` or `ID` to prevent ABBA deadlocks.
- **Never** upgrade `RLock → Lock` without releasing first.
- `World.mu` is always the outermost lock in `pkg/game`. Never call World methods that re-acquire `World.mu` while holding `Player.mu`, `MobInstance.mu`, or `Clan.mu`.
- Use `defer Unlock()` for simple critical sections. Use explicit `Unlock()` for multi-lock or conditional-unlock patterns.

### Verified safe nested patterns

```
World.mu → MobInstance.mu          (save.go — deserialization)
Player.mu → Equipment.mu           (death.go — death cleanup)
Clan.mu → Player.mu                (item_transfer.go — gold transfer)
World.mu.RLock → Player/Mob.mu     (party.go — group handling)
```

---

## Package Overview

| Package | Role |
|---|---|
| `cmd/server` | Entry point; wires all dependencies in init order |
| `pkg/session` | WebSocket lifecycle, login, command dispatch, agent variable push |
| `pkg/game` | World state, rooms, mobs, objects, players, zone resets |
| `pkg/combat` | 2-second combat tick, damage formulas, THAC0 |
| `pkg/scripting` | Sandboxed Lua VM (gopher-lua), ~60 registered API functions |
| `pkg/parser` | ROM area file parser (`.wld`, `.mob`, `.obj`, `.zon`) |
| `pkg/command` | Command registry, skill handlers |
| `pkg/db` | Postgres persistence (characters, agent keys, decision logs) |
| `pkg/events` | Timer-based event queue + in-process pub/sub bus |
| `pkg/agent` | Agent variable subscription, dirty-tracking, push flush |
| `pkg/auth` | JWT generation/validation, IP rate limiter |
| `pkg/telnet` | Raw TCP listener with IAC negotiation |
| `pkg/audit` | Security and admin event logging |
| `pkg/metrics` | Prometheus exposition endpoint |
| `pkg/moderation` | Mute, ban, word filter, spam detection |

See **[Architecture Reference](/docs/development/architecture/)** for the full data-flow diagram and concurrency model.

---

## Contributing

1. Read the Prime Directive above.
2. Run the full verification checklist before every commit.
3. If you touch any lock acquisition, check the hierarchy table and add a comment in `locks.go` for any new nested pattern.
4. Combat formula changes require a side-by-side comparison against the original C `fight.c` logic, documented in your PR.
5. Lua script API changes must preserve backward compatibility with existing `world/lib/scripts/` files.
