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
`5c0f517e8c888f0e1dfe4d22a3c792e69e9232a7`. The exact comparison

```text
git diff --name-status 5c0f517e8c888f0e1dfe4d22a3c792e69e9232a7 HEAD
```

contains only the Phase 6.3 provenance handoff, its original-excerpt evidence,
and the recovered census output. The later merge to current `origin/main`
also changes only `docs/`. No production, scenario, fixture, or runner input
changed between the census checkpoint and this audit base, so the aggregate
is reusable for the unchanged corpus. This is a reuse justification, not a
claim that the recovered stream is complete.

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
reconciled tally: 33/33 selected runs passed, with failed=0, infra=0,
timed_out=0, and stale=0. One seed-1 `info-basic` run was repeated with `--show-oracle`
and its normalized C blocks are preserved at
`/home/zach/dp-phase6-4-oracle-smoke-info-basic-seed1-2026-09-13.log`.

## Validation record

The final documentation checkpoint records `gofumpt -l .`, build, vet, full
tests, game tests, lint, `make fidelity-depth`, and
`make expected-divergences-check`. The exact command results and the
single-baseline census limitation are in the handoff.
