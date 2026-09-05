# Depth-fidelity handoff — `recall`

Date: 2026-09-01

## Queue position

This round began from refreshed `main` after `git pull --ff-only`, a
successful `make fidelity-depth`, and rereading
`docs/fidelity/DEPTH_TESTING.md` plus the latest `reload` handoff. The special-
procedure inventory remains exhausted, the one blocked row
`objmagic.sleep-entry-gates` remains queued after its single cast-sleep
vehicle, and the interpreter sweep advanced from `reload` to `recall`.

The apparent next token `recharge` was not a separate target: it is already
owned by the shared `do_not_here` family row in
`docs/fidelity/depth/do-not-here.tsv`, together with `receive`, `remort`,
`rent`, and `retrieve`. Per R5b/R5c, it was not re-picked. The next actual
unclaimed family is `reroll` at `src/interpreter.c:649`. Between `recall` and
`reroll`, `remove`, the `do_action` social aliases, `reply`/`.`, `rescue`,
`retreat`, `ride`, `roomflags`, and `reallyquit` are already owned by their
shared or dedicated manifests.

Frontier before this slice: 2,949 total; 2,876 proven/delegated; 22 blocked;
51 excluded.

Frontier after this slice: 2,958 total; 2,885 proven/delegated; 22 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:641 */
{ "recall"    , POS_RESTING , do_recall , 0, 0 },
```

`src/act.other.c:1727-1748` first gates immortals and NPCs, the
`ROOM_BFR` anti-recall room, and combat. It then calls
`src/spells.c:124-163`'s `spell_recall`, which selects the default or hometown
room and transfers the player. The command emits the exact origin and
destination room audiences, ignores trailing arguments, and leaves a mount in
the origin after the transfer. The dispatcher enforces `POS_RESTING`, so a
sleeping player receives `In your dreams, or what?` and never reaches
`do_recall`. The awake path ends with the normal look output when the player
is awake.

The Go recall implementation already matched the confirmed C behavior. The
existing direct tests cover the command gates and hometown targets; this slice
adds command-surface evidence for argument handling, transfer, origin and
destination audiences, and the sleeping-position gate.

## Evidence and confirmed divergence

Scenario: `cmd/dp-oracle-diff/scenarios/recall-depth.txt`

Manifest: `docs/fidelity/depth/recall.tsv` (9 rows)

Focused test: `pkg/session/recall_depth_test.go`, pinning the registered
`POS_RESTING` command entry and its command-surface vehicle.

The scenario uses the registered recall-capable player setup, a destination
peer, and a probe actor. It captures the actor's destination look, the primary
actor's disappearance at the origin, the destination appearance, the sleeping
room audience, and the exact sleeping-command refusal. Seeds 1, 2, 3, 5, and
8 were GREEN; seed 1 was also run with `--show-oracle`. No Go behavior change
was required and no `src/` or C-oracle file was edited.

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

Feature branch: `glm/depth-recall`

Feature commit: `79e3fb61e` (`test: add recall depth fidelity coverage`)

Feature PR: #1108 — merged as `fca4304cb`. Hosted lint, security, and test
checks were green in run `33576585919`; build/deploy were skipped by workflow
conditions. The PR was merged only after every reported hosted check was
green.

The earlier qecho, purge, and plot handoff PRs remain open because their
checks did not fire after their permitted exact retries. They are not claimed
or repicked by this handoff.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and ordering), R4 (no invention), R5 (process discipline), R5b
and R5c (shared behavior ownership), and R5e (verify the actual C call path).
The source-order claim for the next target is `reroll` at
`src/interpreter.c:649`, dispatching to `do_wizutil` with `SCMD_REROLL`.
