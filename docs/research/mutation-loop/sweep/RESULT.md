# Round 4 population sweep result

This is a fixed-snapshot work list, not a mutation-loop round. It has no sample seed, headline, interval, or kill-rate claim.

- Sweep source SHA: `c28a41e9d427c6b034aee40eca0c4a4c2199765f`.
- Current-main dependency: the census snapshots remain pinned to the frozen sweep SHA; after freeze, `origin/main` advanced through DP-1622 and the DP-1610 ruling. The result branch is based on current `origin/main`; Round 4 and repairs use the current main snapshot.
- Frozen pool: 189 functions and 998 literal tokens; one `c-unreachable` literal is excluded from the denominator.
- Mutations: first ASCII letter to `Q`, or `Z` when the first letter is `Q`; the same transformed text does not occur twice within any batch.
- A kill requires the mutant line in the Go diff and the original line in that scenario’s normalized C transcript. Each literal is scored from its full batch run; there are no singleton runs.

## Counts by class

| Class | Killed | Survived | Unknown | Excluded | Total |
|---|---:|---:|---:|---:|---:|
| ask-zach | 0 | 1 | 1 | 0 | 2 |
| c-reachable | 688 | 295 | 0 | 0 | 983 |
| c-unreachable | 0 | 0 | 0 | 1 | 1 |
| go-only-in-game | 0 | 12 | 0 | 0 | 12 |
| **Total** | 688 | 308 | 1 | 1 | 998 |

## Counts by package

| Package | Killed | Survived | Unknown | Excluded | Total |
|---|---:|---:|---:|---:|---:|
| boards | 6 | 2 | 0 | 0 | 8 |
| combat | 10 | 3 | 0 | 0 | 13 |
| command | 48 | 6 | 0 | 0 | 54 |
| game | 458 | 261 | 1 | 1 | 721 |
| session | 166 | 36 | 0 | 0 | 202 |
| **Total** | 688 | 308 | 1 | 1 | 998 |

## Census runs

| Kind | Batch | Run | Mutants | Commit | Verdict | Draw parity | Run directory |
|---|---:|---|---:|---|---|---|---|
| mutant | 3 | `luna-sweep-batch-03` | 77 | `f247f01d33a37ba7b83e7a385ece43bdb008bec2` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-21/luna-sweep-batch-03` |
| mutant | 5 | `luna-sweep-batch-05` | 77 | `92b9c7ac22d415d0d3e3f3fab641791dad305e06` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-21/luna-sweep-batch-05` |
| mutant | 8 | `luna-sweep-batch-08` | 77 | `09e007d22257e1f5ec247b1612288891ecda193a` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-21/luna-sweep-batch-08` |
| mutant | 9 | `luna-sweep-batch-09` | 77 | `c8acfd465738774bff4f6a28722baab2c259eba8` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-21/luna-sweep-batch-09` |
| mutant | 1 | `luna-sweep-batch-01-r2` | 77 | `5c62058ab53682ddbb4199eb739858883fde8912` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-01-r2` |
| infra-recheck | 1 | `luna-sweep-batch-01-r2-infra` | 77 | `5c62058ab53682ddbb4199eb739858883fde8912` | CLEAN | oracle-draw-parity: scenarios=9 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-01-r2-infra` |
| mutant | 2 | `luna-sweep-batch-02-r2` | 77 | `98dee96ab604fb3512d16e10ce7c7bde69cc67a1` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-02-r2` |
| infra-recheck | 2 | `luna-sweep-batch-02-r2-infra` | 77 | `98dee96ab604fb3512d16e10ce7c7bde69cc67a1` | CLEAN | oracle-draw-parity: scenarios=5 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-02-r2-infra` |
| infra-recheck | 3 | `luna-sweep-batch-03-infra` | 77 | `f247f01d33a37ba7b83e7a385ece43bdb008bec2` | CLEAN | oracle-draw-parity: scenarios=2 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-03-infra` |
| infra-recheck | 5 | `luna-sweep-batch-05-infra` | 77 | `92b9c7ac22d415d0d3e3f3fab641791dad305e06` | CLEAN | oracle-draw-parity: scenarios=4 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-05-infra` |
| infra-recheck | 8 | `luna-sweep-batch-08-infra` | 77 | `09e007d22257e1f5ec247b1612288891ecda193a` | NOT_CLEAN | oracle-draw-parity: scenarios=3 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-08-infra` |
| infra-recheck | 9 | `luna-sweep-batch-09-infra` | 77 | `c8acfd465738774bff4f6a28722baab2c259eba8` | CLEAN | oracle-draw-parity: scenarios=5 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-09-infra` |
| mutant | 4 | `luna-sweep-batch-04-r2` | 77 | `1c25081231d571beed3b1ec5be3bd68da20d1552` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-04-r2` |
| infra-recheck | 4 | `luna-sweep-batch-04-r2-infra` | 77 | `1c25081231d571beed3b1ec5be3bd68da20d1552` | NOT_CLEAN | oracle-draw-parity: scenarios=5 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-04-r2-infra` |
| mutant | 6 | `luna-sweep-batch-06-r2` | 77 | `93b433c5c69945acd9a62d36fde6a2a411e40d0e` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-06-r2` |
| infra-recheck | 6 | `luna-sweep-batch-06-r2-infra` | 77 | `93b433c5c69945acd9a62d36fde6a2a411e40d0e` | CLEAN | oracle-draw-parity: scenarios=5 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-06-r2-infra` |
| mutant | 7 | `luna-sweep-batch-07-r2` | 77 | `9d0672edebf116a8b66bf68c0f14bf6a8a28df65` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-22/luna-sweep-batch-07-r2` |
| infra-recheck | 7 | `luna-sweep-batch-07-r2-infra` | 77 | `9d0672edebf116a8b66bf68c0f14bf6a8a28df65` | CLEAN | oracle-draw-parity: scenarios=1 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-23/luna-sweep-batch-07-r2-infra` |
| mutant | 10 | `luna-sweep-batch-10-r2` | 77 | `5de15aebdaf6ce563286ad23da79558e9f6d7e39` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-23/luna-sweep-batch-10-r2` |
| infra-recheck | 10 | `luna-sweep-batch-10-r2-infra` | 77 | `5de15aebdaf6ce563286ad23da79558e9f6d7e39` | NOT_CLEAN | oracle-draw-parity: scenarios=1 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-23/luna-sweep-batch-10-r2-infra` |
| mutant | 11 | `luna-sweep-batch-11-r2` | 76 | `714da7ee195c61736f7ced24c935a3f7f6ab4276` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-23/luna-sweep-batch-11-r2` |
| mutant | 12 | `luna-sweep-batch-12-r2` | 76 | `628b78936fd75d60846cf786eaebc24fd442399a` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-23/luna-sweep-batch-12-r2` |
| mutant | 13 | `luna-sweep-batch-13-r2` | 76 | `a8d1009dff2036a122ee7cf26ecc4ba5829b8a57` | NOT_CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-23/luna-sweep-batch-13-r2` |
| infra-recheck | 13 | `luna-sweep-batch-13-r2-infra` | 76 | `a8d1009dff2036a122ee7cf26ecc4ba5829b8a57` | NOT_CLEAN | oracle-draw-parity: scenarios=1 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-23/luna-sweep-batch-13-r2-infra` |
| control | control | `luna-population-sweep-control-c28-r2` | 0 | `c28a41e9d427c6b034aee40eca0c4a4c2199765f` | CLEAN | oracle-draw-parity: scenarios=2455 draw_divergent=0 | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-23/luna-population-sweep-control-c28-r2` |

## Setup refusals

No setup refusals were reported.

## Batch registration

All mutant commits are throwaway direct children of the sweep SHA and are retained only for these censuses. The initially misregistered control on the docs commit was cancelled while queued before any worker ran; see `RUN-NOTES.md`.

## Unknown-infra rows

0 sampled rows remained unknown (infra) after same-commit targeted rechecks; they are excluded from kills and the denominator.
Separate R1a unknown: 1 row(s).

## R1a item

Batch 05 row 148 is `unknown` and excluded from kills and the denominator. C passes a null clan plan to `%s` in `do_clan_info` (`src/clan.c:774`); the pinned oracle prints `(null)` through glibc behavior, which is not game law. The row awaits Zach's ruling on what a planless clan should print. Its archived C and mutated Go lines remain in `probes.tsv`.

## Decisions for Zach

Row 148's expected planless-clan output remains open in `DECISIONS.md`. Row 472 was open at sweep completion and was classified as bucket E in the 2026-11-24 survivor-triage follow-up below.

## For Claude

Please review every `killed` row against its cited C transcript line and Go diff line, then check the aggregate counts and any setup-refusal or infrastructure reruns.

## Survivor-triage follow-up (2026-11-24)

All 308 survivors are assigned in `SURVIVOR-TRIAGE.tsv`. The table below gives the effective follow-up buckets after the corrections in `CLASSIFICATION-CORRECTIONS.md`; the original first-pass assignments remain preserved in the TSV:

| Bucket | Count | Follow-up |
|---|---:|---|
| A. no witness | 261 | Repair in scenario-backed batches. |
| B. masked | 0 | No normalization or masking cases found. |
| C. Go-only branch | 15 | Thirteen unapproved proposals; retain two already-approved divergences (DP-1373 and the alias cap). |
| D. C-unreachable | 27 | Excluded from repair work and the effective denominator. |
| E. not player-facing | 2 | Rows 472 and 498 are collector false positives; exclude from repair work and future draws. |
| F. ruling or harness capability | 3 | Rows 2, 4 and 908 cannot receive an isolated clean paired proof under the current approved behavior or fixture/input boundaries. |

These are follow-up classifications on current `origin/main`; the frozen sweep totals above remain unchanged. The original bucket assignments are preserved in `SURVIVOR-TRIAGE.tsv`; corrections in `CLASSIFICATION-CORRECTIONS.md` supersede them for follow-up work. Batch 05 row 148 remains a separate R1a unknown, outside the 308 survivors.
