# Sweep run notes

- All source snapshots were captured cleanly before queueing; each recorded `go-head.txt` matches `BATCH-RUNS.tsv`, and all mutant commits are direct children of the frozen sweep SHA.
- All 13 mutant commits passed `GOEXPERIMENT=nojsonv2 go build ./...`.
- The 998 probes are balanced across 13 batches of 76–77 because the same transformed player-facing text occurs at 13 source sites; each batch contains unique transformed text.
- An initial control registration named `luna-population-sweep-control` captured the documentation commit `5d9702dad` rather than the sweep SHA. It was cancelled while queued, before a scenario worker ran. The retained attempt is `/home/zach/Archives/darkpawns/oracle-runs/2026-11-21/luna-population-sweep-control`. The measured control is `luna-population-sweep-control-c28` on `c28a41e9d427c6b034aee40eca0c4a4c2199765f`; the cancelled registration is excluded from results.
