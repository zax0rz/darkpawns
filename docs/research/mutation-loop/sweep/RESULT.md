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
| A. no witness | 206 | All 206 killed: 205 by scenario-backed proofs in `REPAIR-LEDGER.tsv`, and row 4 by a unit proof under the 2026-11-11 exception (#3081). Rows 527-530 were killed in #3082. |
| B. masked | 0 | No normalization or masking cases found. |
| C. Go-only branch | 39 | All handled. 38 rows are in `GO-ONLY-REMOVALS.tsv`: 20 removed, 14 silenced, 2 approved divergences (DP-1373 and the alias cap) and 2 ported (647-648, #3074). Row 2 is an approved divergence under DP-1388 (ruling 8), recorded in `CLASSIFICATION-CORRECTIONS.md`. Row 908 was removed in #3084 once structured WebSocket commands got C's line cut. |
| D. C-unreachable | 58 | Excluded from repair work and the effective denominator. Row 144 joined in #3082: C never sets a clan spell. |
| E. not player-facing | 5 | Three collector false positives, plus rows 834 and 838 (`go-only-surface`, ruling 7). Excluded from repair work and future draws. |
| F. ruling or harness capability | 0 | All settled by Zach's 2026-11-25 rulings. |

These are follow-up classifications on current `origin/main`; the frozen sweep totals above remain unchanged. The original bucket assignments are preserved in `SURVIVOR-TRIAGE.tsv`; row 622 is superseded by the Go-only correction in `CLASSIFICATION-CORRECTIONS.md`. Batch 05 row 148 remains a separate R1a unknown, outside the 308 survivors.

## Survivor-triage correction update (2026-11-25)

Rows 665, 677–680, and 687–693 are reclassified from bucket A to bucket D after checking the C call paths and the existing depth manifest. Row 665 is an unreachable stable mount reload failure with the shipped 8021 prototype; the other 11 rows are in special procedures that C does not assign. The first-pass triage rows and the 308-row survivor set are unchanged. Effective counts are now A253/B0/C15/D35/E2/F3. The R1a unknown, row 148, remains separate and outside the survivor count.
Rows 843, 851, 856, and 859 move from bucket A to bucket C because their sampled Go fallback arms follow a successful target or combatant lookup and have no reachable C call path. Row 861 also moves from A to C: the sampled Go nil-equipment fallback cannot run for a live player, and C's corresponding no-wield response comes from the later slot check. The first-pass triage rows and the 308-row survivor set are unchanged. On current `origin/main` before the open corrections in PRs #3057 and #3058, effective counts are A260/B0/C20/D23/E2/F3. Preserve those PRs' corrections when combining the updates.
Rows 739–742 move from bucket A to bucket D: the live `pray_for_items` special is assigned at room 8008, but the reward loop requires an `item_for_<player>` extra description in the adjacent key room. The current world has no such key and no player command can create one. The first-pass triage rows and 308-row survivor set are unchanged. On current `origin/main` before open corrections in PRs #3058 and #3060, effective counts are A261/B0/C15/D27/E2/F3. Preserve both PRs' corrections when combining the updates.
## Repair batch update (2026-11-25)
Rows 402, 403, and 406 are killed by scenario-backed proofs: the NODROP gate for an NPC victim, the shared carry-weight gate for a mobile victim, and the `TO_VICT` give line delivered to a switched mobile descriptor. Row 404 moves from A to C because it is a Go-only transfer-error fallback with no C counterpart. Relative to current `origin/main` before this batch, the effective counts are A261/B0/C16/D23/E2/F3. The open combat-entry and prayer-reward PRs (#3060 and #3061) also update these shared research files; preserve their appended rows and recompute the cumulative counts when combining the diffs.
Rows 889 and 890 are killed by `switched-body-infobar-depth`, which reaches the `mobInfoBarBody` already-on and invalid-argument outputs through a switched descriptor. Rows 884-885 move from A to C: the invalid-mode fallback has no live C path and Go's shared infobar mode is only set to OFF or ON in production. Relative to current `origin/main` before this batch, the effective counts are A261/B0/C17/D23/E2/F3. This update shares correction/result files with open repair PRs #3054–#3062; preserve their rows and recompute cumulative counts when combining the diffs.
## Survivor repair batch: `perform_move` mobile refusals (2026-11-25)
Rows 308–311 are killed by four isolated one-byte mutations through switched-mobile `performMoveMob` scenarios. Each mutant census failed on the changed Go byte while retaining the original C line. The four-scenario clean census passed; all five runs used `--draw-parity`, with zero draw divergence, INFRA, TIMEOUT, hollow steps, or probe setup-refusal steps. The closed-door scenarios close the fixture door in warmup before switching, so the probe reaches the closed refusal itself. Preliminary clean attempts are retained but excluded: `luna-survivors-performmove-clean` and `-clean-r3` passed while the door was still open and the probe moved rooms; `-clean-r2` failed fixture validation because door state `3` is unsupported.
| Row | Scenario | Clean run | Mutant run | Result |
|---:|---|---|---|---|
| 308 | `perform-move-mob-no-exit-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-survivors-performmove-clean-r4` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-survivors-performmove-row-308-mutant` | Killed; C `Alas...`, Go `Qlas...` |
| 309 | `perform-move-mob-hidden-secret-depth` | same | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-survivors-performmove-row-309-mutant` | Killed; C `Alas...`, Go `Qlas...` |
| 310 | `perform-move-mob-closed-keyword-depth` | same | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-survivors-performmove-row-310-mutant` | Killed; C `The gate seems...`, Go `Qhe gate seems...` |
| 311 | `perform-move-mob-closed-unnamed-depth` | same | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-survivors-performmove-row-311-mutant` | Killed; C `It seems...`, Go `Qt seems...` |
Relative to `origin/main` before this batch, the survivor buckets move from A261/B0/C15/D23/E2/F3 to A257/B0/C15/D23/E2/F3. This batch is based on `fe2838ce4`; open repair PRs #3054–#3063 also update shared evidence and need to be unioned before aggregate counts are finalized.
These are follow-up classifications on current `origin/main`; the frozen sweep totals above remain unchanged. The original bucket assignments are preserved in `SURVIVOR-TRIAGE.tsv`; corrections in `CLASSIFICATION-CORRECTIONS.md` supersede them for follow-up work. Rows 527–529 remain unknown within the 308 survivors; batch 05 row 148 is a separate R1a unknown outside that set.

## Consolidated counts after the 2026-11-25 batch merge

Claude merged repair PRs #3054–#3068 together and recomputed the bucket table above from `SURVIVOR-TRIAGE.tsv`, every bucket move in `CLASSIFICATION-CORRECTIONS.md`, and the row ids in `REPAIR-LEDGER.tsv`. The per-PR count sentences above were each computed against a different base and are superseded by the table. Effective buckets: A225 (161 killed, 64 open), B0, C29, D45, E3, F6, total 308.

Open bucket A rows: 16, 17, 20, 34, 40, 60, 62, 72, 79, 87, 88, 89, 111, 126, 144, 150, 154, 164, 179, 220, 221, 222, 224, 237, 297, 298, 299, 345, 357, 370, 383, 386, 485, 487, 488, 496, 505, 506, 530, 545, 608, 609, 612, 614, 617, 780, 787, 808, 809, 810, 812, 834, 837, 838, 870, 878, 902, 909, 918, 919, 928, 960, 986, 994.


## Last-survivor repair batch 2 (2026-11-25; stacked on PR #3071)

Rows 20, 40, 62, 87, 88, and 89 are killed by isolated one-line mutants. Each mutant census failed on the changed Go byte; every clean baseline passed. All six mutants and their clean baselines used `--draw-parity`, with zero draw divergence, INFRA, TIMEOUT, hollow output, or probe setup refusals. Row 20 uses a spoken call-lightning witness that reaches the sampled spell damage helper; rows 87-89 use the switched-horse chosen-guard actor lane.

| Row | Mutant commit | Scenario | Clean run | Mutant run | Result |
|---:|---|---|---|---|---|
| 20 | `d2bd0244ab54` | `spell-wimpy-default-arm-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-20-clean-r2` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-20-mutant-r2` | Killed; the victim receives C's wimpy line through the spell damage tail. |
| 40 | `999de499d712` | `tiger-mounted-refusal-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch2-clean-r3` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-40-mutant` | Killed; C prints “Dismount first!” and the mutant changes the Go line. |
| 62 | `76f4636765c7` | `disarm-gates-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch2-witness-clean` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-62-mutant` | Killed; the named-but-missing target reaches C's “Disarm who?” line. |
| 87 | `c634cad71` | `spec-proc-chosen-guard-switched-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-chosen-switched-clean` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-87-mutant-r2` | Killed; actor's block line changes. |
| 88 | `02f1637cc` | `spec-proc-chosen-guard-switched-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-chosen-switched-clean` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-88-mutant-r2` | Killed; actor's block line changes. |
| 89 | `091a699a4` | `spec-proc-chosen-guard-switched-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-chosen-switched-clean` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-89-mutant` | Killed; actor's challenge line changes. |

Rows 34, 60, 72, 111, and 154 move from A to C (Go-only concurrent-state guards); rows 79, 126, 150, 164, and 179 move from A to D (no player-reachable C call path for the sampled branch); row 144 moves from A to F as an R1a unknown caused by C's overlapping `sprintf`. Rows 87-89 also receive a source-attribution correction in `CLASSIFICATION-CORRECTIONS.md`; the frozen triage rows remain unchanged.

Several preliminary attempts are retained but excluded from the kills: row 20's first melee mutant stayed CLEAN because the transcript reached the `CombatEngine` duplicate in `pkg/combat/engine.go`, not the sampled helper; the first spell fixture set wimpy on the caster and did not reach the victim's wimpy line; row 88's first mutant changed a duplicate in `specCastleGuardActor`, not the sampled `specChosenGuardActor` site; and the player-only chosen-guard scenario does not cover the NPC actor lane. The corrected scenario and mutants above reach each sampled site.

Combined with PR #3071, branch-local effective counts are A208 (177 killed, 31 open), B0, C34, D54, E3, F9. The remaining bucket-A rows are: 16, 17, 237, 297, 298, 299, 345, 357, 370, 383, 386, 485, 488, 496, 505, 506, 530, 545, 608, 609, 612, 614, 617, 780, 787, 810, 902, 918, 919, 986, 994. These branch-local counts do not change the Claude-maintained bucket table above; Claude recomputes that table after the stacked PRs merge.

## Last-survivor repair batch 3 (2026-11-25; stacked on PR #3072)

Rows 16, 17, 297-299, 345, 357, 383, 386, 485, 505, and 506 are killed by isolated one-line mutants. The clean baseline is `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch3-clean-post-main-r1`, run on merged commit `d4203e38d`; it covers all ten scenarios below. Every mutant and the clean baseline used `--draw-parity`; mutant runs failed on the changed Go byte, while the clean baseline passed with zero draw divergence, INFRA, TIMEOUT, hollow output, or probe setup refusals.

| Row | Mutant commit | Scenario | Mutant run | Result |
|---:|---|---|---|---|
| 16 | `7d3578f2cd6d` | `wounded-band-player-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-16-mutant-r2` | Killed; C mortally-wounded line, Go Qou... |
| 17 | `f0d43f65a311` | `wounded-band-player-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-17-mutant` | Killed; C incapacitation line, Go Qou... |
| 297 | `1bc1be2e888b` | `dream-bad-travel-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-297-mutant` | Killed; C You have a dream..., Go Qou... |
| 298 | `e363a0f9aa2c` | `dream-bad-travel-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-298-mutant` | Killed; C fades from existence, Go Qhe... |
| 299 | `9faf44bef66a` | `dream-bad-travel-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-299-mutant` | Killed; C fades into existence, Go Qhe... |
| 345 | `0e17050738aa` | `drink-full-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-345-mutant` | Killed; C You are full., Go Qou... |
| 357 | `9561241e4104` | `pour-out-refind` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-357-mutant` | Killed; C The a tin cup is empty., Go Qhe... |
| 383 | `0d5abf00d24a` | `wear-basic` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-383-mutant` | Killed; C You can't wear a tin cup., Go Qou... |
| 386 | `f9cd7488f37f` | `wear-basic` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-386-mutant` | Killed; second duplicate sampled site reaches its C counterpart |
| 485 | `9bbd2612e394` | `switched-mobile-door-pour-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-485-mutant` | Killed; C actor line preserved, mutated Go $Q leaves it blank |
| 505 | `8b16a2bdb824` | `staff-tap-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-505-mutant-r2` | Killed; C It seems powerless., Go Qt seems powerless. |
| 506 | `09f35da22487` | `staff-tap-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-row-506-mutant-r2` | Killed; C Nothing seems to happen., Go Qothing... |

Rows 488 and 496 move from A to C: the sampled `World.doUse` arms are bypassed by the live session command guards, which emit the same bytes; both isolated mutants stayed CLEAN because no player command reaches those sites. Row 370 moves from A to D because the sampled second wield check is a defensive duplicate after the earlier check. Row 530 moves from A to F as an R1a unknown: C uses overlapping `sprintf` for the no-argument auto listing, so the pinned bytes do not establish the intended string. These corrections and source citations are recorded in `CLASSIFICATION-CORRECTIONS.md`.

Excluded attempts remain in the archive. Row 16's first mutant run reached the drowning duplicate rather than `emitWoundBand`; the corrected direct wounded-band scenario reaches the sampled victim line. Row 505's first mutant run reached `useStaff`, not `useStaffTap`; `staff-tap-depth` drains the staff charge before the second tap and reaches the sampled refusal. The earlier staff-tap clean run is retained; the batch clean suite rechecks it alongside all other witnesses.

Relative to batch 2, this resolves 16 rows: 12 kills, rows 488 and 496 to C, row 370 to D, and row 530 to F. Branch-local effective counts are A204 (189 killed, 15 open), B0, C36, D55, E3, F10. The remaining bucket-A rows are: 237, 545, 608, 609, 612, 614, 617, 780, 787, 810, 902, 918, 919, 986, 994. These branch-local counts do not change the Claude-maintained bucket table above.

## Last-survivor repair batch 4 (2026-11-25; stacked on PR #3073)

Rows 237, 608, 609, 612, 614, 617, 780, 787, 810, 986, and 994 are killed by isolated one-line mutants. The clean baseline `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-clean-merged` ran on merged commit `6baf7376d`: 11/11 CLEAN with zero draw divergence, INFRA, TIMEOUT, hollow output, or probe setup refusals. Each counted mutant run used `--draw-parity` and failed on its changed Go byte with zero draw divergence, INFRA, TIMEOUT, hollow output, or probe setup refusals.

| Row | Mutant commit | Scenario | Mutant run | Result |
|---:|---|---|---|---|
| 237 | `ecaa71d9f516` | `ctell-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-row-237-mutant` | Killed; changed Go byte fails against the original C line. |
| 608 | `389750e609f5` | `switched-body-staff-zero-charge-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-row-608-mutant-r2` | Killed; changed Go byte fails against the original C line. |
| 609 | `14178b8a728c` | `switched-body-behead-slash-last-survivor-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-row-609-mutant` | Killed; changed Go byte fails against the original C line. |
| 612 | `63033b8c7734` | `switched-body-behead-barehands-last-survivor-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-row-612-mutant` | Killed; changed Go byte fails against the original C line. |
| 614 | `af4a3d6d97e9` | `switched-body-shoot-pc-miss-last-survivor-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-row-614-mutant` | Killed; changed Go byte fails against the original C line. |
| 617 | `82bc4f5d5a7a` | `switched-body-shoot-pc-hit-last-survivor-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-row-617-mutant` | Killed; changed Go byte fails against the original C line. |
| 780 | `25f33bd605f9` | `switched-body-cast-smartness-master-last-survivors-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-row-780-mutant` | Killed; changed Go byte fails against the original C line. |
| 787 | `4695365ae646` | `switched-body-cast-master-gate-last-survivor-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-row-787-mutant` | Killed; changed Go byte fails against the original C line. |
| 810 | `5c29ffa6c837` | `cast-sitting-last-survivor-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-row-810-mutant-r2` | Killed; the cast-specific sitting path reaches the sampled `checkCastPosition` line. |
| 986 | `508c6e08cd40` | `switched-body-wizutil-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-row-986-mutant-r2` | Killed; level-9 clone reaches the sampled body-level gate. |
| 994 | `a7f7e29976e0` | `switched-body-wizutil-depth` | `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/luna-last-survivors-batch4-row-994-mutant-r2` | Killed; changed Go byte fails against the original C line. |

Rows 545, 902, and 919 move from A to C, D, and D respectively. Row 918 moves from A to F as an R1a unknown and stays outside kills and the effective denominator. Row 810's earlier quaff-only mutation attempt is retained but excluded because it reached a separate C branch with identical text; the dedicated cast scenario above reaches the sampled path. Row 986's first attempt is also excluded: the clone prototype's default level hit the command gate before the sampled body-level branch, so the corrected fixture sets the clone to level 9. These call-path corrections and citations are recorded in `CLASSIFICATION-CORRECTIONS.md`; row 918 is also recorded in `../DECISIONS.md`.

Relative to batch 3, this resolves the last 15 rows: 11 kills, row 545 to C, rows 902 and 919 to D, and row 918 to F. Branch-local effective buckets are A200 (200 killed, 0 open), B0, C37, D57, E3, F11. The remaining A rows are empty. The aggregate bucket table above remains unchanged for Claude to recompute after the stacked PRs merge.

## Claude's recount after the last-survivor batches (2026-11-25)

Claude merged Luna's last-survivor PRs #3071, #3072, #3073 and #3076 together and recomputed the bucket table above from `SURVIVOR-TRIAGE.tsv`, every move in `CLASSIFICATION-CORRECTIONS.md`, and the row ids in `REPAIR-LEDGER.tsv`. All 39 new repair rows were checked: each mutant run is a one-line commit that FAILs on its scenario with zero draw divergence, and each clean run is CLEAN with zero draw divergence and no refused setup steps. Effective buckets: A201 (200 killed, 1 open), B0, C37, D57, E3, F10, total 308.

Row 918 is not an R1a unknown. The same overlapping `sprintf` in `do_wiznet` (`src/act.wizard.c:1957`) appended `Wizwritingpeer` correctly in the pinned C transcript, so the build does append; the `(Writing)` suffix was missing because the peer was never flagged. C's `do_write` on paper points `ch->desc->str` at the note and never sets `PLR_WRITING` (`src/act.comm.c:1098-1101`), while Go's `beginNoteWrite` does (`pkg/game/act_write.go:179`). Every C `PLR_WRITING` reader (wiznet, who, tell, the channels) therefore sees a different state for a player writing on paper. Row 918 returns to bucket A, open, pending a Go fix; a board-write probe (`src/boards.c:266` sets the flag) can witness the sampled line itself.

## Claude's recount after the ruling work (2026-11-25)

Claude recomputed the bucket table after #3080, #3081, #3082 and #3084 (the
ruling work from Zach's 2026-11-25 rulings in `DECISIONS.md`) and Round 4
(#3083) merged. Starting from A201, B0, C37, D57, E3, F10:

- Rows 527, 528, 529 and 530: F to A, killed (ruling 1, #3082).
- Row 4: F to A, unit-proven (ruling 6, #3081).
- Rows 2 and 908: F to C. Row 2 is an approved divergence (ruling 8); row 908
  was removed (ruling 3, #3084).
- Row 144: F to D (#3082).
- Rows 834 and 838: F to E as `go-only-surface` (ruling 7).

Result: **A206 (all killed), B0, C39 (all handled), D58, E5, F0**, 308 rows.
No survivor is left open. Row 148, outside the 308, is killed by
`clan-info-planless-depth` (#3082) under ruling 2.
