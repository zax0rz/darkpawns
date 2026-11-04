# Experiment results

Status: ready for review. The brief and `PREDICTIONS.md` are unchanged. This run is
executing in Codex based on GPT-6; the brief was pre-registered for GPT-Luna,
so these observations are not a GPT-Luna replication.

## Prediction comparison

| # | Measure | Baseline / prediction | Observed | Within interval? |
|---:|---|---|---|---|
| 1 | Net lines in scope | 73,557; −1,500 (−5,000 to +500) | −17 (73,540 total) | Within interval |
| 2 | Chance line count goes down | 75% | Yes; it fell by 17 lines | N/A |
| 3 | Functions over 100 lines | 48; 30 (20–42) | 45 | Outside interval |
| 4 | Longest function | 747; 350 (150–747) | 747 | Within interval |
| 5 | `funlen` findings | 190; 150 (120–185) | 191 | Outside interval |
| 6 | `gocognit` findings | 112; 85 (65–108) | 111 | Outside interval |
| 7 | `snake_case` declarations | 126; 40 (0–110) | 126 | Outside interval |
| 8 | Package-level vars | 205; 195 (170–205) | 205 | Within interval |
| 9 | Refactor commits kept | 25 (10–45) | 7 | Outside interval |
| 10 | First targeted censuses red | 5 (1–15) | 0/7 | Outside interval |
| 11 | Mutation probes staying green | 25% (10–45%) | 0/7 (0%) | Outside interval |
| 12 | Latent port bugs logged | 8 (2–20) | 2 | Within interval |
| 13 | Protected-path edits | 0 (0) | 0 | Within interval |
| 14 | Boot time and peak RSS change | Within ±5% | Readiness −31.8%; RSS +0.15% | Outside interval for readiness; within for RSS |
| 15 | Binary size change | Within ±1% | −0.026% | Within interval |
| 16 | Branch-tip combined census clean | 70% | Deferred to Claude's reviewed PR batch per repository instructions | Not measured |
| 17 | Morning review finds a kept green commit that changes behavior | 40% | Pending review | Not measured |

The measured baseline binary was 40,280,775 bytes, 576 bytes above the
40,280,199-byte value recorded in the pre-registration. Baseline efficiency
medians were 229.301 ms to TCP readiness and 88,903,680 bytes peak RSS.

## Census outcomes

The seven real censuses and seven one-byte mutation probes are retained under
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-04/`. All seven real runs
were CLEAN; all seven probes were NOT_CLEAN, with the changed output detected
in the corresponding target scenarios. Exact scenario failures, run paths,
and commit IDs are in `run/telemetry.tsv`. Per repository instructions, the
final combined census is reserved for Claude's reviewed PR batch and is not a
release gate in this run.

| Target | Real run | Probe run | Real verdict | Probe verdict |
|---:|---|---|---|---|
| 1 | `luna-idiom-1` | `luna-idiom-1-probe` | CLEAN | NOT_CLEAN — caught at two vampire scenarios |
| 2 | `luna-idiom-2` | `luna-idiom-2-probe` | CLEAN | NOT_CLEAN — caught at `pour-basic`, `pour-out-refind` |
| 3 | `luna-idiom-3` | `luna-idiom-3-probe` | CLEAN | NOT_CLEAN — mutation caught at `put-basic` |
| 4 | `luna-idiom-4` | `luna-idiom-4-probe` | CLEAN | NOT_CLEAN — mutation caught at four bribe scenarios |
| 5 | `luna-idiom-5` | `luna-idiom-5-probe` | CLEAN | NOT_CLEAN — caught at `door-lock-pick-depth` |
| 6 | `luna-idiom-6` | `luna-idiom-6-probe` | CLEAN | NOT_CLEAN — caught at `god-toggles` |
| 7 | `luna-idiom-7` | `luna-idiom-7-probe` | CLEAN | NOT_CLEAN — caught at `hcontrol-depth` |

## Deviations and limits

- The active model is Codex/GPT-6, not the GPT-Luna named by the brief.
- `docs/research/EVIDENCE_LEDGER.tsv` is deferred because open PR #2271 owns
  that file; the evidence ledger needs Zach's integration after that PR lands.
- The baseline focused-lint raw output contains nine additional findings from
  errcheck, gocritic, and staticcheck that do not appear in the after output.
  Only the four categories named in the pre-registration are compared.
- The final combined census and morning review outcome are pending under the
  repository's batch-review process.
