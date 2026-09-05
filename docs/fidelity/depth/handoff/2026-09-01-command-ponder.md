# Depth-fidelity handoff — `ponder`

Date: 2026-09-01

## Queue position

This round began from `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep advanced from `policy` to
`ponder`.

The next unclaimed interpreter row is `poofin` at `src/interpreter.c:613`.

Frontier before this slice: 2,829 total; 2,758 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,836 total; 2,765 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:612 */
{ "ponder"   , POS_RESTING , do_action , 0, 0 },
```

`src/act.social.c:102-151` implements the shared `do_action` path. It finds the
social, rejects `noshout` actions, parses a character target only when the
social has a `char_found` message, and otherwise uses the no-argument pair;
typed arguments are ignored for a social with no `char_found` message. The
social record at `lib/misc/socials:580-583` is:

```text
ponder 1 0
You ponder over matters as they appear to you at this moment.
$n sinks deeply into $s own thoughts.
#
```

The `1 0` record means minimum level 1 and no hide flag in the loader's legacy
field names; it has no victim-position gate. The Go registry at
`pkg/game/socials.go` has the same three messages, `MinLevel: 1`, and
`HideFlag: 0`. The command entry remains the C-level unrestricted entry gate,
with the social-level gate enforced by the shared Go path.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/ponder-depth.txt`

Manifest: `docs/fidelity/depth/ponder.tsv` (7 rows)

Test: `pkg/session/ponder_depth_test.go`

The seed-1 main vehicle was GREEN with `--show-oracle`. It matched the exact
no-argument actor and room-audience messages, including the C behavior that a
typed target, `self`, or missing target with trailing words is ignored because
the social has no `char_found` branch. The matrix at seeds `1,2,3,5,8` was
GREEN with no normalized divergence.

No confirmed divergence was found, so no production Go behavior was changed.
The round adds only the durable scenario, manifest, and focused registration
test. Position-gate and `noshout` behavior are delegated to existing proven
shared-path evidence.

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

Feature branch: `glm/depth-ponder`

Feature commit: `90dd4a28a` (`test: prove ponder depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1076 — merged; main merge commit `c8b3af29c`. Hosted lint,
security, and test checks were green; build-and-push and deploy were skipped by
repository workflow conditions. The PR was merged only after the required
hosted checks were green.

The prior plot handoff PR #1064 remains open because its checks did not fire
after the one permitted exact workflow retry; it was not merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and draw parity), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The source-order inventory and manifest
claim are maintained under R5b/R5c.
