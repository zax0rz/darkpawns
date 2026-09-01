# Depth-fidelity handoff — `newbie`

Date: 2026-09-01

## Frontier

This session began from clean `main` after the `news` handoff at:

```text
Cases: 2638 total, 2568 proven/delegated, 22 blocked, 48 excluded
Actionable completion: 2568/2590 = 99.2%
```

The `newbie` slice is now merged to `main` in PR #1042 (`c976ee60b`). The
post-merge frontier is:

```text
Cases: 2648 total, 2578 proven/delegated, 22 blocked, 48 excluded
Actionable completion: 2578/2600 = 99.2%
```

The special-procedure inventory remains exhausted. The one intentionally
blocked row `objmagic.sleep-entry-gates` remains blocked; its separate cast-
sleep vehicle is already proven and does not clear the blocked direct-entry
row. The interpreter sweep continues.

## Slice proof

The next source-order unclaimed row was `src/interpreter.c:566`:

```c
{ "newbie" , POS_SLEEPING, do_gen_comm , 0, SCMD_NEWBIE },
```

The C call path was read first: `src/act.comm.c:1146-1305`,
`do_gen_comm(ch, argument, cmd, subcmd)`, with `channels[SCMD_NEWBIE] =
PRF_NONEWBIE` and the `com_msgs` row for the exact `newbie` verb. The proof
enumerated the C order: `PLR_NOSHOUT`, `ROOM_SOUNDPROOF`, the level gate
exception for `SCMD_NEWBIE`, zero wisdom/intelligence, sender `PRF_NONEWBIE`,
leading-space/empty argument handling, ANSI deletion, the self echo, and
global recipient filtering for `PRF_NONEWBIE`, `PLR_WRITING`, and soundproof
rooms. Newbie does not spend movement or update gossip history.

Go already routed `newbie` through the shared `DoChannel` implementation with
the matching verb, sender/recipient flag, no minimum level, and global
audience. No behavior change was justified by the oracle comparison (R4).

Added:

- `cmd/dp-oracle-diff/scenarios/newbie-channel-depth.txt`
- `pkg/game/newbie_channel_depth_test.go`
- `pkg/session/newbie_depth_test.go`
- `docs/fidelity/depth/newbie.tsv` (10 manifest rows, including delegated
  shared-channel gates)

The oracle vehicle used an Implementor plus two mortal peers, with the silent
peer toggled `nonewbie`. `--show-oracle --seed 1` confirmed the exact bytes,
including:

```text
You newbie, 'hello from a level one player'\r\n
Nwbactor newbies, 'hello from a level one player'\r\n
Yes, newbie, fine, newbie we must, but WHAT???\r\n
You aren't even on the channel!\r\n
```

Seeds 1, 2, 3, 5, and 8 all returned `no normalized divergence`. Focused
`pkg/game` and `pkg/session` tests passed. Full local gates passed:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Hosted lint, security, and test checks for PR #1042 were green; build and
deploy were skipped by the workflow. Per the standing rule, the PR was merged
only after all required hosted checks were green.

## Fidelity rules applied

- R1: preserved player-facing bytes and absence of recipient bytes.
- R2: preserved the `newbie` command surface and POS_SLEEPING entry gate.
- R3: repeated the oracle comparison across the required deterministic seeds.
- R4: made no Go behavior change without a confirmed divergence.
- R5b/R5c: delegated shared `do_gen_comm` channel behavior only after tracing
  the common call path and existing proof.
- R5e: verified the actual C registration and call path rather than relying on
  a summary.

## Next queue item

The next unclaimed interpreter-table family is `nibble` at
`src/interpreter.c:567`. Its C handler is the next call path to map before
building the `nibble` oracle vehicle. Continue with branch
`glm/depth-nibble`, one family PR, and one dated handoff. Do not re-pick
`newbie`, and preserve the existing claims for `mail`, `social`, and
`murder` while checking the table-order sweep.
