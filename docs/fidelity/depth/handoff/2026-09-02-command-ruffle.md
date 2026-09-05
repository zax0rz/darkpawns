# Depth-fidelity handoff — `ruffle`

Date: 2026-09-02

## Queue position

This round began from `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus
the latest `rsay` handoff. The special-procedure inventory remains exhausted.
The single blocked row `objmagic.sleep-entry-gates` remains blocked after its
one cast-sleep vehicle using the outlaw/reagent arms; the reachable portion is
covered by the existing sleep-spell evidence and the remaining entry-gate
surface is still recorded as blocked. The interpreter sweep advanced from
`rsay` to `ruffle`; `say` and its apostrophe alias are already owned by the
existing `comm.tsv` family. The next un-manifested family is `save` at
`src/interpreter.c:672`.

Frontier before this slice: 3,067 total; 2,990 proven/delegated; 26 blocked;
51 excluded.

Frontier after this slice: 3,077 total; 3,000 proven/delegated; 26 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:668 */
{ "ruffle"   , POS_STANDING, do_action   , 0, 0 },
```

The shared handler is `src/act.social.c:102-151`. It finds the social record,
checks `PLR_NOSHOUT`, parses only the first target token, and selects the
no-argument, target-success, self-target, or target-not-found branch. The
record in `lib/misc/socials:650-658` is `ruffle 0 0`: no minimum level, no
invisibility hiding, and no victim-position restriction. Its target-success
arm therefore reaches actor, non-victim room, and victim audiences even when
the target is sleeping. The record's exact actor, room, victim, missing-target,
self, and no-argument strings were checked against the parsed Go record.

## Evidence and confirmed parity

Scenarios:

- `cmd/dp-oracle-diff/scenarios/ruffle-depth.txt`
- `cmd/dp-oracle-diff/scenarios/ruffle-sleeping-depth.txt`

Manifest: `docs/fidelity/depth/ruffle.tsv` (10 rows)

Focused test:

- `pkg/session/ruffle_depth_test.go`

The clean-main RED/GREEN probe reported no normalized divergence without a Go
behavior change. It covered the no-argument actor/room pair, first-token
target parsing with trailing words, target-success actor/room/victim bytes,
self-target, and missing-target output. The sleeping-target vehicle confirmed
that the record's zero minimum victim position reaches the success arm while
C's audience delivery remains exact. The focused test pins the POS_STANDING
entry gate and all eight parsed social message slots.

Both vehicles reported no normalized divergence at seeds 1, 2, 3, 5, and 8;
the seed-1 core run used `--show-oracle` to verify the intended C block.
No `src/` or oracle-tree file was edited.

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

Feature branch: `glm/depth-ruffle`

Feature commit: `c7e992cc8` (`test: prove ruffle depth fidelity`)

Feature PR: #1133 — no checks were initially reported, so the one permitted
exact retry was run with `gh workflow run "Dark Pawns CI/CD" --ref
glm/depth-ruffle`. Hosted lint, security, and test then passed; conditional
build-and-push and deploy jobs were skipped. The PR was self-merged only
after the required checks were green, as main commit `6d109019d`.

Open no-check PRs remain unmerged: plot #1064, purge #1095, qecho feature
#1096, qecho handoff #1097, and roll handoff #1130.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic oracle matrix), R4 (no invention), R5 (process discipline),
and R5e (verify the actual C call path). Shared do_action gates and social
record behavior were handled as class-level evidence under R5b/R5c.
