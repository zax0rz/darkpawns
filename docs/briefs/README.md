# Briefs — Coding Agent Task System

## Workflow

```
Daeron writes brief → Coding agent branches & implements → Daeron reviews PR → Merge → Linear update
```

### 1. Daeron writes the brief

Create `docs/briefs/BRIEF-YYYY-MM-DD-<slug>.md` following the template below. Each brief should be:
- Self-contained (coding agent has everything it needs)
- Scoped to 1-5 related issues max
- Verified against actual code (read the files, confirm line numbers, confirm the bug)
- Include regression tests where they make sense

### 2. Coding agent implements

Hand the brief to Claude Code (or Gemini, etc.). The agent:
- Creates a branch from `main`
- Implements the fixes
- Runs build gates: `go build ./... && go vet ./... && go test ./...`
- Pushes the branch

### 3. Daeron reviews

Read the PR diff against the brief. Check:
- Does the fix match the brief's description?
- Are lock safety / concurrency concerns addressed?
- Did the agent introduce side effects?
- Are regression tests included where the brief requested them?
- Do all build gates pass?

### 4. Merge & update Linear

After merge:
- Add commit hash comment to each Linear issue
- Move issues to Done

## Brief Template

```markdown
# Brief: <Title> — YYYY-MM-DD

**Workspace:** `/Users/zach/.openclaw/workspace-daeron/darkpawns_repo`
**Repo:** `git@github-darkpawns:zax0rz/darkpawns.git` (branch: `main`)
**Build gate:** `go build ./... && go vet ./... && go test ./...` — ALL THREE MUST PASS.

---

## Fix N: DP-XXX — <Short Title> (<Severity>)

**File:** `<path/to/file.go>` — `<FunctionName>()` (line ~N)

**Problem:**
<What's broken. Be specific — cite line numbers, explain the code path.>

**Fix:**
<Exact code change needed. Show before/after when helpful.>

**Cite:** C source — `<file>:<line>` (function name). <How the Go port differs from C, if applicable.> If no C equivalent exists (Go-only addition), say so explicitly — the coding agent shouldn't waste time hunting for one.

**Regression Test:**
<What test should exist, specific assertions, whether one exists already or needs writing.>
If the bug is hard to unit test, explain why and what integration test would cover it.>

**Verification:** `go build ./... && go vet ./... && go test ./...`

---

## Execution Order

<List fixes in dependency order. Smallest/safest first.>

## After All Fixes

```bash
cd /Users/zach/.openclaw/workspace-daeron/darkpawns_repo
git add -A
git commit -m "fix: <description> (DP-XXX, DP-YYY)"
git push -u origin fix/<branch-name>
gh pr create --title "fix: <description> (DP-XXX, DP-YYY)" --body "Fixes DP-XXX, DP-YYY. See docs/briefs/BRIEF-YYYY-MM-DD-<slug>.md for details."
```

Then wait for Daeron to review and merge. Do NOT merge the PR yourself.

## Linear Updates (after merge)

- DP-XXX: Add comment "Fixed — <what changed>", commit <hash>, move to Done
- DP-YYY: Add comment "Fixed — <what changed>", commit <hash>, move to Done
```

## When to Include Regression Tests

**Always include when:**
- The bug is a concurrency issue (race condition, deadlock) — write a test that exercises concurrent access
- The bug is a silent correctness issue (like DP-527 — affect cleanup that silently does nothing)
- The bug affects a data path (player stats, gold, items)
- The fix changes control flow (if/else → switch, new early return)

**Skip when:**
- The fix is a dependency upgrade (Go version, library bump)
- The fix is a one-liner config change
- The fix removes dead code
- Writing a meaningful test would require mocking the entire game engine

**Format for test section:**
```markdown
**Regression Test:** `pkg/engine/affect_test.go`
- Add `TestTickExpiredAffectCleanup`: create entity, apply poison affect, mock tick to expire it, assert stat reversal fired and wear-off message sent
- Add `TestTickExpiredAffectFlagClearing`: apply affect with AFF_* flag, expire it, assert ClearStatusFlag called
```

## Naming Convention

- `BRIEF-YYYY-MM-DD-<slug>.md` for time-boxed batches
- `BRIEF-<CATEGORY>-<NNN>-<slug>.md` for standalone work
  - Categories: SECURITY, INFRA, CLEANUP, CODE, FEATURE
