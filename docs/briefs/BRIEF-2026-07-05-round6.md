# Brief: Round 6 — Telnet Write Deadline + Audit Init Leak + CORS Guard + Makefile Path + Example Channel Order

**Issues:** DP-804, DP-799, DP-798, DP-829, DP-794
**Date:** 2026-07-05
**Priority:** Medium (4) + Medium (1 example)
**Effort:** M (DP-804), S (rest)

---

## DP-804: Telnet sessions can block forever waiting for writeLoop to drain (MED → goroutine/fd leak)

**Problem:** `pkg/telnet/listener.go:437-454` — On login failure, `handleConn` calls `s.CloseSend()` then does `<-done` to wait for `writeLoop` to exit. `writeLoop` calls `tc.Write(data)` with **no write deadline**. If the client stops reading, the kernel TCP send buffer fills and `tc.Write` blocks indefinitely. The goroutine and file descriptor leak permanently — the `defer rawConn.Close()` never runs.

Two failure paths both have this pattern:
- Line 440: `sendLoginWithPassword` error → `s.CloseSend()` → `<-done`
- Line 451: `!s.IsAuthenticated()` → `s.CloseSend()` → `<-done`

**Fix:** Set a write deadline on the raw TCP connection before waiting for `writeLoop`:

```go
// In both login-failure paths, BEFORE s.CloseSend():
_ = rawConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
s.CloseSend()
<-done
rawConn.SetWriteDeadline(time.Time{}) // clear deadline for subsequent use if needed
```

This ensures `tc.Write` will return with a deadline error after 5 seconds, unblocking `writeLoop`, which drains the channel and exits, unblocking `<-done`.

Search for ALL `<-done` patterns in listener.go — there may be other cleanup paths (e.g., normal disconnect) that have the same issue. Apply the deadline guard consistently.

**Cite:** C source `src/comm.c` uses non-blocking sockets with select(), so this scenario doesn't apply. The Go telnet layer is new and needs explicit write deadline management.

**Regression test:** Hard to unit test (requires a real TCP conn that stops reading), but verify with `go vet ./pkg/telnet/...` and `go test ./pkg/telnet/...`. A manual test: connect, send bad password, kill the client's read side, verify the server goroutine exits within 5 seconds.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## DP-799: Repeated Init calls leak the previous audit log file (MED → fd leak)

**Problem:** `pkg/audit/logger.go:89-96` — `Init()` creates a new `AuditLogger` and overwrites `globalLogger` without closing the existing one. On repeated calls (tests, reload), the old file descriptor leaks.

```go
func Init(filename string) error {
	logger, err := NewAuditLogger(filename)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}
```

**Fix:** Close the previous logger before replacing:

```go
func Init(filename string) error {
	if globalLogger != nil {
		if err := globalLogger.Close(); err != nil {
			slog.Warn("audit logger close on re-init failed", "error", err)
		}
	}
	logger, err := NewAuditLogger(filename)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}
```

**Cite:** No C counterpart — Go audit layer.

**Regression test:** `TestAuditInit_RepeatedCalls` — call Init() twice with different filenames, verify only one file descriptor is open (or at minimum, no panic/error on second Init).

**Verification:** `go build ./pkg/audit/... && go test ./pkg/audit/...`

---

## DP-798: CORS dev-mode all-origins bypass guarded only by ENVIRONMENT env var (MED → security)

**Problem:** `web/cors.go:66-76` — When `ENVIRONMENT=development`, `isOriginAllowed` returns `true` for EVERY origin. The sole guard is a single env var. If misconfigured in production, any website can make credentialed cross-origin requests.

```go
func isDevMode() bool {
	return os.Getenv("ENVIRONMENT") == "development"
}
```

**Fix:** Add a secondary guard — also require that the request is from a local address:

```go
func isDevMode(r *http.Request) bool {
	if os.Getenv("ENVIRONMENT") != "development" {
		return false
	}
	host := r.Host
	// Only allow from localhost / 127.0.0.1 / [::1]
	if strings.HasPrefix(host, "localhost") ||
		strings.HasPrefix(host, "127.0.0.1") ||
		strings.HasPrefix(host, "[::1]") ||
		strings.HasPrefix(host, "0.0.0.0") {
		return true
	}
	log.Printf("[CORS] WARNING: dev mode CORS rejected for non-local origin %q", r.RemoteAddr)
	return false
}
```

This requires changing the `isOriginAllowed` signature to accept `*http.Request` (or at minimum the host string). Check all callers to pass the request through. The `GetAllowedOrigins` / `isOriginAllowed` chain needs the request context propagated from the middleware handler.

**Cite:** No C counterpart — Go web layer.

**Regression test:** `TestCORS_DevModeNonLocalRejected` — set ENVIRONMENT=development, pass origin from 1.2.3.4, verify rejected.

**Verification:** `go build ./web/... && go test ./web/...`

---

## DP-829: Makefile WORLD_DIR defaults to ../darkpawns/lib — breaks non-standard checkout names (MED → build)

**Problem:** `Makefile:3-4`:
```makefile
WORLD_DIR ?= ../darkpawns/lib
```
This only resolves correctly when the checkout directory is literally named `darkpawns`. Anyone cloning to `darkpawns_repo`, `server`, etc. gets a stale path.

**Fix:** Derive from the Makefile's own location:

```makefile
# World directory — resolve relative to repo root, not directory name
WORLD_DIR ?= $(dir $(lastword $(MAKEFILE_LIST)))lib/world

# Or simpler: use shell to get the directory containing this Makefile
REPO_ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
WORLD_DIR  ?= $(REPO_ROOT)lib/world
```

The `lib/world/` directory exists in the repo, so relative-to-Makefile works regardless of checkout name.

**Cite:** No C counterpart — build tooling.

**Regression test:** `make run` / `make test-parse` from a differently-named checkout directory should resolve correctly.

**Verification:** `make test-parse` succeeds.

---

## DP-794: WebSocketOptimizationExample closes channels before unregistering from pool (MED → example panic risk)

**Problem:** `examples/optimization_integration.go:174-178`:
```go
close(session1)
close(session2)
pool.Unregister("session-1")
pool.Unregister("session-2")
```
If the pool sends on a channel between `close()` and `Unregister()`, Go panics with "send on closed channel".

**Fix:** Swap the order — unregister first, then close:
```go
pool.Unregister("session-1")
pool.Unregister("session-2")
close(session1)
close(session2)
```

**Cite:** No C counterpart — Go example code.

**Verification:** `go build ./examples/...`

---

## Build Gate

```bash
go build ./...
go vet ./...
go test ./...
```

All three MUST pass before committing. Commit as:
```
fix: reek round 6 — telnet write deadline, audit init leak, CORS dev guard, Makefile path, example channel order (DP-804, DP-799, DP-798, DP-829, DP-794)
```
