# Brief: Pprof / Profiling Cleanup — 2026-06-12

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-585 — pprof server ignores SIGTERM (MEDIUM)

**File:** `profiling/profiler.go` — `main()` pprof case (line ~698)

**Problem:**
The signal handler only catches `os.Interrupt` (SIGINT). systemd, Docker, and Kubernetes send SIGTERM for graceful shutdown. When SIGTERM arrives, the default handler kills the process immediately, skipping `server.Shutdown()` entirely — no profile data flushed, no clean exit.

Current code:
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt)
<-sigChan
```

**Fix:**
Add `syscall.SIGTERM`:
```go
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
<-sigChan
```

Add `"syscall"` to the import block.

**Cite:** No C equivalent — Go-specific signal handling.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-582 — pprof shutdown has no deadline (MEDIUM)

**File:** `profiling/profiler.go` — `main()` pprof case (line ~702)

**Problem:**
`server.Shutdown(context.Background())` waits indefinitely for active connections to close. A client streaming from `/debug/pprof/profile` (30s default) or a slow TCP connection will block shutdown forever, preventing process exit.

Current code:
```go
if err := server.Shutdown(context.Background()); err != nil {
    slog.Error("pprof server shutdown error", "error", err)
}
```

**Fix:**
Use a context with timeout:
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := server.Shutdown(ctx); err != nil {
    slog.Error("pprof server shutdown error", "error", err)
}
```

**Cite:** No C equivalent — Go HTTP server pattern.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 3: DP-583 — pprof logs ErrServerClosed as error (LOW)

**File:** `profiling/profiler.go` — `StartPProfServer()` goroutine (line ~583)

**Problem:**
`ListenAndServe` returns `http.ErrServerClosed` when `Shutdown()` completes. This is normal shutdown behavior, not an error. Logging it at `slog.Error` level creates false alarms.

Current code:
```go
go func() {
    slog.Info("Starting pprof server", "address", addr)
    if err := server.ListenAndServe(); err != nil {
        slog.Error("pprof server error", "error", err)
    }
}()
```

**Fix:**
Filter out the expected error:
```go
go func() {
    slog.Info("Starting pprof server", "address", addr)
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        slog.Error("pprof server error", "error", err)
    }
}()
```

**Cite:** No C equivalent.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 4: DP-584 — pprof discards file Close errors (MEDIUM)

**File:** `profiling/profiler.go` — 6 locations

**Problem:**
All `f.Close()` calls discard the error with `_ =`. `os.File.Close()` may fsync and can fail due to disk full, quota exceeded, or NFS errors. On failure, profile data is silently lost while the methods log success ("Heap profile written") and return nil.

Locations (all `_ = f.Close()` or `defer func() { _ = f.Close() }()`):
- `StopCPUProfile()` — line ~184
- `WriteHeapProfile()` — line ~206
- `StopBlockProfile()` — line ~233
- `StopMutexProfile()` — line ~270
- `GoroutineDump()` — line ~310
- `StartCPUProfile()` error path — line ~344

**Fix:**
Check and return the error in each location. For the success-path defers, capture and return via named return or error pointer. Example pattern:

```go
// For StopCPUProfile (not deferred):
if err := p.cpuProfile.Close(); err != nil {
    return fmt.Errorf("close cpu profile: %w", err)
}

// For deferred Close in WriteHeapProfile, StopBlockProfile, etc.:
var closeErr error
defer func() {
    if cerr := f.Close(); cerr != nil && closeErr == nil {
        closeErr = fmt.Errorf("close profile: %w", cerr)
    }
}()
// ... existing logic ...
return closeErr
```

**Cite:** No C equivalent — Go file I/O pattern.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## DP-567 — Already Fixed

**Note:** DP-567 (PerformanceMonitor.Stop() double-close panic) is already fixed on `main`. The code already uses `pm.once.Do(func() { close(pm.stopChan) })`. No action needed.

---

## Execution Order

1. **Fix 1** (DP-585) — SIGTERM signal handling — smallest change, adds 1 import + 1 arg
2. **Fix 2** (DP-582) — shutdown deadline — small change, context.WithTimeout
3. **Fix 3** (DP-583) — ErrServerClosed filter — 1-line change
4. **Fix 4** (DP-584) — Close error handling — most locations to touch, but each is mechanical

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
go build ./... && go vet ./... && go test ./...
gofumpt -l .  # verify no formatting issues
git add -A
git commit -m "fix: pprof cleanup — SIGTERM, shutdown deadline, ErrServerClosed, Close errors (DP-582, DP-583, DP-584, DP-585)"
git push -u origin fix/pprof-cleanup-2026-06-12
gh pr create --title "fix: pprof cleanup (DP-582, DP-583, DP-584, DP-585)" --body "Fixes DP-582, DP-583, DP-584, DP-585. See docs/briefs/BRIEF-2026-06-12-pprof-cleanup.md for details."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-582: Add comment "Fixed — shutdown now uses context.WithTimeout(10s)", commit <hash>, move to Done
- DP-583: Add comment "Fixed — ErrServerClosed filtered from error log", commit <hash>, move to Done
- DP-584: Add comment "Fixed — all Close() calls now check and return errors", commit <hash>, move to Done
- DP-585: Add comment "Fixed — signal.Notify now includes SIGTERM", commit <hash>, move to Done
- DP-567: Add comment "Already fixed on main — pm.once.Do already in place", move to Done (no commit needed)
