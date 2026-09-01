# Depth-fidelity handoff — `pardon`

Date: 2026-09-01  
Branch: `glm/depth-pardon`  
Feature commit: `a30174d8a`  
Feature PR: #1059 (merged to `main` as `2936c2e67`)

## Frontier

The clean-main frontier before this slice was 2,732 total cases: 2,661
proven/delegated, 22 blocked, and 49 excluded. After merging `pardon`, it is
2,746 total: 2,675 proven/delegated, 22 blocked, and 49 excluded. Actionable
completion is 2,675/2,697 (99.2%). The special-procedure inventory remains
exhausted; `objmagic.sleep-entry-gates` remains the single explicitly blocked
vehicle; the command-table sweep continues after `pardon`.

## C-first call path

The registration at `src/interpreter.c:601` is `{ "pardon", POS_DEAD,
do_wizutil, 1, SCMD_PARDON }`. The shared handler is
`src/act.wizard.c:2077-2122`:

1. A caller below `LVL_IMMORT` without `PLR_CHOSEN` receives `Huh?!?` from
   the inner do_wizutil authority gate, despite the table's level-1 row.
2. `one_argument` skips fill words and lowercases one target token. Empty,
   missing, visible-NPC, and higher-immortal targets reach their exact shared
   responses before the pardon switch.
3. `SCMD_PARDON` rejects a target without `PLR_OUTLAW` as `Your victim is not
   flagged.`. Otherwise it removes `PLR_OUTLAW`, sends `Pardoned.` to the actor,
   and sends `You have been pardoned by the Gods!` to the target before the
   non-player-facing save/log path.

The shared do_wizutil target resolution and protection machinery remains owned
by the verified `mute.target-gates` boundary under R5b/R5c; this slice proves
the pardon registration, inner authority, parser boundary, and SCMD_PARDON
state/bytes.

## RED and confirmed fixes

The initial Go handler passed only `args[0]` into `wizutilDispatch`. The C
vehicle was RED only for `pardon the <player> ...`: C `one_argument` skips the
leading fill word and resolves the second token, while Go looked up `the` and
reported no such player. The confirmed fix passes the complete tokenized
argument remainder into the existing C-compatible parser. No oracle or C
source was edited.

## Evidence

- Scenarios: `cmd/dp-oracle-diff/scenarios/pardon-depth.txt` and
  `pardon-gates-depth.txt`.
- Manifest: `docs/fidelity/depth/pardon.tsv` (14/14 proven or delegated rows).
- Focused tests: `pkg/session/pardon_depth_test.go`.
- Oracle matrix: seeds `1, 2, 3, 5, 8` for both scenarios, all
  `result: no normalized divergence`.
- `--show-oracle --seed 1` confirmed the C inner authority, NPC, non-outlaw,
  success, fill-word, case-insensitive, and self-target blocks.
- Local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`,
  `go test ./...`, `golangci-lint run ./...`, and clean `gofumpt -l .`.
- Hosted checks for PR #1059 were green after the one permitted exact
  workflow retry (`Dark Pawns CI/CD`); lint, security, and test passed, with
  build/deploy skipped by the workflow.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (the registered command
surface), R3 (deterministic lookup across the seed matrix), R4 (no invented
branches), and R5/R5e (verify the actual C call path). Shared target behavior
is retained under R5b/R5c rather than duplicated.

## Next queue position

Return to clean `main`, pull, run `make fidelity-depth`, reread the depth guide
and this newest handoff, then resweep `src/interpreter.c` against all depth
manifests. `pardon` is claimed; `peer` is a shared `do_action` social already
covered by the social family, and `pick` is owned by the door manifest. The
next unclaimed command family in table order is `peek` at `interpreter.c:602`.
Do not re-pick any command already owned by a manifest or delegated boundary.
