# 2026-10-17 — Spell matrix and skill-message closeout

This handoff records Luna's DP-1371 matrix-and-messages goal, measured against `origin/main` `a185332db373a0012100e64659900c326db2419c`. Two draft PRs remain open for Claude's Tuesday batch review. Nothing was merged or deployed; no full census was run.

## Inventory result

| Surface | `origin/main` units | Draft result | Change |
|---|---:|---:|---:|
| Skill-message corpus | 51 | 1 | -50 |
| Spell matrix routes | 349 | 261 | -88 |
| **Total project blocked inventory** | **494 / 1,392** | **356 / 1,392** | **-138** |

The remaining skill-message unit is the DP-1448 Call Lightning NPC miss lane. Ten other message records are now source-audited C-unreachable. The spell matrix retains 260 reachable gaps and one existing blocked spoken cell. Other inventory owners were left unchanged.

## Skill-message corpus

On origin main, 32 records were proven, 50 were gap-reachable, and one was blocked. The draft result is 72 proven, zero gap-reachable, one DP-1448 gap-blocked, and ten C-unreachable. Outcome-audience cells move from 451 gap-reachable, 351 proven, 3 blocked, 27 unreachable, and 164 `none` to zero gap-reachable, 674 proven, 3 blocked, 155 unreachable, and 164 `none`.

The five broad weapon batches each pass all 36 selected scenarios with draw parity and zero divergent streams: `luna-dp1371-skill-weapon-seed1-final`, `...seed2-final`, `...seed3-final`, `...seed5-final`, and `...seed8-final`. The corrected Whip God fixture has five dedicated targeted CLEAN draw-parity runs: `luna-dp1371-skill-302-god-seed1-draw-final`, `...seed2-draw-final`, `...seed3-draw-final`, `...seed5-draw-final`, and `...seed8-draw-final`. Each 16-hit run selects both M@0475 and M@0489 and retains the attacker, victim, and room lines for each; every draw-parity result is zero. C `do_dark` resets the fight between fresh registered `force` -> `hit` dispatches. This closes the last selector-coverage hole in the weapon batch.

C path: `src/interpreter.c:408,438,495`; `src/act.wizard.c:1856-1881,3220-3235`; `src/act.offensive.c:98-129`; `src/fight.c:1023-1095,1314-1545,1763-1805`; message records in `lib/misc/messages`. The existing M@0189 Meteor Swarm miss-attacker gap is closed through the live Lua mob-oncmd route; its pre-fix red control is retained. M@0045's Call Lightning miss remains under DP-1448 and was not edited.

## Spell matrix

The 299-row matrix changes from 96 proven, 70 delegated, 348 gap, one blocked, and 980 C-unreachable cells to 168 proven, 86 delegated, 260 gap, one blocked, and 980 C-unreachable cells. Of the 88 closed gaps, 72 are direct live proofs and 16 are source-audited delegations. Inventory by route changes as follows:

| Route | Before | After |
|---|---:|---:|
| CAST_SPELL | 10 | 1 |
| CAST_POTION | 89 | 59 |
| CAST_SCROLL | 79 | 58 |
| CAST_WAND | 78 | 58 |
| CAST_STAFF | 93 | 85 |

The five corrected `luna-dp1371-spell-matrix-seedN-final-corrected` runs (N = 1, 2, 3, 5, 8) cover 29 scenarios / 30 cases with zero draw divergence; seed 5 is CLEAN_AFTER_RECHECK after its lone infrastructure row was rechecked alone. The retained group-recall red run showed Go recalling an ungrouped caster while C returned at `mag_groups`'s `AFF_GROUP` guard (`src/magic.c:1465-1475`). Go now has the same caster guard. A positive grouped wand fixture and an ungrouped-caster no-op fixture both pass. Regression controls are `TestMagGroupsGroupRecallRequiresGroupedCaster`, `TestMagGroupsGroupRecallDispatchesSleepingMember`, and the two live wand scenarios. Red controls for Calliope and Mass Dominate are also retained.

World-item call paths run through registered `do_quaff`, `do_recite`, and `do_use` dispatches in `src/act.other.c:897-973` and `src/spell_parser.c:544-817`. Delegations cite the common body and same C arguments, including Teleport (`src/spells.c:168-213`), Clone's `mag_summons` no-op (`src/magic.c:1672-1707`), Word Recall (`src/spells.c:124-165`), Lycanthropy (`src/spells.c:662-685`), Vampirism (`src/spells.c:766-791`), and Mass Dominate (`src/magic.c:1594-1620`; `src/spells.c:407-471`).

## Checks and evidence handling

All required gates passed individually in both worktrees: `make fmt`, `go build ./...`, `go vet ./...`, `go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...`, `make deadcode-ratchet`, `make fidelity-depth`, `make fidelity-units`, `make string-census`, `gosec -severity=high -quiet ./...`, and the manifest proof-token check. Spell `fidelity-units` passed 1,948 claims; skill passed 1,941. String census passed its ratchet with the existing stale-generated-report warning; generated reports remain untouched. Proof fields contain scenario@seed tokens or test symbols; census run names live in notes. The proof-token check printed no missing scenarios.

Evidence is retained under `/home/zach/Archives/darkpawns/oracle-runs/2026-10-17/`, including the five corrected spell censuses, five broad weapon censuses, five corrected Whip God censuses, red controls, and the seed-1 normalized C coverage dump. Final goal elapsed time: 9h 08m. PR count: 2.

## Draft PRs

- [#2095 — skill-message corpus](https://github.com/zax0rz/darkpawns/pull/2095)
- [#2096 — spell matrix](https://github.com/zax0rz/darkpawns/pull/2096)

Both remain drafts on fresh `origin/main` worktrees. Claude's batch should stack them and run the single combined full census; a failing row can then be attributed by rerunning that scenario against each PR.
