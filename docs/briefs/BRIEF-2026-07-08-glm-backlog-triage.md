# Brief: DP Backlog Triage & Cleanup

**For:** GLM 5.2 (zai coding plan)
**From:** Claude
**Requested by:** Zach
**Date:** 2026-07-08
**Repo:** `zax0rz/darkpawns`, branch `main` at commit `73bad7f` or later

---

## Goal

The DP team backlog has ~108 open issues (Backlog/Todo). A lot of it is stale: bugs
already fixed by later work, duplicate findings, or coverage numbers that moved
since the ticket was filed. Bring the backlog down to a number that reflects
*actual* remaining work — close what's done, fix what's small, leave what's
genuinely a bigger call to a human.

This is a bounded, mostly-mechanical task: verify a claim against the current
code (`go test <pkg> -cover`, `grep`/`Read` the referenced file/line), then act.
Don't guess — every close/update below should cite the command or code you
actually ran.

Use whatever Linear access you have (MCP tools or `linear-cli`, see
`docs/briefs/linear-cleanup-brief.md` for install notes) to read/update issues
on the **DP** team only. Don't touch CHAD, HH, or LAB teams.

---

## Part 1: Close — already verified stale by Claude (2026-07-08)

These four are confirmed fixed already. Comment with the evidence below, then
set status to **Done** (DP-775/776) or **Done** with an updated-number comment
(DP-606/607, since coverage isn't literally "0%" anymore, just not great):

| Issue | Claim | Verified reality |
|---|---|---|
| DP-775 | Lock ordering deadlock between `processBuy`/`processSell` (`pkg/game/systems/shop_manager.go:173`) | Both functions take a single `player.Lock()` each — no nested/ABBA lock acquisition present. This is a re-filed duplicate of already-**Done** DP-613/DP-660, which fixed this exact deadlock. Close as duplicate. |
| DP-776 | `ObjectPool.TryGet` deadlocks calling `Get` while holding the pool mutex (`pkg/optimization/object_pool.go:150`) | `TryGet` calls `op.getLocked()` (an internal, non-locking helper), not the public `Get()`. Duplicate of already-**Done** DP-614/DP-659, which fixed this. Close as duplicate. |
| DP-606 | `pkg/optimization` — 0% test coverage | `go test ./pkg/optimization/... -cover` → **48.2%**. Update with real number; close or downgrade to a much lower priority "improve further" ticket, executor's judgment. |
| DP-607 | `pkg/telnet/listener.go` — zero test coverage on connection entry point | `listener_test.go` exists; `go tool cover -func` shows most exported functions (`Listen` 68%, `ipFromAddr` 100%, `effectiveBanLevel` 95%, `handleConn` 41%, etc.) covered. Some functions are still 0% (`promptContainsMenu`, `renderCharCreateOptions`, `sendMSSP`, `buildGMCPFrame`, `handleIncomingGMCP`) — note those as a follow-up rather than reopening the whole ticket. |

---

## Part 2: Re-verify the 2026-06-29 Reek batch (bulk of the backlog)

45 of the ~108 open issues are `reek`-labeled findings, almost all filed on a
single day (2026-06-29). Given how much test-writing has happened since (DP-962
kill-payout tests, various coverage pushes), a lot of these are probably stale
too — but each needs its own check, they weren't all touched.

**Method per issue:**
1. Read the file/line referenced (below) or `get_issue` for the full description if the snippet isn't enough context.
2. If it's a coverage claim: re-run `go test <package> -cover` (or `-coverprofile` + `go tool cover -func` for per-function detail like above). If coverage is now substantially higher and a `_test.go` file exists for the named file, close with the updated number.
3. If it's a real code-quality/bug claim (not coverage): re-read the current code at that location. If already fixed, close citing what changed. If still true and it's a small, single-function fix, just fix it (see Part 3 — many of these overlap). If it's still true but nontrivial, leave open, no state change needed.
4. Always leave a comment citing the command you ran or the code you read — don't close on vibes.

**Full list to work through:**

| Issue | File ref | Title |
|---|---|---|
| DP-658 | `pkg/command/` | already updated 2026-07-07 to 13.5%, downgraded to Medium — no action needed, skip |
| DP-892 | (main game loop) | No context cancellation in main game loop |
| DP-608 | `pkg/command/skill_commands.go` | 40+ skill commands, zero test coverage — note: `skill_commands_test.go` now exists (178 lines), re-check whether it actually covers the 40+ commands claimed or just a subset |
| DP-661 | (persistence package) | untested at 17.8% coverage |
| DP-666 | (pkg/game) | untested critical path at 14.9% — re-check: now 26.1% per `go test ./pkg/game/... -cover` |
| DP-863 | (cryptography/secrets package) | zero test coverage |
| DP-881 | (concurrent transactions, Door.Reset) | missing test coverage |
| DP-870 | (pkg/storage) | no tests |
| DP-871 | (PIIHandler, WebSocketLogger, slog chain) | no test coverage |
| DP-867 | (examples package) | no tests |
| DP-872 | (pkg/optimization) | no tests — likely same underlying fix as DP-606, cross-reference |
| DP-702 | `pkg/scripting/engine.go:167` | mutex held during Lua DoFile/PCall blocks engine |
| DP-662 | `pkg/spells` | untested at 14.1% |
| DP-610 | `metrics_test.go` | pkg/metrics tests check names not values |
| DP-737 | `pkg/secrets/manager.go:30` | raw byte-string key from ENCRYPTION_KEY, no KDF |
| DP-746 | `examples/optimization_integration.go:39` | WorkerPoolExample closes pool while tasks queued |
| DP-755 | `pkg/agentcli/fsm.go:19` | dead code in FSMDecision, unreachable loop |
| DP-759 | `pkg/storage/interface.go:8` | Store interface lacks context.Context |
| DP-767 | `pkg/db/decision_log.go:267` | DecisionLogWriter.Stop returns before flush completes |
| DP-768 | `pkg/telnet/listener.go:230` | password prompts ignore disconnects |
| DP-772 | `pkg/events/queue.go:132` | TimeUntilNext reports cancelled events as pending |
| DP-778 | `tests/e2e/web/test_web_client.py:18` | missing base_url pytest fixture, all e2e tests broken |
| DP-784 | `scripts/ai_optimizer.py:170` | deprecated asyncio.get_event_loop(), breaks on 3.14+ |
| DP-785 | `Makefile:99` | deploy-site safety checks bypassed by default creds/host |
| DP-787 | `pkg/privacy/middleware.go:35` | middleware buffers full body before truncating logs |
| DP-788 | `requirements.txt:1` | unbounded lower-bound-only dependency constraints |
| DP-790 | `pkg/combat/formulas.go:230` | strength indexing can panic on out-of-range stats |
| DP-808 | `examples/optimization_integration.go:96` | AiBatchProcessingExample discards Submit errors |
| DP-809 | `examples/metrics_integration.go:41` | hardcoded Prometheus/Grafana URLs, no real binding |
| DP-810 | `scripts/trace_path.py:3` | tied to one developer's absolute path |
| DP-811 | `pkg/optimization/advanced_pool.go:272` | AdvancedWorkerPool.Resize can't reduce workers |
| DP-812 | `pkg/db/player.go:57` | DB.New leaks connections on init failure |
| DP-813 | `pkg/scripting/engine.go:1` | AGENTS.md placeholder references `go test` for typechecking |
| DP-814 | `pkg/storage/interface.go:36` | no compile-time assertion SQLiteBackend satisfies FullBackend |
| DP-815 | `pkg/secrets/manager.go:57` | path built before traversal check in GetSecret |
| DP-816 | `requirements.txt:1` | Python deps unbounded above, not reproducible |
| DP-817 | `examples/optimization_integration.go:47` | example suppresses errors with #nosec G104 |
| DP-818 | `examples/optimization_integration.go:214` | dead commented-out Go in a doc comment |
| DP-819 | `pkg/moderation/manager.go:344` | regex word filters recompiled every message |
| DP-820 | `pkg/game/systems/door_manager.go:96` | GetDoorsInRoom suppresses unused var with `_` |
| DP-821 | `scripts/emotion_llm_classifier.py:224` | shared mutable dict defaults in batch_classify |

---

## Part 3: Just fix these (small, well-scoped, still-real bugs)

Based on the ref lines above, these read like genuine single-function fixes,
not "add a test suite" projects. If your Part 2 pass confirms they're still
live, fix them directly rather than just filing a note — same standard as the
DP-1007 fix (verify against source, make the minimal correct change, run
tests, commit):

- DP-772 — `TimeUntilNext` cancelled-event bug
- DP-767 — `DecisionLogWriter.Stop()` flush ordering
- DP-768 — telnet password prompt disconnect handling
- DP-790 — strength indexing panic guard
- DP-811 — `AdvancedWorkerPool.Resize` can't shrink
- DP-812 — `DB.New` connection leak on init failure
- DP-814 — missing compile-time interface assertion (one-line `var _ FullBackend = (*SQLiteBackend)(nil)`)
- DP-815 — reorder path traversal check before path construction
- DP-819 — hoist regex compilation out of the hot path
- DP-820 — fix the suppressed unused-var warning properly
- DP-755 — dead code in FSMDecision
- DP-737 — decode ENCRYPTION_KEY properly (hex/base64 + KDF) instead of raw bytes

Each should be its own small commit (or one commit per logical group) so
they're easy to review — don't bundle all of these into one giant diff.

---

## Part 4: Do NOT touch — deferred product/research backlog

These are large, multi-week feature or research epics, not bugs. They're
intentionally parked (game-port fidelity work takes priority over client/agent
work per current project priorities) — leave them in Backlog untouched, don't
close them, don't re-prioritize them:

- Admin Phase 6.x (DP-50, 52, 55–57)
- Website v2 / Hugo rebuild / map-database features (DP-67, 70, 78, 310, 317–331)
- Agent Layer epics (DP-214–230)
- Client Sprint epics (DP-193–199)
- Research write-ups (DP-71, 74–75, 190, 227–228, 241)
- DP-189 Clawpatch Lua Mapper, DP-191 archive scripts, DP-529 interface{} type safety sweep, DP-601 database table ownership, DP-516/522–524 client platform phases, DP-645 differential harness (explicitly deferred already)

If any of these look genuinely stale (e.g. references removed infra), flag
them for Zach rather than closing unilaterally — these are scope calls, not
verification calls.

---

## Part 5: Next real fidelity bug (flag, don't fix — unless you have time)

**DP-1008** — `pkg/spells/affect_spells.go:552-620`, Animate Dead is missing
three validation checks that C's `magic.c:1692-1760` has (charm-flag check,
follower-cap check, random pfail roll). This is real and unverified-by-Claude
(only read the truncated description, not the full ticket or source). If you
have bandwidth after Parts 1–3, this is the same shape of fix as DP-1007 —
read the full ticket, verify against `src/magic.c`, port the three checks,
add/update tests.

---

## Verification / Wrap-up

1. `go build ./...` and `go test ./...` must pass before any commit.
2. Commit per logical group, not one mega-commit.
3. After closing issues, re-run a `list_issues` query for team DP, state
   Backlog/Todo, to get the new total — report the before/after count.
4. Leave a comment on every issue you touch citing what you actually verified
   (command output, file/line read) — no silent closes.
