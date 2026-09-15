# Phase 6.5 ignored-error cleanup — 2026-09-15

Base checkpoint: `ebb5feff8b4b6f5a7d76492ae10b6fed3fdd7309` (`origin/main`,
merged #1466). Implementation branch: `glm/modernize-phase6-5-batch`.

PR #1467 (`glm/phase6-4-disposition`) was still open when this work started.
It records the accepted Phase 6.4 weather/mail/ban/spec-registry
retained/deferred dispositions. This batch does not duplicate its roadmap or
disposition edits, reopen #1451, or change mail, weather, ban authority,
registry ownership, save formats, schemas, CI, deployment, or website files.

## Scope and inventory method

The working inventory was made before implementation from `pkg/` and
`cmd/server/` production Go files, excluding `*_test.go`. The raw
`\b_\s*=` sweep found 297 lines across 1,012 Go files. Each site was then
classified by the actual returned value and call path:

- a blank result that was an error, including multi-result calls;
- an intentional non-error discard, such as a map assertion, a required RNG
  draw, an unused C-compatible parameter, or an existence probe;
- parser numeric conversion where the error is intentionally folded into the
  current C-compatible default semantics;
- player output or network/session policy where a new failure policy would
  change the observed contract; or
- tests, examples, support tooling, generated data, or a behavior-sensitive
  game operation needing a separately named proof.

The exact handled sites below use line numbers from the frozen base checkpoint
above. The source locations may move as the implementation is reviewed; the
base hash makes every row reproducible.

## Implemented sites

All handled sites either use an existing error return path or add a diagnostic
only. Cleanup diagnostics preserve the primary result and carry only safe
operation context; they do not log passwords, mail bodies, private payloads,
or serialized records.

| exact base site(s) | operation | policy and proof |
|---|---|---|
| `cmd/server/main.go:129,157` | `os.Setenv` for the persistence directory and development JWT secret | Fail closed during boot with the existing `slog.Error`/`os.Exit` startup policy. |
| `cmd/server/main.go:201` | database shutdown close | Log a shutdown diagnostic while preserving the already-completed server result. |
| `pkg/agent/memory_hooks.go:92-93,109-110` | request/response body read, drain, and close | Return request-body read failure; log drain/close failures without changing retry/status behavior. `TestDoMemoryHookWithRetryReturnsRequestBodyReadError` covers the failure path. |
| `pkg/agentcli/llm.go:28,41,43` | request marshal, response read, response close | Return marshal/read failures; log only the safe response-close diagnostic. |
| `pkg/agentcli/daemon.go:325,341,357,364` | event journal append | Return the existing persistence error channel to `handleMessage`; in-memory state remains updated. `TestHandleEventReturnsEventPersistenceError` covers the failure path. |
| `pkg/agentcli/daemon.go:426` | local CLI response write | Diagnostic-only `slog.Warn`; the daemon has no response error channel and does not alter MUD session policy. |
| `pkg/parser/obj.go:66`, `pkg/parser/wld.go:126`, `pkg/parser/mob.go:149`, `pkg/parser/zon.go:42` | world-data file close | Shared `pkg/errlog.Close` diagnostic helper; parser primary errors and C-compatible defaults remain unchanged. |
| `pkg/combat/fight_messages.go:62` | fight-message file close | Same diagnostic-only cleanup policy; message parsing and ordering are unchanged. |
| `pkg/privacy/client.go:123,126` | privacy-filter error-body drain and close | Drain errors are diagnostics; the existing fallback and returned status error remain primary. |
| `pkg/session/discord.go:47` | Discord response close | Diagnostic-only cleanup for an asynchronous operator webhook; no player output changes. |
| `pkg/storage/sqlite.go:47,53,142` | SQLite init cleanup and player-list rows close | Preserve the ping/migration primary errors; report close failures and keep list return semantics. |
| `pkg/db/player.go:314` | player-name listing rows close | Diagnostic-only cleanup; query/scan/rows errors remain the returned database contract. |
| `pkg/db/narrative_memory.go:153,173,196,233` | narrative query rows close | Diagnostic-only cleanup with operation names only; no agent text or record data is logged. |
| `pkg/moderation/manager.go:144,186,487` | penalty/filter/report rows close | Diagnostic-only cleanup; moderation scan and iteration policy is unchanged. |
| `pkg/admin/handlers.go:1656,2293` | admin query rows close | Diagnostic-only cleanup; response bytes and handler error policy are unchanged. |
| `pkg/optimization/database.go:147` | query-analysis rows close | Diagnostic-only cleanup; recommendations and query errors are unchanged. |
| `pkg/optimization/python_ai.go:430` | async AI batch shutdown | Return the existing `Close() error` channel instead of silently dropping a flush failure; worker cleanup still runs. |
| `pkg/engine/gameloop.go:208` | no-return game-loop shutdown wrapper | Log a stop timeout/error while preserving the established no-return API. |
| `pkg/game/level.go:461` | level-up event publication | Log the existing event-bus error after the level/save/player-facing operation has completed; no player bytes or state order change. |

The shared helper is covered by `pkg/errlog/close_test.go`, including an
injected failing closer and safe operation context. The changed package tests
and the full repository gates are the primary verification; these sites do not
change the telnet transcript, so no new oracle scenario is minted for a
diagnostic-only path.

## Retained or deferred sites

These rows remain in the inventory with an explicit reason. “Retained” means
the existing discard is intentional or its current policy is safer to preserve;
“deferred” means the site needs a separate proof or a newly authorized policy.

| exact site family | disposition | reason / smallest next experiment |
|---|---|---|
| `pkg/parser/obj.go:147,162,171-173,236,253-254`; `pkg/parser/wld.go:216,312,314-315`; `pkg/parser/mob.go:331,371-384,387-388,445-446,460-462,621`; `pkg/parser/zon.go:74-76,129-180` | deferred | Ignored numeric parse errors are part of the current parser/default behavior. Do not invent rejection or fallback changes without C parser proof; next experiment is a separate parser-semantics inventory and fixture. |
| `pkg/session/wiz_info.go:414,428,442,454,491,501,511,521,535`; `pkg/session/wiz_player.go:359`; `pkg/game/bans.go:140`; `pkg/game/aliases.go:85` | deferred | These writes feed player-visible reports or player/admin output. R1 forbids replacing them with slog; next experiment is a writer-failure/session-policy proof. |
| `pkg/telnet/listener.go:153,159,175,190,291,293,312,361,374,380,393,455,673,805,829,882-885,928,1097`; `pkg/session/session_pump.go:45,47,109,111,135,138`; `pkg/session/manager.go:994-995,1004-1005`; `pkg/session/session_idle.go:73`; `pkg/session/session_manager.go:80`; `pkg/session/session_player.go:144` | deferred | Network write, deadline, and disconnect failures can change session policy and byte delivery. Keep the current best-effort behavior until a named transport policy test exists. |
| `pkg/grapevine/client.go:88,95,115,149,170,260,338`; `pkg/agentcli/client.go:93,118,127,131,146,391`; `pkg/agentcli/reconnect.go:112`; `pkg/agentcli/ws.go:45,51,62`; `pkg/agentcli/daemon.go:93,99,118,126-128,159,162,167,226` | deferred | External connection cleanup/deadline sites are network/session policy, even when not telnet-facing. Next experiment is a connection-state matrix that proves close/write failure handling before changing policy. |
| `pkg/game/save.go:153,716`; `pkg/game/house_save.go:143`; `pkg/game/other_settings.go:164`; `pkg/game/aliases.go:64,77,112`; `pkg/game/bans.go:94,133,246`; `pkg/boards/boards.go:142,235-236`; `pkg/game/dns.go:218` | deferred | Persistence and filesystem cleanup can affect saved state, reloadability, or administrative state. Preserve the primary storage contract; next experiment is operation-specific injected-close/write proof. |
| `pkg/game/mail.go:187,200,219,254` | retained/deferred under Phase 6.4 | Mail free-list, corruption, multi-block, and concurrency contracts remain explicitly open after #1466. Do not fold mail cleanup into this batch; reopen only with the named lifecycle proof. |
| `pkg/db/narrative_memory.go:273,284` | deferred | `RowsAffected` errors can make persistence counts unknowable. The next experiment must define the database result/count contract rather than silently logging or inventing counts. |
| `pkg/game/inventory.go:290`; `pkg/game/clans.go:334`; `pkg/game/spec_procs_missing.go:65`; equipment lookups in `pkg/game/act_movement.go:782`, `pkg/game/skill_c10_combat.go:21`, `pkg/game/skill_special.go:591`, `pkg/game/skill_special.go:625`, `pkg/game/spec_paladin.go:62`, `pkg/game/skills2.go:201` | retained/deferred | Transfer/equipment/existence-probe results are behavior-sensitive. Trace the C caller and add failure-path state/rollback proof before handling any error. |
| `pkg/game/skill_combat.go:579`; `pkg/game/combat_wire.go:477`; `pkg/game/spec_procs.go:233`; `pkg/session/gen_ps_cmds.go:80` | deferred | Combat entry, corpse extraction, room cleanup, and help reload can change state or player output. Next experiment is a named call-path/failure-policy test, not a lint-only slog. |
| `pkg/game/logging.go:73,96,383`; `pkg/dprng/drawlog.go:49` | retained | These are diagnostic/draw-log writes. Changing their failure path can recurse into logging or conceal draw evidence; keep the current behavior until the diagnostic sink contract is named. |
| `pkg/testutil/helpers.go:273` and all `*_test.go`/example sites | out of production scope | Test support, fixtures, and examples are inventoried separately from the production `pkg/` cleanup batch. No production behavior or test fixture was changed to suppress an error. |
| interface assertions, unused C-compatible parameters, `OneArgument`/`halfChop` remainder strings, map assertions, required RNG draws, and `fmt.Sscanf` validation probes | retained intentional non-error discards | These do not discard an error result. They preserve type checks, C argument consumption, draw count, or boolean probe semantics. |

No site in the retained/deferred table was “fixed” by adding a suppression.
The parser, combat/RNG, transfer/equipment/rollback, persistence, mail, and
transport classes remain open debt rather than a false Phase 6.5 closure claim.

## Tracking and proof boundary

This is one implementation batch, not a claim that all Phase 6.5 production
inventory is complete. The full production census remains the table above plus
the retained intentional/non-error classes. Phase 6.4 remains ruled but not
implemented; PR #1467 is the source of that accepted disposition and is not
duplicated here.

Coverage citations for the implementation checkpoint:

- `pkg/errlog`: `TestCloseReportsFailureWithoutReturningIt` and
  `TestCloseAcceptsNormalCloser`.
- `pkg/agent`: `TestDoMemoryHookWithRetryReturnsRequestBodyReadError`.
- `pkg/agentcli`: `TestHandleEventReturnsEventPersistenceError` plus the
  existing daemon, LLM, and reconnect tests.
- all changed packages: focused tests, then `go build ./...`, `go vet ./...`,
  `go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...`,
  `make fidelity-depth`, and `make expected-divergences-check`.
- final proof: one `make oracle-regression` from a frozen post-implementation
  checkpoint, with the per-attempt logs, input manifest, scenario identity
  reconciliation, and retry review stored outside the repository and linked in
  the final PR description.

## Frozen checkpoint and final census

The implementation and census inputs were frozen at
`3f106fe9a30039fbf0b1da99c10ec57eaa7db872`, with a clean worktree and
`origin/main` at `ebb5feff8b4b6f5a7d76492ae10b6fed3fdd7309`. The SHA-256
manifest covers all 5,430 tracked repository files; its digest is
`697773f2c5164e69a7f3ac3f2088ddcf63305ef283eb0754917d9628840f27a3`.
Production Go, tests, fixtures, scenarios, and drivers were not changed after
that freeze. The only post-census repository change is this handoff update.

Exactly one full `make oracle-regression` ran from that frozen checkpoint:

```text
scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
elapsed=7061.881s
```

Exit status 2 is the established baseline status for the one human-cleared
`accuse-noarg-depth` unpinnable scenario. The nine `EXPECTED` scenarios were
`accuse-depth`, `force-mob`, `medit-entry-depth`, `medit-session-depth`,
`redit-entry-depth`, `redit-session-depth`, `sedit-entry-depth`,
`sedit-session-depth`, and `shoot-target-depth`. There were eight
infrastructure-shaped first attempts (`apologize-sleeping-depth`,
`bounce-depth`, `diagnose-depth`, `give-gold`, `informative-residual-depth`,
`movement-mounted`, `peek-depth`, and `spec-proc-tattoo2-price`); every one
recovered to `PASS` within the runner’s existing retry budget. No scenario
ended `FAIL`, `INFRA`, `TIMEOUT`, or `STALE`.

The final classification identities reconcile exactly: 941 scenario files,
941 unique classified scenario names, no missing names, and no unexpected
names. The durable evidence contains 959 per-attempt logs for those 941
scenarios, the complete console transcript, the freeze check, the input
manifest, and the reconciliation report.

## Measured delta and gates

At the frozen implementation checkpoint, the batch was 25 tracked files, 313
insertions, and 42 deletions: 22 production Go files, two focused test files,
and this handoff document. The final handoff-only proof update adds 64 lines
and removes 3 lines, so the aggregate branch diff is 25 tracked files, 374
insertions, and 42 deletions. No scenario, fixture, driver, oracle, save
format, schema, roadmap, or deployment file changed.

The frozen implementation checkpoint passed:

- `make fmt` and `make check-fmt`;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `go test ./pkg/game/...`;
- `golangci-lint run ./... --color never` (`0 issues`);
- `make fidelity-depth`;
- `make expected-divergences-check`;
- focused changed-package tests and `go test -race ./pkg/agent ./pkg/agentcli ./pkg/optimization ./pkg/errlog`; and
- named oracle smoke checks for `look-start-room`, `advance-depth`, and `dns-depth`, with no normalized divergence.

Durable evidence is outside the repository at
`/home/zach/dp-phase6-5-evidence-20260915-3f106fe9/`, including
`01-make-fmt.log` through `10-race.log`,
`full-oracle-regression-console-run1.log`,
`oracle-attempt-logs-run1/`, `frozen-input-manifest.tsv`,
`freeze-check.log`, and `census-reconciliation.log`.

This remains one coherent implementation batch, not a claim that all Phase 6.5
deferred classes are closed. Deliver the branch as one unmerged PR and stop
for human review; do not merge or deploy.
