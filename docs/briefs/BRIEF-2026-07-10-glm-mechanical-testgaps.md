# BRIEF 2026-07-10 — Mechanical test-gap & hygiene sweep (GLM)

## Who / what / why

You are the **GLM coding agent** (zai plan). This is a **bounded execution brief**:
a batch of small, well-scoped, low-risk fixes for clawpatch findings that have been
**verified real against current `main`** by two independent passes (Reek's reconcile
+ Claude's spot-checks). None of these require fidelity judgment against the C source —
they are Python test-gaps, `examples/` hygiene, docs/comment fixes, and small Go
contract fixes.

**Working repo:** the Dark Pawns dev clone (origin `git@github-darkpawns:zax0rz/darkpawns.git`).
Branch off latest `main`: `git checkout main && git pull && git checkout -b glm/mechanical-testgaps-2026-07-10`.

### Rules (same as the 2026-07-08 batch that worked well)
1. **One fix = one commit.** Conventional message, cite the file and the DP ticket if listed
   (e.g. `test(scripts): assert in emotion classifier pytest tests (DP-1011)`).
2. **Every behavioral fix gets a test** (or fixes an existing false-passing test). For the
   Python test-gap items the fix *is* making the tests actually assert — verify they now
   FAIL when the underlying behavior is broken (introduce a temporary break, confirm red,
   revert).
3. **Run the suite before each commit:** `go build ./... && go vet ./... && go test ./...`
   for Go changes; `pytest <file>` (or `python -m pytest`) for Python changes. Do not commit
   red.
4. **Match surrounding style** — comment density, naming, idiom. gofumpt Go files
   (`gofumpt -w <file>`); the CI runs `gofumpt -l .` and fails on any diff.
5. **If an item is ambiguous or turns out bigger than described, STOP and flag it back**
   with what you found — do not guess or expand scope. A flagged item is a success, not a
   failure.
6. **Do NOT touch the EXCLUDE list below.**

### EXCLUDE — do not work these (handled elsewhere / need judgment)
- `pkg/parser/parser.go:122` (room vnum 0 guard) — **fidelity decision** (NOWHERE=-1 vs 0),
  Claude will take it against the C source. Leave it.
- Anything in `pkg/agentcli/daemon.go`, `pkg/agentcli/ws.go`, `pkg/optimization/database.go`,
  `pkg/optimization/room_cache.go`, `pkg/session/act_comm.go`, `pkg/engine/affect.go` —
  already fixed in open PRs #122/#124/#126. Skip.

---

## Part 1 — Python test-gaps (tests that pass while the code is broken)

These are the highest-value items: pytest-collected functions that `return`/`print`
instead of `assert`, so `pytest` reports green even when the behavior is wrong. Fix =
make them assert; confirm they fail on a real break.

| # | File:line | Problem | Fix | DP |
|---|---|---|---|---|
| 1 | `scripts/test_emotion_classifier.py:94` | test functions `return passed==total`; pytest ignores return values → false pass | Replace boolean returns in `test_*` functions with `assert`. Keep the `__main__`/`run_all_tests()` convenience path separate (rename those helpers so pytest doesn't collect them). | DP-1011 |
| 2 | `test_direct_memory.py:12` & `:91` | `test_*` use `print`-based checks, no `assert` → pytest false success | Replace prints with `assert` in each `test_*`; move cleanup into fixtures/finalizers; drop `main()` or rename it so it isn't collected. | — |
| 3 | `tests/e2e/web/test_web_client.py:627` | `test_security_headers` only counts present headers & accepts insecure outcomes | Assert the real contract: require the configured security headers; reject `500` on injection payloads; assert content on safe `200`s; remove the unreachable/always-pass branch. | — |
| 4 | `tests/e2e/web/test_web_client.py:482` | bare `except: pass` in cleanup hides `NameError` etc. | Guard each close: `if 'ws1' in locals(): ws1.close()`; use `except Exception:` not bare `except:`. | — |
| 5 | `tests/integration/python/test_ai_integration.py:16` | suite skips itself / only tests local mocks | If it's genuinely a mock unit test, **rename/mark it as such** (`test_*_mock`) and add a `@pytest.mark.integration` real test that is skipped (with a clear reason) when the Go bridge/binary isn't available — don't fake integration. If unsure how to reach the real boundary, **flag back**. | — |
| 6 | `web/test_onboarding.py:18` | `requests.get()` calls have no timeout → can hang forever | Add `timeout=DEFAULT_TIMEOUT` (module const, e.g. `10`) to every `requests.*` call. | — |
| 7 | `web/test_onboarding.py:9` | pytest depends on a manually-running `localhost:4350` | Read `BASE_URL` from env; `pytest.skip("set BASE_URL to run onboarding e2e")` when unset, so the file doesn't fail in CI. | — |
| 8 | `web/test_onboarding.py:182` | `generate_agent_code` overwrites files with no error handling | `os.path.exists()` check (prompt or unique filename) + wrap write in try/except with a friendly message. | — |

## Part 2 — Python scripts (real small bugs)

| # | File:line | Problem | Fix | DP |
|---|---|---|---|---|
| 9 | `scripts/ai_optimizer.py:46` | cache hit returns `AIResponse` with the **cached** `request_id` (stale) | On hit, return a copy with the current request's id: `AIResponse(request_id=request.request_id, text=cached.text, tokens=cached.tokens, latency=0, model=cached.model)`. Add a test asserting the returned id matches the new request. | — |
| 10 | `scripts/dp_playtester.py:273` | `--turns` parsed but ignored | Thread it through: `async def run(self, turns: int = 50)`, loop `while turn < turns`, call `await bot.run(args.turns)`. | — |
| 11 | `scripts/wire_scripts.py:83` | mob rewrite regex can insert `Script` lines at the wrong terminator (**can corrupt mob files**) | Bound each mob block from `#<vnum>` to the next `\n#<number>` or EOF, find the final standalone `E` **inside that bounded block**, insert there. Write via temp file + atomic replace. Add a test with a multi-mob fixture asserting the Script line lands in the right block. **Higher risk — test thoroughly; flag back if the block grammar is unclear.** | — |

## Part 3 — `examples/` hygiene

| # | File:line | Problem | Fix | DP |
|---|---|---|---|---|
| 12 | `examples/optimization_integration.go:143` | `WebsocketOptimizationExample` panics on msgs < 20 bytes (teaches a panicking idiom) | `msg[:min(20, len(msg))]`. | — |
| 13 | `examples/optimization_integration.go:47` | `#nosec G104` suppresses unchecked errors | Handle with `slog.Warn`/`log.Printf`; remove the `#nosec` annotations. | DP-817 |
| 14 | `examples/optimization_integration.go:214` | `IntegrateWithServer` is dead commented-out Go that won't compile-check | Delete it (unused & unbuildable) — or move the sample into a real `_test.go`/markdown doc. Prefer deletion unless it's referenced. | DP-818 |
| 15 | `examples/metrics_integration.go:41`/`:50` | prints a hardcoded `localhost:4350/metrics` URL but binds no server | Either start `go http.ListenAndServe(":4350", metrics.Handler())` so the URL serves, OR annotate clearly that it's demonstrative and remove the misleading printed URL. Pick one and be consistent. | DP-809 |
| 16 | `examples/metrics_integration.go` (pkg) | `examples/` has no tests | Add `TestExamples` calling `MetricsIntegration()` and `OptimizationIntegration()` asserting they return without panic. | DP-867 |

## Part 4 — Small Go contract / hygiene fixes

| # | File:line | Problem | Fix | DP |
|---|---|---|---|---|
| 17 | `pkg/agentcli/state.go:115` | `StateFile.Update` does manual lock/unlock — leaks mutex on panic | `sf.mu.Lock(); defer sf.mu.Unlock()`. | — |
| 18 | `pkg/agentcli/session.go:71` | `WriteJSONL` doc says "bytes written" but returns entry count | Fix the doc comment to "number of log entries written". | — |
| 19 | `pkg/agentcli/events.go:192` | `CompactionWindow` uses wrong JSON path for `state` events | Use the same nested `Room`/`name` path as `context.go:96-101`, not the top-level `ROOM_NAME`. Add a test with a real state-event payload. | — |
| 20 | `pkg/auth/ratelimit.go:24/28` | `SetTrustedProxies` silently drops invalid CIDRs, always returns nil | Collect invalid CIDR strings and return them in the error; apply valid ones. If the `sync.Once` behavior is kept, document that only the first call takes effect. Test: invalid CIDR → non-nil error; valid ones still applied. | — |
| 21 | `pkg/command/admin_commands.go:693` | `lvlGod` hardcoded `34`, can drift from `game.LVL_GOD` | Reference the shared constant instead of a literal (import `game.LVL_GOD`, or thread it through). If a package import cycle blocks this, **flag back**. | — |
| 22 | `pkg/optimization/advanced_pool.go:62` | priority-queue items silently discarded on `Close()` | Before closing, drain `priorityQueue` into `taskQueue` (or process inline). Add a test: submit prio items, Close, assert none lost. | — |
| 23 | `pkg/privacy/config.go:27` | `Config.Timeout`/`Config.Fallback` are dead (loaded, never used) | Either wire them into the `Client`, or remove the fields + their env loaders. Prefer wiring if a consumer exists; else remove. **Flag back if unclear which.** | — |
| 24 | `pkg/privacy/client_test.go:110` | `TestConfig_LoadFromEnv` uses `os.Setenv`/`Unsetenv`, leaks env to other tests | Switch to `t.Setenv` (auto-cleanup). | — |

## Part 5 — Dependency hygiene

| # | File | Problem | Fix | DP |
|---|---|---|---|---|
| 25 | `requirements.txt` | lower-bound-only, unbounded above, not reproducible | Add upper bounds (e.g. `openai>=1.90.0,<2.0.0`) for each dep **and** commit a resolved lockfile (`pip freeze > requirements-lock.txt`, or a `requirements.in` → `pip-compile`). Keep loose ranges only in a dev `.in` if you split them. | DP-788 / DP-816 |

---

## Linear bookkeeping
For each item with a DP ticket (DP-809/817/818/867/788/816/1011), add a short comment on
the Linear issue citing the fixing commit, and move it to the appropriate done/review
state per team convention. Items without a ticket don't need one — they're covered by the
commit + this brief.

## Done criteria
- All non-excluded, non-flagged items fixed, each its own commit with a test where behavioral.
- `go build ./... && go vet ./... && go test ./...` green; `gofumpt -l .` clean; touched
  Python files pass `pytest`.
- A short summary at the end: what was fixed, what was flagged back (with why), and the
  branch name for the PR.
