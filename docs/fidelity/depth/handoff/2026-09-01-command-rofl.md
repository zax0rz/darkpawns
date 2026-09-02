# Depth-fidelity handoff — `rofl`

Date: 2026-09-01

## Queue position

This round began from `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `roar` handoff. The special-procedure inventory remains exhausted. The
single blocked row `objmagic.sleep-entry-gates` remains blocked after its one
cast-sleep vehicle using the outlaw/reagent arms; the reachable portion is
covered by the existing sleep-spell evidence and the remaining entry-gate
surface is still recorded as blocked. The interpreter sweep advanced from
`roar` to `rofl`. The source-order audit confirms that `ride` is owned by the
`mount` manifest and `roomflags` by `gen-tog`. The next unclaimed family is
`roll` at `src/interpreter.c:664`.

Frontier before this slice: 3,032 total; 2,955 proven/delegated; 26 blocked;
51 excluded.

Frontier after this slice: 3,042 total; 2,965 proven/delegated; 26 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:663 */
{ "rofl" , POS_RESTING, do_action, 0, 0 },
```

The shared handler is `src/act.social.c:102-151`. It enforces the command
position and `PLR_NOSHOUT` gates, parses the first target token, checks target
visibility and position, and selects the social record's no-argument,
target, self-target, or target-not-found branch. The social record in
`lib/misc/socials:1247-1255` is `rofl 0 5`: hide level 0, minimum victim
position `POS_RESTING`, and eight authored message slots. The target-success
branch has actor, room, and victim audiences; no-argument and self-target
branches have actor and room audiences; missing targets are actor-only.

The actual C self-target branch sends `char_auto` directly with
`send_to_char()`. Therefore its authored `$n` and `$mself` codes remain
literal bytes. Other social branches use `act()` substitution. The existing Go
dispatcher incorrectly sent the self-target `char_auto` through `Act`,
expanding those codes before the fix.

## Evidence and confirmed parity

Scenarios:

- `cmd/dp-oracle-diff/scenarios/rofl-depth.txt`
- `cmd/dp-oracle-diff/scenarios/rofl-sleeping-depth.txt`

Manifest: `docs/fidelity/depth/rofl.tsv` (10 rows)

Focused tests:

- `pkg/game/rofl_depth_test.go`
- `pkg/session/rofl_depth_test.go`

The clean-main RED baseline matched C for the normal no-argument, target,
missing-target, and most self-target output, but exposed the exact self-target
divergence: C emitted
`$n rolls on the floor laughing at $mself.\r\n`, while Go expanded it to
`Roflactor rolls on the floor laughing at himself.\r\n`. The shared Go fix in
`pkg/game/act_social.go` preserves the direct-send bytes for `char_auto`; no
oracle or `src/` files were edited.

The normal scenario proves the no-argument, first-token parsing, target
success/audiences, self-target, and missing-target branches. The sleeping
target scenario proves the social record's victim-position gate. The
registration test proves `POS_RESTING`, level 0, and the parsed C social
metadata, including `MinLevel=0`, `HideFlag=5`, and no explicit override.
The normal and sleeping scenarios were run at seeds 1, 2, 3, 5, and 8; seed 1
was also run with `--show-oracle`. Every run reported no normalized
divergence.

## Verification and integration

All required local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature branch: `glm/depth-rofl`

Feature commit: `ac5744114` (`fix: preserve C rofl self-target bytes`)

Feature PR: #1127 — hosted lint, security, and test checks were green; the
workflow's build-and-push and deploy jobs were skipped by conditions. The
workflow completed successfully and the PR was self-merged only after all
hosted checks were green, as main commit `98648e881e397201148b6c8398d89dc7444095aa`.

The earlier open PRs for `plot`, `purge`, and `qecho` remain open because
their checks did not fire after their one permitted exact workflow retry; none
was merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic oracle matrix), R4 (no invention), R5 (process discipline),
and R5e (verify the actual C call path). The direct-send self-target behavior
was audited as a shared social-class rule under R5b/R5c.
