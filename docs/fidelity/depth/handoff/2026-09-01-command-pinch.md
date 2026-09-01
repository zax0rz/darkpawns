# Depth-fidelity handoff — `pinch`

Date: 2026-09-01

## Queue position

This session began from `main` after `git pull --ff-only` and `make fidelity-depth`.
The special-procedure inventory is exhausted, the one blocked row
`objmagic.sleep-entry-gates` remains queued for its single cast-sleep vehicle, and
the interpreter sweep reached `pinch` in source-table order. The next unclaimed
interpreter row is `piss` at `src/interpreter.c:608`; `peer`, `pick`, and `plot`
are already claimed.

Frontier before this slice: 2,776 total; 2,705 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,787 total; 2,716 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:605 */
{ "pinch", POS_RESTING, do_action, 0, 0 },
```

The command therefore enters the shared social dispatcher at
`src/act.social.c:102-151`. The authoritative social record is
`lib/misc/socials:1466-1474`:

```text
pinch 0 0
Pinch who?
#
You pinch $M right on the butt.
$n pinches $N on the butt.
$n pinches your butt.
Pinch who?
You pinch yourself to see if you're dreaming.
$n pinches $mself to makes sure $e's awake.
```

The first social flag is `0` (not hidden) and the minimum victim position is
`0`. The path includes no-argument handling, target lookup, missing-target
handling, named self and `self` alias handling, player and NPC target audience
messages, and the parser's first-token behavior for extra words.

## Evidence

The slice is evidence-only: existing Go `pkg/game/act_social.go` behavior matched
the C path, so no Go fix was justified under R1/R4/R5e.

Scenario: `cmd/dp-oracle-diff/scenarios/pinch-depth.txt`

Manifest: `docs/fidelity/depth/pinch.tsv` (11 rows)

Test: `pkg/session/pinch_depth_test.go`

The oracle vehicle covers:

- no argument;
- a named player target and room audience;
- an NPC target and room audience;
- named self and the `self` alias;
- a missing target; and
- a one-argument command with trailing words ignored by the shared parser.

`pinch-depth --show-oracle --seed 1` was GREEN, and the matrix at seeds
`1,2,3,5,8` was GREEN. The scenario's C and Go player-facing bytes and room
audiences matched in every covered branch.

## Verification and integration

Local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Feature branch: `glm/depth-pinch`

Feature commit: `e4d78d27d` (`test: prove pinch depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1065 — merged; main merge commit `301fbf445c38764ca996fd626cbf78db1ba5a40b`.
Hosted lint, security, and test checks were green; build-and-push and deploy were
skipped by repository workflow conditions. The PR was merged only after the
required hosted checks were green.

The prior plot handoff PR #1064 remains open because its checks did not fire
after the one permitted exact workflow retry; it was not merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and draw parity), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The source-order inventory and manifest
claim are maintained under R5b/R5c.
