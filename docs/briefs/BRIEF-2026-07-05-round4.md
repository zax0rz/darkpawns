# Brief: Round 4 — Combat Target Fallback + AES Sentinel + Proxy CIDRs + ConnectionMonitor Stats + Dead Callback + Spec-Proc Health Races

**Issues:** DP-846, DP-861, DP-860, DP-859, DP-849, DP-883
**Date:** 2026-07-05
**Priority:** Low (5) + Medium (1)
**Effort:** S each

---

## DP-846: Combat skill commands ignore the current fighting target (LOW → gameplay impact)

**Problem:** `pkg/command/skill_commands.go` — Bash (line 604), Kick (line 639), Trip (line 674), Headbutt (line 709) all check `ch.GetFighting() != "" && len(args) == 0` and immediately return "who?" instead of falling back to the fighting target. The comment at line 600 even says `// Find target — if in combat, default to fighting target` but the code does the opposite.

Disembowel (line 777) and DragonKick (line 808) use the **correct** pattern: check args first, then fall back to `FindTargetInRoom(ch.GetFighting())`, then "who?".

**Fix:** Restructure all four handlers to match the Disembowel pattern:
```go
// Correct pattern (from CmdDisembowel):
if len(args) > 0 {
    target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), strings.Join(args, " "), ch)
    if !found { return s.SendMessage("Bash who?\r\n") }
} else if ch.GetFighting() != "" {
    target, _, found = game.FindTargetInRoom(world, ch.GetRoom(), ch.GetFighting(), ch)
    if !found { return s.SendMessage("Bash who?\r\n") }
} else {
    return s.SendMessage("Bash who?\r\n")
}
```

**Cite:** C source — `src/` combat skill handlers. The C code uses `FIGHTING(ch)` macro to resolve the default target when no argument is provided. The Go port inverted the branch logic for these four skills.

**Regression Test:** Unit test: player in combat with no args → skill targets fighting opponent. Player not in combat with no args → "who?".

**Verification:** `go build ./... && go vet ./... && go test ./pkg/command/...`

---

## DP-861: AES-GCM authentication failures don't return documented sentinel error (LOW)

**Problem:** `pkg/secrets/manager.go:121-123` — `decrypt()` returns raw `gcm.Open` error instead of wrapping it as `ErrDecryptionFailed`. The sentinel error (line 20) is only used for the short-ciphertext check (line 118). Callers using `errors.Is(err, ErrDecryptionFailed)` won't match actual decryption failures.

**Fix:** Line 123: change `return "", err` to `return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)`.

**Cite:** No C equivalent — Go-only crypto package.

**Regression Test:** Test with wrong key ciphertext, verify `errors.Is(err, ErrDecryptionFailed)` returns true.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/secrets/...`

---

## DP-860: Invalid trusted proxy CIDRs silently ignored (LOW)

**Problem:** `pkg/auth/ratelimit.go:29-45` — `SetTrustedProxies` returns `error` but always returns `nil` (line 44). Malformed CIDRs are silently skipped (line 37-39) with only a TODO comment "log in real init." Callers can't detect misconfiguration.

**Fix:** Either return an error on bad CIDRs, or at minimum log the skipped entries:
```go
if err != nil {
    slog.Warn("skipping invalid trusted proxy CIDR", "cidr", c, "error", err)
    continue
}
```
Ideally collect errors and return them if any CIDRs failed.

**Cite:** No C equivalent — Go-only auth middleware.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/auth/...`

---

## DP-859: ConnectionMonitor stats fields always zero (LOW)

**Problem:** `pkg/optimization/database.go:376-412` — `checkHealth()` queries PostgreSQL `pg_stat_activity` and sets `OpenConnections`, `LastCheck`, `Healthy`, but never sets `InUse`, `Idle`, `WaitCount`, `WaitDuration`. These remain zero-valued forever. The struct promises them but `sql.DBStats()` is never called.

**Fix:** Add `dbStats := cm.db.Stats()` inside `checkHealth()` and populate:
```go
dbStats := cm.db.Stats()
cm.stats.InUse = dbStats.InUse
cm.stats.Idle = dbStats.Idle
cm.stats.WaitCount = dbStats.WaitCount
cm.stats.WaitDuration = dbStats.WaitDuration
```

**Cite:** No C equivalent — Go-only monitoring infrastructure.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/optimization/...`

---

## DP-849: AIRequest.Callback field defined but never invoked (LOW — dead code)

**Problem:** `pkg/optimization/python_ai.go:19` — `AIRequest.Callback func(AIResponse, error)` is declared but never called anywhere. `AsyncProcessor.Process()` takes a standalone `callback` parameter instead. The struct field is dead code and breaks the API contract it implies.

**Fix:** Remove the `Callback` field from `AIRequest` struct. If async callback support is desired in the future, it should be added as a first-class feature with actual invocation.

**Cite:** No C equivalent — Go-only AI integration.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/optimization/...`

---

## DP-883: Data races on char.Health in spec_procs and death handlers (LOW)

**Problem:** Three locations bypass the `Combatant` interface and directly access `p.Health` without locks:

1. `pkg/game/spec_procs3.go:725` — `vict.Health -= dam` (no lock). Races with combat engine's `TakeDamage()`.
2. `pkg/game/death.go:884-887` — `ch.MaxHealth++`, `ch.Health = ch.MaxHealth` (no lock). Races with combat regen.
3. `pkg/game/death.go:900` — `p.Health = p.MaxHealth` for ALL connected players (no lock). Broadest race surface.

The primary combat path through the `Combatant` interface (GetHP/TakeDamage/Heal) is properly locked. These are secondary bypasses.

**Fix:** Wrap the direct field accesses in `p.mu.Lock()`/`p.mu.Unlock()`, or route through existing `SetHP()`/`TakeDamage()` methods. For death.go line 900, iterate with individual locks.

**Cite:** C source — the C MUD is single-threaded so no locking needed. The Go port added concurrency via goroutines but missed these direct field accesses.

**Regression Test:** `go test -race ./pkg/game/...`

**Verification:** `go build ./... && go vet ./... && go test -race ./pkg/game/...`
