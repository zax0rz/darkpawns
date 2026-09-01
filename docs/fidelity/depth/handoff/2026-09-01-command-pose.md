# Depth-fidelity handoff — `pose`

Date: 2026-09-01

## Queue position

This round began from `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep advanced from `poofout` to
`pose`.

The next unclaimed interpreter row is `pour` at `src/interpreter.c:616`.

Frontier before this slice: 2,847 total; 2,776 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,854 total; 2,783 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:615 */
{ "pose"     , POS_STANDING, do_action   , 0, 0 },
```

`src/act.social.c:102-151` first resolves the social, rejects
`PLR_NOSHOUT`, and then checks whether the record has a `char_found` message.
The `lib/misc/socials:585-588` record is `pose 1 0`, followed by the actor
message `You stand as tall as the gods themselves.`, the room message
`$n suffers from a severe case of schizophrenic megalomania.`, and `#`.
Because `char_found` is absent, C ignores all typed arguments and emits the
no-argument actor/room pair. The command dispatcher requires standing but has
no authority gate; shared position and noshout refusals are delegated to
existing social-boundary proofs.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/pose-depth.txt`

Manifest: `docs/fidelity/depth/pose.tsv` (7 rows)

Test: `pkg/session/pose_depth_test.go`

The baseline `pose` proof was GREEN after `--show-oracle` confirmed execution
of the intended self-only C block. Actor and peer output matched for no
argument, a named peer, the `self` argument, and an unresolved target with
trailing words. The matrix at seeds `1,2,3,5,8` was GREEN with no normalized
divergence.

No production Go divergence was confirmed, so this slice adds only the
oracle scenario, manifest rows, and a focused registration/social metadata
test. No C source or oracle checkout was edited.

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

Feature branch: `glm/depth-pose`

Feature commit: `d0beec96a` (`test: prove pose depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1082 — merged; main commit `9486b4f36`. Hosted lint, security,
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
