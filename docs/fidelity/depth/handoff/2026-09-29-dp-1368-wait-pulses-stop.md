# DP-1368 — wait-counts-pulses candidate, stopped on pacing proof

Worktrees: `/home/zach/dp-wait-pulses` (main repository) and
`/home/zach/dp-oracle-wait` (oracle), both `sol/wait-pulses`.
Oracle commit: `d2ace874868f34315280bb3a33b68a53d322de4b`.
Main baseline: `75bf33847`. No PR is ready; PR 2 is not started.

## Implementation and boundaries

Under DP_CLOCK, idle passes observe player wait without decrementing it. Each
pumped pulse decrements wait, executes at most one queued command per descriptor
through shared dispatch, then runs heartbeat (R3b/R3c). Dispatch retains the
production command body, with the original post-command wait=1 restricted to
non-DP_CLOCK. The pristine source sites are `src/comm.c:597-628`, with the
heartbeat phase at `src/comm.c:677-679` (read this round; R5g).

The pre-boot C self-test constructs an editor descriptor and character. Twenty
idle phases preserve wait=2; two pumped command phases release exactly the first
of two queued lines through real shared dispatch. A patch test applies the mirror
to d2cb13e, verifies pulse phases and the wait gates, and compares the factored
production dispatch body with pristine C. README build guidance now uses O0 and
isolated checkouts. Main src/, Go server code, existing scenarios, divergence
ledgers/pins and the shared reference binary are unchanged.

## Verified evidence

Evidence: `/home/zach/Archives/darkpawns/oracle-runs/2026-09-29/dp-1368-development/`.
Old and candidate binaries, patch, manifest, build logs and transcripts are kept
there. The reference SHA-256 equals the checked-in pin:
`3200165e4e82d26a7403dd91ce544dfd8b2a23ad61a14a6220cfdc8119627916`.
Candidate and two independent clean checkout builds of the oracle commit match:
`49a0799cd7bb107ea846bdd2768a85c100a75803fa2d9d8ded9337aa427bb76b`.
Promotion and the reference pin update remain deferred.

`dpclock-wait-counts-pulses` uses nosummon, whose WAIT_STATE is
PULSE_VIOLENCE*2 (`src/act.other.c:1210`; `src/structs.h:630,634`). Look stays queued
through 39 pumped pulses and appears on pulse 40. It passes on the candidate and
fails on the reference (R5h); candidate normalized C blocks are identical at
150/300/600ms. An initial three-peer aid vehicle coincidentally passed the old
binary due to peer pacing, so it was replaced rather than counted as proof.

Reference `aid-failure-depth` diverges at 150ms and passes at 300/600ms, confirming
the old pacing-dependent green. Candidate aid and flesh-alter vehicles pass all
three and have identical C blocks. The full candidate matrix was stopped as
required by the brief; completed observations are in pacing-candidate-partial.tsv.

Reference census `dp-1368-reference` completed:

```text
scenarios=1036 passed=1029 expected=6 unpinnable=0 stale=0 failed=0
infra=0 timed_out=0 unstable=1 elapsed=878.446s verdict=CLEAN
```

Candidate full census `dp-1368-candidate` was started with explicit candidate
DP_ORACLE_BIN and CENSUS_ALLOW_NONREFERENCE=1. It remains running at this handoff;
use scripts/census.sh wait, then its summary and completed dumps. The archive
contains compare-census-dumps.py to enumerate scope once both censuses finish.
No existing-scenario updates were made before that scope measurement.

All gates passed: make fmt, go build ./..., go vet ./..., go test ./...
(with DP_ORACLE_BIN pointing to the candidate), golangci-lint cache clean,
golangci-lint run ./..., git diff --check, and make fidelity-depth.

## Stop condition — candidate pacing differs

Candidate `parry-depth` results:

| Quiescence | Result |
|---|---|
| 150ms | normalized divergence detected |
| 300ms | no normalized divergence |
| 600ms | no normalized divergence |

At 150ms, C's first `parry ignored argument [actor]` block includes eight warmup
combat lines before `Parry with what? You're unarmed!`; Go contains only the
parry message. At the final pulse 40, Go's actor receives five combat lines absent
from C's actor block, while the C observer receives corresponding combat output.
The three retained logs are pacing-candidate-parry-depth-{150ms,300ms,600ms}.log.
This is an output-block discrepancy; its root cause has not been traced (R5e).
Do not describe it as a confirmed Go command-queue defect.

The brief explicitly says stop if any candidate pacing-table result differs.
Remaining diagnostic jobs were cancelled; the census was left running for its
retained evidence. No retry, pump adjustment, quiescence adjustment, Go fix,
divergence pin or ledger change was made. Resume requires a revised brief or a
resolution of this stop condition before claiming PR 1 ready.

This candidate addresses the pulse-vs-wall-clock wait issue described in the
repository's DP-1202 input-timing brief. It does not establish full resolution of
DP-1202: the candidate pacing proof is blocked, and Linear issue retrieval needed
reauthentication. Zach still owns merge and binary promotion.
