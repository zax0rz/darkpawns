# Round 1 draw

- Draw SHA: `8590800fe1d41b0c57daf606a9dbee97dbce963c`
- Sample seed: `20261120`
- Population scope: `pkg/boards`, `pkg/combat`, `pkg/command`, `pkg/game`, `pkg/session`, `pkg/spells`.
- Excluded prior sampled functions from `docs/research/idiom-experiment/round3/run/sample.tsv`, `docs/research/mutation-sample-2026-11-15/sample.tsv`, and `docs/research/mutation-sample-2026-11-18/sample.tsv`.
- The collector found 666 functions before exclusions; the prior sample files contain 200 rows covering 178 distinct functions.
- Filtered population: 488 functions and 2440 message literals.
- Sampled: 100 functions.
- Per-package population: `pkg/boards` 5, `pkg/combat` 8, `pkg/command` 33, `pkg/game` 339, `pkg/session` 101, `pkg/spells` 2.
- Per-package sample: `pkg/boards` 1, `pkg/combat` 3, `pkg/command` 7, `pkg/game` 70, `pkg/session` 19.

The sample uses `random.Random(seed).sample(population, 100)` in the collector’s stable order, then `random.Random(seed + i).choice(literals)` for each sampled function. The first ASCII letter is changed to `Q` or `Z` (using `Z` only when the original letter is `Q`).

SHA-256 values for the frozen input and draw TSVs are in `SHA256SUMS`.
