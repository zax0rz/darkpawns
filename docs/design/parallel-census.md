# Concurrent censuses with one shared worker budget

Design proposal, 2026-10-25. Source baseline: `4fc5c8ab2` (`origin/main`
when this worktree was created). This PR changes this document alone. It does
not authorize changing the harness, scheduler, oracle, or governing documents.
R1/R3 require transcript and draw fidelity; R5h requires a concurrency proof
that can fail, rather than assuming isolation proves timing safety.

## Recommendation

Replace the census-long exclusive lock with a host-wide admission coordinator
and **72 shared scenario-worker permits**, initially, keeping the initial
per-run maximum at the measured **36**. The authorized experiment below
measured 1.94× end-to-end throughput with two concurrent copies and identical
captured transcripts and draw rows. At 108 workers, five transient differences
required rechecks, reducing throughput to 1.70× and exposing listener identity
failures. The data support 72, not the earlier total ceiling of 36 or an
unqualified increase to 108.

Keep release `--combined` runs exclusive, draining active scenarios before
admitting the release. Queue ordinary runs, print their identity and position
immediately, and make `wait --run <directory>` the unambiguous reporting
interface. These two policy choices are unchanged.

This is a provisional operating budget for this host and corpus, with a memory
pressure guard; it is not proof of flake-free operation. Fix and test listener
ownership before enabling ordinary concurrency. An implementation may change
startup costs, so remeasure its throughput rather than promising the exact
experimental speedup. Implementation and deployment still require a separate
approved tooling brief. Individual runs above 36 workers were not measured.

## Initial serial resource profile

The brief requested one targeted and one full census, serially and without
bypassing the current lock. Both were launched through `scripts/census.sh` from
clean `4fc5c8ab2`, with the unchanged reference oracle. These initial profiles
preceded the separately authorized concurrency experiment below.
The full run here is a profiling exception explicitly requested by the brief,
not a release certification or the final combined batch census.

Evidence root: `/home/zach/Archives/darkpawns/oracle-runs/2026-10-25/`.
Each run retains the wrapper's `MANIFEST.md`, `go-head.txt`, `git-status.txt`,
`results.tsv`, `census-dump/`, log, summary, and any recheck evidence. Additional
`resource-samples.jsonl`, `resource-processes.json`, and
`resource-summary.json` retain the measurements beside each run. The sampler
source and post-run analyzer are retained at
`sol-parallel-census-instrumentation/{sample,analyze}.py` under that root;
they are deliberately outside this one-file PR. A final
`resource-receipt.txt` records sizes and SHA-256s of the supplemental files,
which finish after the wrapper manifest is written; that manifest is not edited.

Commands, executed sequentially:

```bash
scripts/census.sh start --name sol-parallel-profile-target \
  --scenarios lifecycle-quit-reenter,restart-gossip-review,flee-audience-success,lifecycle-idle-observer \
  --jobs 4
scripts/census.sh wait --run <root>/sol-parallel-profile-target --max-seconds 45
scripts/census.sh start --name sol-parallel-profile-full --jobs 36
scripts/census.sh wait --run <root>/sol-parallel-profile-full --max-seconds 45
```

`<root>` denotes the absolute evidence root above. Bounded waits may be repeated;
no running census log was inspected. The targeted set includes a real SQLite
relogin, process restart, passive audience, and clock-driven lifecycle behavior.
It is a resource sample, not a representative estimate of all command costs.

Both runs used seed 1 and reference SHA-256
`bd6c548c6fbc41d35eede384e4ce2c5a858ff2ce0f568a379d7fb366a9051b07`.

The workstation has an Intel i9-9900K (8 cores, 16 logical CPUs), 32,001 MiB
RAM and 20,479 MiB swap. Before the targeted run, `free -m` reported 10,139 MiB
available and 13,932 MiB swap already used. Desktop applications occupy much
of this host: total installed RAM is not the census's usable memory budget.
`findmnt -T /tmp` identifies the ext4 root filesystem, with about 813 GiB
available at the mid-run check; `/tmp` is not tmpfs on this host.

| Metric | Targeted, 4 jobs | Full, 36 jobs |
|---|---:|---:|
| Scenarios; final verdict | 4; CLEAN | 1,610; CLEAN |
| Regression elapsed | 49.076 s | 1,102.128 s (18m 22s) |
| Final PASS / EXPECTED | 4 / 0 | 1,609 / 1 |
| Internal infrastructure RETRY lines | 0 | 12 |
| Final INFRA / TIMEOUT; wrapper rechecks | 0 / 0; none | 0 / 0; none |
| C process HWM, median / maximum | 11.89 / 13.82 MiB | 11.84 / 34.75 MiB |
| Go process HWM, median / maximum | 72.61 / 84.13 MiB | 69.38 / 86.79 MiB |
| Harness process HWM, median / maximum | 20.09 / 21.66 MiB | 19.78 / 33.41 MiB |
| Peak simultaneous tree RSS, including build/helpers | 445.55 MiB | 3,877.88 MiB (3.79 GiB) |
| Observed C / Go CPU seconds | 0.34 / 4.01 | 94.08 / 931.00 |
| Observed harness / other CPU seconds | 0.64 / 18.03 | 243.44 / 2.74 |
| Sampled peak tree CPU, logical cores | 8.36 | 9.24 |
| Minimum host MemAvailable during samples | 10,002.70 MiB | 6,575.37 MiB |
| Maximum simultaneous C+Go engine processes | 8 | 72 |

The census summaries and manifests establish the verdicts; resource summaries
establish the sampled numbers. The full run started at 09:20:56 and finished
at 09:39:18 EDT. Its completed `census.log` contains 12 infrastructure-shaped
internal RETRY lines, despite zero final INFRA rows. CLEAN therefore does not
mean every first attempt succeeded. The ordinary regression driver deletes its
temporary attempt logs on completion; this run cannot establish each retry's
cause. Future concurrency acceptance must retain those attempts.

At the sampled maxima, C+Go+harness total about 155 MiB per worker before
shells, copying/cache and build reserves. Multiplying separate process maxima
is conservative and does not predict a simultaneous peak: 36 such envelopes
would be about 5.45 GiB, versus the observed 3.79 GiB tree peak. Average observed
tree CPU over the full sampling interval was about 1.15 logical cores, with
short bursts near 9.24. Much worker time is waiting for output/quiescence;
logical CPU count alone is not a useful worker limit. These serial measurements
did not establish a capacity ceiling. The concurrency experiment below supplies
the missing throughput and flake evidence.

Sampling reads `/proc` every approximately 0.5 seconds, follows the detached
runner's descendant tree, and distinguishes C `circle`, Go `dp-server-prebuilt`,
and `dp-oracle-diff` from build/shell helpers. Per-process `VmHWM` records the
resident high-water mark observed before exit; medians are over observed engine
processes, including restart replacements. Tree peak is the maximum simultaneous
RSS sum, including helpers/builds; it double-counts shared pages. CPU seconds
come from observed process user+system ticks; very short-lived processes can be
missed. Peak CPU is a sampled interval expressed in logical cores (1 = 100% of
one core). These are observations, not exact lifetime maxima, PSS measurements,
or a statistical scaling benchmark. File cache and tmp storage add pressure
outside the process RSS sum. Sampling itself adds a small unquantified cost.

The wrapper defaults to 36 (`scripts/census.sh:do_start`); the underlying
regression script defaults to four when invoked directly. The 36-worker
isolation history is in commit `9df3416cc` and
`scripts/oracle_regression_isolation.sh`. The wrapper was introduced with 36
in `7a55e62c5`; its path history contains no committed switch to 48. The brief
reports that 48 was tried and tabled for RAM. That report is a constraint, not
a measured 48-worker result established by the serial profiles. The later
72/108-worker experiment, rather than this historical report, governs the
revised recommendation.

## Authorized 1 / 2 / 3-copy experiment

Zach authorized one experiment in the review of #2149: a fixed target of at
least 200 scenarios, seed 1 and `--draw-parity`, first alone, then two copies,
then three. Exactly these six censuses ran; no case selection, timeout, pin,
seed or game behavior changed between stages. All used clean repository HEAD
`af989b4f4e0dcc779eaaf716eb9ad8006c1869f3`, whose only change from the initial
profile baseline is this design document, and the same reference oracle SHA.

Evidence is under the same dated archive root:

- `sol-parallel-experiment/`: `PROTOCOL.md`, `experiment.json`, exact
  `scenarios.txt` and `scenario-hashes.tsv`, driver/sampler, diagnostic sources,
  start commands/environment overrides, stage resource samples, `throughput.json`,
  `comparison.json`, `comparison-detail.json`, `flake-details.json`,
  `flake-port-evidence.json`, and checksummed receipts.
- `sol-parallel-exp-1x-1/`, `sol-parallel-exp-2x-{1,2}/`, and
  `sol-parallel-exp-3x-{1,2,3}/`: each census's tool-written manifest, captured
  HEAD/status, logs/results/rechecks, C dump, retained attempt logs, normalized
  C **and Go** blocks, raw draw logs and engine logs, including red attempts.

Selection was frozen before the first run: eight named witnesses plus 192
additional cases chosen by deterministic SHA-256 ranking of scenario names,
then ordered by descending static cost. The witnesses include relogin,
restart, passive/named-peer audiences, lifecycle pulses, `wizard-zreset-depth`,
`comm-depth`, the whirlpool special and appraise. The sample also includes
`crash-autosave-survives`. This is not a claim to have exercised the full
corpus or every transport at high concurrency.

**Authorized lock bypass:** the experiment driver held the ordinary global
`dp-census.lock` throughout all three stages, keeping unrelated censuses out.
Each experimental copy received a distinct archived `XDG_RUNTIME_DIR`, giving
its unchanged `census.sh` a private lock and last-run pointer. Every census
still started and waited through `census.sh`, with a distinct name, seed 1,
`--jobs 36 --draw-parity`, and the normal reference hash check. The bypass
ended with the experiment; it is not permission for ordinary agents to bypass
the lock. No repository tooling changed.

A retained `CENSUS_RUNNER` wrapper delegated to `oracle_regression.sh`, retaining
all attempt logs. A Go `-overlay` added diagnostic writes only after both
captures finished and the existing comparator built normalized blocks. It
copied both sides' blocks, draw logs and engine logs before cleanup. Original
and instrumented source are retained; the same overlay was used in all six
runs. Server input/output, timing windows, normalization, draw comparison and
worker classification were unchanged. Manifests identify repository HEAD;
`experiment.json` and the protocol disclose the additional diagnostic overlay.

| Metric | One copy / 36 workers | Two copies / 72 workers | Three copies / 108 workers |
|---|---:|---:|---:|
| Scenario/seed pairs | 200 | 400 | 600 |
| Each initial regression, seconds | 140.965 | 143.881 / 144.094 | 149.289 / 150.605 / 152.670 |
| End-to-end execution makespan, seconds | 150 | 155 | 265 |
| Pairs per second, including required rechecks | 1.333 | 2.581 | 2.264 |
| Throughput relative to solo | 1.00× | **1.94×** | **1.70×** |
| Final wrapper verdicts | CLEAN | CLEAN / CLEAN | CLEAN / CLEAN_AFTER_RECHECK / CLEAN_AFTER_RECHECK |
| Initial INFRA rows requiring wrapper recheck | 0 | 0 | 5 |
| Internal infrastructure RETRY lines | 3 | 4 | 4 |
| Captured attempts differing from solo | 0 | 0 | 5 |
| Final transcript / draw-row differences | 0 / 0 | 0 / 0 | 0 / 0 |
| Final draw-divergent rows | 0 | 0 | 0 |
| Peak aggregate process-tree RSS | 3.79 GiB | 7.52 GiB | 11.37 GiB |
| Minimum host MemAvailable | 6.20 GiB | 4.20 GiB | 2.97 GiB |
| Average / peak sampled tree CPU, logical cores | 1.22 / 11.48 | 2.64 / 16.12 | 2.60 / 14.28 |
| Host swap-out during sample window | 12.1 MiB | 1,302.1 MiB | 3,236.8 MiB |

Execution makespan is earliest runner start to latest manifest finish,
including enumeration/builds and required rechecks, at one-second timestamp
resolution. It excludes the observer's polling delay. The driver/sampler
windows were 165.33, 165.51 and 271.05 seconds; CPU averages use those windows.
Sampling every approximately 0.75 seconds has the same RSS/short-process limits
as the initial profiler. Peak CPU is an approximate staggered `/proc` sample;
16.12 does not mean this 16-thread host acquired additional cores. Swap counters
and PSI are host-wide and include desktop activity; they establish observed
pressure, not exclusive attribution or the cause of a particular flake.

The 72-worker stage completed 400 pairs in only five seconds more than 200
alone. Comparing only the three-copy initial regression times would suggest
near-linear scaling too, but it would omit the serial recheck tail: two runs
finished at 253 and 265 seconds, while the third finished at 159. The complete
108-worker stage was slower per pair than 72 and had less memory headroom.

### Transcript, draw and flake evidence

Every final run contained 3,317 normalized probe/audience blocks, including
1,497 whitespace-only blocks per engine. For all 1,200 final pair executions, both engines' blocks, labels/order, each side's raw `DRAW` rows,
and final MATCH reports matched the solo counterpart exactly. No new
normalization was applied. The reset witness retained the 13-byte `zreset *`
response; the flee actor/origin/destination witnesses stayed nonempty. Final
classification after rechecks was PASS for every pair, but initial
classifications and wrapper verdicts were **not** all equal.

The 36- and 72-worker stages had startup/infrastructure retries but no captured
transcript/draw differences and no wrapper rechecks. At 108, these five failed
attempts were retained and became INFRA when they did not reproduce:

| Run suffix / scenario | Retained first-attempt difference |
|---|---|
| `3x-2` / `meteor-swarm-message-miss-attacker-depth` | Transcripts still matched; Go draw 4112 changed from `Number(100,160) = 131` to `Number(120,180) = 151`, with 4,218 rows rather than 4,128. |
| `3x-2` / `skillmsg-breath-zero-lua-depth` | Go emitted character-entry/infirmary text instead of the breath probe messages; its draw log was shorter. |
| `3x-2` / `gsay-depth` | Go gained foreign `WmsgB`/`WmsgAtt` link-loss messages and a different draw stream. |
| `3x-3` / `fightmsg-weapon-300-god-depth` | C returned `Huh?!?` instead of the forced-combat result; its draw stream changed. |
| `3x-3` / `cutthroat-skill-message-hit-depth` | C returned `You're not trained in slitting throats!` instead of the successful cutthroat result; its draw stream changed. |

The later matched attempts and the five successful wrapper rechecks do not
erase these reds (R3/R5h). Final `draw_divergent=0` does not mean every draw
attempt matched. These are not five proven instances of a first-byte deadline
being exceeded; the retained evidence demonstrates an isolation failure:

- In `3x-2`, meteor's Go process PID 509499 logged a successful telnet bind to
  **39763**. Breath's Go process PID 509545 was assigned the same port and
  logged `Telnet listener failed ... bind: address already in use`, yet its
  harness continued through setup/probes. Meteor's draw stream changed while
  breath's transcript contained the wrong state. Distinct roots did not
  prevent two harnesses from driving one live endpoint.
- The failed `gsay-depth` Go server on **40921** logged foreign `Wmsg*` player
  creation. The process inventory assigns that same telnet port to the
  concurrent `fightmsg-weapon-305-god-depth` worker in `3x-3`. This is a
  cross-run witness; the 39763 collision was between cases within one run.

The audited call path explains why this can escape a clean bind-error abort:
`cmd/server/main.go:978-983` logs a telnet bind error and continues;
`dialWhenReady` in `cmd/dp-oracle-diff/main.go:1877` checks process liveness and
TCP reachability, not that the responding socket belongs to that process.
The close-to-bind window is therefore more serious than the
`allocatePorts` comment's claim that a collision cannot produce a wrong
verdict. Ownership verification must precede fixture login, including C's
readiness connection and every peer/restart connection. Do not fix this by
widening capture windows or retry budgets. This evidence attributes the named
port-sharing witnesses; it does not prove the complete cause of all five reds.

The implicit Go WHOD gap also remains real: retained engine logs
contained 14, 44 and 108 WHOD failure log lines across the three stages
(the server can log an event twice). Even the solo stage had failures, so a
72-worker budget is not a substitute for all-listener isolation.

This is one ordered experiment at seed 1, not repeated trials or proof that
72 has no rare flakes. It supports a provisional 72-worker total with an
initial per-run cap of 36, a memory admission guard, and a listener-ownership
prerequisite. It does not justify enabling 108 or generalizing the measured
speedup to all full/claims corpora. No additional experiment was run after the
five reds; the original artifacts and all retries remain retained.

## Isolation audit: what really serializes runs

| Surface | Current ownership and consequence |
|---|---|
| Global lock | `${XDG_RUNTIME_DIR:-/tmp}/dp-census.lock`, `flock` on FD 9, held by the detached runner. This explicitly refuses a second census (exit 5). It is an admission policy, not a fixed-port necessity. |
| Reporting pointers | The lock body names one PID/run; `dp-census.last` stores one last run. `resolve_run` can therefore report another agent's run unless callers specify `--run` or assert `--name`. These need redesign alongside admission. |
| Runtime roots | `execute` uses `os.MkdirTemp("", "dp-oracle-diff-")`, normally `/tmp/dp-oracle-diff-<random>/`. Each attempt owns `oracle-lib/`, `go-world/`, `go-work/`, fixture files and optional draw logs. Deferred cleanup removes its root. No scenario-name-only shared runtime path was found. |
| Databases | Store-requiring scenarios create SQLite state beneath their own `go-work/`. Others use the intentionally unreachable `/nonexistent-darkpawns-oracle-diff/relogin.db`, not a shared writable database. |
| Builds and logs | `oracle_regression.sh` creates a unique `dp-oracle-regression.XXXXXX` directory or an `attempt.XXXXXX` under a run-specific log root. It builds one harness and server per invocation; workers share those immutable binaries only within that invocation. Concurrent builds still consume RAM/CPU even before scenarios start. |
| Ports | `allocatePorts` reserves four TCP numbers: C P and WHOD P+1, plus independent Go telnet T and HTTP H. Go also defaults its WHOD listener to T+1, which the harness neither reserves nor disables. Five listener numbers are therefore intended, with only four reserved. Reservations close just before boot; close-to-bind and restart races remain. Failed optional Go WHOD binds are logged without terminating the server. |
| Reference check | `do_start` hashes the executable and reads the repository's pinned SHA. This is read-only and does not require run serialization. Replacing the binary during active runs is unsafe; promotion must wait for all active runs and follow the existing promotion procedure. |
| Retained evidence | Run directories are separate, but two starts with the same date/name currently rely on the global lock to serialize the emptiness check. Concurrent admission must reserve a unique directory atomically. |

The source paths behind this table are `scripts/census.sh`,
`scripts/census_runner.sh`, `scripts/oracle_regression{,_worker}.sh`,
`scripts/{claims_census,combined_census}.py`, and
`cmd/dp-oracle-diff/main.go` (`execute`, `allocatePorts`, `prepareGoReloginDB`).
The kernel's ephemeral range on this host is 32768–60999; each live worker
intends five listening port numbers (four explicitly reserved), plus client
ephemeral sockets. The Go default is verified at `cmd/server/main.go:294`
and its startup path at `cmd/server/main.go:998-1008`; C derives its WHOD port
in `src/whod.c:247-253`, called by `src/comm.c:274`. Ports are
not an argument for a one-census limit, but reservation races, TIME_WAIT, and
boot failures must remain observable.

The targeted kernel samples observed four distinct scratch roots, 16 configured
port numbers across the four reserved roles, no overlap among those sampled
roles, no surviving
observed scratch roots after completion, and five C and five Go process lifetimes (the restart replaces engines). For example,
`/tmp/dp-oracle-diff-45311637/` used C 41635, WHOD 41636, Go telnet 36493 and
HTTP 45045; its SQLite file was `go-work/relogin.db`. The Go restart reused
those ports and that database as intended. Another root,
`/tmp/dp-oracle-diff-4171848834/`, used C 39993, WHOD 39994, Go telnet 33423
and HTTP 38341 with the deliberately unreachable store. WHOD numbers are
derived from the audited P+1 rule, not sampled socket ownership. The retained
argv/cwd records identify the C/Go engine pair by its shared scratch root.
The full sample observed 1,614 scratch roots across attempts, no surviving
observed roots, 1,623 C and 1,624 Go process lifetimes, and 5,011 distinct
configured port numbers across the four explicitly reserved roles. Ports can
be reused by later workers. In 105 sample intervals, live process argv records
contained an overlapping configured number. These observations include boot
and shutdown periods and do not prove two bound sockets simultaneously shared
an endpoint. Socket-inode ownership was not sampled, and the analyzer does not
include the implicit Go WHOD fifth port. They refute a claim that this profile
proved collision-free operation; they do not attribute all 12 retries to binds.
The existing isolation script also inventories only four configured roles, so
its historic green cannot certify the additional Go WHOD listener. A future
isolation proof must enumerate actual socket owners and all five roles.
The authorized experiment above supplies additional engine-log witnesses of
actual foreign-endpoint use; the initial argv-only samples could not establish it.

Conclusion: the global lock is the direct single-census barrier. The unreserved
Go WHOD port, shared reporting/registration state and aggregate resource
pressure prevent safely
removing it alone. The current 36-way fan-out already tests many independent
engine pairs; separate census ownership does not itself add engine isolation.

## Proposed admission and scheduling

Use one coordinator per host/user for every worktree, with one canonical
runtime directory, for example `/run/user/<uid>/dp-census/`. Do not let different
`XDG_RUNTIME_DIR`, `TMPDIR`, or checkout paths silently create independent
budgets. Persist run identity in the archive; runtime permits can disappear
at reboot. A short registry mutex protects admission, queue order and state
transitions. Never hold it while building, waiting for a permit, or executing a
scenario. Keep process-held locks/handles for liveness rather than trusting a
PID alone (PID reuse requires process start identity).

1. **Register:** atomically reserve a new run directory; reject an existing
   nonempty run. Capture HEAD, dirty status, selection, seeds, oracle path/hash,
   requested jobs, priority and submitted time before queueing. Freeze the
   selected inputs and binaries before dispatch: queued work must not silently
   build a later checkout revision. Record binary/input digests and verify the
   reference again before first execution. A changed tree or oracle fails
   admission explicitly; never relabel it as the captured HEAD.
2. **Prepare:** serialize census prebuilds through a separate single build
   slot. A build uses memory outside the worker permits; only admit it with
   memory headroom, and defer it while an exclusive release is running. Reuse
   the existing per-run prebuilt-server approach. Do not rebuild per worker.
3. **Dispatch:** one permit covers an entire scenario worker, including its
   harness, C/Go engines, restarts and internal attempts, until descendants are
   reaped. Acquire before fixture copies/boot, release after cleanup. `--jobs`
   is a per-run upper bound; it never reserves idle permits. At most **72**
   workers across all censuses may be active, initially no more than 36 per
   run, matching the experiment. Two 36-job runs can progress together; an
   arriving targeted run shares freed permits fairly across those owners.
4. **Share:** rotate ready ordinary runs round-robin at scenario boundaries,
   lending unused capacity to other runs. A newly arrived small run gets the
   next freed permit rather than waiting for a whole full run to finish.
   Per-run long-first ordering stays intact; a running scenario is never
   paused or killed to redistribute its permit. No promise of equal wall time:
   scenario lengths differ.
5. **Release priority:** an explicit release class (a proposed option, not an
   existing CLI flag) closes ordinary dispatch, lets active workers and builds
   finish, then admits only the release `--combined` run. Ordinary queued work
   remains registered. Existing scenarios are not interrupted, and ordinary
   retries do not leapfrog the release barrier. Multiple releases use FIFO.
   Priority must be explicit; ordinary full/combined runs are not automatically
   releases. Draining can take minutes: the current worker permits up to three
   attempts with a default 240-second timeout per attempt. Repeated releases may
   delay ordinary work; report that fact.
6. **Retry and recovery:** the full, targeted, claims and combined drivers,
   including infrastructure rechecks, must all use the same admission layer.
   Keep the existing serial recheck limit and verdict rules. No pool-per-seed
   or retry path may bypass the global count. A coordinator restart stops new
   admission until it reconciles live process groups and their permits. A
   heartbeat timeout alone must never reclaim a permit from a still-live worker.

An FD-backed permit must live as long as its engine descendants; conversely,
it must not leak into unrelated long-lived children. Cancellation kills and
reaps only that run's process group before releasing its permits. SIGKILL,
coordinator death, orphan descendants, PID reuse, reboot and disk-full need
explicit recovery tests. Do not erase another run's lock/body from a generic
EXIT trap, as the current single-run cleanup would do if copied unchanged.

Memory admission supplements the 72-worker ceiling; it never raises it.
Use a conservative measured per-worker envelope including harness/copies and
build reserve; stop admitting new workers below the selected available-memory
floor, then resume only with hysteresis. Leave running scenarios alone and
report pressure. Reserve desktop/OS headroom and track swap-in/out and memory
PSI; existing swap occupancy alone does not prove active thrashing. A provisional
4 GiB MemAvailable floor matches the remaining headroom observed at 72; pause
new dispatch below it and resume only above a higher hysteresis threshold.
This is an admission guard, not a promise that current workers cannot dip below
it. Validate burst admission and headroom tuning in the implementation; a busier
desktop can reduce usable capacity below 72.
A permit count is not a hard RAM guarantee.

## Implementation boundary and rollout

A future implementation would touch `census.sh` for registration/reporting,
`census_runner.sh` for per-run lifecycle, and the full/targeted, claims and
combined dispatch paths for permit acquisition. The coordinator must own fair
worker dispatch: placing an independent semaphore wait inside each existing
`xargs` pool alone does not establish fairness or release priority. Before
ordinary concurrency is enabled, reserve all five listener roles, close the
reservation-to-bind ownership gap, and verify the expected child owns every
endpoint before sending login/setup. A host-wide boot lease through verified
bind is one option; its throughput cost is unmeasured. An open TCP endpoint or
a process-wide readiness marker alone is insufficient. Abort an attempt on a
failed required listener rather than driving another worker's socket.
`oracle_regression_worker.sh` remains the classification authority; no
acceptance pin, normalizer or timeout change is part of this design.

Keep an exclusive-admission compatibility mode available for rollout and
rollback. Begin with stub scheduler/lifecycle tests, then the census-backed
acceptance experiments below, then ordinary concurrent targeted runs. Enable
full/claims/combined concurrency only after every driver and retry path obeys
the aggregate invariant. A release still requires its existing exact-commit
combined census. Proposed changes to agent instructions belong in the future
approved brief: this document does not amend AGENTS.md or the fidelity law.

## Evidence and reporting contract

Every run keeps its own
`$ORACLE_RUNS_ROOT/YYYY-MM-DD/<run>/MANIFEST.md`, captured Go HEAD, dirty status,
reference sha256, exact commands, seeds, selection, per-scenario results,
transcript dumps, attempts where retained today, rechecks, summary and exit
code. Combined runs keep their deduplicated pair schedule and separate full
and claims projections. No worker appends to another run's result or dump.

Add scheduling metadata, without replacing existing evidence: coordinator
version, budget/headroom settings, requested/effective maximum workers,
submission/admission/finish times, queue and execution duration, cancellations,
and allocation events. Track queued/running/done/killed/failed states separately;
queued is not dead merely because it has no runner PID yet. Preserve initial
failures and all infrastructure rechecks under their original run identity.
Finalization must atomically publish terminal state only once the summary and
manifest are complete; reporting must not race manifest creation.

`start` returns immediately with the exact directory in one line. With no
available permits, recommended proposed output is:

```text
census queued: /…/2026-10-25/luna-probe @<head> position=2 active=72/72 requested=4 eta=unknown
```

If it can dispatch, use `census started:` with the same identity and allocation.
Queue rank is advisory under release priority. Estimates, when available, must
be labeled estimates based on active remaining work and retained timings;
never invent a deadline or recommend aggressive start/status retry loops.
Support an explicit no-queue mode returning exit 5 with active owners and
`eta=unknown` when no trustworthy estimate exists. This preserves automation
that prefers refusal without making it the ordinary agent workflow.

Keep `wait --run <directory>` bounded with exactly one report line and exit
codes 0 clean, 1 not clean, 3 queued/running, 4 unavailable/dead. The current
draw-parity mode can append a second diagnostic line; fold that tally into the
same report line in the implementation while retaining its diagnostic evidence
files. Do not add scheduler chatter to wait output. `--name` assertions
must still fail on mismatch. Bare `wait` may resolve a single unfinished run,
including a queued run; when multiple unfinished runs exist, refuse with one
line naming the need for `--run`
instead of selecting another agent's result through `dp-census.last`.
A last-run pointer remains convenience only, never ownership evidence.
Agents save the directory returned by start, wait on that identity, and leave
the queue alone. They do not change runtime-directory variables to evade the
budget, start duplicate jobs, inspect growing logs, or cancel others' work.

## Determinism and acceptance before implementation ships

Separate process state, per-run fixtures, explicit seeds, `DP_CLOCK=1`, and
`DP_FIXED_TIME` prevent cross-run game state or RNG sharing. Scheduling order
must change only wall time, not inputs, seeds, normalization, pins or draws
(R1/R3). This is a necessary isolation argument, not a proof that bytes cannot
change under CPU/memory pressure.

The earlier counterexample is retained in
[`first-byte/README.md`](../fidelity/depth/evidence/2026-10-01-combined-census/first-byte/README.md):
at 36 workers, `wizard-zreset-depth@2` captured an empty Go response before
`Reset world.` arrived. The two-second first-byte allowance repaired that
failure, retaining 300ms trailing silence. Actor-first capture precedes passive
peer reads. Do not shorten those windows or extend all silent reads as a
scheduler workaround. Contention can still exceed a finite deadline.

A separately authorized implementation must pass these checks, retaining every
run through `census.sh` and every baseline/concurrent comparison:

- Freeze one revision, oracle SHA, scenario inputs and timing options. Run the
  same targeted set alone, then concurrently in two distinct runs sharing the
  same budget. Repeat at least ten matched trials at 4+4, asymmetric 4+32 and
  aggregate 72, respecting the initial per-run cap; include two simultaneous
  full/combined owners after the targeted proofs. Add a repeat under measured memory pressure without
  exceeding the approved ceiling. This is a proposed acceptance experiment,
  not evidence claimed by this PR.
- Include this profile's four cases plus `wizard-zreset-depth` at seeds 1 and
  2, a named-peer issuer, legitimate silence, WebSocket capture, restart/crash,
  and a known pinned divergence. Enumerate the exact case/seed pairs in the
  experiment manifest before running. Assert intended output is nonempty for
  reset/peer witnesses: two identical missing blocks must not become proof.
- Compare exact per-pair classifications, normalized C **and Go** block bytes,
  audience order and fingerprints against the solo baseline, excluding only
  already-approved normalization. C-only `census-dump/` plus equal verdicts is
  insufficient; the implementation experiment must retain both sides' blocks
  through its census runner. Preserve RNG draw comparison where supported;
  a byte-green result with a new DRAW is not concurrency acceptance.
- Compare first attempts, boot/bind failures, retry counts, raw empty capture
  statistics and capture latencies as well as final verdicts. A new failure
  repaired by an infrastructure recheck still fails this no-new-flake proof.
  Known run-varying C behavior must be declared/excluded explicitly before
  comparing stable-byte hashes; never create a fresh pin to bless a difference.
- Observe ports, runtime roots and SQLite paths across *both* runs from kernel
  state, including restart intervals; check cleanup against a pre-run baseline.
  Reuse the isolation script's principles inside the census experiment rather
  than bypassing `census.sh` or interpreting historical isolation as new proof.
  Cover the implicit Go WHOD listener as well as the four reserved roles;
  resolve its allocation gap in a separately reviewed tooling change before
  claiming all intended listeners are isolated. Do not silently disable a
  player-observable listener as a scheduling workaround.
- Exercise queue fairness, release draining/exclusivity, cancellation while
  queued/running/rechecking, duplicate run names and ambiguous bare wait.
  Include assertion controls that deliberately omit a global permit in a
  disposable test scheduler: the aggregate-budget test must turn red (R5h).
  A known capture-delay regression must also make the transcript comparison red.
  Add a controlled foreign-listener fixture that makes the current ownership
  check fail and verifies that no login/setup is sent to the foreign process.

No finite sample proves an absolute absence of flakes. Acceptance means no new
observed differences or capture failures under the specified trials and a
reviewable argument for input/state isolation. Roll back to exclusive census
admission if concurrent operation produces new transcript/draw differences or
worsening infrastructure failures; retain the red evidence and investigate
rather than widening retries. The authorized 108-worker stage has already
failed this acceptance standard. The 72-worker sample passed the captured-byte
and draw comparison, but listener ownership is still a rollout prerequisite.

## Decisions for Zach

1. **Worker budget:** approve a provisional host-wide ceiling of **72**,
   initially at most 36 per run, with memory admission/build reserves and a
   provisional 4 GiB available-memory floor. The experiment measured 1.94×
   throughput with no captured transcript/draw differences at 72; reject 108
   for now because its five transient differences and recheck tail reduced
   throughput. Listener ownership must be fixed and verified before rollout.
2. **Release exclusivity:** approve exclusive release `--combined` admission
   after non-preemptive draining, with ordinary runs retained in the queue.
3. **Queueing versus refusing:** approve immediate queued registration by
   default, explicit run-bound waits and an optional refusal mode; report
   `eta=unknown` until a defensible estimate exists.
