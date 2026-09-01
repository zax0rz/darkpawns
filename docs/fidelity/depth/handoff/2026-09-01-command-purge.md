# Depth-fidelity handoff — `purge`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
newest prior handoff (`2026-09-01-command-purr.md`). The special-procedure
inventory remains exhausted, the one blocked row `objmagic.sleep-entry-gates`
remains queued after its single cast-sleep vehicle, and the interpreter sweep
advanced from `purr` to `purge`.

The next unmanifested interpreter family is `qecho` at
`src/interpreter.c:627`. `quaff` is already represented by its existing depth
manifests, and `quest` is covered by `gen-tog`; neither was repicked.

Frontier before this slice: 2,892 total; 2,821 proven/delegated; 22 blocked; 49
excluded.

Frontier after this slice: 2,904 total; 2,833 proven/delegated; 22 blocked; 49
excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:624 */
{ "purge"    , POS_DEAD    , do_purge    , LVL_IMMORT+1, 0 },
```

`src/act.wizard.c:1423-1478` first takes one argument with `one_argument`.
Character lookup is visibility-aware and precedes object lookup. A player at
the actor's level or above is protected by the exact `Fuuuuuuuuu!` early
return. A lower-level player follows `close_socket`'s lost-link room
announcement and extraction before the actor receives `Okay.`. An NPC is
disintegrated for the non-victim room audience, extracted through C's deferred
path, and acknowledged with `Okay.`. If no character matches, a visible room
object is destroyed for the room audience and acknowledged; a miss emits
`Nothing here by that name.`.

With no argument, C emits the scorching-flames gesture to the room, calls
`send_to_room` for `The world seems a little cleaner.` to awake players
including the actor, extracts all room NPCs and objects, and emits no actor
acknowledgement. `src/comm.c:2366-2373` makes the cleaner-room line awake-only.

## Evidence and confirmed divergences

Scenarios:

- `cmd/dp-oracle-diff/scenarios/purge-depth.txt`
- `cmd/dp-oracle-diff/scenarios/purge-npc-depth.txt`
- `cmd/dp-oracle-diff/scenarios/purge-object-depth.txt`
- `cmd/dp-oracle-diff/scenarios/purge-player-depth.txt`
- `cmd/dp-oracle-diff/scenarios/purge-protected-player-depth.txt`
- `cmd/dp-oracle-diff/scenarios/purge-low-level-depth.txt`

Manifest: `docs/fidelity/depth/purge.tsv` (12 rows)

Focused test: `pkg/session/purge_depth_test.go`

Clean-main RED probes confirmed that Go used `Ok.` instead of C's `Okay.`,
invented an actor acknowledgement on room purge while omitting C's cleaner
line, used short-description substring matching instead of C's canonical
first-token character/object lookup, and omitted the lower-level player
close/link-loss/extract branch. The Go handler was corrected only along those
confirmed paths; `src/` and the C oracle checkout were not edited.

All six vehicles matched C with `--show-oracle` and at seeds 1, 2, 3, 5, and
8, with no normalized divergence. The protected-player vehicle reaches C's
equal-level refusal; the lower-level vehicle proves the link-loss audience;
the NPC vehicle uses `~dpclock pulse 20` before its post-extraction probe.

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

Feature branch: `glm/depth-purge`

Feature commit: `99ee93a72` (`fix: match purge depth behavior (R1/R2/R3/R5e)`)

Feature PR: #1094 — merged as `15175ef2b`. Hosted lint, security, and test
checks were green; build-and-push and deploy were skipped by repository
workflow conditions. The PR was merged only after the required hosted checks
were green.

The prior plot handoff PR #1064 remains open because its checks did not fire
after the one permitted exact workflow retry; it was not merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and ordering), R4 (no invention), R5 (process discipline), and
R5e (verify the actual C call path). The source-order inventory and manifest
claim are maintained under R5b/R5c.
