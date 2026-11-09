# Predictions for the Sol audit of pkg/spells (pre-registered)

2026-11-06, by Claude. Committed before Sol (gpt-6.1-sol) starts [BRIEF-sol.md](BRIEF-sol.md), blind to the DeepSeek and MiMo ledgers in this folder.

Run Sol at **medium** reasoning effort, the effort it used for combat, so the two packages compare.

| # | Measure | Prediction | 80% interval |
|---:|---|---:|---:|
| 1 | Share of the 191 tests marked R | 77% | 68–87% |
| 2 | Mutations run | 20 | 8–50 |
| 3 | Cohen's kappa, Sol vs MiMo | 0.35 | 0.15–0.60 |
| 4 | Cohen's kappa, Sol vs DeepSeek | 0.35 | 0.15–0.60 |
| 5 | Finds the mirror-image clone level divergence (C creates it at level 1, `src/spells.c:1136`) | yes, 50% | |
| 6 | Flags the invented `spells.AttackTypes` words (no C source) | yes, 50% | |
| 7 | Reports `World.HandleSpellDeath` as having no production caller | yes, 35% | |
| 8 | Reports the reagent bonus (`checkReagents` returns a level bonus; C's `has_reagents` is true or false) | yes, 55% | |
| 9 | Marks D on at least 10 of the helper micro-tests that MiMo marked D | yes, 40% | |
| 10 | Finds a C divergence neither smaller model reported | yes, 35% | |
| 11 | Reports contamination (saw prior audit output) | yes, 15% | |

Reasoning:

- In combat, Sol landed near the smaller models on R (76%) but agreed less with each of them than they did with each other. In spells they agreed with each other at only 0.30, so I expect Sol to sit between them, not closer to either.
- Items 5–8 are the real defects so far. Each was caught by one smaller model, or by both. A frontier pass that finds each about half the time would show that a third audit adds coverage rather than repeating.
- In combat, Sol found the NPC wait-state divergence alone (PF-104, later PF-105). Item 10 is the same bet again.

## Zach's prediction

(Record it here when you hand the brief to Sol.)

## Results (scored 2026-11-06)

Sol (gpt-6.1-sol, medium effort) reported "Contamination: none". It split mixed functions into subtest rows (253 rows); the scores below collapse each function to its weakest mark (R < F < C < D) and compare the 190 functions where DeepSeek and MiMo each have one row.

| # | Prediction | Result | Hit? |
|---:|---|---|---|
| 1 | 77% R (68–87%) | 67.5% (129 of 191) | **no**, just below |
| 2 | 20 mutations (8–50) | 38 test observations (26 green, 12 red) | yes |
| 3 | kappa vs MiMo 0.35 (0.15–0.60) | 0.54 | yes |
| 4 | kappa vs DeepSeek 0.35 (0.15–0.60) | 0.42 | yes |
| 5 | finds mirror image's clone level (50%) | yes | yes |
| 6 | flags the invented `AttackTypes` words (50%) | yes, marks the size and index-zero tests D | yes |
| 7 | reports `HandleSpellDeath` uncalled (35%) | no; it found the uncalled private `World.doDamage` instead | **no** |
| 8 | reports the reagent level bonus (55%) | yes | yes |
| 9 | D on at least 10 of MiMo's 16 helper deletes (40%) | 8 D, 2 F, 6 R | **no** |
| 10 | a C divergence neither smaller model reported (35%) | yes, several (see the field note) | yes |
| 11 | reports contamination (15%) | no | yes |

Eight of eleven. The DeepSeek–MiMo kappa on the same 190 functions is 0.30; Fleiss kappa across all three is 0.43. Unlike combat, Sol agreed with each smaller model more than they agreed with each other.
