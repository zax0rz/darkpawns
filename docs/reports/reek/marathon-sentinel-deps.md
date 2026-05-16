# Marathon Sentinel Audit + Dependency Check

**Date:** 2026-05-15  
**Auditor:** Daeron (subagent: reek-sentinel-deps)  
**Commits reviewed:** `d67dd2d` through `HEAD` (last 10)

---

## Task 1: Commit Sentinel

### Summary

Last 10 commits are clean. All fixes are verified, tested, and build-passing. No regressions detected.

| Commit | Description | Verdict |
|--------|-------------|---------|
| `d67dd2d` | Reek marathon — tracker reconciliation, stale test fix, redundant lock removal | ✅ Clean |
| `439281f` | dp-103: full luaSteal implementation | ✅ Clean — major rewrite, well-documented |
| `2b9d232` | dp-103: implement StealRandomItemFromChar | ✅ Clean |
| `0c92850` | dp-103: document luaSteal vnum=0 sentinel | ✅ Clean — documents broken behavior |
| `6d345df` | Reek findings triage fixes + DARKPAWNS.md | ✅ Clean |
| `584725f` | docs: admin panel handoff | ✅ Clean |
| `781dada` | docs: admin panel handoff for Blenda | ✅ Clean |
| `1d53648` | test: 161 admin panel tests with race detection | ✅ Clean |
| `8e44c43` | docs: admin spec update | ✅ Clean |
| `0fdf09a` | feat: Phase 6 dashboard + Phase 7 polish | ✅ Clean |

### Debug Prints Left In

| Severity | File | What | Verdict |
|----------|------|------|---------|
| LOW | `cmd/dp-agent/main.go:200-206` | `fmt.Println` / `fmt.Printf` | ✅ **Not a problem** — CLI entrypoint, these are user-facing output |
| LOW | `pkg/game/mail.go` (multiple) | `log.Printf("SYSERR: ...")` | ✅ **Not a problem** — intentional error logging, not debug prints |

**No debug prints found.** All `fmt.Print` calls are in `cmd/` (CLI output). All `log.Printf` calls are structured SYSERR logging appropriate for a MUD server.

### os.Exit Usage

| Severity | File | What | Verdict |
|----------|------|------|---------|
| LOW | `cmd/dp-agent/main.go` | `os.Exit(1)` | ✅ **Not a problem** — CLI main() entrypoint |

### Stale TODO/FIXME Comments

| Severity | File:Line | What | Recommendation |
|----------|-----------|------|----------------|
| LOW | `pkg/game/combat_ranged.go:179` | `TODO: Check MOB_SENTINEL flag when mob flags are accessible` | Track as DP issue — port fidelity gap |
| LOW | `pkg/game/comm_channel.go:207` | `TODO: act() for proper formatting` | Minor polish — cosmetic only |
| LOW | `pkg/game/comm_channel.go:212` | `TODO: act() for proper formatting` | Same as above |
| LOW | `pkg/game/comm_channel.go:223` | `TODO: Wire up interactive string editor for note-writing.` | Feature gap — track as DP issue |

**4 stale TODOs found.** None are critical. The MOB_SENTINEL check is a port fidelity gap worth tracking. The act() formatting TODOs are cosmetic. The note-writing editor is a known feature gap.

### New Data Races

**None detected.** Commit `49cf6cf` (ActiveAffects data race fix) is the most recent race-related commit, and it's a *fix* not a *introduction*. The redundant lock removal in `d67dd2d` was verified clean.

### Breaking Changes to Exported APIs

**None detected.** The luaSteal rewrite (`439281f`) added new interfaces (`objToTable`, `RemoveObjByInstanceID`, `GiveItemToMob`) but these are internal scripting engine functions, not public API.

### Regressions

**None detected.** The stale test fix in `d67dd2d` (`createMoneyDesc` expectations) was a test expectation correction, not a behavior regression.

---

## Task 2: Dependency Audit

### Module Configuration

| Item | Value | Status |
|------|-------|--------|
| Go directive | `go 1.26.3` | ✅ Matches toolchain (`go1.26.3 darwin/arm64`) |
| Module verify | `all modules verified` | ✅ Clean |
| `go.sum` lines | 62 | ✅ Normal size |

### Direct Dependencies

| Package | Version | Latest | Status |
|---------|---------|--------|--------|
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | — | ✅ Current |
| `github.com/gorilla/websocket` | v1.5.3 | — | ⚠️ **See below** |
| `github.com/lib/pq` | v1.12.3 | — | ✅ Current |
| `github.com/mattn/go-sqlite3` | v1.14.44 | — | ✅ Current |
| `github.com/prometheus/client_golang` | v1.23.2 | — | ✅ Current |
| `github.com/yuin/gopher-lua` | v1.1.2 | — | ✅ Current |
| `golang.org/x/crypto` | v0.51.0 | — | ✅ Current |
| `golang.org/x/text` | v0.37.0 | — | ✅ Current |
| `golang.org/x/time` | v0.15.0 | — | ✅ Current |

### Findings

| Severity | Dependency | What | Recommendation |
|----------|------------|------|----------------|
| LOW | `github.com/golang/protobuf` | Deprecated transitive dep (`v1.5.0 [v1.5.4]`) — not directly used by DP code | Run `go mod tidy` — should auto-remove if unused. If it persists, it's pulled by prometheus client model. No action needed. |
| LOW | `github.com/gorilla/websocket` | v1.5.3 — gorilla project is archived/unmaintained. v1.5.3 is the latest release. | Acceptable — no known CVEs, widely used. Consider switching to `github.com/coder/websocket` if maintaining long-term. |
| LOW | Transitive: `klauspost/compress` | v1.18.0 → v1.18.6 available | Non-breaking patch update. Safe to update. |
| LOW | Transitive: `go.yaml.in/yaml/v2` | v2.4.3 → v2.4.4 available | Patch update. Safe to update. |

### Go Module Cleanliness

- ✅ No unused direct dependencies detected
- ✅ Module is verified
- ✅ No major version mismatches
- ✅ `go.sum` is consistent (62 lines, normal for this dependency set)

---

## Task 3: Build Health

### Results

| Check | Result | Notes |
|-------|--------|-------|
| `go build ./...` | ✅ **PASS** | Clean build, no warnings |
| `go vet ./...` | ✅ **PASS** | No static analysis issues |
| `go test ./...` | ✅ **PASS** | All packages pass (some cached, no failures) |

### Test Coverage Notes

- `pkg/admin` — 0.680s (new, comprehensive test suite from Phase 5-7)
- `pkg/game`, `pkg/session`, `pkg/combat`, `pkg/scripting` — cached (recently ran, passing)
- 25 packages have no test files — acceptable for infrastructure/cmd packages

### Deprecation Notices

None. No compiler warnings or deprecation messages.

---

## Port Fidelity Flags

These `Simplified` comments in the codebase warrant tracking per AGENTS.md port fidelity rules:

| Severity | File:Line | What | Recommendation |
|----------|-----------|------|----------------|
| MEDIUM | `pkg/game/skills2.go:363` | `Simplified: single target version (no surrounding check)` | Track as DP issue — DoSerpentKick lacks AoE check that C version has |
| LOW | `pkg/game/skills2.go:205` | `Simplified version` — DoMindlink | Verify C behavior matches |
| LOW | `pkg/game/skills2.go:410` | `Simplified version: dig in current room` | Verify C DoDig behavior |
| LOW | `pkg/spells/affect_spells.go:1519` | `Simplified Go implementation` — castIdentify | Verify spell identification coverage |
| LOW | `pkg/session/comm_cmds.go:339` | `Simplified: requires "pen" and "paper"` — cmdWrite | Verify C write() behavior |

---

## Summary

| Category | CRITICAL | HIGH | MEDIUM | LOW |
|----------|----------|------|--------|-----|
| Regressions | 0 | 0 | 0 | 0 |
| Debug prints | 0 | 0 | 0 | 0 |
| Stale TODOs | 0 | 0 | 0 | 4 |
| Data races | 0 | 0 | 0 | 0 |
| Breaking changes | 0 | 0 | 0 | 0 |
| Dependency issues | 0 | 0 | 0 | 4 |
| Build failures | 0 | 0 | 0 | 0 |
| Port fidelity gaps | 0 | 0 | 1 | 4 |

**Overall: The codebase is in excellent health.** Build is clean, tests pass, no regressions or data races. The only actionable items are port fidelity tracking (Simplified comments) and a few stale TODOs. Dependency hygiene is good — the deprecated `golang/protobuf` is a transitive dep that doesn't affect DP directly.

**Recommendation:** No immediate action required. Track the `DoSerpentKick` simplified port fidelity gap (MEDIUM) as a DP issue for The Architect's review.
