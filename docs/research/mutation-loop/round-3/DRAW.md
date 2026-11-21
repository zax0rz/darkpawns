# Round 3 draw

- Draw SHA: `cc94e6d25b1ea099421a12a0a88c6a08d20d7ae0`
- Sample seed: `20261122`
- Population scope: `pkg/boards`, `pkg/combat`, `pkg/command`, `pkg/game`, `pkg/session`, `pkg/spells`.
- Collector: `docs/research/idiom-experiment/round2/run/population.go`, run at the draw SHA over all six packages. Stable order is file, function line, then function name.
- Excluded all functions sampled in the 2026-11-08, 2026-11-15, and 2026-11-18 samples and mutation-loop Rounds 1 and 2. Inputs: `docs/research/idiom-experiment/round3/run/sample.tsv`, `docs/research/mutation-sample-2026-11-15/sample.tsv`, `docs/research/mutation-sample-2026-11-18/sample.tsv`, `docs/research/mutation-loop/round-1/sample.tsv`, and `docs/research/mutation-loop/round-2/sample.tsv`.
- Prior sample files contain 400 rows and 392 distinct function identities; 23 identities are absent from the current collector population.
- Raw collector population: 654 functions and 3239 message literals. Filtered population: 285 functions and 1406 message literals.
- Sampled: 100 functions, then one literal per function. Selection uses `random.Random(20261122).sample(population, 100)` and `random.Random(20261122 + i).choice(literals)` for row `i`, retaining collector order for both lists.
- The first ASCII letter in each selected Go string token is changed to `Q`, or to `Z` if that letter is `Q`. `source_offset` is the Go token's opening-byte offset; `mutation_offset` is the exact source byte to change. All 100 token and byte offsets were checked against the files at the draw SHA.
- Per-package filtered population: pkg/boards 1, pkg/combat 5, pkg/command 22, pkg/game 198, pkg/session 59, pkg/spells 0.
- Per-package sample: pkg/boards 0, pkg/combat 1, pkg/command 6, pkg/game 73, pkg/session 20, pkg/spells 0.

The pre-classification `sample.tsv` SHA-256 is `5158eb62e06b6d300d2d1c3ee733020b34391d42918c91ba1301325b3cda2392`. The population, literals, and pre-classification sample hashes are recorded in `SHA256SUMS`. The next commit adds classification evidence before any census.
