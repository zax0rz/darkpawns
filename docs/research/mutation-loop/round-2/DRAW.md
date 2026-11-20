# Round 2 draw

- Draw SHA: `2d5166cf0de2023e4eefa7af9272d58219feb6bc`
- Sample seed: `20261121`
- Population scope: `pkg/boards`, `pkg/combat`, `pkg/command`, `pkg/game`, `pkg/session`, `pkg/spells`.
- Collector: `docs/research/idiom-experiment/round2/run/population.go`, run at the draw SHA over all six packages. The collector's stable order is file, function line, then function name.
- Excluded every function in the 11-08, 11-15, 11-18, and Round 1 samples. Those files contain 300 rows and 292 distinct functions (8 overlaps).
- Raw collector population: 666 functions and 3,330 message literals. Filtered population: 388 functions and 1,926 message literals.
- Sampled: 100 functions, then one literal per function. Selection uses `random.Random(20261121).sample(population, 100)` and `random.Random(20261121 + i).choice(literals)` for row `i`, retaining collector order for both lists.
- The first ASCII letter in each selected Go string token is changed to `Q`, or to `Z` if that letter is `Q`. `source_offset` is the Go token's opening-byte offset; `mutation_offset` is the exact source byte to change. All 100 token and byte offsets were checked against the files at the draw SHA.
- Per-package filtered population: `pkg/boards` 4, `pkg/combat` 5, `pkg/command` 26, `pkg/game` 269, `pkg/session` 82, `pkg/spells` 2.
- Per-package sample: `pkg/boards` 3, `pkg/combat` 0, `pkg/command` 3, `pkg/game` 71, `pkg/session` 21, `pkg/spells` 2.

This commit freezes the draw before classification. Classes and evidence are added in the next commit, before any census.
