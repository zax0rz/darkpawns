# Depth-fidelity handoff — `salute`

Date: 2026-09-02

## Queue position

This round began from `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus
the latest `save` handoff. The special-procedure inventory remains
exhausted. The single blocked row `objmagic.sleep-entry-gates` remains
blocked after its one cast-sleep vehicle using the outlaw/reagent arms; the
reachable portion is covered by the existing sleep-spell evidence and the
remaining entry-gate surface is still recorded as blocked. The interpreter
sweep advanced from `save` to `salute`; `score` is already owned by the
existing `info` family. The next un-manifested family is `scoff` at
`src/interpreter.c:675`.

Frontier before this slice: 3,084 total; 3,006 proven/delegated; 26 blocked;
52 excluded.

Frontier after this slice: 3,094 total; 3,016 proven/delegated; 26 blocked;
52 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:673 */
{ "salute"   , POS_RESTING , do_action   , 0, 0 },
```

The shared handler is `src/act.social.c:102-151`. It resolves the `salute`
record, checks `PLR_NOSHOUT`, parses only the first target token, and selects
the no-argument, target-success, self-target, or target-not-found branch. The
record in `lib/misc/socials:1262-1270` is `salute 0 5`: no minimum level, a
minimum victim position of `POS_RESTING`, and exact actor, room, victim,
missing-target, self-target, and no-argument strings. The Go social metadata
stores that legacy record as `MinLevel=0`, `HideFlag=POS_RESTING`, and
`MinVictimPosition=0`; its derived victim-position gate is therefore the
record's `5`.

The target-position vehicle confirmed that a sleeping victim receives no
social bytes while the actor receives C's exact improper-position response.
No Go behavior change was warranted after the clean-main RED/GREEN
comparison. No `src/` or oracle-tree file was edited.

## Evidence and confirmed parity

Scenarios:

- `cmd/dp-oracle-diff/scenarios/salute-depth.txt`
- `cmd/dp-oracle-diff/scenarios/salute-sleeping-depth.txt`

Manifest: `docs/fidelity/depth/salute.tsv` (10 rows)

Focused test: `pkg/session/salute_depth_test.go`

The oracle vehicles cover no argument, first-token parsing with trailing
words, target-success actor/room/victim audiences, self-target,
target-not-found, and the sleeping-target position gate. Both scenarios
reported no normalized divergence at seeds 1, 2, 3, 5, and 8; the seed-1
core run used `--show-oracle` to verify the intended C blocks. The focused
test pins the POS_RESTING entry gate and all eight parsed social message
slots. An initial focused-test failure was corrected as a test expectation
against the repository's legacy social-field representation; it did not
indicate a live Go/C divergence.

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

Feature branch: `glm/depth-salute`

Feature commits: `b5266df5c` (`test: pin salute social metadata`) and
`f3e82d45d` (`test: prove salute depth fidelity`)

Feature PR: #1137 — hosted lint, security, and test passed; conditional
build-and-push and deploy jobs were skipped. The PR was self-merged only
after all required checks were green, as main commit `c6731fdef`.

Open no-check PRs remain unmerged: plot #1064, purge #1095, qecho feature
#1096, qecho handoff #1097, and roll handoff #1130.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic oracle matrix), R4 (no invention), R5 (process discipline),
and R5e (verify the actual C call path). Shared do_action gates and social
record behavior were handled as class-level evidence under R5b/R5c.
