# Depth-fidelity handoff — `purr`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep advanced from `punch` to `purr`.

The next unmanifested interpreter family is `purge` at
`src/interpreter.c:624`.

Frontier before this slice: 2,885 total; 2,814 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,892 total; 2,821 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:623 */
{ "purr"     , POS_RESTING , do_action   , 0, 0 },
```

`src/act.social.c:102-127` resolves the social and rejects `PLR_NOSHOUT`, then
checks whether the social has a `char_found` message. The
`lib/misc/socials:625-628` record is `purr 0 0` with an actor line,
`$n` room line, and `#` in the `char_found` slot. Because that slot is absent,
the C path emits only the no-argument actor/room pair and ignores every typed
argument; it never performs target lookup, self-target selection, or victim
position checking. The command dispatcher requires resting with no authority
gate; shared position and noshout behavior are delegated to existing social
boundary proofs.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/purr-depth.txt`

Manifest: `docs/fidelity/depth/purr.tsv` (7 rows)

Test: `pkg/session/purr_depth_test.go`

The clean-main `purr` proof was GREEN after `--show-oracle` confirmed execution
of the intended self-only social path. Actor and peer output matched for no
argument, a real peer plus trailing words, `self`, and an unknown target; each
probe produced the exact actor and room lines. The matrix at seeds `1,2,3,5,8`
was GREEN with no normalized divergence.

No production Go divergence was confirmed, so this slice adds only the
command-specific oracle scenario, manifest rows, and a focused registration and
self-only social-record test. No C source or oracle checkout was edited.

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

Feature branch: `glm/depth-purr`

Feature commit: `af45dd927` (`test: prove purr depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1092 — merged; main commit `b52aab23e`. Hosted lint, security,
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
