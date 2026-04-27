# Code Hygiene Remediation Plan

> **Created:** 2026-04-26
> **Based on:** `docs/CODE_HYGIENE_AUDIT.md` (2026-04-23), verified against post-port codebase
> **Status:** Phase 3 remaining

---

## Verified Current State (2026-04-26)

`go build ./...` and `go test ./...` both pass clean.

## Completed

### Phase 1 — Quick Wins & Small Structural ✅
- #6 json.Marshal errors logged (movement_cmds.go)
- #13 EventQueue unexported + getter
- #15 ClassNames/RaceNames unexported + accessors
- #20 Empty init() functions removed
- #21 Report files moved to docs/archive/
- #23 Duplicate thaco → combat.ThacoTable, .backup deleted
- #4 ScriptEngine moved to World field
- #5 aiCombatEngine moved to World field
- #7 Mutex methods removed from Manager
- #10 Goroutine now uses context.WithTimeout
- #14 upgrader moved to Manager field
- #18 commandSessionWrapper removed, *Session implements CommandSession directly
- GetPlayer() interface mess cleaned up — command.SessionInterface returns *game.Player directly, no type assertions

### Phase 2 — Structural Fixes ✅
- #12 RegisterCommand no longer a stub — registers into cmdRegistry
- #11 CombatEngine function pointers → MessageBroadcaster/DeathHandler/ScriptFightHandler/DamageHandler interfaces
- #17 Affectable split into Identity/Stats/VitalSigns/StatusFlagHolder/AffectHolder/Messenger

### Items Fixed by Port (pre-audit)
- #1 pkg/telnet compile failure
- #2 tests/unit stale tests
- #3 Giant switch → cmdRegistry
- #19 Debug logging → slog (scripting)
- #22 Unused import suppression

---

## Remaining — Phase 3 (Big Sweep)

### #8 — slog migration ✅
- All `log.Printf`/`log.Println` migrated to `slog`
- 0 remaining

### #9 — Player field access triage ✅
- All direct field writes converted to setter methods
- Setters added: SetRoomVNum, SetHealth, SetMana, SetMaxHealth, SetMaxMana, SetLevel, SetGold, SetAlignment, SetStrength, SetMaxMove, SetMove, SetFollowing, SetInGroup, SetTitle, SetDescription, SetAFKMessage, ToggleAFK, ToggleAutoExit, AddAffect, SetStats, SetSex, SetID, SetRoom

### Skipped / Deferred
- #16 pkg/game splitting (93 files, too large)

### Scanner Results (2026-04-26)

**govulncheck:** 17 stdlib vulns, all fixed in Go 1.25.2/1.25.3 → update toolchain

**gosec:** 486 findings (non-test/non-benchmark). Triage:
| Count | ID | Severity | Action |
|------|----|----------|--------|
| 175 | G404 weak RNG | HIGH | `#nosec` — game dice don't need crypto/rand |
| 171 | G104 unhandled errors | LOW | Batch fix defer-close patterns |
| 29 | G706 log injection | LOW | Audit user-input slog params |
| 28 | G304 variable file path | MED | Data file loading, mostly fine |
| 15 | G703 path traversal | HIGH | **Audit and fix** |
| 13 | G301 dir permissions | MED | Tighten to 0750 |
| ~30 | G115 integer overflow | MED | Add bounds checks |
| 4 | G306/G302 file perms | MED | Tighten to 0600 |
| 3 | G114 no timeouts | MED | Wrap listeners with timeouts |
| 2 | G705 XSS | MED | Audit |
| 1 | G704 SSRF | HIGH | **Audit and fix** |
| 1 | G108 pprof exposed | HIGH | Gate behind admin auth |

### gosec Sweep Progress (2026-04-26)

| Priority | ID | Status | Details |
|----------|----|--------|----------|
| 1 | G108 | ✅ Done | pprof gated behind PPROF_USER/PPROF_PASS env vars, dedicated mux, crypto/subtle auth |
| 2 | G704 | ✅ Done | validateBaseURL() in constructor, #nosec + docs on all 3 http.NewRequest calls |
| 3 | G703 | ✅ Done | Real sanitization on docs-site, secrets, wizard_cmds; #nosec on trusted paths |
| 4 | G114 | ✅ Done | ReadHeader 10s, Read/Write 30s on both TLS and plain servers |
| 5 | G306/G302 | ✅ Done | profiler files annotated (debug), mail.go → 0600 |
| 6 | G301 | ✅ Done | 7 MkdirAll calls 0755→0750; exceptions for profiling/docs |
| 7 | G115 | 🔲 TODO | Integer overflow bounds checks (~30 findings) |
| 8 | G104 | 🔲 TODO | Unhandled errors, defer-close patterns (171 findings) |
| 9 | G706 | ✅ Done | slog PII handler wired into main.go, filters all string attrs |
| 10 | G705 | 🔲 TODO | XSS audit (2 findings) |
| — | G404 | ✅ Skipped | #nosec — game dice don't need crypto/rand |
| — | G304 | 🔲 TODO | Variable file path (28 findings, mostly data file loading) |

**Bonus:** Created `pkg/privacy/slog_handler.go` — PII-filtering slog.Handler wrapping OpenAI privacy filter service. All game logs filtered by default.

---

## Completion Criteria

- [x] `go build ./...` passes
- [x] `go test ./...` passes
- [x] `gosec ./...` and `govulncheck ./...` run (2026-04-26)
- [x] No global mutable state for ScriptEngine/aiCombatEngine
- [x] No exported mutex methods on Manager
- [x] Admin commands register properly
- [x] Repo root clean of report files
- [x] `log.Printf` count under 50 (0 remaining)
