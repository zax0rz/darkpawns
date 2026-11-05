# Setup notes

- Worktree: `/home/zach/wt/luna-idiom-go` on `luna/idiomatic-go-2026-11-03`.
- Base: `dce50f5bf398ec858a5f3bf92e8b4b61b2664357`, fetched from
  `origin/main` on 2026-11-04. No rebase.
- The brief's `go run ./docs/research/idiom-experiment/metrics . ...` command
  fails here because `metrics/` has its own `go.mod`. The same script was run
  successfully from that nested module:
  `go run . ../../../../ pkg/game pkg/combat pkg/command`.
- The pre-registered lint command returned exit 1 on the baseline, with the
  target findings at dupl 26, funlen 190, gocognit 112, and gocyclo 4. Its raw
  output is `lint-baseline.txt`; it also reports other existing linter
  categories and source-cache warnings naming `/home/zach/wt/claude-idiom-exp`.
  A focused rerun at the exact baseline SHA is retained in
  `lint-baseline-focused.txt`. It preserved the four pre-registered counts
  and also printed errcheck 4, gocritic 1, and staticcheck 4; the after output
  printed only the four selected categories. The report compares only the four
  categories named by `PREDICTIONS.md`, not the inconsistent extra categories.
- Open PR #2271's files are listed in `open-pr-files.md` and remain excluded.
  That PR also edits `docs/research/EVIDENCE_LEDGER.tsv`, so the ledger update
  required for a research result is deferred until the PR no longer owns that
  file; the experiment's claim and evidence will be recorded in `REPORT.md`.
- The repository instructions reserve the final full/combined census for
  Claude's reviewed PR batch. This coding run will complete its targeted real
  and mutation-probe censuses, then leave the branch-tip combined gate for that
  batch; it will not use `--release`.
- Runtime identity: this is a Codex run based on GPT-6, although the
  pre-registered brief is labeled for GPT-Luna. Report the work as Codex/GPT-6
  evidence, not a GPT-Luna replication.
