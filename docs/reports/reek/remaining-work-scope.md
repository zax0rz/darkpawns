# Remaining Work — Scoped for BRENDA69

**Scopist:** Daeron
**Date:** 2026-05-07
**Codebase state:** Clean build, all tests passing, HEAD: c6f9c9d

---

## Tier 1: Quick Wins (mechanical, low risk, ~1-2 hours each)

### 1a. Lock Ordering Documentation (HIGH-001)
**What:** The codebase has 394 lock acquisitions across `pkg/game/` with no documented hierarchy. This is a deadlock waiting to happen. Need a comment block in `pkg/game/` (probably `world.go` or a new `locks.go`) documenting the lock acquisition order.

**Known lock holders:** `World.mu`, `Player.mu`, `MobInstance.mu`, `Equipment.mu`, `Inventory.mu`, `ObjectInstance.mu` (if any)

**Approach:** Audit lock acquisitions, document the hierarchy, add a comment. If a violation is found, fix it. If not, the comment prevents future violations.

**Risk:** Low — documentation only unless a real ordering violation is found.

---

### 1b. Unused Code Cleanup (U1000 — 268 items)
**What:** 268 functions/variables/types that are ported from C but not yet wired to the command registry. These are NOT bugs — they're the migration backlog. But they clutter staticcheck output and make real findings harder to spot.

**Breakdown by file (top 10):**
| File | Count | What's there |
|---|---|---|
| pkg/game/item_helpers.go | 46 | Item manipulation functions |
| pkg/session/display_cmds.go | 40 | Display/info commands |
| pkg/game/info_commands.go | 21 | Information commands |
| pkg/game/act_movement.go | 18 | Movement commands |
| pkg/game/act_comm.go | 9 | Communication commands |
| pkg/game/look.go | 7 | Look/examine |
| pkg/game/combat_basic.go | 7 | Basic combat |
| pkg/game/remort_helpers.go | 6 | Remort system |
| pkg/game/modify.go | 6 | Player modification |
| pkg/game/item_equipment.go | 6 | Equipment handling |

**Approach (two options):**

**Option A — Build tags (safe):** Wrap unwired functions in `//go:build ignore` or move to a `legacy/` directory. Preserves code, removes from staticcheck.

**Option B — Delete (aggressive):** Remove entirely. They're in git history if needed. Cleaner codebase, but loses the "ready to wire" ported code.

**My recommendation:** Option A for now. These functions are the ported C source. Deleting them means re-porting when we need them. Build tags or a `legacy/` directory keeps them available without noise.

**Risk:** Low — no runtime impact either way.

---

### 1c. SA4004 / SA4000 Suppression Cleanup
**What:** Two intentional findings that Reek will keep flagging:
- `equipment.go:235` — loop unconditionally terminated (SA4004). Already has `//nolint:staticcheck` comment.
- `spec_procs3.go:903` — identical && expressions (SA4000). Already has `//nolint:staticcheck` comment.

**Status:** Already suppressed. Reek shouldn't flag these again if the nolint comments are correct. If Reek does flag them, they're false positives.

---

## Tier 2: Medium Effort (design decisions, ~4-8 hours each)

### 2a. Duplicated Entry Points (HIGH-003)
**What:** `cmd/server/main.go` and `cmd/server/main_web.go` share ~90% of their code (TLS config, signal handling, World creation, etc.). Any fix to one must be manually replicated to the other — which is exactly how CRITICAL-001 got missed in one but not the other.

**Files:** `cmd/server/main.go`, `cmd/server/main_web.go`

**Approach:** Extract shared startup/shutdown into a `server.Launch(config)` function. Both entry points call it with their specific config (TLS vs non-TLS). The duplicated signal handling, World creation, and listener setup become one path.

**Risk:** Medium — refactoring entry points requires careful testing of both startup paths.

**Dependencies:** None. Can be done independently.

---

### 2b. Non-TLS Default (HIGH-005)
**What:** `main_web.go` defaults to TLS (`:443`), `main.go` defaults to non-TLS (`:7777`). The MUD port 7777 is non-TLS by design (telnet clients can't do TLS). But the web admin panel should probably default to TLS in production.

**Decision needed from Architect:** Is the web admin meant to be accessed over TLS? If yes, default to TLS and document how to override. If no, document why.

**Risk:** Low — configuration only.

---

## Tier 3: Architecture (big scope, needs planning)

### 3a. Command Registry Wiring
**What:** The 268 U1000 functions need to be wired to the command registry. This is the actual remaining port work — connecting the ported C functions to the Go command dispatcher.

**Scope:** Large. Each function needs:
1. A command entry in the registry
2. Parameter parsing aligned with the Go parser
3. Integration testing with the existing command infrastructure

**My recommendation:** Don't tackle this all at once. Wire commands as they're needed or as Reek identifies specific gaps.

---

## Summary for BRENDA

| Priority | Item | Effort | Risk |
|---|---|---|---|
| Do first | Lock ordering docs (1a) | 1-2h | Low |
| Do first | U1000 cleanup (1b) | 2-4h | Low |
| Skip | SA4004/SA4000 (1c) | Done | None |
| Plan it | Entry point refactor (2a) | 4-8h | Medium |
| Decide | TLS default (2b) | 1h | Low |
| Later | Command wiring (3a) | Weeks | High |

**What Reek will find tonight:** Unless 1a and 1b are done, Reek will flag the same 268 U1000 items and potentially lock ordering concerns. If 1a/1b are done, Reek's report should be nearly clean.

---

*Scoped by Daeron. The machine has her orders.*
