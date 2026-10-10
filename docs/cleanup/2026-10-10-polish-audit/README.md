# Pre-launch polish audit — 2026-10-10 (GLM-5.3-flash, report only)

**Base:** `origin/main` at `496462d6d` (post PR #1899 dead-code-b). **Scope:** reports only — no code changed outside this directory. Brief: `~/BRIEF-2026-10-10-polish-audit-flash.md`.

## Files

| File | Rows | What it is |
|---|---|---|
| `uncalled.tsv` | 95 | Registered-but-never-called exported symbols: subsystem findings first, then dead wrappers/twins/accessors. Every row's `callers_outside_tests` is 0, verified by grep in this session. |
| `deadcode-delta.tsv` | 12 | `deadcode -test ./...` rows **not** already classified in `docs/cleanup/2026-10-09-dead-code-decisions.md` |
| `simplify.tsv` | 4 | Efficiency/simplification with fidelity-risk labels |
| `idiomatic.tsv` | 2 | "C in Go" patterns with fidelity-risk labels |
| `exported-symbol-census.py` | — | The comment-stripping caller-census tool that produced report 1's candidates (rerunnable) |

## Counts behind deadcode-delta

`deadcode -test ./...` (exact form, at `496462d6d`) reports **62** unreachable funcs. 24 match the 2026-10-09 decisions doc by name; another 26 are covered by its scope rulings (the whole `pkg/command/admin_commands.go` "parallel moderation layer (24 funcs)", the 10 `pkg/events` unpublished `.Type` markers assigned to Claude's mudlog-push filing batch, and the class-(d) `dprng` draw-log hooks `DrawLogIndex`/`DrawLogEnabled`). The 12 unclassified rows are the delta.

## Top 10

1. **`HouseSaveAll` parity trap (proved dead in C, not a bug).** Zero callers — but C's `House_save_all` is an empty function (body commented, `src/house.c:710-725`) and its heartbeat call is commented (`src/comm.c:834`). Go matches C by never calling it. **Do not wire without an R4 ruling.** This finding is the audit working as intended: R5e (verify the call path into `src/`) killed a very tempting false positive.
2. **`ExecQcomm`/`doQcomm` fully uncalled** (`act_comm_bridge.go:28`, `comm_channel.go:291`) — and the 2026-10-09 doc's claim that qcomm "is live as `World.ExecQcomm`" is stale. Open fidelity question: C has no `qcomm` entry in `interpreter.c` either; is `do_qcomm` reachable in C at all?
3. **`AITick` superseded driver** (`ai.go:31`) — mob AI runs solely via the game loop's `OnMobileActivity` → `MobileActivity` (`main.go:591`); the `StopAITicker` comment (`world.go:1228`) still claims an AI tick loop exists.
4. **`ExecuteZoneReset` is a test-only door** (`spawner.go:224`) — prod always calls unexported `executeZoneResetLocked`; **44 test call sites ride the wrapper**, the exact test/prod-seam hazard the 2026-10-09 doc's own finding describes. Thin wrapper, so logic is shared — cleanup only.
5. **`OnPlayerEnterRoom` unwired** (`world_player.go:10`) — event-driven aggro trigger, zero callers (live aggro is per-pulse); contains a `go func` StartCombat that can never run.
6. **Dead wrappers with stale comments:** `MovePlayer` (`world.go:1091`; `death.go:741` still says "Called from MovePlayer") and `RemovePlayer` (`world.go:562`; live twin `RemovePlayerBody` at `manager.go:1363,1488`).
7. **`ReplaceParsedWorld` dead twin** (`world.go:389`) — its comment says "Used by the reload wizard command", but `cmdReload` (`wiz_system.go:181`) never calls it.
8. **Snapshot subsystem built, never consumed** — `GetSnapshotManager`/`SnapshotGeneration` zero callers; the package's own comment (`snapshot.go:14-15`) says full snapshots are unnecessary.
9. **No vnum→proto index + 9 linear scans, one quadratic** — `houseLoad` rescans all parsed objs per house item (`house_save.go:69`, boot path); `world_oedit.go:74,87,144`, `wiz_object.go:158`, `wiz_zone.go:177`, `wiz_stats.go:551`, `world.go:261`, `bridge.go:408` repeat the scan. One `map[int]*parser.Obj` fixes all of them.
10. **Superseded session accessors:** `IsWizlocked`/`SetWizlock` (login gates on `WizlockLevel()`, `entry_restrictions.go:12`), and `UpdateActivity`/`LastActive` — the field has **no writer after construction and no reader anywhere**.

## Method

1. **Report 1 (uncalled):** `exported-symbol-census.py` indexes every exported func/method declared in non-test files under `cmd/ pkg/ web/ internal/ admin-ui/ archive/ tools/`, strips comments (line, block, string, char, raw-string aware), and counts identifier references in non-test vs test files. 1,735 exported symbols; 161 zero-live candidates. Each candidate was hand-verified with targeted greps and traced to a live twin or ruling. 47 were already classified in the 2026-10-09 doc and excluded. Non-function sweeps: all 10 server flags read; all 4 config structs constructed; 56 env vars read in code with no doc-only strays; goroutine/ticker sites checked (`gameloop.go:191`, `main.go:662` metrics ticker, oracle-diff workers).
2. **Report 2 (deadcode-delta):** `go install golang.org/x/tools/cmd/deadcode@latest`; `deadcode -test ./...` from the worktree root; delta computed mechanically against the decisions doc, then each delta row's twin checked by grep.
3. **Reports 3/4:** targeted sweeps (linear scans, nested boot loops, sentinels, globals, hot-path allocations, unchecked errors, switch sizes, pointer out-params) plus follow-up reading. Rows are few by design — see honest negatives below.

## Honest negatives (checked, deliberately not reported)

- **Copy-paste duplication was not hunted.** In a 1:1 port most textual repetition *is* the C structure (R1); flagging it would generate churn, not cleanup. The repeated "Alas, you cannot go that way..." sites are C bytes.
- **`act()`-family `interface{}` parameters** mirror C's `void *` act-argument design — deliberate seam, not an idiom violation.
- **Unchecked-error sweep** produced only void-signature false positives (`Session.Close()`, `http.Flusher.Flush()` are `error`-less).
- **`pkg/events`, `pkg/command/admin_commands.go`, `web/` middleware, `pkg/audit`, `pkg/metrics`, `pkg/validation`, roller cluster, legacy save layer, dprng draw-log hooks:** already classified 2026-10-09 — excluded everywhere, not re-reported.
- **`testutil`** is test-only by design; only its zero-caller members are filed.
- **Env vars/flags/config fields:** all wired; no rows.
- The `fidelity_risk` column is `none` on every simplify/idiomatic row: no finding touches a player-visible byte path or an RNG draw path. Per the brief, anything on an R1/R3 path without provable byte/draw-identical proof was dropped (e.g. `print_object_location`-style formatters are reported only as *uncalled*, never as "wire it").

## Census coordination

No census was started by this audit. The shared slot ran `dp-1386-inactive-prompt-combined` during the reading phase (final verdict `CLEAN_AFTER_RECHECK`); per brief rule 2 no builds were run until it cleared, then `go install`/`deadcode` ran in the clear window. `go build`/`go vet`/`go test` were never needed beyond what `deadcode -test` itself compiles.

## Commands run (beyond the per-row `evidence_cmd` greps)

```
git worktree add /home/zach/dp-polish-audit -b glm-flash/polish-audit origin/main
python3 docs/cleanup/2026-10-10-polish-audit/exported-symbol-census.py /home/zach/dp-polish-audit
go install golang.org/x/tools/cmd/deadcode@latest
deadcode -test ./...
scripts/census.sh status   # coordination only
scripts/census.sh wait     # coordination only
```
