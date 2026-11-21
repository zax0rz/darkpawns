# Round 4 population sweep freeze

- Sweep source SHA: `c28a41e9d427c6b034aee40eca0c4a4c2199765f` (`origin/main` after Round 3 repair PRs #2977, #2978, and #2982 merged).
- Scope and collector: the Round 3 collector `docs/research/idiom-experiment/round2/run/population.go`, over `pkg/boards`, `pkg/combat`, `pkg/command`, `pkg/game`, `pkg/session`, and `pkg/spells`. The collector's syntactic output-call filters are unchanged.
- Command: `GOEXPERIMENT=nojsonv2 go run ./docs/research/idiom-experiment/round2/run/population.go --root . --packages pkg/boards,pkg/combat,pkg/command,pkg/game,pkg/session,pkg/spells --out-dir <raw-output-dir>`.
- Historical exclusions: remove every `(func,file)` identity present in the 2026-11-08, 2026-11-15, 2026-11-18, and mutation-loop Round 1, Round 2, or Round 3 `sample.tsv`. The six source sample paths and their hashes are in `SHA256SUMS`. Function line numbers may move; identity is keyed by function name and file.
- Raw collector result: 654 functions and 3,239 message literals. The historical samples contain 500 rows and 492 distinct function identities; 465 are present in the raw current population and excluded, while 27 are absent.
- Frozen Round 4 pool: **189 functions and 998 message literals**. The pool contains 1 function / 8 literals in `pkg/boards`, 4 / 13 in `pkg/combat`, 16 / 54 in `pkg/command`, 125 / 721 in `pkg/game`, 43 / 202 in `pkg/session`, and 0 / 0 in `pkg/spells`.
- `collector-functions.tsv` and `collector-literals.tsv` retain the full raw collector output. `functions.tsv` and `literals.tsv` contain the filtered pool; `excluded.tsv` lists current functions removed by the historical filter, and `excluded-absent.tsv` lists sampled identities not in the current raw pool.
- Verified all 998 frozen literal tokens against their source-file byte offsets and line numbers at this SHA. SHA-256 values for every collector output, filtered file, exclusion list, and input sample are in `SHA256SUMS`.
- The sweep has no sample seed and is not a measurement round. Round 4's draw seed remains `20261123` and is unused here.

This freeze was committed and pushed before classification and before any sweep census.
