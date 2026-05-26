# Audit Report: act.movement.c vs Go Movement Engines

**C file:** `src/act.movement.c` (951 lines)
**Go file(s):** `pkg/game/act_movement.go` (24,815 bytes), `pkg/game/movement.go` (6,310 bytes), `pkg/session/cmd_movement.go` (3,689 bytes), `pkg/session/movement_cmds.go` (12,557 bytes)
**Mapping type:** 1:N
**Functions audited:** 11 C commands / ~10 Go command entrypoints & helpers

---

## Logic Drift & Missing Side Effects

### [FINDING-001]: Entire Position and Regeneration System is Dead Code
- **Location:** `pkg/session/commands.go` (command registry) and `pkg/session/movement_cmds.go` (entire file).
- **C behavior:** In C, characters have dynamic positions (`standing`, `sitting`, `resting`, `sleeping`). Commands like `stand`, `sit`, `rest`, `sleep`, and `wake` allow players to transition between states. Sleeping and resting multiply the character's HP, Mana, and Move regeneration rates, which is a fundamental MUD core mechanic.
- **Go behavior:** The Go codebase has fully ported the position logic into `cmdStand()`, `cmdSit()`, `cmdRest()`, `cmdSleep()`, and `cmdWake()`. However, these are **completely dead and un-wired** (marked with `//lint:file-ignore U1000 ... not yet wired to command registry`). None of them are registered in the active player command registry in `pkg/session/commands.go`.
- **Discrepancy:** Players are permanently locked into the default standing position. Because they cannot sit, rest, or sleep, they are unable to accelerate their vital regeneration loops, causing the gameplay flow to slow down dramatically.
- **Severity:** HIGH
- **Type:** STUB

### [FINDING-002]: Enter and Leave Commands are Completely Omitted
- **Location:** `pkg/session/commands.go` (command registry) and `pkg/game/act_movement.go:682` (`doEnter()` and `doLeave()`).
- **C behavior:** In C `act.movement.c:642` and `676` (`do_enter` / `do_leave`), players can enter custom objects, portals, carriages, or keyword exits: `enter portal`.
- **Go behavior:** Ported in the game layer but **completely un-registered** in the player command registry.
- **Discrepancy:** Players have no way to enter portals or interact with custom movement objects, breaking progress in zones that rely on these exits.
- **Severity:** HIGH
- **Type:** STUB

### [FINDING-003]: Mount Commands (`ride`, `dismount`, `yank`) are Completely Omitted
- **Location:** `pkg/session/commands.go` (command registry) and `pkg/game/act_movement.go:698`.
- **C behavior:** C supports mounting mechanics (`do_dismount` and associated mount actions).
- **Go behavior:** Go has **no active session command registrations** for `ride`, `dismount`, or `yank`, leaving them entirely disabled.
- **Discrepancy:** Despite the help files documenting `"mounts"` syntax, players have no way to interact with the mounting system.
- **Severity:** HIGH
- **Type:** STUB

---

## Type & Boundary Vulnerabilities

### [FINDING-004]: Gender Pronoun Inversion in Position Messages
- **Location:** `pkg/session/movement_cmds.go:28` inside `cmdStand()`.
- **C behavior:** Gender pronoun macros (like `HSHR`) are used to display appropriate gendered text.
- **Go behavior:** Go uses a custom helper `genderHisHer()` in `movement_cmds.go:418`.
- **Risk:** Type constant mismatch. `genderHisHer()` switches:
  - `0` → `"his"` (neutral in Go is 2? actually structs.h neutral=0, male=1, female=2).
  - `1` → `"her"`
  - `default` → `"its"`
  This maps male (`1` in structs.h) to `"her"`, female (`2`) to `"its"`, and neutral (`0`) to `"his"`. This leads to broken pronoun output during position transitions (e.g. "Bob clambers to her feet" or "Alice clambers to its feet").
- **Severity:** MEDIUM
- **Type:** DRIFT / BUG

---

## Concurrency & Mutex Safety

### [FINDING-005]: Concurrency Data Race in Follower Movement Dragging
- **Location:** `pkg/session/cmd_movement.go:78` inside `cmdMove()`.
- **C behavior:** Strictly single-threaded loop; thread-safe.
- **Go behavior:** When a player moves, the routine loops through and moves all followers concurrently:
  ```go
  for _, follower := range followers {
      if _, ferr := s.manager.world.MovePlayer(follower, direction); ferr == nil {
          follower.SendMessage(...)
          ...
      }
  }
  ```
  These followers are modified concurrently on separate player session goroutines without acquiring the follower's lock (`follower.mu`). This constitutes a classic write data race.
- **Severity:** HIGH
- **Type:** CONCURRENCY

---

## Unported Functions

All behavioral features in `act.movement.c` have been structurally ported to Go, with the exception of the `enter`, `leave`, and mount commands mapping.

---

## Summary

- **Total findings:** 5
- **Critical:** 0
- **High:** 4
- **Medium:** 1
- **Low:** 0
- **Unported functions:** 0

---

## Verification Plan

### Automated Verification
Verify the compilation safety of the movement commands:
```bash
go build ./pkg/session/...
go build ./pkg/game/...
```
