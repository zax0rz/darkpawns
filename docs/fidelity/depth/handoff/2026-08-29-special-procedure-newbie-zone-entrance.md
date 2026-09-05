# 2026-08-29 — `newbie_zone_entrance` depth slice

## Frontier and queue

- Started from the clean `main` boundary after the `start_room` slice, pulled
  `6c342fadf`, ran `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff.
- Baseline for this slice: 924 total cases; 897 proven/delegated; 6 blocked;
  21 excluded; actionable completion 897/903 (99.3%).
- This slice adds six cases. Current `main` after integration is 930 total
  cases; 903 proven/delegated; 6 blocked; 21 excluded; actionable completion
  903/909 (99.3%).

## C call path and branch census

- `SPECIAL(newbie_zone_entrance)` is defined at `src/spec_procs.c:2262-2276`
  and assigned to room 16300 at `src/spec_assign.c:607`. Room-special
  dispatch runs before ordinary command dispatch at
  `src/interpreter.c:1407-1415`; a `FALSE` result falls through to the normal
  command handler.
- The C procedure first requires the exact `south` command. It then blocks
  levels 11 through 30 with the exact line
  `Nah, you're too much of a badass to go in there!\r\n`, returning `TRUE`.
  Levels below 11 and immortals (31+) return `FALSE`, with no special output,
  so ordinary movement remains responsible for the result.

## RED/GREEN evidence and port result

- RED on `main`: a level-40 immortal vehicle at room 16300 was blocked by Go,
  while the C oracle fell through to ordinary movement. The focused unit test
  also exposed Go's extra CRLF when the C line ending was passed through the
  writer helper.
- GREEN after the fix: the level-11 blocking vehicle, level-40 immortal
  fallthrough vehicle, and level-10 low-level fallthrough vehicle all report
  no normalized divergence under `--show-oracle`. The focused
  `TestSpecNewbieZoneEntrance_Gates` test covers the command, level, exact
  output, and return gates.
- Manifest rows added: `room.newbie-zone-command-gate`,
  `room.newbie-zone-level-gate`, `room.newbie-zone-block`,
  `room.newbie-zone-return-intercept`,
  `room.newbie-zone-immortal-fallthrough`, and
  `room.newbie-zone-low-level-fallthrough`.
- No `src/` or `darkpawns-c-oracle/` file was edited.

## Verification and integration

- Local gates passed on the feature branch: `make fidelity-depth`,
  `go build ./...`, `go vet ./...`, `go test ./...`,
  `golangci-lint run ./...`, and clean `gofumpt -l .`.
- PR #748 (`glm/spec-newbie-zone-entrance`) received green lint, security,
  and test checks after the required single workflow retry (build/deploy were
  skipped by workflow policy), and was squash-merged as `38083adc1`.

This slice applies R1 (player-facing bytes), R2 (registered room command
surface), R4 (no invented output), and R5/R5e (actual C call path and oracle
behavior).

## Next queue item

Continue special-procedure source order with the active room procedure
`suck_in`, assigned to room 20073 at `src/spec_assign.c:616`. Do not repick
`newbie_zone_entrance`, `start_room`, `pray_for_items`, `fearface`, or earlier
claimed procedures. After active specials, attempt the single blocked
`objmagic.sleep-entry-gates` vehicle, then sweep un-manifested command families
in `src/interpreter.c` table order.

The unrelated untracked brief
`docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved.
