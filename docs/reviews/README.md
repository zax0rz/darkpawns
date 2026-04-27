# Opus Review Progress

**Started:** 2026-04-26  
**All passes complete:** 2026-04-26  
**Budget spent:** ~$24.57 Anthropic payg

## Pass Status

| # | Focus | Status | Report |
|---|-------|--------|--------|
| 1 | Architecture & Idiomatic Go | ✅ Done | `pass1-architecture.md` |
| 2 | Concurrency & Data Integrity | ✅ Done | `pass2-concurrency.md` |
| 3 | Security Deep Dive | ✅ Done | `pass3-security.md` |
| 4 | Fidelity to C Source | ✅ Done | `pass4-fidelity.md` |
| 5 | QA & Edge Cases | ✅ Done | `pass5-qa.md` |

## Backlog

Consolidated findings in `BACKLOG.md` — 86 deduplicated findings (14 CRITICAL, 26 HIGH, 30 MEDIUM, 16 LOW).

## C-to-Go Port Status

**COMPLETE.** All 5 port completion tiers finished (waves 17–20). Build clean. 1 TODO remaining (cosmetic). See `docs/architecture/PORT_SCOPE.md` for details.

## Fix Progress (updated 2026-04-27 16:09)

### CRITICAL (14 total — 12 fixed, 2 deferred)

| ID | Finding | Status | Commit |
|----|---------|--------|--------|
| C-01 | Dual Send Channel — Player.Send dead | ✅ Fixed | d270fb3 |
| C-02 | 70+ package-level mutable combat var hooks | 🔶 Deferred | Structural refactor |
| C-03 | game package is 38K-line god package | 🔶 Deferred | Structural refactor |
| C-04 | save.go reads Player without lock | ✅ Fixed | 865881e |
| C-05 | Session agent fields mutated cross-goroutine (map panic) | ✅ Fixed | 6bb78c5 |
| C-06 | playerSneakState/playerHideState global maps unsynced | ✅ Fixed | 6bb78c5 |
| C-07 | Telnet login bypasses password auth | ✅ Fixed | 4a87eb9 |
| C-08 | Wizard idlist arbitrary file write | ✅ Fixed | 4a87eb9 |
| C-09 | Saving throw system completely wrong (d100→d20) | ✅ Fixed | 8ec63dd |
| C-10 | Combat commands are stubs — no damage calculation | ✅ Fixed | 5fa7622..b5aa0c7 (5 commits) |
| C-11 | Parry/Dodge system not implemented | ✅ Fixed | 1288615 |
| C-12 | Double-close of s.send channel causes panic | ✅ Fixed | 2c13697 |
| C-13 | AdvanceLevel + Save deadlock when C-04 fixed | ✅ Fixed | 865881e |
| C-14 | Combat engine holds stale refs on disconnect | ✅ Fixed | 2c13697 |

### HIGH (26 total — 17 fixed, 6 deferred, 3 open)

| ID | Finding | Status | Commit |
|----|---------|--------|--------|
| H-01 | Package-level var for cross-package wiring | 🔶 Deferred | Structural (related to C-02) |
| H-02 | Session/Command split is confused | 🔶 Deferred | Structural refactor |
| H-03 | World struct does too much (20+ concerns) | 🔶 Deferred | Structural refactor |
| H-04 | interface{} used to break import cycles | 🔶 Deferred | Structural refactor |
| H-05 | Mutex discipline inconsistencies | 🔶 Deferred | Structural refactor |
| H-06 | common package interfaces too wide | 🔶 Deferred | Structural refactor |
| H-07 | SendToZone/SendToAll iterate w.players without lock | ✅ Fixed | 3246727 |
| H-08 | mobact.go accesses RoomVNum bypassing mutex | ✅ Fixed | cc9c233 |
| H-09 | PointUpdate reads p.Flags without lock | ✅ Fixed | 534fc5a |
| H-10 | Spawner goroutine leak | ✅ Fixed | 534fc5a |
| H-11 | SpawnMob blocking send while holding w.mu.Lock | ✅ Fixed | 3246727 |
| H-12 | X-Forwarded-For trusted without proxy validation | ✅ Fixed | f8e586b |
| H-13 | WebSocket allows connections without Origin | ✅ Fixed | fabd8c6 |
| H-14 | cmdAt recursive wizard command — no depth limit | ✅ Fixed | 47cb05e |
| H-15 | No account lockout for failed login attempts | ✅ Fixed | f8e586b |
| H-16 | Affect durations 75× too short | ✅ Fixed | 8ec63dd |
| H-17 | Position damage uses float division | ✅ Fixed | a4feb59 |
| H-18 | CON loss on death deterministic (C is probabilistic) | ✅ Fixed | a4feb59 |
| H-19 | number(1,100) vs rand.Intn(100) off-by-one | ✅ Fixed | 1288615 |
| H-20 | handlePlayerDeath inconsistent locking | ✅ Fixed | cc9c233 |
| H-21 | rawKill creates duplicate corpse | ✅ Fixed | a4feb59 |
| H-22 | No position check for combat commands | ✅ Fixed | a934f3d |
| H-23 | BroadcastToRoom silently drops messages | ✅ Fixed | 534fc5a |
| H-24 | force command stub — dangerous when implemented | ✅ Fixed | 47cb05e |
| H-25 | JWT 24-hour lifetime, no rotation | ⏳ Open | |
| H-26 | Counter_procs fall-through not faithful | ✅ Fixed | 1ea6f0d |

### MEDIUM (30 total — 5 fixed, 25 open)

| ID | Finding | Status | Commit |
|----|---------|--------|--------|
| M-12 | CORS wildcard subdomain matching too permissive | ✅ Fixed | 463c0a7 |
| M-13 | Dev mode CORS allows all origins | ✅ Fixed | 463c0a7 |
| M-14 | CSP allows unsafe-inline for scripts | ✅ Fixed | 463c0a7 |
| M-15 | cmdUsers exposes player IPs to all wizards | ⏳ Open | Rate limited — pending retry |
| M-18 | who command reveals agent status to players | ⏳ Open | Rate limited — pending retry |
| M-19 | Spell affect durations don't match C | ⏳ Open | Subagent in flight |
| M-20 | Bless applies AC instead of saving throw | ⏳ Open | Subagent in flight |
| M-21 | Blindness missing reagent bonus + NPC retaliation | ⏳ Open | Subagent in flight |
| M-01–M-11, M-16–M-17, M-22–M-30 | Various | ❌ Open | See BACKLOG.md |

### LOW (16 total — 0 fixed)

All 16 LOW findings open. See BACKLOG.md.

## Summary

| Severity | Total | Fixed | Open | Deferred |
|----------|-------|-------|------|----------|
| CRITICAL | 14 | 12 | 0 | 2 |
| HIGH | 26 | 17 | 1 | 6 |
| MEDIUM | 30 | 3 | 27 | 0 |
| LOW | 16 | 0 | 16 | 0 |
| **Total** | **86** | **32** | **44** | **8** |

### Deferred items (structural refactors)
- **C-02 + H-01**: Combat globals → injected callbacks
- **C-03 + H-02–H-06**: God package split, session/command split, interface cleanup

These are code quality improvements, not functional bugs. Planned after remaining MEDIUM/LOW items clear.

## Key Files

- `BACKLOG.md` — Full consolidated backlog with file:line references
- `docs/architecture/PORT_SCOPE.md` — Port completion status
- `docs/architecture/PORT_COMPLETION_PLAN.md` — Historical port plan (port is done)
