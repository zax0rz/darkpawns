# DP-1371 Phase 2b mutation audit

The committed `selection.json` is locked to main `0fadb33c6b9f8e1e31886ed0a027f45dd2bacc8e`,
seed `20260930`. It contains the full-frame SHA-256 and 100 selected anchor locators. The full
frame remains in Archives at the path recorded in the report.
`plans.json` describes 100 temporary production mutations; none is a proposed fix.

Read the [classified report](../../docs/fidelity/depth/handoff/2026-09-30-dp-1371-mutation-report.md)
for findings, limits, costs and the extension proposal.

```bash
python3 scripts/fidelity_mutation_sample.py \
  --root /path/to/main-at-recorded-head \
  --baseline /path/to/passing-fidelity-units-output \
  --out /path/to/regenerated-full-frame.json --seed 20260930 --size 100 \
  --created-utc 2026-09-30T19:50:52.284416+00:00 \
  --selection-out /path/to/regenerated-selection.json

python3 scripts/fidelity_mutations/run.py \
  --sample /path/to/regenerated-full-frame.json \
  --plans scripts/fidelity_mutations/plans.json \
  --root /path/to/repository \
  --trees /path/to/fresh-mutant-worktrees \
  --out /path/to/archive/mutations --jobs 3

python3 scripts/fidelity_mutations/report.py \
  --sample /path/to/regenerated-full-frame.json \
  --evidence /path/to/archive --out /path/to/review-artifacts

python3 -m unittest discover -s scripts -p test_fidelity_mutations.py
```

Use fresh archive paths: sampling and execution refuse to overwrite retained
case evidence. `--ids M001,M002` selects a subset for execution; the report
requires the complete locked sample. Detached worktrees have the sampled HEAD;
only one worker mutates a given package tree. Exact source bytes are restored
on exceptions as well as completed executions. Keep these worktrees dedicated
to the audit until all workers have finished.

`evidence/results.tsv` retains every selected anchor and its actual assertion
mapping. `evidence/mutations.json` is a lossless JSON bundle of independent
temporary diffs. Extract one entry’s `diff` to inspect or reproduce that mutant;
the mutations are mutually exclusive. The full frame and raw runs
remain in the local archive identified in `evidence/summary.json`, with checksums
in `evidence/evidence-sha256.tsv`. Production source and tests remain unchanged.
