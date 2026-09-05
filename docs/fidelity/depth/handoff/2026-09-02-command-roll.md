# Depth-fidelity handoff — `roll`

Date: 2026-09-02

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `rofl` handoff. The special-procedure inventory remains exhausted. The
single blocked row `objmagic.sleep-entry-gates` remains blocked after its one
cast-sleep vehicle using the outlaw/reagent arms; the reachable portion is
covered by the existing sleep-spell evidence and the remaining entry-gate
surface is still recorded as blocked. The interpreter sweep advanced from
`rofl` to `roll`. The source-order audit confirms that `ride` is owned by the
`mount` manifest and `roomflags` by `gen-tog`; the next unclaimed family is
`rsay` at `src/interpreter.c:667`.

Frontier before this slice: 3,042 total; 2,965 proven/delegated; 26 blocked;
51 excluded.

Frontier after this slice: 3,053 total; 2,976 proven/delegated; 26 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:664 */
{ "roll" , POS_DEAD, do_roll, 0, 0 },
```

The handler is `src/act.other.c:1927-1946`. It consumes one argument with
`one_argument()`, defaults a missing, zero, or malformed integer to 100 via
`atoi()`, draws once with `number(1, max_roll)`, and emits one actor line plus
one room line. The parser keeps a decimal prefix, ignores trailing words, and
accepts a leading plus. Because `max_roll` is unsigned, a negative parsed
`int` follows C's conversion path into the unsigned range before the single
`number()` call; the oracle scenario preserves that stream behavior.

`src/utils.c:53-64` is the RNG call path. Its bounds swap when the lower bound
exceeds the upper bound and its result is the C float-based inclusive draw.
The Go implementation was compared against that path rather than inferred
from the command text.

## Evidence and confirmed parity

Scenario: `cmd/dp-oracle-diff/scenarios/roll-depth.txt`

Manifest: `docs/fidelity/depth/roll.tsv` (11 rows)

Focused tests:

- `pkg/game/roll_depth_test.go`
- `pkg/session/roll_depth_test.go`

The clean-main RED baseline matched C for no argument, an explicit maximum,
trailing words, and the actor/room audience. It exposed three confirmed
divergences: zero was incorrectly clamped to `(1-1)` instead of defaulting to
100; malformed input returned a Go parse error and omitted the room line
instead of following C's zero/default path; and the malformed branch desynced
the subsequent `6junk` draw. Additional probes confirmed C's decimal-prefix,
leading-plus, and unsigned-negative behavior.

The Go fix in `pkg/game/other_utility.go` uses the C argument parser and
defaulting rules, performs the unsigned/int32 conversion with explicit range
checks, and consumes exactly one draw. No `src/` or oracle-tree file was
edited. The normal scenario ran at seeds 1, 2, 3, 5, and 8; seed 1 also ran
with `--show-oracle`. Every run reported no normalized divergence.

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
gosec -severity=high -quiet ./...  # clean
```

Feature branch: `glm/depth-roll`

Feature commits: `fd8f89dcd` and corrective `ec934cc99` (`fix: match C roll
argument and unsigned conversion`)

Feature PR: #1129. The initial hosted run found a real G115 finding in the
new conversion code; the corrective code removed that finding without adding
a suppression. Corrected hosted run #33590431654 had lint, security, and test
checks green; build-and-push and deploy were skipped by workflow conditions.
The PR was self-merged only after the hosted checks were green, as main commit
`3d87b17f228b5557dc48846aff6910653573ade4`.

The earlier open PRs for `plot`, `purge`, and `qecho` remain open because their
checks did not fire after their one permitted exact workflow retry; none was
merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic draw parity), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The C integer-conversion behavior was
checked as a shared parsing/RNG class under R5b/R5c.
