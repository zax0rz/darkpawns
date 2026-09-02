# Depth-fidelity handoff — `roar`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `rlist` handoff. The special-procedure inventory remains exhausted, the
single blocked row `objmagic.sleep-entry-gates` remains queued after its one
cast-sleep vehicle, and the interpreter sweep advanced from `rlist` to `roar`.
The source-order audit confirms that `ride` is owned by the `mount` manifest
and `roomflags` by `gen-tog`. The next unclaimed interpreter-table family is
`rofl` at `src/interpreter.c:663`.

Frontier before this slice: 3,023 total; 2,946 proven/delegated; 26 blocked;
51 excluded.

Frontier after this slice: 3,032 total; 2,955 proven/delegated; 26 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:662 */
{ "roar"      , POS_RESTING , do_action, 0, 0 },
```

`src/act.social.c:102-151` implements the shared `do_action` path. It first
rejects unsupported action records and `PLR_NOSHOUT`, then uses the social
record's `char_found` presence to decide whether to consume the first
`one_argument()` token. Roar has `hide=0`, minimum victim position 0, and all
eight authored message slots in `lib/misc/socials:1546-1554`. The reachable
branches are no argument (actor plus room), visible target (actor, other-room,
and victim), self target (actor plus room), and missing target (actor only).
The command table supplies the shared `POS_RESTING` entry gate; the social
record adds no victim-position gate and hides no actor. The room and victim
paths use the shared `act()` audience and substitution behavior.

## Evidence and confirmed parity

Scenario: `cmd/dp-oracle-diff/scenarios/roar-depth.txt`

Manifest: `docs/fidelity/depth/roar.tsv` (9 rows)

Focused test: `pkg/session/roar_depth_test.go`

The clean-main RED/GREEN baseline was GREEN: the existing Go social dispatch,
the parsed roar record, and the shared Act engine already matched C. The
scenario with a room observer and named target proves no-argument output,
first-token parsing with ignored trailing words, all three target audiences,
self-target output, and the exact missing-target byte. It was run with
`--show-oracle` at seed 1 to confirm the intended C blocks, then at seeds 1,
2, 3, 5, and 8; every run reported no normalized divergence. No `src/` or
C-oracle file was edited.

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

Feature branch: `glm/depth-roar`

Feature commit: `17ff86d4a` (`test: prove roar depth fidelity`)

Feature PR: #1125 — hosted lint, security, and test checks were green; the
workflow's build-and-push and deploy jobs were skipped by conditions. The
automatic workflow fired normally. The PR was self-merged as main commit
`bd30fc37623a` only after all required hosted checks were green.

The earlier open PRs for `plot`, `purge`, and `qecho` remain open because
their checks did not fire after their one permitted exact workflow retry; none
was merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(deterministic oracle matrix), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). Shared social entry gates, Act audience
routing, and ownership remain under R5b/R5c.
