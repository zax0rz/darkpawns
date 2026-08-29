# 2026-08-29 — `fearface` depth slice

## Frontier and queue

- Started from a fresh `main` boundary, pulled with no changes, ran
  `make fidelity-depth`, and reread `docs/fidelity/DEPTH_TESTING.md` plus the
  newest `pray_for_items` handoff.
- Baseline frontier: 913 total cases; 888 proven/delegated; 6 blocked; 19
  excluded; actionable completion 888/894 (99.3%).
- This documentation-only slice adds one explicit D5 `excluded` case. The
  post-slice frontier is 914 total cases; 888 proven/delegated; 6 blocked; 20
  excluded; actionable completion remains 888/894 (99.3%).

## C call path and registration census

- `SPECIAL(fearface)` is defined at `src/spec_procs.c:2171-2202`. Its body
  rejects asleep/non-NPC/command/nonpositive-HP calls, then has a commandless
  fighting branch that either stands the mob or selects a visible mortal,
  checks `mag_savingthrow`, emits the room stare/panic line, and calls
  `do_flee` one to five times.
- The complete mobile special declaration/assignment section in
  `src/spec_assign.c:85-126` contains no `SPECIAL(fearface)`, and the complete
  `assign_mobiles` table has no `ASSIGNMOB(..., fearface)`. The room assignment
  table likewise has no fearface entry.
- `src/mobact.c:68-93` invokes autonomous specials only through assigned mob
  function pointers, while `src/interpreter.c:1407-1456` invokes only the
  registered room/mobile pointers on player-command dispatch. Therefore the C
  dispatch surface for `fearface` is empty despite the latent function body.

## Manifest result and verification

- Added `mob.fearface-unassigned` to `docs/fidelity/depth/spec-procs.tsv` as
  D5 `excluded`, with no oracle scenario and no synthetic spawn. The Go
  registry's latent `specFearface` function does not establish C reachability.
- No Go behavior changed and no file under `src/` or
  `darkpawns-c-oracle/` was edited. `make fidelity-depth`, `go build ./...`,
  `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .` all pass on the branch.
- This slice is governed by R2 (registered surface), R4 (no invented
  behavior), and R5/R5e (verify the actual assignment and call path); no R1
  player-facing claim is made.

## Integration and next queue item

- The required branch is `glm/spec-fearface`; open one PR for this slice and
  merge only if every GitHub check is green. If checks do not fire, issue the
  one permitted workflow retry and leave the PR open if it remains not-green.
- The next source-order special is the assigned room `start_room` at room
  8099. Begin the next session from a clean `main`/pull/frontier boundary and
  do not repick `fearface` or earlier claimed procedures. After active specials
  are exhausted, attempt the one blocked `objmagic.sleep-entry-gates` vehicle,
  then sweep un-manifested interpreter command families in table order.
