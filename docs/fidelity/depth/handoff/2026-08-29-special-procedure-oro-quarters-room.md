# 2026-08-29 — `oro_quarters_room` depth slice

## Frontier and queue

- Started from the clean `main` boundary after the `suck_in` handoff, ran
  `git checkout main && git pull --ff-only && make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff.
- At the slice start, main reported 930 total cases; 903
  proven/delegated; 6 blocked; 21 excluded; actionable completion 903/909
  (99.3%). The feature branch added six cases and reported 936 total;
  909 proven/delegated; 6 blocked; 21 excluded; actionable completion 909/915
  (99.3%). During the final boundary refresh, GitHub main advanced to
  `771c0272d`, incorporating the previously claimed `suck_in` slice; current
  main reports 937 total; 910 proven/delegated; 6 blocked; 21 excluded;
  actionable completion 910/916 (99.3%).
- The unrelated untracked brief
  `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved.

## C call path and branch census

- `SPECIAL(oro_quarters_room)` is assigned to room 18397 at
  `src/spec_assign.c:614` and defined at `src/spec_procs.c:2302-2318`.
  Room-special dispatch runs before ordinary command dispatch at
  `src/interpreter.c:1407-1415`; a `FALSE` result reaches the normal movement
  handler.
- C first excludes NPC command actors and any command other than exact
  `south`. For a player whose exact name is not `Orodreth`, it sends the
  actor-only warning, emits the TO_ROOM jolt with `$n`/`$s` substitution and
  actor exclusion, halves `GET_HIT` with integer division, and returns `TRUE`.
  Exact `Orodreth` returns `FALSE` with no special output, leaving ordinary
  south dispatch responsible. The focused test covers the player-visible
  command/name gates, ignored argument, exact audience, odd-HP state, and
  no-output owner fallthrough. The NPC gate has no player-visible output and
  is not falsely claimed as a separate live player vehicle.
- Room 18397 is present in authoritative `lib/world/wld/183.wld` and
  `183.wld` is indexed in both worlds, so the live vehicle uses production
  world data without a synthetic room or C-tree edit.

## RED/GREEN evidence and port result

- RED on pre-fix `main`: the valid room vehicle produced C's actor jolt but Go
  emitted only `Alas, you cannot go that way...` and sent no peer jolt. The
  focused test also exposed the pre-fix nested player mutex lock in the HP
  halving path.
- GREEN on `glm/spec-oro-quarters-room`:
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario spec-proc-oro-quarters-room --show-oracle`
  reports `result: no normalized divergence`; the C blocks show the exact
  actor warning and peer jolt. `TestSpecOroQuartersRoom_GatesAudienceHPAndFallthrough`
  pins the exact bytes, actor exclusion, command/name fallthrough, argument
  ignorance, and integer HP transition.
- No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification and integration

- Feature-branch gates passed: `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .`.
- PR #750 (`glm/spec-oro-quarters-room`) is open with commit `fb53b6916`.
  No checks fired initially; the required single retry was issued with
  `gh workflow run "Dark Pawns CI/CD" --ref glm/spec-oro-quarters-room`, and
  no checks were reported afterward. Under the standing rule, this PR is
  treated as not-green and was not merged.

This slice applies R1 (player-facing bytes), R2 (registered room command
surface), R3 (deterministic integer state transition), R4 (no invented
output), and R5/R5e (actual C dispatch/call path and oracle verification).
The open PR claims the slice so it must not be re-picked.

## Next queue item

Continue special-procedure source order with `oro_study_room`, defined at
`src/spec_procs.c:2321` and assigned to room 18399 at
`src/spec_assign.c:613`. Do not re-pick `oro_quarters_room`, `suck_in`, or
any earlier claimed procedure. After the active special-procedure inventory is
exhausted, attempt the single blocked `objmagic.sleep-entry-gates` vehicle,
then sweep remaining un-manifested command families in `src/interpreter.c`
table order.
