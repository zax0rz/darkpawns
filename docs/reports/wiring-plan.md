# Wiring Plan — Last Mile to 100%

**For:** BRENDA69
**Scopist:** Daeron
**Date:** 2026-05-08
**Goal:** 100% feature parity with CircleMUD before admin panels/agent layers

---

## Architecture Overview

Dark Pawns has **two command layers:**

1. **Session layer** (`pkg/session/`) — thin wrappers adapted to the `CommandSession` interface. These are **already wired** via `cmdRegistry.Register()` in `commands.go`. They handle player input and delegate to game logic.

2. **Game layer** (`pkg/game/`) — the actual C-ported logic with signature `(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool`. These match the original C spec_proc pattern. **40 files are unwired.**

**Key insight:** Many session-layer commands already work. The game-layer files are the C-ported versions sitting unused. The work is NOT "rewrite everything" — it's:
- Wire the unique systems (mob AI, pathfinding, mail, clans, gates)
- Fix the 3 critical no-ops
- Complete the house system persistence
- Verify session-layer implementations match C behavior

---

## Priority 1: Fix Active No-Ops (3 items, ~4 hours)

These are called by live code and silently broken.

### 1a. `executeCommand` — charmed mob orders
**File:** `pkg/game/damage_stubs.go:90`
**Currently:** Returns true, does nothing.
**Called by:** `pkg/game/combat_control.go:64,74` — when a player orders a charmed/following mob.
**Fix:** Parse the command string and dispatch to the mob's available commands. In C, this calls `command_interpreter(ch, argument)`. In Go, it should call `w.executeCommand(vict, message)` which needs to actually dispatch.
**C source:** `act.comm.c do_order()` → `command_interpreter(vict, message)`

### 1b. `mobHasFlag` — mob AI detection
**File:** `pkg/game/graph.go:351`
**Currently:** Always returns false.
**Called by:** `pkg/game/graph.go:163` — checks MOB_SENTINEL flag.
**Fix:** Look up the mob's prototype and check its action flags. The mob prototype has `ActionBitvector` (or equivalent). Check `parser.Mob.ActionFlags` against the requested flag.
**C source:** `struct.h IS_SET(MOB_FLAGS(ch), flag)`

### 1c. `doMurder` — pk murder
**File:** `pkg/game/damage_stubs.go:102`
**Currently:** Returns true, does nothing.
**Fix:** Implement the murder command — allows a player to attack another player. Similar to `do_hit` but flags the attacker as a murderer. Check PK rules, alignment consequences.
**C source:** `act.offensive.c do_murder()`

---

## Priority 2: Wire Mob AI System (~8 hours)

This is the most impactful unwired system. Without it, mobs just stand there.

### 2a. `mobact.go` — Mobile Activity
**File:** `pkg/game/mobact.go` (139 lines, unwired)
**Contains:** `MobileActivity()` — the main mob AI tick function. Scans all active mobs, calls spec procs, handles aggressive mobs, memory-based仇恨, scavenging, etc.
**Wire:** Call `w.MobileActivity()` from the main game loop tick (same place `PointUpdate()` is called).
**C source:** `mobact.c mobile_activity()`

### 2b. `graph.go` — Pathfinding + Hunt
**File:** `pkg/game/graph.go` (353 lines, unwired)
**Contains:** `FindFirstStep()`, `doTrack()`, `huntVictim()`, `huntTrashTalk()`
**Wire:** `doTrack` needs to be registered as a command. `huntVictim` is called from `MobileActivity()` for MOB_HUNTER mobs.
**Depends on:** `mobHasFlag` (Priority 1b) must be fixed first.
**C source:** `graph.c do_track()`, `graph.c hunt_victim()`

### 2c. `combat_control.go` — Order command
**File:** `pkg/game/combat_control.go` (partially wired via session)
**Contains:** Order/command charmed mobs
**Wire:** Session layer has `order` registered but delegates to `executeCommand` which is a no-op. Fix Priority 1a and this works.

---

## Priority 3: Wire Communication Systems (~6 hours)

### 3a. `mail.go` — Mail System
**File:** `pkg/game/mail.go` (unwired)
**Contains:** File-backed mail with BLOCK_SIZE=100 byte records, LE64 headers, linked blocks.
**Wire:** Register `mail` command, wire `InitMailSystem()` at boot.
**C source:** `mail.c`

### 3b. `comm_channel.go`, `comm_say.go`, `comm_tell.go`
**File:** `pkg/game/comm_*.go` (unwired)
**Note:** Session layer already has say/tell/gossip wired. These game-layer files may be redundant or more complete. Compare and merge if the game-layer versions have features the session versions lack.
**Action:** Audit both versions, keep the more complete one.

### 3c. `act_comm.go` — Communication Commands
**File:** `pkg/game/act_comm.go` (unwired)
**Contains:** shout, whisper, ask, write, page, gen_comm, qcomm, think, ctell
**Note:** Session layer already has some of these. Check for completeness.

---

## Priority 4: Complete House System (~8 hours)

### 4a. House Load/Save
**File:** `pkg/game/house_save.go`
**Status:** 14 files of code exist, but `houseLoad()` is a no-op and `ObjFromStore`/`ObjToStore` are stubs.
**Fix:** Implement file I/O for house data. The C version uses `house.save` files with binary records.
**C source:** `house.c house_save_table[]`, `house.c store_object()`, `house.c unstore_object()`

### 4b. House Player Commands
**File:** `pkg/game/house_player.go`
**Status:** Code exists but depends on load/save being functional.
**Wire:** Register `house`, `home`, `hcontrol` commands (some may already be in the registry).

---

## Priority 5: Wire Remaining Systems (~10 hours)

### 5a. `clans.go` — Clan System
**File:** `pkg/game/clans.go` (unwired)
**Contains:** Clan manager, save/load, member management
**Wire:** Register `clan` commands, wire clan system at boot.
**C source:** `clan.c`

### 5b. `gates.go` — Portal/Spell Gate
**File:** `pkg/game/gates.go` (unwired)
**Contains:** `LoadNightGate()`, `RemoveNightGate()`, `SpellGate()`
**Wire:** Register `gate` spell, wire gate loading at boot.
**C source:** `act.other.c do_gate()`

### 5c. `mobprogs.go` — Mob Program System
**File:** `pkg/game/mobprogs.go` (unwired)
**Contains:** Lua-triggered mob behaviors
**Wire:** Connect to the Lua scripting engine.
**Depends on:** Scripting engine (already working)

### 5d. `remort_helpers.go` — Remort System
**File:** `pkg/game/remort_helpers.go` (unwired)
**Contains:** Remort/riser mechanics
**Wire:** Register remort commands.

### 5e. `mapcode.go` — Map Display
**File:** `pkg/game/mapcode.go` (unwired)
**Contains:** ASCII map generation
**Wire:** Register `map` command (may already be in session layer).

---

## Priority 6: Verify Session-Layer Completeness (~8 hours)

Audit each session-layer command against the C source to ensure no features were lost in the thin-wrapper adaptation.

### High-priority session commands to audit:
- `cmdLook` — does it handle all C cases (infravision, dark rooms, smoke, etc.)?
- `cmdGet`/`cmdDrop`/`cmdGive` — container handling, gold, all/all.item
- `cmdEat`/`cmdDrink` — poison, drunkenness, nutrition values
- `cmdOpen`/`cmdClose`/`cmdLock`/`cmdUnlock` — door states, key requirements
- `cmdWear`/`cmdRemove` — all equipment slots, anti-alignment checks
- `cmdCast` — all spell types (affect, damage, summon, portal, etc.)
- `cmdPractice` — skill/spell practice system
- `cmdGroup` — group mechanics, follow/leader

---

## Summary

| Priority | What | Hours | Depends on |
|---|---|---|---|
| 1 | Fix 3 active no-ops | 4 | Nothing |
| 2 | Wire mob AI | 8 | Priority 1b |
| 3 | Wire communication | 6 | Nothing |
| 4 | Complete house system | 8 | Nothing |
| 5 | Wire remaining systems | 10 | Nothing |
| 6 | Verify session completeness | 8 | Nothing |
| **Total** | | **44** | |

**Note:** This is an estimate. Some items may be simpler than expected (the code is already ported). Some may reveal gaps that need more work. The 44 hours is a floor, not a ceiling.

---

## How to Approach This

1. **Fix the no-ops first** (Priority 1) — these are bugs, not features
2. **Wire mob AI** (Priority 2) — this is the biggest behavioral gap
3. **Then work through Priorities 3-6** in any order
4. **After each priority:** `go build ./... && go vet ./... && go test ./...`
5. **Commit after each priority** with descriptive message

**Do NOT try to wire everything at once.** One priority at a time. Build. Test. Commit. Move on.

---

*The bones are all here. Connect them one by one.*
