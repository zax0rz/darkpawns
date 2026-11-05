# Idiomatic-Go experiment, round two: results

Measured on the branch based on `5e9556a8789c4125bed8913d34d16e6f2ed68fac`
(round-one PR #2275). The full metrics and four-linter counts are in
[`run/after.tsv`](run/after.tsv); the frozen sample outcomes are in
[`run/probes.tsv`](run/probes.tsv).

| # | Measure | Predicted | Observed | Assessment |
|---:|---|---:|---:|---|
| 1 | Empty `interface{}` occurrences | 0 (0–5) | 0 | Inside interval |
| 2 | snake_case declarations | 15 (0–60) | 126 | Outside interval; the audit found no eligible unexported rename, and exported/C-named identifiers were left alone |
| 3 | Net lines in scope | −300 (+1,000 to −2,500) | +101 (73,540 → 73,641) | Outside interval |
| 4 | Functions over 100 lines | 30 (15–43) | 33 (45 → 33) | Inside interval |
| 5 | Longest function | 400 (150–747) | 747 (unchanged) | At interval boundary |
| 6 | Tier B refactor commits kept | 20 (5–50) | 21 | Inside interval |
| 7 | Ranked inventory skipped as “unsure” | 45% (20–75%) | 0 of 12 rank rows (0%) | Outside interval; other skips have named measurement, route, PR, or static-data reasons |
| 8 | Kept Tier B commits whose first real census went red | 3 (0–10) | 0 of 21 | Inside interval |
| 9 | Sample probes that stayed green | 30% (12 of 40; 15–50%) | 17 of 40 (42.5%) | Inside interval |
| 10 | Sample functions no scenario reaches | 20% (5–40%) | 12 of 40 were operationally classified as no-route predictions (blank predicted scenarios plus green probes) | Inside interval as a probe-based classification; the run did not collect function-coverage traces, so this is not a direct reach count |
| 11 | Latent game-port bugs logged | 6 (1–20) | 0 | Outside interval; one separate population-inventory limitation was found and is documented in `run/REPORT.md` |
| 12 | Protected-path edits | 0 (0–0) | 0 | As predicted |
| 13 | Kept green commit later found to change behavior | 30% | Pending Claude review | Open |

## Stage 1 probe results

The frozen 40-function sample caught 23 mutations (23/40, **57.5%**) and left
17 green. There were eight unrelated `INFRA` rows across seven probe runs:

- Probes 0, 19: `attitude-loot-switched-output-depth`.
- Probes 2, 4, 23: `attitude-loot-gold-depth`.
- Probes 27: `attitude-loot-switched-output-depth`.
- Probe 37: both scenarios above.

The wrapper rechecked the infrastructure rows. Probe 23's recheck hit the same
infra scenario, so its required full retry was run and clean. The final probe
outcomes count only completed mutation verdicts. The initially registered,
unmodified probe-0 run was canceled before execution; the retained
`probe-0-attempt-2` is the actual mutated attempt.

Against the scenarios written before each census, the agent predictions were
**23 exact, 7 partly correct, and 10 incorrect**. Five blank scenario
predictions were disproved by caught mutations. The 12 blank predictions that
stayed green are the probe-based classification reported in row 10; green
alone cannot distinguish an unreachable function from an unexercised output
branch.

The Stage 1 population remains the preregistered 402 rows and its recorded
SHA-256. Post-run review found that its same-function call catalog misses at
least `Dream` and `DreamTravelFn`: `DreamContext.SendToChar` and `SendToRoom`
have one explicit argument, but `population.go` maps both names to argument
index 1, while selector calls expose the message at `call.Args[0]`. The frozen
sample and denominator were not regenerated after this was discovered.
