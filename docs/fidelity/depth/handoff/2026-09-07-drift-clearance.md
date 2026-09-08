# Drift-clearance round — final handoff

Date: 2026-09-07
Branch: `glm/drift-clearance`
Base: `origin/main` (`dc81d42e06e5b13bc67c3d694db6090adc06673c`)

## Binding state and scope

This round followed `/home/zach/dp-handoff-brief-2026-09-05.md`,
`docs/fidelity/RULEBOOK.md`, and `docs/fidelity/DEPTH_TESTING.md`. The C source
under `src/` and `darkpawns-c-oracle/` were read-only throughout.

The prior-session WIP directory `/tmp/dp-fidelity-fixes` was gone. Its boards
and combat work was already present in merged PR #1396 (`2b639c18a`), so no
salvage was copied forward.

## Seventeen-scenario clearance

The live census evidence is under
`/home/zach/dp-census-real-gate-2026-09-05-live/triage-premod-2026-09-06/`.
Its printed diffs supplied the work order and the two-attempt pre-modernization
record. On this branch the final targeted run was:

```text
oracle-regression: scenarios=17 passed=17 expected=0 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0
```

The cleared scenarios, in the census order, were:

`afk-depth`, `backstab-aware-trio`, `ban-depth`, `boards-depth`,
`combat-entry-rescue-fighting`, `combat-trip-opener`, `commands-depth`,
`help-immortal-depth`, `invis-depth`, `invis-threshold-depth`,
`lines-infobar-depth`, `movement-followers`, `player-state-toggles`,
`spec-proc-cityguard-breed`, `spec-proc-cityguard`,
`spec-proc-dragon-breath-combat`, and `spec-proc-suck-in`.

The first fourteen already had green/proven coverage in the current depth
ledger after the earlier modernization work; this round re-ran them and
preserved their green status. The three special-procedure ledger rows that
were still blocked were promoted only after five-seed proof:

- `mob.cityguard-breed-killer` → `oracle-green-multiseed`,
  `spec-proc-cityguard-breed@1,2,3,5,8`.
- `mob.cityguard-pulse-dispatch` → `oracle-green-multiseed`,
  `spec-proc-cityguard@1,2,3,5,8`.
- `mob.dragon-breath-combat-transcript` → `oracle-green-multiseed`,
  `spec-proc-dragon-breath-combat@1,2,3,5,8`.

Before/after ledger count for those rows: `blocked 3` → `blocked 0`,
`oracle-green-multiseed 99` → `102`. The other blocked/excluded rows were not
reclassified.

## Fixes and evidence

- `cmd/dp-oracle-diff/main.go`: every scenario now gets a disposable Go
  `lib/{world,text}` copy, preventing `data/world_state.json` from leaking
  between runs. `room-desc-exits` was manually checked at seeds 1 and 2; its
  prior suffix shimmer was runtime-state contamination, not a lookup bug.
- `pkg/combat/engine.go` and `pkg/combat/engine_test.go`: preserve C's hit and
  damage draws while suppressing a second death transcript when a later mobile
  special sees a `POS_DEAD` victim. Coverage is the cityguard seed-5 diff and
  the five-seed cityguard/dragon combat matrix.
- `pkg/session/wiz_stats.go`: report the already one-based no-database player
  ID directly, matching C `GET_IDNUM`; `wizard-valid-reports-depth` passes at
  seeds 1 and 2 and in the corpus.
- `cmd/dp-oracle-diff/scenarios/house-depth.txt` and the two key-seller
  scenarios: updated stale fixture comments/owners from the old no-database
  ID 0 to the current C-shaped ID 1. `house-depth` and both key-seller cases
  pass in focused and full-corpus runs.
- `scripts/census-nightly.sh`: runs a timestamped full census from a detached
  `origin/main` worktree, stores per-run output under
  `/home/zach/dp-census-nightly/`, and appends exit summaries to
  `exit-codes.log`. It was smoke-tested with `room-desc-exits`; no cron was
  installed. Zach can install the desired schedule, for example:
  `0 3 * * * /home/zach/darkpawns/scripts/census-nightly.sh`.

## Corpus and gates

The final build-once corpus run completed all 934 scenarios:

```text
oracle-regression: scenarios=934 passed=924 expected=9 unpinnable=1 stale=0 failed=0 infra=0 timed_out=0
```

The one unpinnable result is the pre-existing `accuse-noarg-depth` run-varying
shape; the driver exits 2 for that known human-clearance item. There were no
new failed, stale, infrastructure, or timed-out results. The prior census
baseline was `passed=922 expected=9 unpinnable=1 stale=0 failed=0 infra=2
timed_out=0`, so the final corpus delta is `922` → `924` PASS and `infra 2` →
`0`.

`make expected-divergences-check` passes with 26 ledger rows across 10
scenarios and zero unresolved proof references. `make fidelity-depth` reports
4,760 cases: 4,658 proven/delegated, 51 blocked, and 51 excluded (98.9%
actionable completion).

The standard gates all pass:

- `/usr/local/go/bin/go build ./...`
- `/usr/local/go/bin/go vet ./...`
- `/usr/local/go/bin/go test ./...`
- `golangci-lint run ./...` — 0 issues
- `gofumpt -l .` — clean
- `git diff --check` — clean

No RED-set scenario was touched without the documented source/evidence review.
