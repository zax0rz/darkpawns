# BRIEF — COV-5: e2e smoke test in CI — login/look/move/kill/save/quit (DP-966)

**Linear:** DP-966 (COV-5: e2e smoke test in CI — login/look/move/kill/save/quit)
**Effort:** M
**Agent:** Kimi
**Source of truth:** docs/reports/REVIEW-2026-07-05-full-audit.md — §3C item 5

## Goal

Wire an end-to-end smoke test into CI that boots the real server binary, connects over a real transport, logs in, moves, kills a mob, saves, reconnects, and verifies persistence. This closes the "tests green, game broken" gap permanently.

## Why this matters

All current CI tests are unit tests (or telnet e2e that deliberately use a dead database). Every unit test can pass while the actual game is unplayable — that exact scenario happened twice before (DP-589 nil DB panic, DP-590 mob-AI deadlock). The only test that catches those is launching the real binary.

The existing Go telnet e2e tests (`tests/e2e/telnet_smoke_test.go`) cover login, movement, combat, skills, and spells — but deliberately use a dead DB URL (`postgres://x:x@127.0.0.1:1/nope`) so persistence is untested in CI. A persistence round-trip test exists (`TestTelnetSmoke_PersistenceRoundTrip`) but is gated on `DP_TEST_DB_URL` and CI doesn't set it.

## Two complementary approaches

### Option A (Go-native, preferred): Enable the existing persistence test in CI

The infrastructure is already built. `TestTelnetSmoke_PersistenceRoundTrip` (at `tests/e2e/telnet_smoke_test.go:286`) does exactly what we need:
1. Creates a new character via telnet (full character creation flow)
2. Quits (persists to DB)
3. Reconnects with wrong password → asserts rejection (DP-591)
4. Reconnects with correct password → asserts character loads back

It just needs CI to set `DP_TEST_DB_URL` pointing at the existing Postgres service container. One CI workflow edit.

### Option B (Python, as specified in Linear): Wire `scripts/smoke_test_2b.py` into CI

The Python script (`scripts/smoke_test_2b.py`, 230 lines) does a WebSocket-based smoke test:
1. Creates warrior via WebSocket login
2. Walks rooms, finds a mob
3. Kills the mob, loots corpse, wields weapon
4. Quits, reconnects, verifies equipment persisted

This exercises a different transport path (WebSocket JSON vs raw telnet) which is valuable — but requires more CI wiring (Python websockets dependency, server startup, port coordination).

### Recommendation: Do both

Option A is a one-line CI change. Option B adds WebSocket transport coverage. Together they validate that both transport paths work end-to-end with persistence.

## Current CI state

**File:** `.github/workflows/ci.yml`

The `test` job already runs:
1. `go test -race $(go list ./... | grep -v /tests/unit) -v -timeout 120s` — this already runs the telnet e2e tests (they're in `tests/e2e/`, not excluded)
2. A weak server startup check: `timeout 10s ./server -port 9999 & sleep 3; curl /health` — no `-world`, no `-db`, no telnet, just HTTP health

The `test` job has a **PostgreSQL 16 service** already running (port 5432, user/pass/db all `postgres`). This is unused by the Go tests but is exactly what the persistence test needs.

**What's missing:** No `DP_TEST_DB_URL` env var is set, so `TestTelnetSmoke_PersistenceRoundTrip` skips. No Python e2e runs.

## Existing test infrastructure

### Go telnet e2e (`tests/e2e/telnet_smoke_test.go`, 700 lines)

Six tests, all run in CI (not gated by `-short`):

| Test | What it exercises |
|------|-------------------|
| `TestTelnetSmoke_GuestEntersWorld` | Guest login, MOTD, room vnum/exits, east/west movement |
| `TestTelnetSmoke_CharacterCreation` | Full char creation state machine |
| `TestTelnetSmoke_Combat` | Create warrior, walk to Temple Square, `hit` NPC, combat rounds |
| `TestTelnetSmoke_SkillKick` | `kick` skill on NPC |
| `TestTelnetSmoke_SpellCasting` | `infravision`, `magic missile`, `fireball` level gate |
| `TestTelnetSmoke_PersistenceRoundTrip` | **SKIPPED in CI** — needs `DP_TEST_DB_URL` |

Helper functions:
- `launchAndDial(t)` — builds binary, starts server with dead DB, dials telnet
- `launchAndDialDB(t, dbURL)` — same but with a real DB URL
- `createWarrior(t, conn, r, name, pw)` — drives full character creation
- `createChar(t, conn, r, name, pw, class)` — creates character of any class
- `deleteTestPlayer(t, dbURL, name)` — cleanup
- `readUntil(t, conn, r, marker, timeout)` — reads telnet output until marker found
- `mustWrite(t, conn, data)` — writes to telnet connection
- `serverBinary(t, root)` — caches `go build -o $tmp/dp-server ./cmd/server` in TestMain

### Python smoke test (`scripts/smoke_test_2b.py`, 230 lines)

WebSocket-based, uses `websockets` library. Connects to `ws://localhost:4350/ws` by default.

Protocol: sends JSON `{"type": "command", "data": {"command": "look", "args": []}}`, receives JSON `{"type": "text", "data": {"text": "..."}}`.

Flow: create warrior → walk rooms → find mob → kill → loot → wield → quit → reconnect → verify equipment persisted.

Note: This script expects the server's WebSocket endpoint (`/ws`) which is the HTTP server path, not telnet. It needs the HTTP port, not the telnet port.

## Fix

### Part 1: Enable Go persistence test in CI (Option A)

Edit `.github/workflows/ci.yml` — in the `test` job, set `DP_TEST_DB_URL` env var pointing at the existing Postgres service container:

```yaml
  test:
    runs-on: ubuntu-latest
    env:
      DP_TEST_DB_URL: "postgres://postgres:postgres@localhost:5432/darkpawns?sslmode=disable"
    services:
      postgres:
        # ... existing config unchanged ...
```

The `TestTelnetSmoke_PersistenceRoundTrip` test will now run. It:
- Creates a unique-named character (`Rt<timestamp>`)
- Cleans up after itself via `deleteTestPlayer`
- Tests: create → quit → wrong password → correct password → load

**IMPORTANT:** The persistence test calls `launchAndDialDB` which starts a fresh server instance with the real DB. This is separate from the other e2e tests that use `launchAndDial` (dead DB). Multiple server instances may run concurrently in the test — each gets its own ports. This is fine.

**Risk:** If the `players` table doesn't exist in the CI database, `sql.Open` will succeed but queries will fail. The server creates tables on boot via `db.New()`. Verify the server's migration/bootstrap code creates the `players` table.

### Part 2: Wire Python smoke test into CI (Option B)

Add a new step in the `test` job after "Build Go binary":

```yaml
    - name: Start server for Python smoke test
      run: |
        ./server -world lib/world -port 4350 -telnet-port 0 -db "${DP_TEST_DB_URL}" &
        SERVER_PID=$!
        # Wait for server to be ready
        for i in $(seq 1 30); do
          if curl -sf http://localhost:4350/health > /dev/null 2>&1; then
            break
          fi
          sleep 1
        done
        echo "SERVER_PID=$SERVER_PID" >> $GITHUB_ENV

    - name: Run Python e2e smoke test
      run: |
        pip install websockets
        python3 scripts/smoke_test_2b.py --ws-url ws://localhost:4350/ws

    - name: Stop server
      if: always()
      run: kill $SERVER_PID 2>/dev/null || true
```

Key details:
- `-telnet-port 0` disables telnet (we only need HTTP/WebSocket)
- `-port 4350` matches the smoke test's default URL
- `DP_TEST_DB_URL` is already set from Part 1
- `JWT_SECRET` — the server generates an ephemeral one in development mode if not set, so this should work. But to be safe, add `JWT_SECRET=ci-smoke-test-secret-at-least-32-chars-long` to the env.

**Risk:** The Python smoke test is more fragile than the Go tests because it does actual combat (kill a mob, loot corpse). The world layout must have a reachable mob from the starting room. The test walks north/south/east/west looking for a mob — if no mob is reachable, it will fail with "Found a mob: ✗".

### Part 3 (optional but recommended): Make the Go persistence test use CI's Postgres directly

Currently `TestTelnetSmoke_PersistenceRoundTrip` starts its own server with the DB URL. If the CI Postgres is available, this works. But verify:

1. The server's `db.New()` connects successfully to `postgres://postgres:postgres@localhost:5432/darkpawns`
2. The `players` table gets created (auto-migration on boot)
3. The test's cleanup (`DELETE FROM players WHERE name = $1`) works

If any of these fail, the test will fail with a clear error, which is exactly the point.

## Files

| File | Change |
|---|---|
| `.github/workflows/ci.yml` | Add `DP_TEST_DB_URL` env var + Python smoke test step |

That's it — no production code changes.

## Build gate

After editing the workflow, verify locally that the Go persistence test works with a local Postgres:

```bash
# If you have Postgres running locally:
DP_TEST_DB_URL="postgres://postgres:postgres@localhost:5432/darkpawns?sslmode=disable" \
  go test -race ./tests/e2e/ -run TestTelnetSmoke_PersistenceRoundTrip -v -timeout 120s
```

Then run the standard gate:
```bash
go build ./...
go vet ./...
go test -race $(go list ./... | grep -v /tests/unit) -timeout 120s
gofumpt -l .
golangci-lint run ./...
```

## Constraints

1. **Do NOT modify any Go source code.** This is CI wiring only.
2. **Do NOT modify `scripts/smoke_test_2b.py`** unless absolutely necessary for CI compatibility (e.g., URL parameterization).
3. **The Python smoke test must not be the gate.** If it flakes (mob not found, timing issue), it should not block PRs. Consider running it as a separate CI job or adding `continue-on-error: true` until it's stable.
4. **The Go persistence test IS the gate.** It's deterministic, uses the same binary, and tests a critical path (save/load/wrong-password).
5. **Preserve existing test behavior.** The six existing telnet e2e tests use dead DB and must continue to pass unchanged.
6. Single PR. Commit message: `ci: wire e2e persistence smoke test + Python WebSocket smoke into CI (COV-5)`

## What success looks like

After this PR merges, every CI run on `main` will:
1. Run all existing unit tests (unchanged)
2. Run 6 telnet e2e tests (unchanged)
3. **NEW:** Run `TestTelnetSmoke_PersistenceRoundTrip` against a real Postgres — verifies character create → save → wrong password → correct password → load
4. **NEW:** Run `scripts/smoke_test_2b.py` — WebSocket login → walk → kill → loot → equip → reconnect → verify persistence

If any of these fail, the PR is blocked. The "tests green, game broken" gap is closed.

## Documentation value

The CI workflow itself becomes the contract: "a PR must pass login, movement, combat, skills, spells, character creation, persistence, AND wrong-password rejection — over both telnet and WebSocket transports."
