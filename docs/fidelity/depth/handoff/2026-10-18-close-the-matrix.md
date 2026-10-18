# 2026-10-18 — Close the spell matrix

This handoff records Luna's DP-1371 spell-matrix closeout against `origin/main` `de3ab7340cc70a9b09a9bad593416369f825a456`. The work is in one draft PR for Claude's batch review. No full census was run; Claude owns the next combined census.

## Inventory result

| Surface | Before | Draft result | Change |
|---|---:|---:|---:|
| Spell-matrix route units | 261 | 3 | -258 |
| **Total project blocked inventory** | **356 / 1,254** | **98 / 996** | **-258** |

The remaining three matrix units are the existing blocked spoken spell-105 cell and the potion/staff Locate Object cells classified `gap-blocked` under R1a. The other 95 blocked units remain with their existing owners: activity sites, switched-mob `fight.c` sites, DP-1448, lifecycle, and shops.

The 299-by-five matrix changes from 168 proven, 86 delegated, 260 gap-reachable, 1 gap-blocked, and 980 C-unreachable cells on `origin/main` to 426 proven, 86 delegated, 0 gap-reachable, 3 gap-blocked, and 980 C-unreachable cells. Two Locate Object cells moved from gap-reachable to gap-blocked after the C undefined behavior was verified. Route units change as follows:

| Route | Before | Draft result |
|---|---:|---:|
| CAST_SPELL | 1 | 1 |
| CAST_POTION | 59 | 1 |
| CAST_SCROLL | 58 | 0 |
| CAST_WAND | 58 | 0 |
| CAST_STAFF | 85 | 1 |

## Spell proofs and source paths

Eight indexed world items prove their item-specific route behavior: one scroll, three staffs, and four wands. The 52-scenario builder batch covers the remaining OLC-configured route cases, including potion, scroll, wand, and staff dispatch. Across the two batches, all 258 reachable item-route cells have a distinct case ID and five-seed transcript and draw-parity evidence. The 277 added `cast.tsv` claims have unique case IDs; proof columns contain only `scenario@1,2,3,5,8` tokens.

Every item route reaches the registered command and `mag_objectmagic` path in `src/act.other.c:897-974` and `src/spell_parser.c:592-817`. C passes potion item level from `GET_OBJ_VAL(obj, 0)` in `src/spell_parser.c:699-718`; scrolls reach `CAST_SCROLL` in `src/spell_parser.c:660-698`; wand and staff use the `CAST_WAND` / `CAST_STAFF` branches in `src/spell_parser.c:610-658,724-817`, including staff room fan-out. The eight world-item scenarios exercise those paths on their indexed prototypes.

The OLC fixtures use a separate `ItemBuilder` at level 35 to edit through its own descriptor while `ItemMage` remains level 30 for item use. This follows C's `oedit` command and builder gate at `src/interpreter.c:592`, `src/olc.h:54`, and `src/structs.h:620`; the edited object's value 0 supplies item spell level.

The Drowning potion case exposed a real return-contract divergence. C accepts spell numbers through `TOP_SPELL_DEFINE` (`src/spells.h:260`), applies the sitting gate, runs no routines for inactive spell 103, and returns success (`src/spell_parser.c:411-434,472-502`). The potion loop continues after that successful no-op (`src/spell_parser.c:711-715`). Go previously treated its missing `SpellInfo` entry as failure and stopped the potion's later slots. Go now preserves C's no-op and return value. The pre-fix red control `luna-dp1371-potion-drowning-revert-control-seed1` reproduces the divergence with zero draw divergence; the corrected live potion case passes in the five-seed builder batch.

The potion and staff Locate Object cells remain `gap-blocked`. C's potion path passes a null object to `spell_locate_object` at `src/spell_parser.c:713`; staff paths pass null at `src/spell_parser.c:602,605,804,809`. `spell_locate_object` dereferences `obj->name` at `src/spells.c:361`. These scenarios were kept out of the census corpus under R1a. The narrowly approved tooling change in `scripts/spell_matrix.py` allows `gap-blocked` only for these two `(spell_id, route)` pairs and the pre-existing spell-105 spoken exception, verifies their required citations, and self-tests that a third item cell is rejected.

## Retained oracle evidence

Each corrected builder run selects 52 scenarios and is CLEAN with zero draw divergence: `luna-dp1371-matrix-olc-builder-seed1-final`, `luna-dp1371-matrix-olc-builder-seed2`, `luna-dp1371-matrix-olc-builder-seed3`, `luna-dp1371-matrix-olc-builder-seed5`, and `luna-dp1371-matrix-olc-builder-seed8`. The five world-item runs (`luna-dp1371-matrix-world-seed1`, `seed2`, `seed3`, `seed5`, and `seed8`) are also CLEAN with zero draw divergence; only the eight indexed world-item scenarios are counted from that batch. The prior combined Infravision fixture attempted OLC with a level-30 caster and is not counted; its corrected separate-builder fixture passes in the five-seed builder batch.

The Drowning revert control is retained at `/home/zach/Archives/darkpawns/oracle-runs/2026-10-18/luna-dp1371-potion-drowning-revert-control-seed1`. The focused corrected Infravision seed-1 run is `/home/zach/Archives/darkpawns/oracle-runs/2026-10-18/luna-dp1371-itemroute-infravision-flushed-seed1`. All census artifacts are retained under `/home/zach/Archives/darkpawns/oracle-runs/2026-10-18/`.

## Checks

All required gates passed individually: `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...`, `make deadcode-ratchet`, `make fidelity-depth`, `make fidelity-units`, `make string-census`, `gosec -severity=high -quiet ./...`, and the proof-token check. Fidelity-depth reports 6,672 cases with no validation errors; fidelity-units passes 1,918 claims across 14 packages. String-census passes its ratchet with the existing stale-generated-report warning; generated reports were left untouched. The proof-token check prints no missing scenarios.

Elapsed time at handoff capture: 2h 04m (goal timer). PR count: 1.

## Draft PR

- One long-lived draft PR from `origin/main` `de3ab7340cc70a9b09a9bad593416369f825a456` is open for Claude's batch review. The separate matrix-validator tooling commit is called out explicitly in the PR body.

