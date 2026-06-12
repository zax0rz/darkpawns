# Brief: Critical Bug Batch — 2026-06-12

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS before commit.
**Model:** Gemini 3.5 Flash

---

## Fix 1: DP-527 — Tick() drops expired affect cleanup (CRITICAL)

**File:** `pkg/engine/affect_manager.go` — `Tick()` method (line ~340)

**Problem:** Phase 1 (under lock) collects expired affects and removes them from `am.affects[entityID]`. Phase 2 calls `RemoveAffectsBySpell()` to process the removal — but `RemoveAffectsBySpell()` re-acquires the lock and iterates `am.affects[entityID]`, which no longer contains the expired affects (they were already removed in Phase 1). So it finds nothing.

**Result:** Expired affects are silently dropped. No `removeAffectImmediate()` fires (stat reversals lost), no `ClearStatusFlag()` fires (status flags stuck on), no wear-off message sent. Example: poison expires but the stat penalty stays forever.

**Fix:** Don't call `RemoveAffectsBySpell` in Phase 2. Instead, for each expired affect, call `removeAffectImmediate()` and `ClearStatusFlag()` directly (these don't need the lock — they're called from within the lock elsewhere). Then send the wear-off message. Keep the lock ordering (collect under lock, process outside lock) but do the actual work inline instead of through the public method.

**Key code path:**
```go
// Phase 2 should look like this (NOT calling RemoveAffectsBySpell):
for _, entry := range expiredAffects {
    entityID := am.getEntityID(entry.entity)
    
    // Reverse stat changes
    am.removeAffectImmediate(entry.entity, entry.aff)
    
    // Decrement flag refs and clear if zero
    if entry.aff.Flags != 0 {
        if refs, ok := am.flagRefs[entityID]; ok {
            refs[entry.aff.Flags]--
            if refs[entry.aff.Flags] <= 0 {
                entry.entity.ClearStatusFlag(entry.aff.Flags)
                delete(refs, entry.aff.Flags)
            }
        }
    }
    
    // Send wear-off message
    am.sendAffectMessage(entry.entity, entry.aff, false)
}
```

**Verification:** `go build ./... && go vet ./... && go test ./...`
**Cite:** C source — no direct equivalent; affect tick is a Go addition.

---

## Fix 2: DP-561 — Shop.Restock() deadlocks (HIGH)

**File:** `pkg/game/systems/shop.go` — `Restock()` method (line 379)

**Problem:** `Restock()` acquires `s.mu.Lock()` at line 382, then calls `s.CanSellType()` at line 395 inside the loop. `CanSellType()` (line 165) immediately tries `s.mu.RLock()`. Go's `sync.RWMutex` is not re-entrant — the `RLock` blocks forever waiting for the writer (itself). Deadlock.

**Fix:** Option A (preferred) — duplicate the type check inline without calling `CanSellType()`:
```go
// Instead of s.CanSellType(proto.TypeFlag), check directly:
sellsThisType := false
for _, t := range s.ShopTypes {
    if t == proto.TypeFlag {
        sellsThisType = true
        break
    }
}
if !sellsThisType {
    continue
}
```

Option B — release the lock before the loop, re-acquire for each item. More complex, more lock contention. Don't do this.

**Cite:** C source — `src/shop.c:800-815` (stock_mobiles). C uses a simple function call without mutex (single-threaded). The Go port added mutex but didn't account for re-entrancy.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 3: DP-558 — Go stdlib vulns (CRITICAL)

**File:** `go.mod` — change `go 1.26.3` to `go 1.26.4`

**Problem:** Three standard library CVEs affect the codebase. All fixed in Go 1.26.4:
- GO-2026-5039: net/textproto — arbitrary inputs in errors without escaping
- GO-2026-5044: net/http — request smuggling via invalid Transfer-Encoding
- GO-2026-5045: crypto/ecdh — P256 secret scalar not zeroed after use

**Fix:**
```bash
# In the repo root:
go mod edit -go=1.26.4
go mod tidy
go build ./... && go vet ./... && go test ./...
```

**Also fix DP-559 (HIGH):** After Go upgrade, update golang-jwt:
```bash
go get github.com/golang-jwt/jwt/v5@latest
go mod tidy
go build ./... && go vet ./... && go test ./...
```
This fixes CVE-2025-30204 (DoS via asymmetric resource consumption).

**Also fix DP-560 (MEDIUM):** After Go upgrade, update x/crypto:
```bash
go get golang.org/x/crypto@latest
go mod tidy
go build ./... && go vet ./... && go test ./...
```
This fixes 14 SSH vulnerabilities in x/crypto v0.51.0.

**Verification:** `go build ./... && go vet ./... && go test ./...` — all three must pass.

---

## Execution Order

1. **DP-558** (Go upgrade + deps) — smallest diff, unblocks crypto fixes
2. **DP-561** (Shop deadlock) — straightforward, prevents zone reset hangs
3. **DP-527** (Affect cleanup) — most complex, needs careful testing

## After All Three

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
go build ./... && go vet ./... && go test ./...
git add -A
git commit -m "fix: critical batch — affect cleanup, shop deadlock, Go upgrade (DP-527, DP-561, DP-558, DP-559, DP-560)"
git push -u origin fix/critical-batch-2026-06-12
gh pr create --title "fix: critical batch (DP-527, DP-561, DP-558, DP-559, DP-560)" --body "See docs/briefs/BRIEF-2026-06-12-critical-batch.md for details."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.

## Linear Updates (do these AFTER merge)

- DP-558: Add comment "Fixed — upgraded to Go 1.26.4", move to Done
- DP-559: Add comment "Fixed — golang-jwt updated to latest", move to Done
- DP-560: Add comment "Fixed — x/crypto updated to latest", move to Done
- DP-561: Add comment "Fixed — inlined CanSellType check to avoid re-entrant lock", move to Done
- DP-527: Add comment "Fixed — Phase 2 now calls removeAffectImmediate/ClearStatusFlag directly", move to Done
