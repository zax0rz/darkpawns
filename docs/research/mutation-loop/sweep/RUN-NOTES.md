# Sweep run notes

- All source snapshots were captured cleanly before queueing; each recorded `go-head.txt` matches `BATCH-RUNS.tsv`, and all mutant commits are direct children of the frozen sweep SHA.
- All 13 mutant commits passed `GOEXPERIMENT=nojsonv2 go build ./...`.
- The 998 probes are balanced across 13 batches of 76–77 because the same transformed player-facing text occurs at 13 source sites; each batch contains unique transformed text.
- `BATCH-PLAN.tsv` SHA-256: `bbe5f9a9379d98bf1fb1dd1f2c2cbb95533907274bdaafd06e926e53ad2a5f58`.
- An initial control registration named `luna-population-sweep-control` captured the documentation commit `5d9702dad` rather than the sweep SHA. It was cancelled while queued, before a scenario worker ran. The retained attempt is `/home/zach/Archives/darkpawns/oracle-runs/2026-11-21/luna-population-sweep-control`. The measured control is `luna-population-sweep-control-c28` on `c28a41e9d427c6b034aee40eca0c4a4c2199765f`; the cancelled registration is excluded from results.
- After the sweep freeze, `origin/main` advanced to `02a6241c9` through the DP-1622 typed-dollar fix (`9227c5c0c`), the DP-1610 ruling note (`3229914ce`), and the session audit merge (`02a6241c9`). Sweep registrations and source snapshots remain pinned to `c28a41e9d427c6b034aee40eca0c4a4c2199765f`; the report will present this as fixed-snapshot evidence, and Round 4 plus follow-on repairs use current `origin/main`.
