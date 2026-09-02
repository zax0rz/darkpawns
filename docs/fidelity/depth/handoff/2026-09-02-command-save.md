# Depth-fidelity handoff — `save`

Date: 2026-09-02

## Queue position

This round began from `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus
the latest `ruffle` handoff. The special-procedure inventory remains
exhausted. The single blocked row `objmagic.sleep-entry-gates` remains
blocked after its one cast-sleep vehicle using the outlaw/reagent arms; the
reachable portion is covered by the existing sleep-spell evidence and the
remaining entry-gate surface is still recorded as blocked. The interpreter
sweep advanced from `ruffle` to `save`; the next un-manifested family is
`salute` at `src/interpreter.c:673`.

Frontier before this slice: 3,077 total; 3,000 proven/delegated; 26 blocked;
51 excluded.

Frontier after this slice: 3,084 total; 3,006 proven/delegated; 26 blocked;
52 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:672 */
{ "save"     , POS_SLEEPING, do_save     , 0, 0 },
```

The handler is `src/act.other.c:186-203`. It returns silently for an NPC or
descriptor-less character. For a player descriptor, a command invocation
emits `Saving <name>.\r\n`, then calls the C alias writer, immortal poof
writer, player-record save, and crash inventory save. The command does not
use its trailing argument, has no room audience, and is legal at
`POS_SLEEPING`.

The existing Go session path already matched the reachable C acknowledgement
and persistence call boundary. No Go behavior change was warranted after
the clean-main RED/GREEN comparison. The NPC/no-descriptor branch is
unreachable from a player descriptor command and is recorded as excluded;
the shared player-record/crash save seam is delegated to the existing quit
persistence proof. No `src/` or oracle-tree file was edited.

## Evidence and confirmed parity

Scenario: `cmd/dp-oracle-diff/scenarios/save-depth.txt`

Manifest: `docs/fidelity/depth/save.tsv` (7 rows)

Focused test: `pkg/session/save_depth_test.go`

The oracle vehicle proves the exact acknowledgement, ignored trailing text,
and sleeping entry. It reported no normalized divergence at seeds 1, 2, 3,
5, and 8; seed 1 was run with `--show-oracle` and showed each intended C
save block. The focused test pins the POS_SLEEPING, level-0 command gate.

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

Feature branch: `glm/depth-save`

Feature commit: `bf180b3e0` (`test: prove save depth fidelity`)

Feature PR: #1135 — no checks initially reported, so the one permitted exact
retry was run with `gh workflow run "Dark Pawns CI/CD" --ref glm/depth-save`.
Hosted lint, security, and test passed; conditional build-and-push and deploy
jobs were skipped. The PR was self-merged only after the required checks were
green, as main commit `2983ef655`.

Open no-check PRs remain unmerged: plot #1064, purge #1095, qecho feature
#1096, qecho handoff #1097, and roll handoff #1130.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic oracle matrix), R4 (no invention), R5 (process discipline),
and R5e (verify the actual C call path). The shared persistence boundary was
delegated under R5b/R5c.
