# Depth-fidelity handoff — `review`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus the
latest `return` handoff. The special-procedure inventory remains exhausted, the
single blocked row `objmagic.sleep-entry-gates` remains queued after its one
cast-sleep vehicle, and the interpreter sweep advanced from `reallyquit` to
`review`.

The source-order audit confirms that `ride` is already owned by the `mount`
manifest. The next unclaimed interpreter-table family is `rin` at
`src/interpreter.c:660`, which shares `do_kuji_kiri` with the other Kuji Kiri
command aliases and must be audited as that call path requires.

Frontier before this slice: 2,994 total; 2,917 proven/delegated; 26 blocked;
51 excluded.

Frontier after this slice: 3,001 total; 2,924 proven/delegated; 26 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:658 */
{ "review"   , POS_DEAD    , do_review   , 0, 0 },
```

`src/new_cmds.c:1369-1391` builds the exact `Last Gossips:` header, walks all
25 fixed review slots from slot 24 down to slot 0, substitutes `Someone
invisible` when the stored speaker invisibility exceeds the viewer level, and
pages the result to the actor's descriptor. `update_review()` at
`src/new_cmds.c:1341-1358` shifts the existing entries toward slot 24 and
inserts each new gossip at slot 0. The command ignores its argument. The
command's POS_DEAD/level-0 dispatcher gate is shared with the existing quit
proof.

## Evidence and confirmed divergence

Scenario: `cmd/dp-oracle-diff/scenarios/review-depth.txt`

Manifest: `docs/fidelity/depth/review.tsv` (seven rows)

Focused tests: `pkg/game/review_depth_test.go`

The clean-main RED scenario matched C for empty history, gossip updates, and
ignored trailing words, but exposed a player-visible ordering divergence: Go
iterated its newest-first slice directly, while C rendered the older entry
first by traversing the fixed array backward. The Go fix changes only
`World.ReviewGossip` traversal order; no `src/` or C-oracle file was edited.
Focused tests pin the C invisibility comparison and 25-entry drop-oldest
window.

The corrected scenario is GREEN with `--show-oracle` and across seeds 1, 2, 3,
5, and 8. The actor-only pager path is covered by the same live vehicle; the
shared pager mechanics remain delegated to the existing help pager proof under
R5b/R5c.

## Verification and integration

All required local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
git diff --check
```

Feature branch: `glm/depth-review`

Feature commit: `d58ad068b` (`fix: preserve C review gossip order`)

Feature PR: #1119 — hosted lint, security, and test checks were green; the
workflow's build-and-push and deploy jobs were skipped by conditions. It was
self-merged as main commit `3b91fa61b7a5` only after the required hosted checks
were green.

The earlier open PRs for `plot`, `purge`, and `qecho` remain open because their
checks did not fire after their one permitted exact workflow retry; none was
merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism across the oracle matrix), R4 (no invention), R5 (process
discipline), and R5e (verify the actual C call path). The fixed-buffer state
and shared pager/source-order ownership are maintained under R5b/R5c.
