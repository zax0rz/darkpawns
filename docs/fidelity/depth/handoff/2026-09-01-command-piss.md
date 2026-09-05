# Depth-fidelity handoff — `piss`

Date: 2026-09-01

## Queue position

This session began from `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep reached `piss` in source-table
order. The next unclaimed interpreter row is `point` at `src/interpreter.c:609`.

Frontier before this slice: 2,787 total; 2,716 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,798 total; 2,727 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:608 */
{ "piss", POS_RESTING, do_action, 0, 0 },
```

The command enters the shared social dispatcher at `src/act.social.c:102-151`.
The authoritative social record is `lib/misc/socials:1073-1081`:

```text
piss 1 0
You take a piss in the corner.
$n takes a piss.
You piss on $M.
$n lifts up $s hind leg and pisses on $N.
$n lifts up $s hind leg and pisses on you.
There's no one by that name around.
You piss on yourself.
$n pisses all over $mself.
```

The record's first value is minimum social level `1` and its hide flag is `0`;
the minimum victim position is `0` because the social record has no separate
position gate. The reachable surface includes the interpreter resting-position
gate, the shared `PLR_NOSHOUT` refusal, no-argument actor/room bytes, visible
player and NPC target audiences, named self and `self` alias handling, missing
targets, and `one_argument` first-token parsing. Shared visibility and hide-flag
delivery remain owned by the social matrix.

## Evidence

The slice is evidence-only: existing Go `pkg/game/act_social.go` behavior
matched the verified C path, so no Go fix was justified under R1/R4/R5e.

Scenario: `cmd/dp-oracle-diff/scenarios/piss-depth.txt`

Manifest: `docs/fidelity/depth/piss.tsv` (11 rows)

Test: `pkg/session/piss_depth_test.go`

The C-first vehicle uses a scriptless generic mob and a named peer in room 8004
and covers:

- no argument;
- a named player target and room audience;
- an NPC target and room audience;
- named self and the `self` alias;
- a missing target; and
- leading fill words with trailing input ignored by `one_argument`.

The initial vehicle setup used a rejected oracle account name containing the
command substring and therefore never reached gameplay; after renaming the
fixture identity, `piss-depth --show-oracle --seed 1` was GREEN. The matrix at
seeds `1,2,3,5,8` was GREEN with no normalized divergence, and the focused
registration test pins the exact `1,0` social metadata and all eight authored
messages.

## Verification and integration

Local gates passed after correcting that fixture/test transcription:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Feature branch: `glm/depth-piss`

Feature commit: `5252bdc43` (`test: prove piss depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1067 — merged; main merge commit `584b26dac`.
Hosted lint, security, and test checks were green; build-and-push and deploy
were skipped by repository workflow conditions. The PR was merged only after
the required hosted checks were green.

The prior plot handoff PR #1064 remains open because its checks did not fire
after the one permitted exact workflow retry; it was not merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and draw parity), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The source-order inventory and manifest
claim are maintained under R5b/R5c.
