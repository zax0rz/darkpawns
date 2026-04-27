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

## Fix Progress (as of 2026-04-27)

### CRITICAL (14 total)

| ID | Finding | Status |
|----|---------|--------|
| C-01 | Dual Send Channel — Player.Send dead | ❌ Open |
| C-02 | 70+ package-level mutable combat var hooks | ❌ Open |
| C-03 | game package is 38K-line god package | ❌ Open |
| C-04 | save.go reads Player without lock | ❌ Open |
| C-05 | Session agent fields mutated cross-goroutine (map panic) | ❌ Open |
| C-06 | playerSneakState/playerHideState global maps unsynced | ❌ Open |
| C-07 | Telnet login bypasses password auth | ❌ Open |
| C-08 | Wizard idlist arbitrary file write | ⚠️ Partial (filename hardcoded, no filepath.Base guard) |
| C-09 | Saving throw system completely wrong (d100→d20) | ❌ Open |
| C-10 | Combat commands are stubs — no damage calculation | ❌ Open |
| C-11 | Parry/Dodge system not implemented | ❌ Open |
| C-12 | Double-close of s.send channel causes panic | ✅ Fixed (2c13697) |
| C-13 | AdvanceLevel + Save deadlock when C-04 fixed | ❌ Open (blocked on C-04) |
| C-14 | Combat engine holds stale refs on disconnect | ✅ Fixed (2c13697) |

### Recommended Fix Order (from BACKLOG.md)

1. ~~C-12 + M-25~~ ✅ Done
2. **C-14 + C-01** — Combat stale refs + dual send channel (C-14 done, C-01 open)
3. **C-04 + C-13** — Save locking + AdvanceLevel restructure
4. **C-05 + C-06** — Cross-goroutine map panics
5. **C-08 + C-07** — Security (file write + telnet auth)
6. **C-09 + H-16** — Saving throws + affect duration
7. **C-10 + C-11** — Combat command stubs + parry/dodge
8. **H-12 + H-15** — Rate limiting bypass + account lockout
9. **H-22 + M-27** — Position check enforcement + death position reset
10. **C-02 + H-01** — Combat globals → injected callbacks

## Key Files

- `BACKLOG.md` — Full consolidated backlog with file:line references
- `docs/architecture/PORT_SCOPE.md` — Port completion status
- `docs/architecture/PORT_COMPLETION_PLAN.md` — Historical port plan (outdated — port is done)
- `docs/plans/` — Historical planning docs
- `docs/architecture/` — Design docs, protocol specs
- `docs/operational/` — CI, security, monitoring
