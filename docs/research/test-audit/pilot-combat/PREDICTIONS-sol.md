# Predictions for the Sol audit (pre-registered)

2026-11-05, by Claude, committed before Sol (gpt-6.1-sol) starts the same brief ([BRIEF-sol.md](BRIEF-sol.md)), blind to the MiMo and DeepSeek ledgers in this folder.

| # | Measure | Prediction | 80% interval |
|---:|---|---:|---:|
| 1 | Share of the 208 tests marked R | 80% | 70–90% |
| 2 | Mutations run | 25 | 10–60 |
| 3 | Cohen's kappa, Sol vs MiMo | 0.65 | 0.45–0.85 |
| 4 | Cohen's kappa, Sol vs DeepSeek | 0.70 | 0.50–0.85 |
| 5 | Finds the `counter_procs` duplicate's C divergence | yes, 80% | |
| 6 | Marks `TestCalculateHitChance_Natural20` F or D (it can't fail) | yes, 60% | |
| 7 | Every F/C/D row predates the oracle (2026-07-12), as both others found | yes, 70% | |
| 8 | Finds a C divergence neither cheap model reported | yes, 35% | |
| 9 | Reports contamination (saw prior audit output) | yes, 20% | |

Reasoning: a frontier model should run more mutations and so mark more F than MiMo (7) but about as many as DeepSeek (15). Its marks should agree with DeepSeek's slightly more, because DeepSeek ran a mutation for every F claim, and so did the brief's strictest reading.

## Zach's prediction

Recorded 2026-11-05, as Zach delivered the brief to Sol: "it'll find the same things the smaller models found with nits. this guy ALWAYS has nits and LOVES to add tests." Scored as two checks:

- **Same findings:** Sol reports both the `counter_procs` divergence and the vacuous natural-20 test, and nothing major beyond them.
- **Nits and tests:** Sol marks more F than either smaller model (more than 15), and its notes recommend adding new tests.

## Results (scored 2026-11-05)

Sol (gpt-6.1-sol, **medium** reasoning effort) finished in about 8 minutes and reported "Contamination: none". It split one function by subtest (209 rows); scores below collapse that to the weaker mark per function.

| # | Prediction | Result | Hit? |
|---:|---|---|---|
| 1 | 80% R (70–90%) | 76.4% (159 of 208) | yes |
| 2 | 25 mutations (10–60) | 23 | yes |
| 3 | kappa vs MiMo 0.65 (0.45–0.85) | 0.49 | yes, at the low end |
| 4 | kappa vs DeepSeek 0.70 (0.50–0.85) | 0.44 | **no** |
| 5 | finds the `counter_procs` divergence (80%) | yes | yes |
| 6 | marks Natural20 F or D (60%) | marked C (merge) | **no** |
| 7 | every F/C/D predates the oracle (70%) | no: 3 F and 3 C after 2026-07-12 | **no** |
| 8 | finds a C divergence neither smaller model reported (35%) | yes: NPC wait state | yes |
| 9 | reports contamination (20%) | no | yes |

**Zach's prediction:**

- **Same findings as the smaller models, with nits:** half right. It found `counter_procs`, and it had plenty of nits (28 C, the most consolidation marks of the three). But it also found two major things the smaller models missed: the wait-state divergence, and the history of the AC golden table.
- **More F than either smaller model:** yes, barely (16 against DeepSeek's 15).
- **Recommends new tests:** no. Its notes propose no new tests.
