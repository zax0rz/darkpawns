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

## Results

(Scored by Claude after the run.)
