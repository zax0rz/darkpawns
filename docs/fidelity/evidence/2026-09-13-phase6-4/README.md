# Phase 6.4 audit evidence — 2026-09-13

This evidence package supports the dated Phase 6.4 audit handoff. It contains
no production, test, scenario, fixture, runner, CI, deploy, website, or
save-format change.

## Source and worktree checkpoints

| item | value |
|---|---|
| worktree | `/home/zach/darkpawns-audit-phase6-4` |
| branch | `glm/audit-phase6-4` |
| base | fresh `origin/main` at `57414fb998f2d1257afd17a62eecb5894a79a482` (merge of #1454) |
| audit date | 2026-09-13 |
| primary checkout edit | `docs/specs/tui-setup-wizard.md` remained untouched |
| open modernization PRs | none returned by `gh pr list --state open` at audit start |
| read-only trees | `src/` and `darkpawns-c-oracle/` |

The relevant current-source hashes at this checkpoint are:

| file | SHA-256 |
|---|---|
| `pkg/game/mail.go` | `586d94b659dd567b9424fd1316c339ef783d4d6c97dd6f065b57aded27b892f4` |
| `pkg/game/weather.go` | `fdf7acc2cf16feb66156e78f7d5e430d953c249b76932d13e4b372bb30664496` |
| `pkg/game/merge_bridge.go` | `613f66575e4462a6ac429b3ac3554fc18ef613da4a72b869e3ae28dacfdb30e4` |
| `pkg/game/bans.go` | `ef7d229df2c85575dc4da13e8acec3e553d18c0f9df49f343b0392b3e66ccef6` |
| `pkg/game/spec_assign.go` | `e9f1f80de94307c658c08d51172b980846b222e933c17222975c5d19252e931e` |

Historical report excerpts and their hashes are preserved in
[`original-excerpts.md`](original-excerpts.md).

## Census reuse boundary

The recovered Phase 6.3 census was run at documentation checkpoint
`ab916e4b21d4b99a1740521846c21b9b90b7dd23`. The exact comparison

```text
git diff --name-status ab916e4b21d4b99a1740521846c21b9b90b7dd23 HEAD
```

contains only documentation paths: the Phase 6.3 provenance handoff and
evidence, the recovered census output, this Phase 6.4 handoff and evidence,
and the roadmap. The later merge to current `origin/main` also changes only
`docs/`. No production, scenario, fixture, or runner input changed between
the actual census checkpoint and this audit base, so the aggregate is reusable
for the unchanged corpus. This is a reuse justification, not a claim that the
recovered stream is complete.

The exact path set in the reviewed diff is:

```text
M  docs/fidelity/depth/handoff/2026-09-13-modernization-phase6-3-provenance.md
A  docs/fidelity/depth/handoff/2026-09-13-modernization-phase6-4-audit.md
M  docs/fidelity/evidence/2026-09-13-phase6-3-provenance/original-excerpts.md
A  docs/fidelity/evidence/2026-09-13-phase6-3-provenance/recovered-census-output.txt
A  docs/fidelity/evidence/2026-09-13-phase6-4/README.md
A  docs/fidelity/evidence/2026-09-13-phase6-4/original-excerpts.md
M  docs/modernization/06-roadmap.md
```

The durable recovered output is
[`recovered-census-output.txt`](../2026-09-13-phase6-3-provenance/recovered-census-output.txt).
Its exact aggregate is:

```text
oracle-regression: scenarios=941 passed=931 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0 elapsed=7066.739s started=2026-09-13T08:49:05-0400 finished=2026-09-13T10:46:52-0400
```

The recovered output preserves 856 status lines (847 PASS, 8 EXPECTED, 1
UNPINNABLE) and the final aggregate, but 85 scenario status lines and all
temporary per-attempt logs are absent. Therefore it supports the aggregate
and the known human-cleared `accuse-noarg-depth` baseline, but cannot
independently reconcile all 941 scenario identities or inspect deleted
attempt logs. No missing output is reconstructed. The audit treats this as
aggregate-only evidence and uses fresh focused runs below for the named
families.

## Fresh focused unit evidence

Commands and complete output are preserved outside self-cleaning directories:

| package | command family | result | durable output |
|---|---|---|---|
| `pkg/game` | named mail, weather, ban, registry, object-special, and bridge tests | PASS, `0.079s` | `/home/zach/dp-phase6-4-focused-game-2026-09-13.log` |
| `pkg/session` | named time/weather, manager-ban, and active-character tests | PASS, `0.024s` | `/home/zach/dp-phase6-4-focused-session-2026-09-13.log` |
| `pkg/game` | registry smoke, bank contract, and object-receiver tests | PASS, `0.021s` | `/home/zach/dp-phase6-4-focused-spec-2026-09-13.log` |

The selected test names and proof limits are listed in the handoff; these are
unit/fixture seams, not a substitute for end-to-end restart, transport, or
full-pulse evidence.

## Fresh focused oracle evidence

The frozen matrix was executed with:

```text
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle
/home/zach/dp-phase6-4-oracle-diff-2026-09-13 --scenario <name> --seed <seed>
```

The complete output is preserved at
`/home/zach/dp-phase6-4-oracle-focused-2026-09-13.log`. The matrix is
recorded in the handoff with scenario names, seed sets, and the final
reconciled tally: 33 unique selected runs passed, with failed=0, infra=0,
timed_out=0, and stale=0. The preserved base log has 32 completed PASS
report blocks: `spec-proc-bank` contains seeds 1, 2, 5, and 8, so seed 3 was
missing. It also contains one incomplete C-oracle readiness diagnostic
(`bind: Address already in use`, lines 86-228), which has no result row and
is not counted as a completed run. No completed scenario/seed pair is
duplicated. Seed 3 was rerun on unchanged inputs; its complete result is
preserved at
`/home/zach/dp-phase6-4-oracle-spec-proc-bank-seed3-2026-09-13.log` and is
`result: no normalized divergence`. The repeated seed-1 `info-basic` smoke
run does not substitute for the missing bank seed. One seed-1 `info-basic`
run was repeated with `--show-oracle` and its normalized C blocks are
preserved at
`/home/zach/dp-phase6-4-oracle-smoke-info-basic-seed1-2026-09-13.log`.

The rerun used the same built harness and frozen scenario input:

```text
PATH=/usr/local/go/bin:$PATH \
DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle \
/home/zach/dp-phase6-4-oracle-diff-2026-09-13 --scenario spec-proc-bank --seed 3
```

Harness SHA-256: `67cf2b002b521256f49bd4b668f76210bae12bd35b3347600dd474192b07a5a0`.
Scenario SHA-256: `66c180a56d53533d7ed14ee3e02e79e66a582625fd363352273a331e35930a45`.

## Review correction: weather lock re-entry

The current live path has a separate concrete locking defect. The production
heartbeat invokes `OnWeatherAndTime` every 63 seconds at
`pkg/engine/gameloop.go:325-328`; `cmd/server/main.go:324-326` calls
`game.WeatherAndTime(true, ...)`; and `pkg/game/weather.go:313-319` holds
`weatherMu.Lock()` across `AnotherHour`. When the pre-increment hour is 4 or
20, `AnotherHour` reaches the hour-5 or hour-21 event helpers, each of which
calls `weatherMu.RLock()` before checking `weatherWorld` (`weather.go:329-351,
533-603`). A Go `sync.RWMutex` cannot be re-entered this way, so the heartbeat
blocks. The manual `tick` path reaches the same call at
`pkg/session/wiz_system.go:732-740`.

The existing tests miss this because they call `AnotherHour` or the event
helpers directly without the enclosing write lock, or call
`WeatherAndTime(false)` from hour 8. The selected info pulse advances hour 14
to 15 and does not reach an event hour. The C comparison is
`src/comm.c:825-831` → `src/weather.c:41-80`; no production fix or
weather/RNG/scheduler refactor is included here. This defect is now the
highest-priority proof/triage task, ahead of the mail lifecycle proof.

## Validation record

The final documentation checkpoint passed every required documentation-PR
gate:

| gate | result |
|---|---|
| `gofumpt -l .` | PASS; no files listed |
| `git diff --check` | PASS |
| `/usr/local/go/bin/go build ./...` | PASS |
| `/usr/local/go/bin/go vet ./...` | PASS |
| `/usr/local/go/bin/go test ./...` | PASS |
| `/usr/local/go/bin/go test ./pkg/game/...` | PASS |
| `golangci-lint run ./...` | PASS; 0 issues |
| `make fidelity-depth` | PASS; 4816 total, 4697 proven/delegated, 68 blocked, 51 excluded |
| `make expected-divergences-check` | PASS; 26 rows across 10 scenarios; pins OK |

Complete command outputs are preserved at:

- `/home/zach/dp-phase6-4-gofumpt-review-2026-09-13.log`
- `/home/zach/dp-phase6-4-diff-check-review-2026-09-13.log`
- `/home/zach/dp-phase6-4-build-review-2026-09-13.log`
- `/home/zach/dp-phase6-4-vet-review-2026-09-13.log`
- `/home/zach/dp-phase6-4-test-all-review-2026-09-13.log`
- `/home/zach/dp-phase6-4-test-game-review-2026-09-13.log`
- `/home/zach/dp-phase6-4-lint-review-2026-09-13.log`
- `/home/zach/dp-phase6-4-fidelity-depth-review-2026-09-13.log`
- `/home/zach/dp-phase6-4-expected-divergences-review-2026-09-13.log`
