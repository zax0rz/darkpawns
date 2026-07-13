# BRIEF 2026-07-11 — Kimi: Python fix + coverage batch + ticket hygiene

**Executor:** Kimi (K2.7-code, has Linear access). **Branch:** `kimi/python-coverage-2026-07-11`
(fresh off current `main`). **One PR** for code; Linear edits are separate.

## Ground rules
1. **Work in your OWN clone or `git worktree`** — never share a working dir/HEAD
   with GLM or Claude. `git status` clean before every commit.
2. **Verify each item against current `main` before changing it.** Your reconcile
   already confirmed these are live in main — good — but re-read the exact lines,
   they may have moved. Read `src/*.c` only if fidelity is involved (none here).
3. One commit per item, regression test where testable (Python: real `assert`s,
   not boolean returns). `go`/`python` must build; run the affected tests.
4. **Do NOT edit any Go interface** — Claude is changing `pkg/storage` (DP-759)
   and `pkg/spells` (DP-1008) in parallel. Stay in Python + test-only Go.

## Part A — Python bug (do first)

### A1. `scripts/ai_optimizer.py` — AIBatchProcessor non-reentrant lock across await (DP-1010)
Lines ~138-186: `AIBatchProcessor` holds a non-reentrant lock across an `await`.
If the awaited coroutine (or a callback) re-enters a method that takes the same
lock, it deadlocks; at minimum it serializes the whole batch defeating the point.
**Fix:** release the lock before the `await` (guard only the shared-state mutation),
or switch to `asyncio.Lock` used correctly (never held across an await that can
re-enter). Mirror the pattern from the earlier ai_optimizer fixes (commits
ad4cc26 `get_running_loop`, ce17e52 `independent context dicts`). Test: a unit
test that exercises concurrent `process()` calls and asserts no deadlock + correct
per-request results. Comment DP-1010 with the fix.

## Part B — coverage batch (legit improvement work; one commit per package)

Your reconcile already measured current coverage — use it as the baseline and
raise each. Prioritize the *untested* hot files named, not raw % chasing.

- **DP-608 / DP-658** — `pkg/command` at 13.4%: `skill_commands.go` still untested.
  Add table tests for the skill command handlers (happy path + guard/failure
  branches). Read the C `do_*` for the skills you cover so assertions match
  original semantics.
- **DP-661** — `pkg/db` at 35.1%: add tests for the untested query paths.
- **DP-662** — `pkg/spells` at 36.3%: add tests for untested spell effects.
  **Skip `affect_spells.go` Animate Dead (DP-1008) — Claude owns that.**
- **DP-871** — PIIHandler / WebSocketLogger / slog handler chain untested: add a
  test that drives a log record through the handler chain and asserts PII is
  masked and the record reaches the sink.

These are Go tests — run `go test -race ./pkg/command/ ./pkg/db/ ./pkg/spells/ ...`.

## Part C — Linear ticket hygiene (you have access)

You closed 0 last round because everything was genuinely real — good, don't
force-close. This round:
1. **When you FIX a ticket above (DP-1010, and each coverage ticket you raise
   meaningfully), move it to Done** with a comment citing the commit SHA + the
   new coverage number. That's the close criterion — a fix, not a re-triage.
2. **DP-813** (`.clawpatch` typecheck misconfig) — GLM is fixing the config;
   after GLM's PR merges, verify and close DP-813.
3. **DP-503** (split-brain `pkg/game/shop.go`) — do NOT act; leave a comment
   noting it blocks DP-881 shop tests, flagged for Claude's judgment call.
4. Leave DP-892 (game-loop ctx cancellation) flagged for the architecture pass.

## Do NOT touch (other lanes)
- **Claude:** DP-759 storage context.Context, DP-1008 Animate Dead fidelity,
  DP-503 judgment.
- **GLM:** the Go bugs (ratelimit, privacy/config, advanced_pool, events, dream,
  admin_commands, Makefile DP-785, DP-813 config) + shop/storage coverage +
  privacy client_test t.Setenv + test-race Kender.
- Excluded (already fixed in PR #132 / earlier): DP-809/817/818/867/788/816/1011.

When done: PR `kimi/python-coverage-2026-07-11 → main`, body lists each item +
verdict. Claude verifies diff + Linear state.
