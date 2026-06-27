# BRIEF: Validation Cleanup & File Permissions

**Date:** 2026-06-27
**Source:** Reek/Clawpatch findings batch
**Issues:** DP-570, DP-569, DP-597, DP-573 + file permissions sweep
**Assignee:** Reek
**Estimated scope:** 12 changes across 7 files, single commit

---

## 0. File Permissions Sweep — `0o644` → `0o600` (LOW)

**Source:** Expanded audit after DP-597

16 write sites across the codebase use `0o644` (world-readable). The agent_store (DP-597) is already covered above. These are the remaining sites that should be `0o600`:

### Sensitive — must fix

| File | Line | What it writes |
|------|------|----------------|
| `pkg/agentcli/state.go` | 96 | Agent character state (session data, vitals, room) |
| `pkg/agentcli/events.go` | 126 | Agent event buffer (append-only log of everything the agent observed) |
| `pkg/agentcli/client.go` | 392 | Agent client config/credentials |
| `pkg/game/clans.go` | 160 | Clan data (player organization metadata) |
| `pkg/dreaming/dream.go` | 137 | Dream graph (narrative memory graph output) |
| `pkg/dreaming/dream.go` | 143 | Dream summary (narrative memory summary) |
| `pkg/dreaming/dream.go` | 150 | Dream result (narrative memory result JSON) |

### Probably fine as `0o644` — no change needed

| File | Line | What it writes |
|------|------|----------------|
| `cmd/dp-goat/internal/cache/cache.go` | 48 | Local API cache (ephemeral) |
| `cmd/dp-goat/internal/client/client.go` | 193 | API response cache |
| `pkg/game/other_settings.go` | 155 | Game settings file |
| `profiling/profiler.go` | 57, 114, 155, 199, 237 | Profile output files (diagnostic, not secrets) |

### Fix

Change `0o644` → `0o600` for all 7 sensitive sites listed above. Each is a single-character change (`6` → `0` in the third octal digit).

**Note:** `profiling/profiler.go` also has `0o644` on 5 sites — leave those alone. Profile files are diagnostic output, not secrets. The dp-goat cache files are also fine as-is (ephemeral local cache, no credentials).

---

## 1. DP-570 — SanitizePlayerName contract mismatch (MEDIUM)

**File:** `pkg/validation/validation.go:43-59`

`SanitizePlayerName` strips invalid characters and truncates to `maxPlayerNameLength` (32), but does **not** enforce `minPlayerNameLength` (2). Input `"a!"` sanitizes to `"a"` (1 rune), which fails `IsValidPlayerName`. The two functions don't compose — sanitize-then-validate produces false rejections.

**Fix:** Add a minimum length check to `SanitizePlayerName`. After stripping invalid chars, if the result is shorter than `minPlayerNameLength`, return empty string (let the caller handle it). This guarantees `SanitizePlayerName` output either passes `IsValidPlayerName` or is empty.

```go
// In SanitizePlayerName, after the existing truncation block:
if utf8.RuneCountInString(sanitized) < minPlayerNameLength {
    return ""
}
```

**Test:** Add test cases for `"a!"` → `""`, `"!b"` → `"b"`, `"!"` → `""`, `"ab"` → `"ab"`.

---

## 2. DP-569 — SanitizeInput truncates after HTML escaping (MEDIUM)

**File:** `pkg/validation/input.go:75-96`

`SanitizeInput` escapes HTML entities first, then truncates to 1000 runes. Each `&` expands to `&amp;` (5 runes), `<` to `&lt;` (4 runes). This means:
1. Effective input limit is much lower than 1000 runes for strings with escapable chars
2. Truncation can split an HTML entity mid-sequence (e.g., `&amp;` → `&am`), producing invalid HTML

**Fix:** Reorder — truncate to 1000 runes on the raw input, *then* escape HTML. Move the length check above the HTML escaping block.

```go
func SanitizeInput(input string) string {
	// Remove control characters
	input = strings.Map(func(r rune) rune {
		if r < 32 && r != 9 && r != 10 && r != 13 {
			return -1
		}
		return r
	}, input)

	// Truncate BEFORE escaping (DP-569)
	if utf8.RuneCountInString(input) > 1000 {
		input = string([]rune(input)[:1000])
	}

	// Escape HTML
	input = strings.ReplaceAll(input, "&", "&amp;")
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#39;")

	return input
}
```

**Test:** Add test with 999 `&` chars + 2 normal chars → should produce 999 `&amp;` + 2 chars (not truncated mid-entity). Also test that a 1001-char input with no special chars gets truncated to 1000.

---

## 3. DP-597 — Agent store file permissions (LOW)

**File:** `pkg/admin/agent_store.go:96,142,304`

Three issues:

### 3a. World-readable file writes (lines 142, 304)

`os.WriteFile(tmp, data, 0o644)` — world-readable. Contains agent keys and findings metadata.

**Fix:** Change both instances to `0o600`.

### 3b. Silent MkdirAll error (line 96)

`_ = os.MkdirAll(dir, 0o755)` — discards the error. Subsequent file operations fail with confusing root cause.

**Fix:** Log the error:

```go
if err := os.MkdirAll(dir, 0o755); err != nil {
	slog.Error("failed to create agent store directory", "dir", dir, "error", err)
}
```

---

## 4. DP-573 — Validation package cleanup (LOW)

**Files:** `pkg/validation/input.go`, `pkg/validation/validation_test.go`, `Makefile`

### 4a. Deprecated comment on active function (input.go:40)

`ValidateInput` is marked `// Deprecated: unused` but is actively called by `ValidateCommand` (line 89). Remove the deprecation comment.

### 4b. Missing test coverage

Zero tests for `ValidateCommand`, `SanitizePlayerName`, and `SanitizeInput`. Add:

- `TestSanitizePlayerName` — valid names, invalid chars, too short, too long, empty input
- `TestSanitizeInput` — normal text, HTML entities, control chars, length limit, empty input
- `TestValidateCommand` — valid command, XSS in command, XSS in args, empty command

### 4c. Makefile .PHONY gaps (Makefile:1)

The first `.PHONY` line is missing these targets that exist in the Makefile:
- `monitoring-restart`
- `privacy-logs`
- `privacy-build`
- `up-with-privacy`
- `test-parse`

Add them to the existing `.PHONY` declaration on line 1.

### 4d. Stale closed-file references — FALSE POSITIVE

Reek claims `WriteHeapProfile`, `StopBlockProfile`, `StopMutexProfile` save `*os.File` to struct fields after deferred Close. **Verified incorrect.** The Profiler struct only has `cpuProfile *os.File` (line 23). The other three functions use local variables only. Reject this sub-finding.

---

## Execution Order

1. **File permissions sweep** (0 sites) — change `0o644` → `0o600` in 7 files
2. **DP-597** (agent_store) — already covered in step 0 + MkdirAll fix
3. **DP-569** (truncate order) — reorder 2 blocks in `input.go`, add test
4. **DP-570** (min length) — add 3 lines to `validation.go`, add test
5. **DP-573** (cleanup) — remove deprecation comment, add tests, fix Makefile

All changes are in `pkg/validation/`, `pkg/admin/`, `pkg/agentcli/`, `pkg/game/`, `pkg/dreaming/`, and `Makefile`. No cross-package dependencies. Single commit is fine.

## Verification

After all changes, run from repo root:

```bash
go build ./...
go vet ./...
go test ./pkg/validation/... ./pkg/admin/... ./pkg/agentcli/... ./pkg/game/... ./pkg/dreaming/...
golangci-lint run ./pkg/validation/... ./pkg/admin/... ./pkg/agentcli/... ./pkg/game/... ./pkg/dreaming/...
```

All must pass before committing.

## Linear Issues

Update each issue with a comment linking to the commit when done:
- DP-570: `[Reek] MEDIUM: SanitizePlayerName contract mismatch`
- DP-569: `[Reek] MEDIUM: SanitizeInput truncates after HTML escaping`
- DP-597: `[Reek] LOW: Agent Store File Permissions`
- DP-573: `[Clawpatch] LOW: validation package cleanup`
