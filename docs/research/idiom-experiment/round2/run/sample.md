# Round-two population fingerprint

- Code base: `5e9556a8789c4125bed8913d34d16e6f2ed68fac` (round-one PR #2275 tip; no source changes before sampling).
- Population script: `population.go`; it records each non-test function, method, and function literal that passes a source string literal to an explicit player-output call/callback or through a same-function message value or output builder. The exact syntactic rule and call catalog are at the top of the script.
- Population rows: 402 functions, sorted by file and declaration line.
- `population.tsv` SHA-256: `a790e7a2d87f334eb37495f1433714de54b388d8d7d9f865ce1f13c52451b631`.
- Auxiliary literal inventory SHA-256: `a1e5843136853f5080b68fd3c101521603a5eff98c4ce30ed6eaa01310b34c22`.
- Reproduce the hashes from the repository root with `sha256sum docs/research/idiom-experiment/round2/run/population.tsv docs/research/idiom-experiment/round2/run/population-literals.tsv`.

The inventory counts only message argument positions and follows only string-like local expressions, avoiding non-message arguments such as the blank Act substitution argument. Literals without an ASCII letter (such as line endings and layout-only separators) are omitted because the required mutation targets the first ASCII letter. Every sampled function therefore has at least one probe-eligible literal. This fingerprint is committed before drawing the seeded sample. `sample.tsv` and probe predictions will be added in the next commit.
