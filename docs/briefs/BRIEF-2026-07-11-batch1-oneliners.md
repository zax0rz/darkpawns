# Brief: Batch 1 One-Liners — 2026-07-11

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## CRITICAL: Verify Against C Source Before Fixing

**Do NOT just apply the fix described below.** Read the C source file at the path specified in each `**Cite:**` field FIRST. Confirm the C behavior matches what this brief describes. If the C source says something different from what's written here, STOP and report the discrepancy. Fidelity to C is the entire point.

---

## Fix 1: DP-1044 — Mob HP regen rounds up for odd levels (LOW)

**File:** `pkg/game/limits_gain.go:154` — `GainHP()` (mob branch)

**Problem:**
C integer truncation rounds DOWN. Go's `(lvl*5 + 1) / 2` rounds UP. An odd-level mob (e.g. level 9) gets 1 extra HP/tick in Go vs C.

- C (limits.c:133): `gain = 2.5*GET_LEVEL(ch)` — double 22.5 assigned to int → truncates to 22
- Go (limits_gain.go:154): `(lvl*5 + 1) / 2` → (45+1)/2 = 23

**Fix:**
Change line 154 from:
```go
return (lvl*5 + 1) / 2 // integer approximation of 2.5×level
```
to:
```go
return lvl * 5 / 2 // integer approximation of 2.5×level (truncates, matching C)
```

**Cite:** `src/limits.c:133-135` — `gain = 2.5*GET_LEVEL(ch)` assigned to `int gain`. C integer truncation drops the fractional part. Go's `+1` before `/2` was an incorrect attempt to match this — it over-corrects into rounding up.

**Regression Test:**
Add to `pkg/game/limits_gain_test.go` (or create if missing):
- `TestMobHPRegenOddLevel`: level 9 → expect 22 (not 23)
- `TestMobHPRegenEvenLevel`: level 10 → expect 25 (same in both)
- `TestMobHPRegenHighLevel`: level 22 → expect 55 (last level using 2.5x before the 4x branch at 23)

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-1039 — Slowed characters still get 1 attack; C allows 0 (LOW)

**File:** `pkg/combat/formulas.go:631-634` — `CalcAttacks()`

**Problem:**
C's sanity check allows 0 attacks per round (a slowed low-level mob/player skips a round entirely). Go's floor of 1 means SLOW can never fully deny a round.

- C (fight.c:2006-2007): `if (attacks < 0) attacks = 0;` — allows zero attacks
- Go (formulas.go:631): `if attacks < 1 { attacks = 1 }` — floors at 1

**Fix:**
Change line 631 from:
```go
if attacks < 1 {
    attacks = 1
}
```
to:
```go
if attacks < 0 {
    attacks = 0
}
```

**Cite:** `src/fight.c:2006-2007` — `if (attacks < 0)  /* sanity check for slow -rparet */ attacks = 0;`. The C comment even credits the original author (-rparet). Go's floor at 1 is a port error — someone assumed "at least 1 attack" but C explicitly allows 0.

**Regression Test:**
Add to `pkg/combat/formulas_test.go` (or create if missing):
- `TestSlowCanDenyRound`: a level 1 mob (base attacks=1) with AFF_SLOW → attacks goes to 0, expect 0 (not 1)
- `TestSlowOnHighLevelStillAttacks`: a level 40 mob (base attacks=5+) with AFF_SLOW → expect >0

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. Fix 1 (DP-1044) — isolated, no dependencies
2. Fix 2 (DP-1039) — isolated, no dependencies
3. Run build gate
4. Commit both with tests

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
git add pkg/game/limits_gain.go pkg/combat/formulas.go
git add -u  # pick up any test files
git commit -m "fix: mob HP regen truncation + slow attack floor (DP-1044, DP-1039)"
git push -u origin fix/batch1-oneliners
gh pr create --title "fix: mob HP regen truncation + slow attack floor (DP-1044, DP-1039)" --body "Fixes DP-1044, DP-1039. Both are C fidelity one-liners verified against src/limits.c and src/fight.c."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.
