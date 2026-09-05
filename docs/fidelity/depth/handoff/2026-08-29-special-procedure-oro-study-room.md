# 2026-08-29 — `oro_study_room` depth slice

## Frontier and queue

- Started from the clean `main` boundary after the `oro_quarters_room`
  handoff, ran `git checkout main && git pull --ff-only && make fidelity-depth`,
  and reread `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff.
- Main baseline for this slice was 937 total cases; 910
  proven/delegated; 6 blocked; 21 excluded; actionable completion 910/916
  (99.3%). This slice added six cases; after PR integration main reports 943
  total; 916 proven/delegated; 6 blocked; 21 excluded; actionable completion
  916/922 (99.3%).
- The unrelated untracked brief
  `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved.

## C call path and branch census

- `SPECIAL(oro_study_room)` is assigned to room 18399 at
  `src/spec_assign.c:613` and defined at `src/spec_procs.c:2321-2337`.
  Room-special dispatch runs before ordinary command dispatch at
  `src/interpreter.c:1407-1415`; a `FALSE` result reaches the normal movement
  handler.
- C first excludes NPC command actors and any command other than exact
  `north`. For a player whose exact name is not `Orodreth`, it emits the
  actor-only warning, sends the `$n`/`$s` jolt through `TO_ROOM` with actor
  exclusion, halves `GET_HIT` with integer division, and returns `TRUE`.
  Exact `Orodreth` returns `FALSE` with no special output, leaving ordinary
  north dispatch responsible. The focused test covers the player-visible
  command/name gates, ignored argument, exact audience, odd-HP state, and
  no-output owner fallthrough. The NPC gate has no player-visible output and
  is not falsely claimed as a separate live player vehicle.
- Room 18399 is present in authoritative `lib/world/wld/183.wld` and
  `183.wld` is indexed in both worlds, so the live vehicle uses production
  world data without a synthetic room or C-tree edit.

## RED/GREEN evidence and port result

- RED on pre-fix `main`: the valid room vehicle produced C's jolt block but Go
  fell through to `Alas, you cannot go that way...` and sent no peer jolt.
  The focused test also exposed the pre-fix nested player mutex lock in the
  HP-halving path.
- GREEN on `glm/spec-oro-study-room`:
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario spec-proc-oro-study-room --show-oracle`
  reports `result: no normalized divergence`; the C blocks show the exact
  actor warning and peer jolt. `TestSpecOroStudyRoom_GatesAudienceHPAndFallthrough`
  pins the exact bytes, actor exclusion, command/name fallthrough, argument
  ignorance, and integer HP transition.
- No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification and integration

- Feature-branch gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .`.
- PR #751 (`glm/spec-oro-study-room`) received green lint, security, and test
  checks; build/deploy were skipped by workflow policy. It was squash-merged
  as `a5fa1676b`. The separate PR #750 for `oro_quarters_room` remains open
  and ungreen; its claimed slice was not re-picked.

This slice applies R1 (player-facing bytes), R2 (registered room command
surface), R3 (deterministic integer state transition), R4 (no invented
output), and R5/R5e (actual C dispatch/call path and oracle verification).

## Next queue item

Continue the `src/spec_procs.c` special-procedure inventory with `bank`,
defined at `src/spec_procs.c:2345` and registered for object vnums 8034 and
18224 at `src/spec_assign.c:535` and `src/spec_assign.c:561`. Do not re-pick
`oro_study_room`, `oro_quarters_room`, `suck_in`, or earlier claimed
procedures. After the active special-procedure inventory is exhausted, attempt
the single blocked `objmagic.sleep-entry-gates` vehicle, then sweep remaining
un-manifested command families in `src/interpreter.c` table order.
