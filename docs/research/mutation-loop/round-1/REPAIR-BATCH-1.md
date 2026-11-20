# Round 1 repair batch 1

Repair commit: `c2fab6011273f637abd74ce9b56ed25a077787f8`.

## C-first witnesses

The targeted census ran against behavior witness commit
`c2fab6011273f637abd74ce9b56ed25a077787f8` (the exact Go SHA recorded in its manifest):
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-19/luna-r1-repair1-final-witnesses-v2`.
It is CLEAN: 3/3 scenarios passed, draw divergent 0.

| Sample row | Scenario | C transcript line |
|---:|---|---|
| 1 | `switched-body-kuji-kiri-restored-skill-depth` | `Satan Klaus interlaces his fingers and slowly fades from view.` |
| 10 | `mobile-pick-force-notice-depth` | `You'll need to hold a set of lockpicks before you can pick a lock!` |
| 14 | `mobile-pick-force-notice-depth` | `Lockwitness has forced you to 'sit'.` |
| 15 | `spec-proc-guild-guard-mobile-actor-depth` | `The guard humiliates you, and blocks your way.` |

The captured C and Go blocks match each line. The same run also covers all three
changed scenarios with draw parity enabled.

After correcting the fixture header, a fresh census of that scenario used the
frozen working-tree inputs recorded at `bbcf262a0` in
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-19/luna-r1-repair1-solreview-header`.
It is CLEAN: 1/1 passed, draw divergent 0.

## Classification corrections

Rows 3 and 18 are recorded in `CLASSIFICATION-CORRECTIONS.md`. Their frozen
sample classes and the measured Round 1 result remain unchanged. The existing
singleton runs are `/home/zach/Archives/darkpawns/oracle-runs/2026-11-19/luna-r1-isolate-3`
and `/home/zach/Archives/darkpawns/oracle-runs/2026-11-19/luna-r1-isolate-18`.

## Revert proofs

Each one-byte mutant commit passed `GOEXPERIMENT=nojsonv2 go build ./...`
before its targeted census.

| Sample row | Mutant commit | Run directory | Result | Mutated-byte diff |
|---:|---|---|---|---|
| 1 | `699c445ae946ca2822af80767b6e0c1837aca3c8` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-19/luna-r1-repair1-mutant-row1` | NOT_CLEAN; draw divergent 0 | Go changed `$n interlaces...` to `$Q interlaces...`; observer C says `Satan Klaus interlaces his fingers and slowly fades from view.` while Go emits ` interlaces his fingers and slowly fades from view.` |
| 10 | `8d071f4600a07d938f1c489c86cc4653fcac16d4` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-19/luna-r1-repair1-mutant-row10` | NOT_CLEAN; draw divergent 0 | Go changed `You'll need...` to `Qou'll need...`; C and Go show those distinct bytes on `send:peer pick gate`. |
| 14 | `2c05f64f038b7b50fb9ddab9ee33c6cd2a321843` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-19/luna-r1-repair1-mutant-row14` | NOT_CLEAN; draw divergent 0 | Go changed `$n has forced you...` to `$Q has forced you...`; C emits `Lockwitness has forced you to 'sit'.` while Go emits ` has forced you to 'sit'.` |
| 15 | `4a5ef1af580f973627d3bc8cdc782ea34ddf5c62` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-19/luna-r1-repair1-mutant-row15` | NOT_CLEAN; draw divergent 0 | Go changed `The guard humiliates...` to `Qhe guard humiliates...`; C emits `The guard humiliates you, and blocks your way.` while Go emits `Qhe guard humiliates you, and blocks your way.` |
