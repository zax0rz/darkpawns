# Depth-fidelity handoff — `poke`

Date: 2026-09-01

## Queue position

This round began from `main` after `git pull --ff-only` and a successful
`make fidelity-depth`. The special-procedure inventory is exhausted, the one
blocked row `objmagic.sleep-entry-gates` remains queued for its single
cast-sleep vehicle, and the interpreter sweep advanced from `point` to `poke`.
The next unclaimed interpreter row is `policy` at `src/interpreter.c:611`.

Frontier before this slice: 2,812 total; 2,741 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,823 total; 2,752 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:610 */
{ "poke"     , POS_RESTING , do_action   , 0, 0 },
```

The handler is `src/act.social.c:102-151`. Its actual path is:

1. `find_action(cmd)` resolves the registered social record.
2. `PLR_NOSHOUT` rejects the command before argument parsing.
3. When `char_found` exists, `one_argument` extracts the first non-fill-word
   target token; otherwise the social is no-argument-only.
4. No target emits `char_no_arg` to the actor and `others_no_arg` to the room.
5. `get_char_room_vis` resolves a visible mob or player in the actor's room.
6. A missing target emits `not_found`; the actor target emits `char_auto` and
   `others_auto`; another target checks `min_victim_position`, then emits the
   actor, non-victim room, and victim branches.

The poke record at `lib/misc/socials:570-578` has no hide flag and no minimum
victim position. Its reachable player-visible branches are therefore the
POS_RESTING entry gate, no argument, visible player target, visible NPC target,
named self, the `self` alias, missing target, and fill-word/trailing-input
parsing. The shared social visibility and pager-independent audience logic is
claimed or delegated in the existing social manifests.

## Evidence and confirmed divergences

Scenario: `cmd/dp-oracle-diff/scenarios/poke-depth.txt`

Manifest: `docs/fidelity/depth/poke.tsv` (11 rows)

Test: `pkg/session/poke_depth_test.go`

The seed-1 main vehicle was GREEN with `--show-oracle`. It matched the C
oracle for the actor-only no-argument line, the player victim audience, the
generic mob victim audience, named self and `self` alias, missing target, and
`one_argument` fill-word/trailing-input behavior. The matrix at seeds
`1,2,3,5,8` was GREEN with no normalized divergence.

No confirmed divergence was found, so no production Go behavior was changed.
The round adds only the durable scenario, manifest, and focused registration
test. This follows R4: the existing Go social implementation already matched
the actual C call path and authored bytes.

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

Feature branch: `glm/depth-poke`

Feature commit: `186c3d1aa` (`test: prove poke depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1071 — merged; main merge commit `9c5ccde79`. Hosted lint,
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
