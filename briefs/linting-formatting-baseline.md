# Brief: Linting & Formatting Baseline

**Date:** 2026-05-27
**Requested by:** The Architect
**Execute via:** Claude Code or Gemini (Antigravity)

## Goal

Establish a clean Go quality baseline for the Dark Pawns codebase. After the fidelity audit, this is the "stop the bleeding" pass — formatting, linting, dead code removal, error handling. One commit. No behavior changes.

## Pre-Flight

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
go build ./... && go vet ./... && go test ./...
```

All three MUST pass before you start. If they don't, stop and report.

## Step 1: Install gofumpt

```bash
go install mvdan.cc/gofumpt@latest
```

Binary lands at `~/go/bin/gofumpt`. Add `~/go/bin` to PATH if needed.

## Step 2: Create `.golangci.yml`

Create this file at the repo root. This is a conservative config — only linters that catch real bugs or enforce basics. No style police.

```yaml
run:
  timeout: 5m
  skip-dirs:
    - admin-ui
    - node_modules
    - website

linters:
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused
    - gosimple
    - gocritic

linters-settings:
  errcheck:
    check-type-assertions: false
    check-blank: false
    exclude-functions:
      - (*os.File).Close
      - (io.Closer).Close
      - (net.Conn).Close
  gocritic:
    enabled-tags:
      - diagnostic
      - performance
    disabled-tags:
      - style
      - experimental
      - opinionated

issues:
  exclude-rules:
    # Test files get more lenient error handling
    - path: _test\.go
      linters:
        - errcheck
    # Third-party code
    - path: admin-ui/
      linters:
        - errcheck
        - govet
        - unused
    - path: node_modules/
      linters:
        - errcheck
        - govet
        - unused
  max-issues-per-linter: 0
  max-same-issues: 0
```

**Why these linters:**
- `errcheck` — unchecked errors are the #1 source of silent bugs in Go
- `govet` — catches suspicious constructs (shadowed variables, printf args)
- `ineffassign` — dead stores that indicate logic errors
- `unused` — dead code that confuses readers
- `staticcheck` — catches deprecated usage, unnecessary code, simplification opportunities
- `gosimple` — suggests simpler equivalent expressions
- `gocritic` — catches common mistakes (diagnostic + performance tags only, not style)

**Why NOT these linters (yet):**
- `wrapcheck` — requires error wrapping everywhere, too noisy for a ported codebase
- `exhaustruct` — requires all struct fields be set, incompatible with many patterns
- `errorlint` — requires specific error comparison patterns, would need massive refactoring
- `stylecheck` / `revive` — style enforcement, Boy Scout rule handles this going forward

## Step 3: Run gofumpt (one-time formatting pass)

```bash
gofumpt -w .
```

This will reformat 240 files. It's a no-behavior-change formatting commit. The changes are things like:
- Removing unnecessary braces around single-case switches
- Normalizing argument spacing
- Removing blank lines in short functions
- Consistent handling of grouped imports

**Do NOT review individual changes.** This is mechanical formatting. Trust the tool.

## Step 4: Fix all lint findings

Run `golangci-lint run ./...` and fix every finding. Here's the full list by category:

### 4a. errcheck — 35 findings (production code only, tests excluded by config)

The `.golangci.yml` above excludes test files and `(*os.File).Close` / `(io.Closer).Close`. After applying those exclusions, you should have ~15-20 production-code errcheck findings to fix.

For each one:
- **`defer f.Close()`** → If the close error doesn't matter (reading a file you already read from), use `defer func() { _ = f.Close() }()`. If it does matter (writing), log or propagate.
- **`json.Unmarshal`** → Check the error. In handlers, return it. In init code, log.Fatal.
- **`conn.Close()`** → `defer func() { _ = conn.Close() }()` is usually fine.
- **`d.events.Append()`** → These are fire-and-forget event logging. `_ = d.events.Append(...)` is acceptable.
- **`os.Setenv` / `os.Unsetenv`** → In test setup, these can't fail. Use `_ =`.

### 4b. unused — 12 findings (dead code)

These are functions/variables that are defined but never called. Remove them all:
- `packWeightLabel` in `cmd_info.go`
- `cmdSocial` in `cmd_social.go`
- `infobarClearHit`, `infobarClearMana`, `infobarClearMove`, `infobarClearExpPoints`, `infobarClearNeededExpPoints`, `infobarClearLevel`, `infobarClearGold` in `display_cmds.go`
- `cmdInfoBarUpdate` in `display_cmds.go`
- `sendGMCP` in `telnet/listener.go`
- `main` in `benchmarks/websocket_benchmark_test.go` — this is a benchmark file using `func main()` instead of `func Benchmark*`. Leave this one alone or convert to proper benchmark.

**Before removing:** Check if any of these are registered as command handlers or referenced via string lookup (common in MUD codebases). If `cmdSocial` is registered by name in a command map, it's not dead — the linter just can't see the string reference. Check `command_map.go` or similar.

### 4c. staticcheck — 10 findings

- **QF1012** (3 findings): `WriteString(fmt.Sprintf(...))` → `fmt.Fprintf(w, ...)` — cleaner, avoids intermediate string allocation
- **S1039** (2 findings): unnecessary `fmt.Sprintf` — just use the string directly
- **QF1003** (3 findings): if/else chain → tagged switch — cleaner, more idiomatic
- **SA1019** (2 findings): `a.Type` deprecated in `save.go` — these are in the affect serialization path. **DO NOT CHANGE** without understanding the full affect system. Add `//nolint:staticcheck` with a comment explaining why: `"SA1019: a.Type is used for backward-compatible deserialization of existing save files"`. This is a known deprecation that can't be fixed without a save file migration.

### 4d. ineffassign — 3 findings

- `cmd/test-race/main.go:95` — `className` assigned but overwritten. Fix the logic.
- `pkg/session/cmd_info.go:131` — `ratio` assigned but overwritten. Fix the logic.
- `pkg/spells/affect_spells.go:146` — `aff` assigned but overwritten. Fix the logic.

### 4e. govet — 2 findings

Both are in `admin-ui/node_modules/flatted/` — third-party code. Skip. The `skip-dirs` in `.golangci.yml` should handle this.

## Step 5: Update Makefile

Add a `lint-full` target and update `lint`:

```makefile
fmt:
	gofumpt -w .

check-fmt:
	@test -z "$(gofumpt -l .)" || (echo "Files need gofumpt. Run: gofumpt -w ." && gofumpt -l . && exit 1)

vet:
	go vet ./...

lint: check-fmt vet
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...
```

Change the existing `fmt` target from `go fmt ./...` to `gofumpt -w .` (gofumpt is a superset of go fmt).

## Step 6: Add CI checks

Add these steps to `.github/workflows/ci.yml` AFTER the existing test job, as a new job:

```yaml
  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v6
    - uses: actions/setup-go@v6
      with:
        go-version: '1.26.3'

    - name: Install golangci-lint
      uses: golangci/golangci-lint-action@v7
      with:
        version: latest

    - name: Install gofumpt
      run: go install mvdan.cc/gofumpt@latest

    - name: Check formatting
      run: test -z "$(gofumpt -l .)"

    - name: Run golangci-lint
      run: golangci-lint run ./...
```

## Step 7: Verify

After ALL changes:

```bash
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
```

ALL five must pass. If any fail, fix before committing.

## Commit

Single commit: `chore: establish linting and formatting baseline`

Include:
- `.golangci.yml` (new)
- `gofumpt -w .` formatting changes (mechanical, don't review line-by-line)
- Lint fixes (errcheck, unused, staticcheck, ineffassign)
- Updated `Makefile`
- Updated CI workflow

## What This Does NOT Do

- Does not change any game behavior
- Does not refactor for idioms (Boy Scout rule going forward)
- Does not add new tests
- Does not touch infrastructure
- Does not touch Lua scripts
- Does not touch world files

This is purely a code quality baseline pass. Clean slate for the next phase.
