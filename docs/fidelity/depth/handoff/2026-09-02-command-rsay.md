# Depth-fidelity handoff — `rsay`

Date: 2026-09-02

## Queue position

This round began from `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus
the latest `rofl` handoff. The special-procedure inventory remains exhausted.
The single blocked row `objmagic.sleep-entry-gates` remains blocked after its
one cast-sleep vehicle using the outlaw/reagent arms; the reachable portion is
covered by the existing sleep-spell evidence and the remaining entry-gate
surface is still recorded as blocked. The interpreter sweep advanced from
`roll` to `rsay`; source order puts `ruffle` next at `src/interpreter.c:668`.

Frontier before this slice: 3,053 total; 2,976 proven/delegated; 26 blocked;
51 excluded.

Frontier after this slice: 3,067 total; 2,990 proven/delegated; 26 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:667 */
{ "rsay"     , POS_RESTING , do_race_say , 0, 0 },
```

The handler is `src/act.comm.c:635-755`. It strips leading literal spaces,
then checks zero wisdom/intelligence and `PLR_NOSHOUT` before the empty-
argument prompt. Non-empty speech selects one of the seven player-race
language tables (`speak_rakshasan`, `speak_elven`, `speak_human`,
`speak_dwarven`, `speak_kender`, `speak_minotaur`, or `speak_ssaur`). The room
loop sends translated text only to awake mortal players of another race;
same-race players, awake immortals, NPCs on the visible descriptor path, and
the actor's self-copy use the original text with the C race label. Punctuation
selects `exclaims`, `asks`, `states`, or `says`; `PRF_NOREPEAT` changes only
the actor confirmation to `Ok.\n\r`. Sleeping recipients are suppressed by
the C `AWAKE` checks.

The Go command wrapper now runs the game prechecks even for empty arguments,
preserves the raw post-command remainder for internal spacing, uses the actual
C player-race constants and labels, and emits the exact C line endings for
the special gate and no-repeat responses. The obsolete invented NPC language
scaffolding was removed once the corrected player-race path made it
unreachable. No `src/` or oracle-tree file was edited.

## Evidence and confirmed parity

Scenarios:

- `cmd/dp-oracle-diff/scenarios/rsay-depth.txt`
- `cmd/dp-oracle-diff/scenarios/rsay-immortal-depth.txt`
- `cmd/dp-oracle-diff/scenarios/rsay-sleeping-depth.txt`
- `cmd/dp-oracle-diff/scenarios/rsay-noshout.txt`

Manifest: `docs/fidelity/depth/rsay.tsv` (14 rows)

Focused tests:

- `pkg/game/rsay_depth_test.go`
- `pkg/session/rsay_depth_test.go`

The clean-main RED baseline exposed the old Go race-number mapping, a session
early return that bypassed the C stupid/noshout gates, the wrong Go
`PLR_NOSHOUT` bit, incomplete player language tables, an incorrect actor
`asks` spelling, missing awake-recipient filtering, and helper-added line
endings that changed C's exact LFCR responses. Each was corrected against
the verified C call path (R1/R2/R5e). A separate compared setup command
exposed an unrelated existing `set <player> level` wording mismatch; it was
not fixed forward because it is outside this slice (R4/R5b/R5c).

The core, immortal-recipient, and sleeping-recipient scenarios all reported
no normalized divergence at seeds 1, 2, 3, 5, and 8. The dedicated noshout
vehicle reported no normalized divergence at seed 1. The focused game and
session tests passed, including exact C player-language golden values and
gate ordering.

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

The optional local `gosec` binary was unavailable; hosted security passed.

Feature branch: `glm/depth-rsay`

Feature commit: `0f7fe457d` (`fix: match rsay depth fidelity`)

Feature PR: #1131 — hosted lint, security, and test checks were green; the
workflow's build-and-push and deploy jobs were skipped by conditions. The PR
was self-merged only after all hosted checks were green, as main commit
`f7693fe41`.

The earlier open PRs for `plot`, `purge`, and `qecho` remain open because
their checks did not fire after their one permitted exact workflow retry; none
was merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic oracle matrix), R4 (no invention), R5 (process discipline),
and R5e (verify the actual C call path). The shared player-flag and language
mapping corrections were audited as class-level behavior under R5b/R5c.
