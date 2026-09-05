# Depth-fidelity handoff — `reroll`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `recall` handoff. The special-procedure inventory remains exhausted, the
one blocked row `objmagic.sleep-entry-gates` remains queued after its single
cast-sleep vehicle, and the interpreter sweep advanced from `recall` to
`reroll`.

The source-order audit did not repick `rescue`, `retreat`, `retrieve`, or the
other intervening rows already owned by dedicated/shared manifests. The next
unclaimed interpreter-table family is `restore` at `src/interpreter.c:650`.

Frontier before this slice: 2,958 total; 2,885 proven/delegated; 22 blocked;
51 excluded.

Frontier after this slice: 2,968 total; 2,895 proven/delegated; 22 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:649 */
{ "reroll"   , POS_DEAD    , do_wizutil  , LVL_GRGOD, SCMD_REROLL },
```

`src/act.wizard.c:2077-2111` first applies the shared `do_wizutil` gates:
non-Implementor callers without `PLR_CHOSEN` receive `Huh?!?`, a missing
argument receives `Yes, but for whom?!?`, an unresolved visible target receives
`There is no such player.`, a visible NPC receives `You can't do that to a
mob!`, and a higher immortal target receives `Hmmm...you'd better not.`.
Target parsing uses C `one_argument`, so leading fill words are skipped and
trailing words are ignored.

The `SCMD_REROLL` branch calls `roll_real_abils()` from `src/class.c:380-497`,
copies the resulting class/race-dependent abilities into the target, updates
`GET_ORIG_CON`, emits `Rerolled...` followed by the exact new-stat line to the
actor only, and saves the target without additional player-facing output.
There is no victim or room announcement.

## Evidence and confirmed divergence

Scenario: `cmd/dp-oracle-diff/scenarios/reroll-depth.txt`

Manifest: `docs/fidelity/depth/reroll.tsv` (10 rows)

Focused tests: `pkg/session/reroll_depth_test.go`

The first proof vehicle exposed a harness capacity/setup error and was
discarded rather than treated as evidence. The corrected vehicle uses a fresh
Implementor primary, two peers, a level-38 caster, room 8162, and registered
trainee mob 16303; the caster exercises no-argument, missing-target, NPC,
higher-immortal, successful, and fill-word/trailing-input branches.

Clean `main` was RED: Go invented `Usage: reroll <player>` for no arguments,
failed C's fill-word/trailing-input target boundary, emitted `Rerolled!` rather
than `Rerolled...`, omitted the C `StrAdd` field, and did not perform the C
stat/state mutation. The corrected Go path now uses C argument parsing,
`RollRealAbils`, the exceptional-strength value, inventory strength, and
`OrigCon` update. No `src/` or C-oracle file was edited.

Seeds 1, 2, 3, 5, and 8 were GREEN after the fix; seed 1 was also run with
`--show-oracle`. The manifest records the command entry, all reachable target
gates, one-argument boundary, actor-only audience, state mutation, and the
shared target-gate delegation to `mute.target-gates`.

## Verification and integration

All local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Feature branch: `glm/depth-reroll`

Feature commit: `d7d8205d3` (`fix: match reroll depth fidelity`)

Feature PR: #1110 — hosted lint, security, and test checks were green;
build-and-push and deploy were skipped by workflow conditions. It was
self-merged as main commit `f9f046ea4` only after the required hosted checks
were green.

The earlier open documentation PRs for `plot`, `purge`, and `qecho` remain
open because their checks did not fire after their one permitted exact
workflow retry; none was merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and draw parity), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The shared target boundary and source-order
claim are maintained under R5b/R5c.
