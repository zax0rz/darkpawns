# Round-two population fingerprint

- Code base: `5e9556a8789c4125bed8913d34d16e6f2ed68fac` (round-one PR #2275 tip; no source changes before sampling).
- Population script: `population.go`; it records each non-test function, method, and function literal that passes a source string literal to an explicit player-output call/callback or through a same-function message value or output builder. The exact syntactic rule and call catalog are at the top of the script.
- Population rows: 476 functions, sorted by file and declaration line.
- `population.tsv` SHA-256: `de9ed5755a829c11f63fb9f741d441b6cecd50d50523486540864c1aa8b5454c`.
- Auxiliary literal inventory SHA-256: `e8c07051f0fb0d68f545eaa651ce5b12581193917437b7a7f2be8dbd27fa9665`.
- Reproduce the hashes from the repository root with `sha256sum docs/research/idiom-experiment/round2/run/population.tsv docs/research/idiom-experiment/round2/run/population-literals.tsv`.

This fingerprint is committed before drawing the seeded sample. `sample.tsv` and probe predictions will be added in the next commit.
