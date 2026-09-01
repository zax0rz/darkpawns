# Depth-fidelity handoff — `pout`

Date: 2026-09-01

## Queue position

This round began from `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep advanced from `pose` through
the already-complete `pour` family to `pout`.

The previous `pose` handoff pointed at `pour`, but the source-order sweep
confirmed `pour` was already fully manifested (`do_pour: 23/23`), so it was
not re-picked. The next actual unmanifested interpreter family is `prompt` at
`src/interpreter.c:619`; `practice` between `pout` and `prompt` is already
covered.

Frontier before this slice: 2,854 total; 2,783 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,861 total; 2,790 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:617 */
{ "pout"     , POS_RESTING , do_action   , 0, 0 },
```

`src/act.social.c:102-151` first resolves the social and rejects
`PLR_NOSHOUT`, then checks whether the record has a `char_found` message. The
`lib/misc/socials:590-593` record is `pout 0 0`, followed by the actor message
`Ah, don't take it so hard.`, the visible room message `$n pouts.`, and `#`.
Because `char_found` is absent, C ignores all typed arguments and emits the
no-argument actor/room pair. The command dispatcher requires resting but has
no authority gate; shared position and noshout refusals are delegated to
existing social-boundary proofs.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/pout-depth.txt`

Manifest: `docs/fidelity/depth/pout.tsv` (7 rows)

Test: `pkg/session/pout_depth_test.go`

The baseline `pout` proof was GREEN after `--show-oracle` confirmed execution
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

Feature branch: `glm/depth-pout`

Feature commit: `a7e6b837f` (`test: prove pout depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1084 — merged; main commit `230ce659f`. Hosted lint, security,
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
