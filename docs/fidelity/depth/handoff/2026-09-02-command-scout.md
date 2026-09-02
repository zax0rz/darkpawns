# Depth-fidelity handoff — `scout`

Date: 2026-09-02

## Queue position

This round began from `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus
the latest `scoff` handoff. The special-procedure inventory remains
exhausted. The single blocked row `objmagic.sleep-entry-gates` remains
blocked after its one cast-sleep vehicle using the outlaw/reagent arms; the
reachable portion is covered by the existing sleep-spell evidence and the
remaining entry-gate surface is still recorded as blocked. The interpreter
sweep advanced from `scoff` to `scout`; `scream` is already represented only
in existing shared evidence and has no command-family manifest, so the next
un-manifested family is `scratch` in the next table position after `scout`.

Frontier before this slice: 3,101 total; 3,023 proven/delegated; 26 blocked;
52 excluded.

Frontier after this slice: 3,118 total; 3,040 proven/delegated; 26 blocked;
52 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:676 */
{ "scout"     , POS_STANDING, do_scout    , 0, 0 },
```

The handler is `src/act.other.c:1826-1922`. It applies C's
`one_argument()` parsing, then gates no argument, invalid direction, missing
skill, and `OUTSIDE(ch)` before checking the destination exit. Success emits
the destination terrain description and then independently reports darkness,
death, tunnel, objects on the ground, and the C `crowd_size[]` people
description. Direction output uses the canonical `dirs[dir]` spelling; the
outdoor macro permits a non-indoors room or a non-inside sector even when an
indoors flag is present. The handler's exact bytes and branch ordering were
read from the C source under R5e.

The clean-main RED vehicles confirmed the Go parser, outdoor-gate, canonical
terrain, warning, object, and crowd divergences. The Go implementation was
updated only for those confirmed mismatches; `src/` and the C oracle tree
were not edited.

## Evidence and confirmed parity

Scenarios:

```text
cmd/dp-oracle-diff/scenarios/scout-depth.txt
cmd/dp-oracle-diff/scenarios/scout-no-skill-depth.txt
cmd/dp-oracle-diff/scenarios/scout-gates-depth.txt
cmd/dp-oracle-diff/scenarios/scout-success-depth.txt
```

Manifest: `docs/fidelity/depth/scout.tsv` (17 rows)

Focused tests:

```text
pkg/game/scout_depth_test.go
pkg/session/scout_depth_test.go
```

All four vehicles reported no normalized divergence at seeds 1, 2, 3, 5,
and 8. Seed 1 was run with `--show-oracle`; it covered the invalid/no-exit
branch, fill-word first-argument parsing, city terrain, darkness, death,
tunnel, object, and people descriptions. The game test pins every C terrain
description and crowd-size boundary; the session test pins the
`POS_STANDING` command gate.

## Verification and integration

All required local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature branch: `glm/depth-scout`

Feature commit: `8c90dc681` (`fix: match scout depth fidelity`)

Feature PR: #1141 — hosted lint, security, and test passed; conditional
build-and-push and deploy jobs were skipped. The PR was self-merged only
after all required checks were green, as main commit `255fc46d4`.

Open no-check PRs remain unmerged: plot #1064, purge #1095, qecho feature
#1096, qecho handoff #1097, and roll handoff #1130.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic oracle matrix), R4 (no invention), R5 (process discipline),
and R5e (verify the actual C call path). The shared command-gate and parser
class evidence was reused under R5b/R5c.
