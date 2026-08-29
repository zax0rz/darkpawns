# Depth-fidelity handoff — 2026-08-29 — `outofjailguard`

## Session and frontier

- Started from `main` after `git pull --ff-only` at `74e18907e` (cleric handoff), reran `make fidelity-depth`, and reread `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff.
- Baseline frontier: 896 total cases; 874 proven/delegated; 6 blocked; 16 excluded.
- Post-slice frontier: 900 total cases; 878 proven/delegated; 6 blocked; 16 excluded; actionable completion 878/884 (99.3%).
- The inventory remains 113 `SPECIAL` definitions, 233 active `ASSIGNMOB` registrations, 228 unique active mob vnums, and 66 final assigned procedure names after later registrations win.

## C call path and branch coverage

`SPECIAL(outofjailguard)` at `src/spec_procs.c:1763-1780` was audited through the command-table dispatch in `src/interpreter.c:1407-1456`, the canonical movement command path, and `src/comm.c:2392-2555` for `act()`. It is registered for mob vnum 8089 at `src/spec_assign.c:299`; the world reset places that guard in room 8117, whose south exit leads to the main holding cell.

The port now follows C's `IS_MOVE`/south gate, mortal and non-hunting gate, `ch->in_room` check, exact `TO_ROOM` substitution and actor exclusion, exact TO_CHAR collar warning, and TRUE return that prevents `do_move()`. The Go world has no player hunting state, so its player query is the C-equivalent false result. The first vehicle run used an immortal after `goto`; that was corrected with the existing `set ... level 1` warmup before claiming the mortal branch.

## RED and GREEN evidence

- Added `cmd/dp-oracle-diff/scenarios/spec-proc-outofjailguard.txt` with `empty-players`, `quiet-mobs`, active vnum 8089, a scriptless guard, a co-located peer, and the south probe in room 8117.
- RED on the pre-fix implementation showed Go leaking the room message to the actor, leaving `$n`/`$s` unsubstituted for the peer, and using the wrong CRLF boundary. The corrected mortal vehicle exposed the exact C actor/peer split.
- GREEN after the fix: `spec-proc-outofjailguard` reported no normalized divergence for seeds 1, 2, 3, 5, and 8.
- Focused tests cover missing-player/autonomous, non-movement, immortal, wrong-room, audience substitution/exclusion, and one-CRLF output.

## Port and manifest result

- Updated `pkg/game/spec_procs4.go` to use the C level/hunting/player-room gates, `Act(..., ToRoom)`, and the exact separate TO_CHAR message.
- Added `pkg/game/spec_outofjailguard_test.go`, the live scenario, and four rows to `docs/fidelity/depth/spec-procs.tsv`.
- No files under `src/` or `darkpawns-c-oracle/` were edited.

## Verification and integration

All required local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and `gofumpt -l .` clean.

PR #738 (`glm/spec-outofjailguard`) passed lint, security, and test checks; build/deploy were skipped by workflow policy. It was squash-merged to `main` at `fd4b9b57d`.

The pre-existing untracked `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved and uncommitted.

## Next queue item

Continue the source-order special-procedure inventory with `jailguard` (`SPECIAL` at `src/spec_procs.c:1781`, registered vnum 8088 at `src/spec_assign.c:298`). Do not repick `outofjailguard` or invent vehicles for any unregistered definitions. After active special procedures are exhausted, attempt `objmagic.sleep-entry-gates` once through the cast-sleep outlaw/reagent vehicle, then sweep unmanifested command families in `src/interpreter.c` table order.
