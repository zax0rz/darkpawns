# Briefs — Coding Agent Task System

## Core Principle: C Fidelity

**Every behavior must match the C source exactly.** The Go port is from DikuMUD/Merc 2.2 (`src/`). C files in `src/` are the single source of truth for all gameplay mechanics — combat formulas, spell tables, skill success rates, spec proc behaviors, saving throws, AC reduction, damage messages, attribute applications, class tables, etc.

- If a brief touches gameplay mechanics, it **must cite the C source** with `**Cite:**` field pointing to the exact function/line
- The coding agent reads the C source directly when given the path
- When Go deliberately diverges from C (e.g., immortal caps, float64 vs integer division), this **must be documented as an explicit deviation** in both the code and the test
- Golden tests (C→Go table transcriptions) are the highest-value test investment — they catch silent drift that no functional test would find

## Workflow

```
Triage Linear issues → Verify against codebase → Write brief → Hand to coding agent → Review → Build gate → Commit/push → Close Linear
```

### 1. Triage

Scan open Linear issues. Classify each as:
- **Already fixed** (close with commit reference)
- **Stale/obsolete** (cancel with reason)
- **Actionable** (verify against code, write brief)
- **Test coverage** (demote to Low)
- **Feature/architecture** (defer to roadmap)
- **Research** (note and defer)

### 2. Verify before briefing

Before writing a brief, **read the actual code**. Confirm:
- The file and line numbers are current
- The bug still exists
- The C source (if applicable) matches what you expect
- No prior fix attempt was made that partially addressed it

### 3. Write the brief

Create `docs/briefs/BRIEF-YYYY-MM-DD-<slug>.md` following the template below. Each brief should be:
- **Self-contained** — coding agent has everything it needs (no "check with Zach" ambiguity)
- Scoped to 1-8 related issues max
- Include `**Cite:**` fields for every C source reference
- Include regression tests where they make sense
- Include exact file:line references

### 4. Hand to coding agent

Pick the best agent for the job (see Agent Rotation below). The agent:
- Reads the brief
- May propose an implementation plan for review before coding (recommended for L+ effort)
- Creates a branch from `main`
- Implements the fixes
- Runs build gates: `go build ./... && go vet ./... && go test ./...`
- Pushes the branch (Kimi commits to main directly — known process issue)

### 5. Review

This is the critical step. **Do not skip review.** Check:
- C source fidelity — does the Go match the cited C behavior?
- Are lock safety / concurrency concerns addressed?
- Did the agent introduce side effects?
- Are regression tests / golden tests included where requested?
- Did the agent handle edge cases (nil guards, empty collections, zero values)?
- Do all build gates pass? (`go build ./... && go vet ./... && go test ./...`)
- For golden tests: spot-check a sample of C→Go transcriptions against source

### 6. Commit, push, close Linear

After review:
- Stage and commit with conventional commit message
- Push to remote
- Add commit hash to each Linear issue
- Move issues to Done

## Agent Rotation

Multiple coding agents are available. Rotate to avoid rate-limit burnout:

| Agent | Strengths | Rate Limits | Notes |
|-------|-----------|-------------|-------|
| **Kimi k2.6/k2.7** | Bug fixes, golden tests, large briefs | 403 after ~7 parallel subagents; ~1hr cooldown | Commits directly to main |
| **Gemini 3.5-flash** | Golden tests, mechanical fixes, code review | Generous — good fallback | Produces clean walkthroughs |
| **Claude Fable 5** | Full audits, architecture review, complex analysis | Expensive — use sparingly | Best for code review, not implementation |
| **Claude Sonnet** | Complex multi-file changes, refactoring | Moderate | Good for PR review |
| **DeepSeek** | Easy/small tasks, cleanup, nits | Cheap, fast | Good for quick wins |

**Dispatch rule:** Don't default to Claude for implementation — save it for review and audit. Rotate Kimi/Gemini for coding work. DeepSeek for trivial fixes.

## Brief Types

### Bug Fix Brief
Standard format (see template below). 1-8 related issues. Include C citations.

### Golden Test Brief
Transcribe C static tables/formulas into Go test assertions. Higher volume, lower risk.
- Include the full C source excerpt (table, formula, or function)
- Specify the Go test file name and package
- Specify expected assertion count
- Note any deliberate Go divergences from C
- Example: `BRIEF-2026-07-05-round8a-spell-golden.md`

### Audit Brief
For full codebase reviews (e.g., Fable). Three-phase: Sweep → Deep Dive → Roadmap.
- Phase 1: Package health survey, architecture risks, port completeness
- Phase 2: Findings with file:line, severity, category, effort, fix approach
- Phase 3: Prioritized roadmap, work streams, coverage targets
- Example: `BRIEF-2026-07-05-fable-full-audit.md`

## Review Checklist (for every PR/commit)

- [ ] Build gate passes: `go build ./... && go vet ./... && go test ./...`
- [ ] Fix matches the brief's description
- [ ] C source citations verified (spot-check at least 2)
- [ ] No side effects or regressions
- [ ] Concurrency safety: locks acquired/released correctly, no check-then-act races
- [ ] Error handling: player-facing paths check errors (no swallowed returns)
- [ ] Golden tests: C values match, Go divergences documented
- [ ] No `fmt.Fprintf` converted to `slog` (those are MUD output)
- [ ] No `CustomData` removed (it's the escape hatch)
- [ ] No C files modified (they're reference only)

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

---

## CI Pipeline

The CI runs on every push to `main` and every PR against `main`. Config: `.github/workflows/ci.yml`. **All checks must pass before merging.**

### Jobs

| Job | What it runs | Gate? |
|-----|-------------|-------|
| **test** | `go test -race ./...` (all packages except tests/unit), `go build ./...`, Python pytest (non-e2e), binary build + 10s startup smoke | ✅ Required |
| **lint** | `golangci-lint run ./...` (v2.12.2, pinned to match Go 1.26.4), `gofumpt` format check | ✅ Required |
| **build-and-push** | Docker image build + push to ghcr.io (main branch only, `zax0rz` owner only) | Conditional |
| **deploy** | kubectl apply to k8s (main branch push only, requires KUBECONFIG secret) | Conditional |

### Build Gate (local, run before every commit)

```bash
go build ./...    # Compilation
go vet ./...      # Static analysis
go test ./...     # All tests
```

CI runs **`go test -race`** which is stricter than local `go test`. If CI fails with a race detection that local tests didn't catch, it means the concurrent code path is only exercised under CI's parallel test runner.

### Lint Rules

- **golangci-lint** uses `.golangci.yml` config (v2-style, matching golangci-lint v2.12.2)
- **gofumpt** — stricter than `gofmt` (enforces parenthesized return types, etc.)
- Common failures:
  - `SA4000` — identical expressions in comparison (e.g., `3.0 / 3.0`)
  - `SA1019` — deprecated type/field usage
  - `gocritic` — comment formatting on `// Deprecated:` (must be in its own paragraph)

### What's NOT in CI (yet)

- **e2e smoke test** (`scripts/smoke_test_2b.py`) — not wired. See COV-5 (DP-966) for the plan.
- **Docker Compose integration** — no compose-based test exists yet
- **Coverage reporting** — no `codecov` or similar; coverage tracked manually via `go test -cover`
