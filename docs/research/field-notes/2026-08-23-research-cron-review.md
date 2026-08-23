# Research Cron Review — 2026-08-23

## Provenance

Condensed from Daeron's
`/Users/zach/.openclaw/workspace-daeron/memory/cron-review-2026-08-23.md`.
Daeron reviewed two OpenClaw jobs: research writing (`dde06fbf`) and weekly
research digest (`df3f924c`). Counts below remain provisional until recomputed
from exported raw run records.

## Reported Configuration and History

| Job | Reported schedule | Runs | Successful | Failed | Current model |
|---|---:|---:|---:|---:|---|
| Research writing | Tue/Thu 18:00 ET | 36 | 30 | 6 | MiMo v2.5 |
| Research digest | Sun/Wed 18:00 ET | 30 | 25 | 5 | MiMo v2.5 |

Older Daeron instructions reportedly describe one weekly run for each job, so
the live schedules and written specification have drifted. The higher frequency
may be intentional; verify before reducing it.

The writing job reportedly produced 31 drafts totaling roughly 300 KB between
May and August. Its topic sequence forms a coherent arc from silent drift and
constraint engineering through differential testing, coordination failure,
ratchets, magnitude errors, and enforcement cost.

## Operational Findings Reported by Daeron

- Model changes correlate with historical failures: context overflow under a
  GLM configuration, timeouts and very high token use under DeepSeek, and fewer
  core-task failures under the current MiMo configuration.
- Both jobs attempt Soviet cross-posts from isolated sessions where the `soviet`
  executable is reportedly absent from `PATH`. These partial failures may be
  logged only as warnings and therefore evade failure alerts.
- Research-writing runs reportedly reached approximately 1.09M, 2.17M, and 2.7M
  input tokens on three dates. This suggests unbounded or repeated context reads,
  but the causal file-access trace has not been preserved.
- Digest output is scattered among `RESEARCH-LOG.md`, GBrain, and report files.
  Multiple jobs also edit `RESEARCH-LOG.md`, creating overwrite and anchor risk.
- Existing completion checks do not establish factual accuracy, novelty, or
  citation integrity.

## Adopted Conclusions

1. **Separate canonical storage from distribution.** Repository field notes and
   the evidence ledger are canonical. Discord, GBrain, and Soviet are mirrors.
2. **Prefer append-only artifacts.** Each digest gets a dated file. Independent
   jobs do not share mutable regions in one monolithic research log.
3. **Bound writing context.** A writing run receives a research track, one
   question, selected ledger claims, and named evidence. It does not crawl every
   draft and the whole repository.
4. **Make partial failure visible.** Artifact write, ledger update, and each
   distribution target receive separate result fields.
5. **Pin and record execution metadata.** Model and schedule changes need a
   rationale; token use and output paths belong in the run record.
6. **Gate claims, not word counts.** File existence and length are smoke checks.
   Citation resolution, claim state, duplication, and contradiction checks are
   the research quality gate.

## Where This Review Differs From Daeron's Recommendation

Daeron proposed consolidating all digest writes into `RESEARCH-LOG.md` with
non-overlapping zones. This still requires isolated agents to coordinate edits
inside one mutable file and does not prevent stale-anchor or lost-update errors.
The repository contract instead uses one dated append-only note per run and a
small structured ledger as the index.

Daeron also described Soviet as always failing, while the review's failure table
names only some runs explicitly. The persistent PATH diagnosis is plausible, but
claim `RO-005` remains `needs-verification` until raw results establish the
denominator.

## Suggested Job Responsibilities

### Evidence/digest job

- collect a bounded time window of commits, PRs, oracle/depth artifacts, and raw
  pipeline results;
- create one dated field note;
- propose or update ledger rows with conservative states;
- publish mirrors only after the canonical write succeeds.

### Research-writing job

- select one research track and question;
- consume verified ledger entries plus explicitly named provisional hypotheses;
- write or revise one draft with claim IDs attached;
- report contradictions and missing evidence rather than filling gaps with prose.

Frequency is a budget and editorial decision. Two writing sessions plus two
digests per week may be useful during an active period, but it should be an
explicit choice measured by novel verified claims per token, not an inherited
cron accident.
