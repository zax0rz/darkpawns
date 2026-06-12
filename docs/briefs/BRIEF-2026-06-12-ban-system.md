# Brief: Ban System Fixes — 2026-06-12

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix 1: DP-553 — Wildcard ban matching broken (HIGH)

**File:** `pkg/game/bans.go` — `IsBanned()` (line 155)

**Problem:** The ban system uses `strings.Contains(hostname, entry.Site)` for matching. This works for exact substrings (e.g., banning `"aol.com"` matches `"user.aol.com"`). But the C code supported wildcard patterns like `"192.168.1.*"` — the Go port never implemented glob/wildcard matching. If someone bans `"192.168.1.*"`, it literally searches for the string `"192.168.1.*"` in the IP, which never matches.

The C code (`src/ban.c:isbanned()`) used `match()` which supported `*` as a wildcard for any characters. The Go port just does substring containment.

**Fix:** Add wildcard matching to `IsBanned()`. When a ban entry contains `*`, use Go's `filepath.Match()` or a simple custom glob matcher. When it doesn't contain `*`, keep the existing substring behavior.

```go
func (bm *BanManager) IsBanned(hostname string) int {
    if hostname == "" {
        return BanNot
    }
    lower := strings.ToLower(hostname)
    maxLevel := BanNot
    for _, entry := range bm.bans {
        matched := false
        if strings.Contains(entry.Site, "*") {
            // Wildcard matching — matches C's match() function
            // Use path.Match (not filepath.Match) for OS-independent behavior
            var err error
            matched, err = path.Match(entry.Site, lower)
            if err != nil {
                slog.Warn("malformed ban wildcard pattern", "pattern", entry.Site, "error", err)
            }
        } else {
            // Substring matching — existing behavior
            matched = strings.Contains(lower, entry.Site)
        }
        if matched && entry.BanType > maxLevel {
            maxLevel = entry.BanType
        }
    }
    return maxLevel
}
```

Note: add `"path"` to the import block (not `"path/filepath"`).

**Regression Test:** `pkg/game/bans_test.go`
- `TestIsBannedWildcard`: add ban `"192.168.1.*"`, assert `IsBanned("192.168.1.100") == BanAll`, assert `IsBanned("10.0.0.1") == BanNot`
- `TestIsBannedSubstring`: add ban `"aol.com"`, assert `IsBanned("user.aol.com") == BanAll` (existing behavior preserved)
- `TestIsBannedEmptyHostname`: assert `IsBanned("") == BanNot`

**Cite:** C source — `src/ban.c:isbanned()` lines 86–104. C used `match()` for glob-style wildcard matching.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Fix 2: DP-554 — ValidName missing online duplicate check (HIGH)

**File:** `pkg/game/merge_bridge.go` — `ValidName()` (line 111)

**Problem:** `ValidName()` checks the invalid name list and name length, but the `HasActiveCharacter` callback is never called. The comment at line 35 says it exists for this purpose, the session manager wires it up at `pkg/session/manager.go:209`, but `ValidName()` never uses it.

```go
// Current code — missing the HasActiveCharacter check:
func ValidName(name string) bool {
    if len(name) < 2 || len(name) > 20 {
        return false
    }
    if banManager != nil && !banManager.ValidName(name) {
        return false
    }
    return true  // ← never checks if name is already online
}
```

**Fix:** Add the `HasActiveCharacter` check:
```go
func ValidName(name string) bool {
    if len(name) < 2 || len(name) > 20 {
        return false
    }
    if banManager != nil && !banManager.ValidName(name) {
        return false
    }
    // Check if character is already online (DP-554)
    if HasActiveCharacter != nil && HasActiveCharacter(name) {
        return false
    }
    return true
}
```

Note: `HasActiveCharacter` is a function variable (not a method), so it can be nil if session manager hasn't wired it up yet. The nil check prevents panics during startup or tests.

**IMPORTANT:** The callback wired in `pkg/session/manager.go:209` uses `m.GetSession(name)` which is a case-sensitive map lookup. If player "Aidan" is online, `GetSession("aidan")` returns false — allowing a duplicate login. The callback must be case-insensitive:
```go
game.HasActiveCharacter = func(name string) bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    for sessName := range m.sessions {
        if strings.EqualFold(sessName, name) {
            return true
        }
    }
    return false
}
```

**Regression Test:** `pkg/game/merge_bridge_test.go` (or `bans_test.go`)
- `TestValidNameOnlineDuplicate`: set `HasActiveCharacter = func(name string) bool { return name == "Aidan" }`, assert `ValidName("Aidan") == false`, assert `ValidName("Other") == true`
- `TestValidNameNilCallback`: set `HasActiveCharacter = nil`, assert `ValidName("Test") == true` (no panic)

**Cite:** C source — `src/ban.c:Valid_Name()` lines 257–286. C checked the invalid list only; the online check was in the login code. Go consolidated both into `ValidName()` but forgot to wire up the callback.

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

1. **DP-553** (wildcard matching) — isolated change to `IsBanned()`
2. **DP-554** (online check) — isolated change to `ValidName()`

## After Both Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
go build ./... && go vet ./... && go test ./...
git add -A
git commit -m "fix: ban system — wildcard matching + online duplicate check (DP-553, DP-554)"
git push -u origin fix/ban-system-2026-06-12
gh pr create --title "fix: ban system (DP-553, DP-554)" --body "See docs/briefs/BRIEF-2026-06-12-ban-system.md for details."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-553: Add comment "Fixed — wildcard matching added to IsBanned() using filepath.Match", commit <hash>, move to Done
- DP-554: Add comment "Fixed — HasActiveCharacter callback now wired into ValidName()", commit <hash>, move to Done
