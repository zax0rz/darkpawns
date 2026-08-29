# Depth-fidelity handoff — 2026-08-29 — `cuchi`

## Session and frontier

- Started from `main` after `git pull --ff-only` at `0a4fcc2c4` (citizen merged).
- Reran `make fidelity-depth` and reread `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff, `2026-08-29-special-procedure-citizen.md`.
- Baseline frontier: 884 total cases; 862 proven/delegated; 6 blocked; 16 excluded.
- The refreshed C inventory remains 113 `SPECIAL` definitions, 233 active `ASSIGNMOB` registrations, 228 unique active mob vnums, and 66 final assigned procedure names after later registrations win.
- In source order, `cuchi` was the next unclaimed active procedure after `citizen`, registered at vnum 18306 (`src/spec_assign.c:415`). The intervening definitions `mini_thief`, `black_undead_knight`, `red_undead_knight`, and `mickey`/`mallory` have no `ASSIGNMOB` or `ASSIGNROOM` registration, so R2 provides no legitimate dispatch vehicle for them. The next active procedure after those is `cleric`.

## C call path and branch coverage

`src/spec_procs.c:1034-1071` was audited through `src/interpreter.c:597` and the special dispatch at `src/interpreter.c:1407-1456`. `CMD_IS("pat")` is the only gate and ignores the argument. The ordinary branch sends the actor two exact `stc()` lines and adds 10 gold; the exact-name `Orodreth` branch instead promotes the actor to `LVL_IMPL`. Both branches then send a `TO_ROOM` pat line and branch-specific purr line, excluding the actor from those room messages, and return `TRUE` so `do_action` does not run a second social. Autonomous `cmd==0` dispatch falls through immediately.

## RED and GREEN evidence

- Added `cmd/dp-oracle-diff/scenarios/spec-proc-cuchi.txt` with scriptless vnum 18306, a primary actor, and a passive room peer.
- RED on the pre-fix implementation, `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle go run ./cmd/dp-oracle-diff --scenario spec-proc-cuchi --seed 1 --show-oracle`: the old Go code leaked the TO_ROOM pat/purr lines to the actor, emitted extra blank lines from doubled CRLFs, and used runtime mob text instead of the C literal `Cuchi`.
- GREEN after the fix: the same seed with `--show-oracle` reported `no normalized divergence`; C's actor-only and peer-only blocks were visible and exact.
- Focused tests cover autonomous/non-pat/missing-player gates, ordinary +10 gold and argument ignorance, exact-name Orodreth promotion, alternate purr text, audience ordering, CRLF bytes, and the TRUE interception result.

## Port and manifest result

- Replaced the invented/random `specCuchi` behavior with the C command gate, literal messages, `Act(..., ToRoom)` audience semantics, exact-name promotion, gold branch, and `TRUE` return in `pkg/game/spec_procs.go`.
- Added six rows to `docs/fidelity/depth/spec-procs.tsv` covering gates, ordinary pat, Orodreth promotion, audience, ignored argument, and command interception.
- No files under `src/` or `darkpawns-c-oracle/` were edited.

## Verification and integration

All required local gates passed: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .` clean, and `make fidelity-depth` exit 0. The post-slice frontier is 890 total cases; 868 proven/delegated; 6 blocked; 16 excluded; actionable completion 868/874 (99.3%).

The cuchi slice is on branch `glm/spec-cuchi`; its PR is to be opened after this handoff is committed. The pre-existing untracked `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved and uncommitted.

## Next queue item

Continue the active special-procedure inventory with `cleric` (`src/spec_procs.c:1425`, first active registration at `src/spec_assign.c:197`, vnum 1305). Do not repick `cuchi` or invent vehicles for the unregistered intervening definitions. After active special procedures are exhausted, attempt `objmagic.sleep-entry-gates` once through the cast-sleep outlaw/reagent vehicle, then sweep unmanifested command families in `src/interpreter.c` table order.
