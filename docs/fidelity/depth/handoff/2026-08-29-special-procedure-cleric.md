# Depth-fidelity handoff — 2026-08-29 — `cleric`

## Session and frontier

- Started from `main` after `git pull --ff-only` at `0bd892f4c`, then reran `make fidelity-depth` and reread `docs/fidelity/DEPTH_TESTING.md` plus the newest handoff, `2026-08-29-special-procedure-cuchi.md`.
- Baseline frontier: 890 total cases; 868 proven/delegated; 6 blocked; 16 excluded.
- Post-slice frontier: 896 total cases; 874 proven/delegated; 6 blocked; 16 excluded; actionable completion 874/880 (99.3%).
- The inventory remains 113 `SPECIAL` definitions, 233 active `ASSIGNMOB` registrations, 228 unique active mob vnums, and 66 final assigned procedure names after later registrations win.

## C call path and branch coverage

`SPECIAL(cleric)` at `src/spec_procs.c:1425-1610` was audited through the autonomous mobile path in `src/mobact.c:68-93` and the combat-time path in `src/fight.c:1898-2032`. Its registered vehicles include vnum 1305 (`src/spec_assign.c:197`) and vnum 11023 (`src/spec_assign.c:321`, repeated later). The live proof uses active, scriptless vnum 11023, a level-20 cleric, so it exercises the native special without the unrelated level-21+ `damage()` switcheroo branch.

The port now covers the C awake/NPC/command/negative-HP gates, sitting/resting `do_stand` room bytes, peaceful-room return, noncombat self-healing through NPC `cast_spell()`, target resolution, lspell calculation and alignment guard, emergency teleport direction, heal/offense split, blindness/curse/poison cleanup order, the intentional lspell 12 and 20-24 holes, weather-gated call lightning, offensive spell mapping, NPC verbal components, and area-spell room narration. The C bitwise-`&` blindness expression's draw is pinned explicitly (R1/R3/R5e).

## RED and GREEN evidence

- Added `cmd/dp-oracle-diff/scenarios/spec-proc-cleric.txt` with `empty-players`, `quiet-mobs`, active vnum 11023, `strip-mob-script`, and a no-exit room. The God explicitly opens combat; `~dpclock pulse 20` pads the wait-setting combat path.
- RED on the pre-fix path exposed missing NPC `say_spell()` bytes, incorrect cleric lspell mapping (including lspell 12), wrong cleanup target/direction, missing weather/sky and stand gates, and absent NPC earthquake room narration.
- GREEN after the fixes: `spec-proc-cleric` reported no normalized divergence for seeds 1, 2, 3, 5, and 8. Seed 3 also pinned the deployed oracle's `SPELL_POISON` target incantation byte `saugab`.
- Added focused unit coverage for entry/stand gates, lspell 12, earthquake room output, call-lightning weather gates, and C's blindness draw order.

## Port and manifest result

- Updated `pkg/game/spec_procs3.go` to follow the C cleric call path and spell branches, using the NPC cast helper so verbal components and cast gates remain intact.
- Updated `pkg/spells/affect_spells.go` for NPC area-room narration and C `act()` capitalization; aligned the deployed oracle's poison incantation in `pkg/spells/say_spell.go` with a regression test.
- Added six rows to `docs/fidelity/depth/spec-procs.tsv`.
- No files under `src/` or `darkpawns-c-oracle/` were edited.

## Verification and integration

All required local gates passed before integration: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and `gofumpt -l .` clean.

PR #737 (`glm/spec-cleric`) passed lint, security, and test checks; build/deploy were skipped by workflow policy. It was squash-merged to `main` at `a4c23f0d0`.

The pre-existing untracked `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved and uncommitted.

## Next queue item

Continue the source-order special-procedure inventory with `outofjailguard` (`SPECIAL` at `src/spec_procs.c:1763`, registered vnum 8089 at `src/spec_assign.c:299`). `conductor` and `brass_dragon` occur first in the source but have no `ASSIGNMOB`/`ASSIGNROOM` registration and must not receive invented vehicles. After the active special inventory, attempt `objmagic.sleep-entry-gates` once, then sweep un-manifested command families in `src/interpreter.c` table order.
