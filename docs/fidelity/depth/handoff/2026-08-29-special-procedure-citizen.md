# Depth-fidelity handoff — 2026-08-29 — `citizen`

## Session and frontier

- Started from `main` after `git pull --ff-only` at `0d560312d`.
- Read `docs/fidelity/DEPTH_TESTING.md`, `docs/fidelity/RULEBOOK.md`, the brief, and the newest prior handoff, `2026-08-28-carve.md`.
- Baseline `make fidelity-depth`: 877 total cases; 855 proven/delegated; 6 blocked; 16 excluded.
- The special-procedure inventory was refreshed from `src/spec_procs.c`, `src/spec_procs2.c`, `src/spec_procs3.c`, and active `ASSIGNMOB` registrations in `src/spec_assign.c`: 113 `SPECIAL` definitions, 233 active registrations, 228 unique active mob vnums, and 66 final assigned procedure names after later registrations win.
- Prior handoffs claimed the special-procedure queue through `dragon_breath`. The next unclaimed procedure in file-and-registration order was `citizen`, registered at vnums 8062 and 18202. No prior handoff claimed it.

## C call path and branch coverage

`src/spec_procs.c:986-1032` was audited against the actual callers: autonomous dispatch in `src/mobact.c:54-93`, combat dispatch in `src/fight.c:1898-2032`, player command dispatch in `src/interpreter.c:1407-1456`, and the C standing messages in `src/act.movement.c:696-730`. The procedure gates non-NPC/player command entry, sleeping and negative-HP mobs; recovers a sitting/resting fighting mob without consuming citizen RNG; otherwise draws `number(0,19)`, then `number(1,10)`, emits the exact eight C room messages (case 2 emits two messages), stays silent for inner cases 9 and 10, and always returns `FALSE`.

## RED and GREEN evidence

- Added `cmd/dp-oracle-diff/scenarios/spec-proc-citizen.txt`, using scriptless active vnum 8062 and padded `~dpclock pulse 40` autonomous dispatch.
- RED before the fix, `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle go run ./cmd/dp-oracle-diff --scenario spec-proc-citizen --seed 1 --show-oracle`: normalized divergence showed the Go-invented `Nice day.` output and missed the C cityguard-bird output.
- GREEN after the fix: seeds 1, 2, 3, 5, and 8 all reported `no normalized divergence`; seed 1 was also checked with `--show-oracle` and showed the exact C room bytes to both actor and peer.
- Added `TestSpecCitizen_EntryGates`, `TestSpecCitizen_StandingRecovery`, `TestSpecCitizen_Sayings`, and `TestSpecCitizen_SilentAndDrawOrder`.

## Port and manifest result

- Replaced the invented Go sayings with the C strings, C gate order, fighting recovery, nested RNG ranges, room audience, and `FALSE` return in `pkg/game/spec_procs.go`.
- Added seven rows to `docs/fidelity/depth/spec-procs.tsv` covering entry gates, recovery, sayings, silent partition/draw order, pulse dispatch, audience, and multiseed RNG parity.
- No files under `src/` or `darkpawns-c-oracle/` were edited.

## Verification and integration

All required local gates passed: `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, `gofumpt -l .` clean, and `make fidelity-depth` exit 0. The post-slice frontier is 884 total cases; 862 proven/delegated; 6 blocked; 16 excluded; actionable completion 862/868 (99.3%).

Commit `2c44dedd5` was pushed as branch `glm/spec-citizen`. PR #735 had green `lint`, `security`, and `test` checks; workflow build/deploy jobs were skipped by policy. It was self-merged cleanly into `main` at `0a4fcc2c4` on 2026-08-29.

The pre-existing untracked `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` was preserved and remains uncommitted.

## Next queue item

Continue the special-procedure inventory with `cuchi` (`src/spec_procs.c:1034`, active registration vnum 18306). Do not repick `citizen`. After the special-procedure inventory is exhausted, attempt the single blocked `objmagic.sleep-entry-gates` row via the cast-sleep outlaw/reagent vehicle, then sweep unmanifested `src/interpreter.c` command families.
