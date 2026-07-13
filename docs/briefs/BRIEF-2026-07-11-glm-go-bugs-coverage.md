# BRIEF 2026-07-11 — GLM: Go bug fixes + coverage (post-reconcile marathon)

**Executor:** GLM (zai coding plan). **Branch:** `glm/go-bugs-2026-07-11` (fresh off
current `main`). **One PR** when the whole list is green.

## Ground rules (same as your last sweep — it went well)
1. **Work in your OWN clone or a `git worktree`.** Never share a working dir /
   HEAD with another agent. Confirm `git status` is clean before you start and
   before every commit. (Last time two agents shared a clone and 11 commits
   scattered — don't repeat it.)
2. **Verify each finding against CURRENT `main` source before fixing.** Several
   clawpatch findings this cycle were *framed wrong but real underneath* — fix the
   REAL divergence, not the paraphrase. If a finding turns out false/already-fixed,
   skip it and say so in the PR body. Read `src/*.c` when fidelity is involved.
3. **One commit per item**, scoped, with a regression test that fails before /
   passes after. End each message with your own `Co-Authored-By:` line.
4. Run `go build ./... && go test -race ./... -timeout 120s` before pushing.
   `gofmt -l` / golangci-lint must be clean (CI runs `lint` + `test` jobs).

## Part A — Go bugs (do these first; highest value)

### A1. `pkg/auth/ratelimit.go` — SetTrustedProxies swallows CIDR errors (SECURITY)
`SetTrustedProxies` (line 29) always `return nil`, even when `net.ParseCIDR`
fails inside the loop (invalid CIDRs are logged + `continue`d, never surfaced).
The signature promises `error`. A misconfigured CIDR silently disables part of
the X-Forwarded-For trust set — a security control. **Fix:** collect parse errors
inside the `Once.Do` and return them joined (`errors.Join`) so a bad CIDR is a
hard startup error, not a silent downgrade. Keep the `sync.Once` semantics (init
once). Note: the finding's "return is outside the Once" phrasing is a red herring
— the real bug is the swallowed errors. Test: pass a bad CIDR, assert non-nil err.

### A2. `pkg/privacy/config.go` — Timeout/Fallback are dead config
`Config.Timeout` (line 30) and `Config.Fallback` (line 33) are defaulted (54/55)
and loaded from env (96/101) but **read by no one** — grep shows zero `.Timeout`
/`.Fallback` uses outside config.go. **Fix:** wire them into the Client: use
`Timeout` (seconds) for the HTTP client timeout, and `Fallback` ("mask" etc.) for
the documented fallback behavior on PII-handler failure. Do NOT just delete —
they're documented knobs. Test: construct a Client from a Config with a custom
Timeout/Fallback and assert they take effect.

### A3. `pkg/optimization/advanced_pool.go` — Close() drops buffered items
`Close()` (line ~338) closes the `priorityQueue` channel; items still buffered in
it are dropped without being processed. **Fix:** drain remaining items before /
after close (follow the WorkerPool/AIBatchProcessor `Wait`+drain pattern already
added in DP-746/808, commit 7101423 — mirror it). Test: enqueue N, Close, assert
all N processed or accounted for.

### A4. `pkg/agentcli/events.go` — CompactionWindow field mismatch
`CompactionWindow` (line ~192) gates on `ev.Type == "state"` but extracts the
room via `ROOM_NAME` while `context.go:97` uses a different field. Verify the two
against each other and the actual event schema; make extraction consistent so
state events actually contribute their room to the compaction window. Test:
feed a state event, assert the room is captured.

### A5. `pkg/dreaming/dream.go` — eventID collision at same nanosecond
`eventID` (line ~97) is `fmt.Sprintf("%s-%s-%d", agentID, kind, nanoTs)`. Two
events of the same kind in the same nanosecond collide. **Fix:** add a
monotonic counter (atomic) or a short random suffix. Test: generate 2 IDs for
same agent/kind back-to-back, assert distinct.

### A6. `pkg/command/admin_commands.go` — hardcoded lvlGod=34 drift
Line ~693 hardcodes `lvlGod = 34` with a "KEEP-IN-SYNC" comment pointing at
`pkg/game/limits.go`. **Fix:** reference the real constant from `pkg/game/limits.go`
(or a shared const) instead of the literal, so it can't drift. Test: assert the
admin-side value equals the limits.go constant.

### A7. `Makefile` (DP-785) — deploy defaults bypass ifndef
Lines 100-101: `DEPLOY_USER ?= root` / `DEPLOY_HOST ?= 192.168.1.121` set defaults
that make the `ifndef` guards further down dead — a deploy with no env set silently
targets a hardcoded host as root. **Fix:** remove the `?=` defaults (or make the
targets hard-error when DEPLOY_USER/DEPLOY_HOST are unset), so deploy requires
explicit host/user. No test (Makefile) — just the guard. Comment DP-785.

### A8. `.clawpatch/project.json` + `.clawpatch/config.json` (DP-813)
Both still have `"typecheck": "go test ./..."` — running the full test suite as a
"typecheck" step. **Fix:** point `typecheck` at `go vet ./...` (or `go build ./...`),
and leave the test suite to the `test` command. Quick config fix. Comment DP-813.

## Part B — coverage (do after Part A; lower priority)

### B1. `pkg/game/systems/shop_test.go` (DP-881)
Current `pkg/game/systems` coverage 65.1%. Add: concurrent-transaction test
(two buyers, `-race`), and a `Door.Reset` test. **Before writing shop tests,
check DP-503** — there may be a split-brain `pkg/game/shop.go` vs
`pkg/game/systems/shop.go`; test the one that's actually wired (ask if unclear,
don't test dead code). Fidelity: read `src/shop.c` for buy/sell/reset semantics.

### B2. `pkg/storage` (DP-870)
Coverage 33.3%, one test only, 194 lines of SQLite backend untested. Add table
tests for the PlayerStore/WorldStore SQLite impl: round-trip save/load, not-found,
and error paths. (Note: DP-759 will add `context.Context` to these interfaces —
**Claude is doing DP-759 in parallel**; coordinate so you don't both edit the
interface. Safe split: you write the *tests* against current signatures; if the
signatures change under you, rebase.)

### B3. `pkg/privacy/client_test.go` — os.Setenv → t.Setenv
Lines ~110-124 use `os.Setenv`/`os.Unsetenv`; env leaks to parallel tests. Swap to
`t.Setenv` (auto-restores, forbids `t.Parallel` correctly). Trivial.

### B4. `cmd/test-race/main.go` — Kender race coverage
Lines ~89-105: the stat-roll test matrix covers 15 of 49 race×class combos and
omits Kender entirely. Add the missing race rows (at least Kender). Low priority.

## Do NOT touch (other lanes)
- **Claude:** DP-759 (storage context.Context), DP-1008 (Animate Dead fidelity),
  DP-503 judgment call (is `pkg/game/shop.go` dead?).
- **Kimi:** DP-1010 (Python ai_optimizer lock), the coverage-only tickets
  DP-608/658/661/662/871, ticket close/link.
- Already fixed / in flight: everything in PRs #122/124/125/126/127/128/129/130/
  131/132 — check `git log` before assuming a finding is open.

When done: open ONE PR `glm/go-bugs-2026-07-11 → main`, list each item + its
verdict (fixed / skipped-why) in the body. Claude verifies the diff + Linear state.
