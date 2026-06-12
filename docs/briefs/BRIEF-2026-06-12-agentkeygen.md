# Brief: Agentkeygen Cleanup — 2026-06-12

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-574 — Move PostgreSQL DSN off the command line (HIGH)

**File:** `cmd/agentkeygen/main.go` — `main()` (lines 18-27)

**Problem:**
The `-db` flag accepts a full PostgreSQL connection string as a CLI argument. On Linux/macOS, command-line arguments are visible to other local users via `ps aux` or `/proc/<pid>/cmdline`. This exposes credentials in plaintext.

Current code:
```go
dsn := flag.String("db", "", "postgres connection string")
```

**Fix:**
Replace the `-db` flag with an environment variable lookup. Use `DATABASE_URL` as the primary source.

```go
// Replace flag lookup with:
dsn := os.Getenv("DATABASE_URL")
if dsn == "" {
    fmt.Fprintln(os.Stderr, "error: DATABASE_URL environment variable is required")
    fmt.Fprintln(os.Stderr, "example: export DATABASE_URL='postgres://user:pass@localhost/darkpawns'")
    os.Exit(1)
}
```

Keep `-name` as a flag since the character name is not sensitive.

Also update the usage comment at the top of the file from:
```go
//   go run ./cmd/agentkeygen -name "brenda69" -db "postgres://..."
```
to:
```go
//   DATABASE_URL="postgres://..." go run ./cmd/agentkeygen -name "brenda69"
```

**Regression Test:** `cmd/agentkeygen/main_test.go`
- Add `TestMainRequiresDatabaseURL`: run the binary with only `-name test` set, no `DATABASE_URL` in env, assert exit code is non-zero and stderr contains `DATABASE_URL environment variable is required`
- Add `TestMainReadsDatabaseURL`: set `DATABASE_URL=postgres://test:test@localhost/test` via `TestMain` env helper, confirm the binary attempts connection (will fail with connection refused, which is acceptable — the point is it didn't reject the input)

**Cite:** No C equivalent — this is a Go-specific CLI tool.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-580 — Connection leak on os.Exit (MEDIUM)

**File:** `cmd/agentkeygen/main.go` — `main()` (lines 39-55)

**Problem:**
`db.New()` opens a PostgreSQL connection. Three `os.Exit(1)` calls follow without ever calling `database.Close()`. Since `os.Exit` skips deferred functions, even adding `defer database.Close()` wouldn't help. Repeated quick invocations could exhaust connection slots.

**Fix:**
Restructure into `run(name, dsn string) error` for lifecycle management and `runWithDB(name string, database db.Database) error` for business logic. This keeps connection setup/teardown in one place and makes the business logic unit-testable with a mocked `db.Database`.

```go
func main() {
    name := flag.String("name", "", "character name to associate the key with")
    flag.Parse()

    if *name == "" {
        fmt.Fprintln(os.Stderr, "error: -name is required")
        flag.Usage()
        os.Exit(1)
    }

    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        fmt.Fprintln(os.Stderr, "error: DATABASE_URL environment variable is required")
        fmt.Fprintln(os.Stderr, "example: export DATABASE_URL='postgres://user:pass@localhost/darkpawns'")
        os.Exit(1)
    }

    if err := run(*name, dsn); err != nil {
        slog.Error("agentkeygen failed", "error", err)
        os.Exit(1)
    }
}

func run(name, dsn string) error {
    database, err := db.New(dsn)
    if err != nil {
        return fmt.Errorf("connect to database: %w", err)
    }
    defer database.Close()

    return runWithDB(name, database)
}

func runWithDB(name string, database db.Database) error {
    if _, err := database.GetPlayer(name); err != nil {
        return fmt.Errorf("get player %q: %w", name, err)
    }

    rawKey, id, err := database.CreateAgentKey(name)
    if err != nil {
        return fmt.Errorf("create agent key: %w", err)
    }

    fmt.Printf("Character: %s\n", name)
    fmt.Printf("Key (id=%d): %s\n", id, rawKey)
    fmt.Println("(shown once — store in Vaultwarden)")
    return nil
}
```

**Regression Test:** `cmd/agentkeygen/main_test.go`
- Add `TestRunReturnsErrorOnBadDSN`: pass invalid DSN, assert returned error is non-nil and contains "connect to database"
- Add `TestRunWithDBSuccess`: pass a mock `db.Database` where `GetPlayer` returns nil and `CreateAgentKey` returns a test key/id, assert no error returned
- Add `TestRunWithDBGetPlayerError`: pass a mock `db.Database` where `GetPlayer` returns an error, assert returned error contains "get player"
- Add `TestRunWithDBCreateKeyError`: pass a mock `db.Database` where `GetPlayer` succeeds but `CreateAgentKey` fails, assert returned error contains "create agent key"
- Add `TestRunWithDBClosesDatabase`: use a mock that records `Close()` calls, assert `Close()` was called exactly once after `run()` completes (success or error path)

**Cite:** No C equivalent — Go-specific CLI tooling pattern.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 3: DP-586 — Misleading error message (LOW)

**File:** `cmd/agentkeygen/main.go` — `main()` (line 47)

**Problem:**
`database.GetPlayer()` can fail for many reasons (connection drop, query timeout, etc.) but the log unconditionally says `"character not found"`. This hides the real cause and makes debugging impossible from tool output alone.

Current code:
```go
if _, err := database.GetPlayer(*name); err != nil {
    slog.Error("character not found", "name", *name)
    os.Exit(1)
}
```

**Fix:**
Include the actual error. This is handled automatically by Fix 2 (the `run()`/`runWithDB()` refactor), which wraps errors with context:

```go
return fmt.Errorf("get player %q: %w", name, err)
```

If not using the refactor, the minimal fix is:
```go
slog.Error("get player", "name", *name, "error", err)
```

**Regression Test:** Covered by `TestRunWithDBGetPlayerError` — the error message now includes the real cause.

**Cite:** No C equivalent.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

All three fixes are in the same file and should be applied together:

1. **Fix 2** (DP-580) — restructure into `run()`/`runWithDB()` with proper cleanup
2. **Fix 1** (DP-574) — replace `-db` flag with `DATABASE_URL` env var
3. **Fix 3** (DP-586) — error message fix falls out of the refactor naturally

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
go build ./... && go vet ./... && go test ./...
git add -A
git commit -m "fix: agentkeygen — move DSN to env var, fix connection leak, improve error messages (DP-574, DP-580, DP-586)"
git push -u origin fix/agentkeygen-cleanup-2026-06-12
gh pr create --title "fix: agentkeygen cleanup (DP-574, DP-580, DP-586)" --body "Fixes DP-574, DP-580, DP-586. See docs/briefs/BRIEF-2026-06-12-agentkeygen.md for details."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-574: Add comment "Fixed — DSN moved to DATABASE_URL env var, no longer visible in ps output", commit <hash>, move to Done
- DP-580: Add comment "Fixed — restructured into run()/runWithDB() with defer database.Close(), connection always cleaned up", commit <hash>, move to Done
- DP-586: Add comment "Fixed — error message now includes actual error from GetPlayer()", commit <hash>, move to Done
