# Brief: Damage Metrics Guard — 2026-06-12

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-577 — DamageDealt accepts negative values (HIGH)

**File:** `pkg/metrics/metrics.go` — `DamageDealt()` (line ~269)

**Problem:**
`DamageDealt(sourceType string, amount int)` passes `amount` directly to `Counter.Add()`. Prometheus Counters are monotonically increasing — negative values corrupt the metric and break dashboards/alerts.

The C source (`src/fight.c:1483`) clamps damage to `[0, 3000]` before applying. The Go function has no guard.

Current code:
```go
func DamageDealt(sourceType string, amount int) {
	damageDealt.WithLabelValues(sourceType).Add(float64(amount))
}
```

**Fix:**
Add a guard at the top of the function:
```go
func DamageDealt(sourceType string, amount int) {
	if amount < 0 {
		return
	}
	damageDealt.WithLabelValues(sourceType).Add(float64(amount))
}
```

This is the minimal safe fix. It matches the C source's intent (no negative damage) without changing the function signature or requiring caller updates.

**Cite:** C source — `src/fight.c:1483` (damage calculation clamps to `[0, 3000]`). The Go port passes damage through without clamping.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## After Fix

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
go build ./... && go vet ./... && go test ./...
gofumpt -l .
git add -A
git commit -m "fix: DamageDealt guards against negative values (DP-577)"
git push -u origin fix/damage-metrics-2026-06-12
gh pr create --title "fix: DamageDealt negative guard (DP-577)" --body "Fixes DP-577. See docs/briefs/BRIEF-2026-06-12-damage-metrics.md for details."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-577: Add comment "Fixed — DamageDealt now returns early on negative amount, prevents Counter corruption", commit <hash>, move to Done
