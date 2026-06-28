# Clawpatch fix brief — Security hardening batch (INFRA)

**Date:** 2026-06-26
**Target:** Kimi K2.7-code
**Repo:** `/Users/zach/darkpawns` (on `main` after tonight's deploy)
**Branch:** `fix/security-hardening-20260627` (create from main)

## Ground rules

Same as previous batch:
- Work one finding at a time
- Build-gate every fix: `go build ./... && go vet ./... && go test ./pkg/...`
- One commit per finding: `fix(security): <short title>` + body with Linear issue ID
- Add regression tests where practical
- DO NOT edit `src/` or `.clawpatch/`

## Findings

### DP-591 [HIGH] Hardcoded Postgres default credentials
**File:** `cmd/server/main.go:69`
**Fix:** Remove the hardcoded `-db` default (`postgres://postgres:postgres@localhost/darkpawns?sslmode=disable`). Make `-db` required — if not provided, check `DATABASE_URL` env var. If neither is set, log a fatal error and exit. The server should never silently connect with well-known creds.
**Caveat:** The systemd unit on CT 120 already passes `-db` explicitly, so this won't break production. But it prevents accidental credential exposure on fresh installs.

### DP-596 [HIGH] WebSocket Upgrader Bypass in Development
**File:** `pkg/session/manager.go:44`
**Fix:** Remove the `ENVIRONMENT=development` bypass in `CheckOrigin`. Always use the explicit origin allowlist. If development needs relaxed CORS, that's a separate configuration mechanism (env var with explicit origins), not a blanket `return true`.
**Caveat:** None — the k8s manifest already sets `ENVIRONMENT=production`.

### DP-557 [HIGH] No DNS hostname resolution for ban checks
**File:** `pkg/telnet/listener.go:109`
**Fix:** After extracting `remoteIP`, resolve it to a hostname via `net.LookupAddr()`. Check both the raw IP and the resolved hostname against the ban list. Add timeout (2-3 seconds max) so a slow DNS server doesn't block connections.
**Caveat:** C's `gethostbyaddr()` was synchronous with no timeout. Adding a timeout is a behavioral improvement, not a regression. Document this.

### DP-595 [MEDIUM] pprof Server Exposes Runtime Internals
**File:** `profiling/profiler.go`
**Fix:** Bind pprof to `127.0.0.1:6060` instead of `:6060`. Add an env var `PPROF_BIND_ADDR` to override (for remote debugging on LAN). Default must be localhost-only.
**Caveat:** This breaks remote pprof access unless `PPROF_BIND_ADDR` is set. Document in the profiling README.

### DP-592 [MEDIUM] Login Rate Limit IP-Only, No Account Lockout
**File:** `pkg/auth/ratelimit.go`
**Fix:** Add account-level tracking. After N failed attempts (suggest 10) for a given username within T minutes (suggest 15), lock that account for T minutes. Return "account locked" instead of "invalid credentials" so the user knows what happened. Store lockout state in the database (not just in-memory) so it survives restarts.
**Caveat:** This interacts with the `darkpawns` database role permissions we just fixed. Verify the `players` table is writable by the `darkpawns` role before implementing.

### DP-550 [MEDIUM] k8s deployment missing JWT_SECRET and ENCRYPTION_KEY
**File:** `k8s/server.yaml`
**Fix:** Add `JWT_SECRET` and `ENCRYPTION_KEY` as Kubernetes Secret references in the deployment spec. Create a template comment showing how to generate them:
```
kubectl create secret generic dp-secrets --from-literal=JWT_SECRET=$(openssl rand -hex 32) --from-literal=ENCRYPTION_KEY=$(openssl rand -hex 32)
```
**Caveat:** This is a k8s manifest change — verify it doesn't conflict with whatever injection mechanism is currently in place (if any).

## When done
- Run full `go build ./... && go vet ./... && go test ./...`
- Summarize: which findings committed, which skipped (and why)
- Leave the branch for Zach to review; do not merge to `main` or push
