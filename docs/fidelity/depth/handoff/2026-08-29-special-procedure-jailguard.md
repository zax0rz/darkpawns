# Depth-fidelity handoff — 2026-08-29 — `jailguard`

## Session and frontier

- Started from `main` after `git pull --ff-only` at `520946321` (out-of-jail guard handoff), reran `make fidelity-depth`, and reread `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff.
- Baseline frontier: 900 total cases; 878 proven/delegated; 6 blocked; 16 excluded.
- Post-slice frontier: 904 total cases; 882 proven/delegated; 6 blocked; 16 excluded; actionable completion 882/888 (99.3%).
- The inventory remains 113 `SPECIAL` definitions, 233 active `ASSIGNMOB` registrations, 228 unique active mob vnums, and 66 final assigned procedure names after later registrations win.

## C call path and branch coverage

`SPECIAL(jailguard)` at `src/spec_procs.c:1781-1798` was audited through the command-table dispatch in `src/interpreter.c:1407-1456`, the canonical movement path, and `src/comm.c:2392-2555` for `act()`. It is registered for mob vnum 8088 at `src/spec_assign.c:298`; the world reset places that script-bearing guard in room 8118, the main holding cell, with north leading back toward room 8117.

The port now follows C's `IS_MOVE`/north gate, mortal and non-hunting gate, `ch->in_room` check, exact `TO_ROOM` name/pronoun substitution and actor exclusion, exact TO_CHAR flabby-hand warning, and TRUE return that prevents `do_move()`. The Go world has no player hunting state, so its player query is the C-equivalent false result. The scriptless fixture was required to isolate the native special from `jailguard.lua`.

## RED and GREEN evidence

- Added `cmd/dp-oracle-diff/scenarios/spec-proc-jailguard.txt` with `empty-players`, `quiet-mobs`, active vnum 8088, `strip-mob-script`, a co-located peer, and the north probe in room 8118.
- RED on the pre-fix implementation showed Go leaking the room message to the actor and leaving `$n`/`$m` unsubstituted for the peer. The mortal vehicle used the existing wizard `set ... level 1` warmup after teleporting the peer.
- GREEN after the fix: `spec-proc-jailguard` reported no normalized divergence for seeds 1, 2, 3, 5, and 8; `--show-oracle` confirmed the intended actor/peer blocks.
- Focused tests cover missing-player/autonomous, non-movement, immortal, wrong-room, audience substitution/exclusion, and exact CRLF output.

## Port and manifest result

- Updated `pkg/game/spec_procs4.go` to use the C level/hunting/player-room gates, `Act(..., ToRoom)`, and the exact separate TO_CHAR message for jailguard.
- Added `pkg/game/spec_jailguard_test.go`, the live scenario, and four rows to `docs/fidelity/depth/spec-procs.tsv`.
- No files under `src/` or `darkpawns-c-oracle/` were edited.

## Verification and integration

All required local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and `gofumpt -l .` clean.

PR #739 (`glm/spec-jailguard`) required the one permitted workflow retry because no checks initially appeared; the retry produced a fully green lint/security/test run, with build/deploy skipped by workflow policy. It was squash-merged to `main` at `d8fb1511a`.

The pre-existing untracked `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved and uncommitted.

## Next queue item

Continue the source-order special-procedure inventory with `dracula` (`SPECIAL` at `src/spec_procs.c:1799`, registered vnums 7903 at `src/spec_assign.c:262` and 14110 at `src/spec_assign.c:432`). Do not repick `jailguard` or invent vehicles for any unregistered definitions. After active special procedures are exhausted, attempt `objmagic.sleep-entry-gates` once through the cast-sleep outlaw/reagent vehicle, then sweep unmanifested command families in `src/interpreter.c` table order.
