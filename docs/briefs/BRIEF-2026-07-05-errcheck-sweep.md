# Brief: Error-Handling Sweep — 2026-07-05

**Workspace:** `/Users/zach/.openclaw/workspace/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-720 — parseDuration accepts invalid suffix strings (MEDIUM)

**File:** `pkg/command/admin_commands.go` — `parseDuration()` (line ~793)

**Problem:**
`parseDuration` strips a trailing `"d"` suffix, then passes the remainder to `fmt.Sscanf(s, "%d", &days)`. `fmt.Sscanf` with `"%d"` matches only a leading prefix — it silently ignores trailing non-numeric characters. This means:

- Input `"5days"` → strips `"d"`, passes `"5day"` to Sscanf → matches `"5"`, returns 5 days
- Input `"5dd"` → strips `"d"`, passes `"5d"`, matches `"5"`, returns 5 days
- Input `"5df"` → strips `"d"`, passes `"5f"`, matches `"5"`, returns 5 days

An admin who fat-fingers a duration for `cmdMute` or `cmdBan` (e.g., `"5df"` for "5 days") gets the intended duration instead of an error. The typo is silently masked.

**Fix:**
After `fmt.Sscanf`, verify that the entire input was consumed by checking the number of items scanned AND that no trailing characters remain. The simplest approach:

```go
func parseDuration(s string) (time.Duration, error) {
	if strings.HasSuffix(s, "d") {
		var days int
		remaining := s[:len(s)-1]
		n, err := fmt.Sscanf(remaining, "%d", &days)
		if err == nil && n == 1 && fmt.Sprintf("%d", days) == remaining {
			return time.Duration(days) * 24 * time.Hour, nil
		}
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q: %w", s, err)
		}
		return 0, fmt.Errorf("invalid duration %q: trailing characters in day specification", s)
	}
	return time.ParseDuration(s)
}
```

Key change: `fmt.Sprintf("%d", days) == remaining` ensures the entire string was numeric — no trailing garbage.

**Cite:** No C equivalent. C's punishment commands are permanent toggles with no duration parsing — `act.wizard.c:2131` (squelch/mute), `act.wizard.c:2139` (freeze), `ban.c:132` (ban). `parseDuration` is a Go-only addition. This is a pure G104 errcheck fix, not a fidelity concern.

**Regression Test:** `pkg/command/admin_commands_test.go`
- Add `TestParseDuration_ValidDays` — assert `parseDuration("5d")` returns 120 hours
- Add `TestParseDuration_InvalidTrailing` — assert `parseDuration("5days")` returns error, `parseDuration("5dd")` returns error, `parseDuration("5df")` returns error
- Add `TestParseDuration_TimeUnits` — assert `parseDuration("2h30m")` returns 2h30m (delegates to time.ParseDuration)
- Add `TestParseDuration_Empty` — assert `parseDuration("")` returns error

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-728 — io.ReadAll error silently discarded in privacy middleware (MEDIUM)

**File:** `pkg/privacy/middleware.go` — `HTTPMiddleware()` (line ~61)

**Problem:**
Line 61 reads the request body with:
```go
bodyBytes, _ := io.ReadAll(r.Body)
```

The error return is discarded. If the body read fails (premature connection close, I/O timeout, transport error), `bodyBytes` will be nil or incomplete. The middleware then replaces `r.Body` with an `io.NopCloser` wrapping whatever was read, so downstream handlers receive a truncated or empty body with no indication of failure.

**Fix:**
Handle the error — log it and either pass through an empty body or let the handler see the error:

```go
bodyBytes, err := io.ReadAll(r.Body)
if err != nil {
    slog.Warn("failed to read request body in privacy middleware",
        "error", err,
        "path", r.URL.Path,
        "method", r.Method,
    )
    // Continue with empty body — the handler will see an empty request
    bodyBytes = nil
}
```

At minimum, log the error so operators can see body-read failures in the server log. Do NOT panic or return an error response — this is a privacy filtering middleware and should be transparent.

**Cite:** No C equivalent. The privacy middleware is a Go-only addition — the C MUD is telnet-only (`comm.c` process_input) with no HTTP stack, no PII filtering concept. This is a pure G104 errcheck fix, not a fidelity concern.

**Regression Test:** `pkg/privacy/middleware_test.go`
- Add `TestHTTPMiddleware_BodyReadFailure` — create a request with a body that returns an error on Read, verify the middleware logs a warning and the downstream handler sees an empty body (no panic)

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. Fix DP-720 first (parseDuration) — self-contained in admin_commands.go, easy to test
2. Fix DP-728 second (io.ReadAll) — different package, independent

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace/darkpawns_repo
git add -A
git commit -m "fix: parseDuration garbage-suffix rejection + privacy middleware body-read error handling (DP-720, DP-728)"
git push -u origin fix/errcheck-sweep
gh pr create --title "fix: errcheck sweep — parseDuration + privacy middleware (DP-720, DP-728)" --body "Fixes DP-720, DP-728. See docs/briefs/BRIEF-2026-07-05-errcheck-sweep.md for details."
```

Then wait for review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-720: Add comment "Fixed — parseDuration now rejects trailing non-numeric characters after day suffix", commit <hash>, move to Done
- DP-728: Add comment "Fixed — io.ReadAll error now logged via slog.Warn in HTTPMiddleware", commit <hash>, move to Done
