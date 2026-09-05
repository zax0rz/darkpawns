# Depth-fidelity handoff — `puke`

Date: 2026-09-01

## Queue position

This round began from `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep advanced from `prompt` to
`puke`.

The next unmanifested interpreter family is `punch` at
`src/interpreter.c:622`.

Frontier before this slice: 2,869 total; 2,798 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,877 total; 2,806 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:621 */
{ "puke"     , POS_RESTING , do_action   , 0, 0 },
```

`src/act.social.c:102-151` resolves the social, rejects `PLR_NOSHOUT`,
extracts the first target argument when `char_found` exists, and then selects
the no-argument, missing-target, self-target, or actor/room/victim trio path.
The `lib/misc/socials:605-612` record is `puke 0 0`, with exact messages for
each of those eight slots and no victim-position gate. The command dispatcher
requires resting with no authority gate; shared position, noshout, and visible
room-target lookup behavior are delegated to existing social-boundary proofs.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/puke-depth.txt`

Manifest: `docs/fidelity/depth/puke.tsv` (8 rows)

Test: `pkg/session/puke_depth_test.go`

The baseline `puke` proof was GREEN after `--show-oracle` confirmed execution
of the intended social path. Actor and peer output matched for no argument,
the target trio including pronoun substitutions, self-target, and the
social-specific missing-target response. The matrix at seeds `1,2,3,5,8` was
GREEN with no normalized divergence.

No production Go divergence was confirmed, so this slice adds only the
command-specific oracle scenario, manifest rows, and a focused registration
and full social-record test. No C source or oracle checkout was edited.

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

Feature branch: `glm/depth-puke`

Feature commit: `2eadcf2bb` (`test: prove puke depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1088 — merged; main commit `919524543`. Hosted lint, security,
and test checks were green; build-and-push and deploy were skipped by
repository workflow conditions. The PR was merged only after the required
hosted checks were green.

The prior plot handoff PR #1064 remains open because its checks did not fire
after the one permitted exact workflow retry; it was not merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and draw parity), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The source-order inventory and manifest
claim are maintained under R5b/R5c.
