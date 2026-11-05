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
