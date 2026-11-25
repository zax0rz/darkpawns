# Round 4 draw

- Draw SHA: `f3d46e951e8c73f154dfecf38770dd4ae130ada2`.
- Sample seed: `20261123`.
- Population scope: `pkg/boards`, `pkg/combat`, `pkg/command`, `pkg/game`, `pkg/session`, `pkg/spells`.
- Collector: `docs/research/idiom-experiment/round2/run/population.go`, run at the draw SHA over all six packages. Stable order is file, function line, then function name.
- Excluded every sampled `(func,file)` identity from the 2026-11-08, 11-15, and 11-18 samples and mutation-loop Rounds 1, 2, and 3: `docs/research/idiom-experiment/round3/run/sample.tsv`, `docs/research/mutation-sample-2026-11-15/sample.tsv`, `docs/research/mutation-sample-2026-11-18/sample.tsv`, and `docs/research/mutation-loop/round-{1,2,3}/sample.tsv`.
- Prior files contain 500 rows and 492 distinct function identities; 31 identities are absent from the raw collector population at the draw SHA.
- Raw collector population: 648 functions and 3182 message literals. Filtered population after exclusions: 187 functions and 976 message literals.
- Sampled: 100 functions, then one literal per function. Selection uses `random.Random(20261123).sample(population, 100)` and `random.Random(20261123 + i).choice(literals)` for row `i`, retaining collector order for both lists.
- Mutation: first ASCII letter in the selected Go string value changes to `Q`, or `Z` if it is already `Q`. `source_offset` is the token-opening byte offset; `mutation_offset` is the source byte for that letter. All 100 tokens and offsets were checked against the draw SHA before classification.
- Per-package raw collector population: `pkg/boards` 5, `pkg/combat` 10, `pkg/command` 34, `pkg/game` 412, `pkg/session` 151, `pkg/spells` 36.
- Per-package filtered population: `pkg/boards` 1, `pkg/combat` 4, `pkg/command` 16, `pkg/game` 125, `pkg/session` 41, `pkg/spells` 0.
- Per-package sample: `pkg/boards` 0, `pkg/combat` 2, `pkg/command` 9, `pkg/game` 71, `pkg/session` 18, `pkg/spells` 0.
- Of the 100 sampled functions, 99 occur in the frozen Round 4 sweep pool and 1 are new to that pool. This does not split the headline.

The pre-classification `sample.tsv` SHA-256 is `0de218da476549a5e6e57f5350dc05216faf15f05d476b8e627eb5595a4cfeb2`. The population, literals, and pre-classification sample hashes are in `SHA256SUMS`. Classification is the next commit and must land before any census.
