# Round-two population fingerprint

- Code base: `5e9556a8789c4125bed8913d34d16e6f2ed68fac` (round-one PR #2275 tip; no source changes before sampling).
- Population script: `population.go`; it records each non-test function, method, and function literal that passes a source string literal to an explicit player-output call/callback or through a same-function message value or output builder. The exact syntactic rule and call catalog are at the top of the script.
- Population rows: 424 functions, sorted by file and declaration line.
- `population.tsv` SHA-256: `55ddde8196f5542893131cc9686b370b32cd3625245b3faf56a670b2a29adf36`.
- Auxiliary literal inventory SHA-256: `e43bef61cf4c78f75dca1c252ea880158b19ebb4ffe1ebbf664fae51894fc864`.
- Reproduce the hashes from the repository root with `sha256sum docs/research/idiom-experiment/round2/run/population.tsv docs/research/idiom-experiment/round2/run/population-literals.tsv`.

The inventory counts only message argument positions and follows only string-like local expressions, avoiding non-message arguments such as the blank Act substitution argument. This fingerprint is committed before drawing the seeded sample. `sample.tsv` and probe predictions will be added in the next commit.
