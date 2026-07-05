# Brief: Round 7 — Cache Integer Division + Audit Logger Race + Shop Interface Dup + promauto Panic + lvlGod Drift + Penalty Audit Trail

**Issues:** DP-769, DP-830, DP-832, DP-827, DP-826, DP-831
**Date:** 2026-07-05
**Priority:** Low (6)
**Effort:** S each, DP-827 is M

---

## DP-769: Integer division truncates avg_access_per_item in cache Stats() (LOW)

**Problem:** [`pkg/optimization/cache.go:114`](pkg/optimization/cache.go#L114) — `stats["avg_access_per_item"] = totalAccess / len(c.items)` uses integer division. With `totalAccess=5` and `len=10`, the result is `0` rather than `0.5`, producing a misleading metric. Meanwhile line 116's `hit_ratio` correctly uses `float64()` casts.

**Fix:** Change line 114 to: `stats["avg_access_per_item"] = float64(totalAccess) / float64(len(c.items))`

**Regression test:** `pkg/optimization/cache_test.go` — Create a cache, access items a known number of times, call `Stats()`, assert `avg_access_per_item` is a float matching `float64(totalAccess)/float64(itemCount)`.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/optimization/...`

---

## DP-830: Global logger access is unsynchronized (LOW)

**Problem:** [`pkg/audit/logger.go:86`](pkg/audit/logger.go#L86) — `globalLogger` is a bare `*AuditLogger` package-level pointer. `Init()` writes it (closing old, replacing new) and `LogEvent()` reads it, both without any mutex. If `Init()` is called concurrently with `LogEvent()`, a use-after-close race occurs.

**Fix:** Add a `sync.RWMutex` to protect `globalLogger`:
```go
var (
    globalLogger *AuditLogger
    loggerMu     sync.RWMutex
)

func Init(filename string) error {
    loggerMu.Lock()
    defer loggerMu.Unlock()
    // ... existing close+create logic
    globalLogger = logger
    return nil
}

func LogEvent(event AuditEvent) {
    loggerMu.RLock()
    defer loggerMu.RUnlock()
    if globalLogger != nil {
        globalLogger.Log(event)
    }
}
```

**Regression test:** `pkg/audit/logger_test.go` — Start a goroutine calling `LogEvent` in a loop, call `Init()` with a new filename from another goroutine, assert no data race with `go test -race`.

**Verification:** `go build ./... && go vet ./... && go test -race ./pkg/audit/...`

---

## DP-832: Duplicate shop manager interfaces split the package API (LOW)

**Problem:** Two shop manager interfaces exist in `pkg/common/`:
- [`common.go:5`](pkg/common/common.go#L5) — `ShopManager` with 4 methods (CreateShop, GetShop, GetShopByNPC, GetShopsInRoom)
- [`shop.go:7`](pkg/common/shop.go#L7) — `ShopManagerInterface` with 8 methods (superset, adds RemoveShop, GetAllShops, ProcessShopRestock, FindShopForTransaction)

Code typed against `ShopManager` cannot use removal, restock, or transaction lookup. Neither is marked deprecated.

**Fix:** Deprecate `ShopManager` in common.go — add a comment `// Deprecated: Use ShopManagerInterface from shop.go instead` and add `//go:deprecated` if tooling supports it. Then grep for all consumers of `ShopManager` (the 4-method one) and switch them to `ShopManagerInterface`. If no consumers exist, delete it entirely.

**Regression test:** `go build ./...` — ensure no compilation errors after removal/merge.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## DP-827: promauto registration panics on re-import with fresh registry (LOW)

**Problem:** [`pkg/metrics/metrics.go`](pkg/metrics/metrics.go) — All 17 metrics are registered via package-level `promauto.NewXxx` calls (lines 15-111). `promauto` registers into the global default registry on import. If any test or alternative init creates a fresh `prometheus.Registry` and the package still uses `promauto`, metrics silently bind to the wrong registry, and a second import panics with "duplicate metrics collector registration".

**Fix:** Replace `promauto.NewXxx(...)` with `prometheus.NewXxx(...)` and move registration into an explicit `Init(r prometheus.Registerer)` function. Provide a `Handler()` function that uses the custom registry. For backward compat, add a package-level `init()` that calls `Init(prometheus.DefaultRegisterer)` if not already initialized.

```go
var (
    connectionsActive *prometheus.Gauge
    // ... declare all as nil pointers
    registerer prometheus.Registerer = prometheus.DefaultRegisterer
    once       sync.Once
)

func Init(r prometheus.Registerer) {
    once.Do(func() {
        registerer = r
        connectionsActive = prometheus.NewGauge(prometheus.GaugeOpts{...})
        registerer.MustRegister(connectionsActive)
        // ... register all 17
    })
}

func Handler() http.Handler {
    return promhttp.Handler()
}
```

**Effort:** M (touches 17 metric declarations + adds Init + backward compat)

**Regression test:** `pkg/metrics/metrics_test.go` — Call `Init(prometheus.NewRegistry())` twice, assert no panic. Create a fresh registry, init, record a metric, gather from that registry, assert value present.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/metrics/...`

---

## DP-826: lvlGod hardcoded constant silently drifts from game.LVL_GOD (LOW)

**Problem:** [`pkg/command/admin_commands.go:751`](pkg/command/admin_commands.go#L751) — `const lvlGod = 34` with comment "mirrors game.LVL_GOD to avoid circular import". Currently both are 34, but the comment on line 754 also says "Primary gate: player level >= LVL_GOD (50)" which is wrong — it's 34. If `game.LVL_GOD` changes, this hardcoded copy drifts silently.

**Fix:** Replace the hardcoded const with a build-time assertion or runtime check:
```go
// lvlGod mirrors game.LVL_GOD to avoid a circular import.
// Keep in sync with pkg/game/limits.go LVL_GOD.
var lvlGod = game.LVL_GOD
```
Wait — this would cause a circular import. The correct fix: use a `//go:generate` or `_ = game.LVL_GOD` approach. Actually, the simplest fix: add a compile-time check. Since circular imports prevent direct use, add an `init()` with a build-tag assertion file in `cmd/server/` that verifies `admin_commands.lvlGod == game.LVL_GOD`, or simply fix the misleading comment and add a `//nolint`-style KEEP-IN-SYNC marker:
```go
// lvlGod mirrors game.LVL_GOD to avoid a circular import.
// KEEP-IN-SYNC: pkg/game/limits.go LVL_GOD (currently 34).
const lvlGod = 34
```
Then fix the stale comment on line 754 from "(50)" to "(34)".

**Regression test:** This is a maintenance marker, not runtime code. The init-assertion approach is best but requires careful import management.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## DP-831: cleanupExpiredPenalties deletes expired rows — no audit trail (LOW)

**Problem:** [`pkg/moderation/manager.go:220`](pkg/moderation/manager.go#L220) — The cleanup goroutine runs every 5 minutes and `DELETE FROM player_penalties WHERE expires_at IS NOT NULL AND expires_at <= NOW()`. After deletion, there is no record that a player was ever muted or banned. Only the admin_log table retains the initial action.

**Fix:** Change the DELETE to an UPDATE that marks the penalty as expired rather than deleting it:
```go
_, err := m.db.Exec(`
    UPDATE player_penalties
    SET status = 'expired', expired_at = NOW()
    WHERE expires_at IS NOT NULL AND expires_at <= NOW()
      AND status != 'expired'
`)
```
This preserves the penalty record for admin investigation while marking it as no longer active. All queries that check active penalties should already filter by `status = 'active'` or similar.

**Regression test:** `pkg/moderation/manager_test.go` — Insert a penalty with past expiry, run cleanup, assert the row still exists with `status = 'expired'` rather than being deleted.

**Verification:** `go build ./... && go vet ./... && go test ./pkg/moderation/...`

---

## Build Gate

```bash
go build ./... && go vet ./... && go test ./...
```

All four must pass before committing.
