# Shared census admission implementation

Implements [the approved design](parallel-census.md). Zach approved the 72-worker
host/user budget, 36 per run, a 4 GiB memory floor, exclusive release admission
after draining, and queued registration by default. These decisions do not
require another policy choice in this PR. AGENTS.md is unchanged; its exact
proposed replacement appears below for Zach's separate approval.

## Operator interface

```bash
scripts/census.sh start --name issue --scenarios a,b --draw-parity
scripts/census.sh wait --run /absolute/directory/printed/by/start --max-seconds 45
scripts/census.sh status --run /absolute/directory/printed/by/start
scripts/census.sh cancel --run /absolute/directory/printed/by/start
scripts/census.sh start --combined --release --name release-candidate
```

`start` returns after snapshot/registration with one started/queued line, captured
HEAD, queue position, active workers, requested maximum, and `eta=unknown`.
`--jobs` accepts 1–36 and caps this run, not the host total. `--no-queue` refuses
with exit 5 when immediate admission is unavailable. `--exclusive` provides
exclusive ordinary admission for rollback/diagnosis; a release additionally
gets FIFO priority and must use `--combined`. Running scenarios are not killed
to make space for a release. New ordinary builds/scenarios wait while it drains.

`wait` and `status` print one line, including draw diagnostics on that same line.
Their exits remain 0 clean, 1 not clean, 3 queued/running, and 4 missing/dead or
ambiguous. Bare reports refuse to select among multiple unfinished runs.
`--name` still asserts identity. Cancellation requires `--run` and only affects
that run. The Makefile wait/status targets require `CENSUS_RUN` explicitly.

## Implementation and recovery

- `census.py` reserves the run directory atomically and copies tracked plus
  non-ignored untracked inputs before queueing, checks for concurrent checkout
  edits, records HEAD/dirty status and file digests, and makes the source copy
  read-only. Builds, embedded scenarios, fixtures and world copies use that
  snapshot. External manifest fixtures and custom runner scripts are retained
  separately. Queued runs cannot silently build a later checkout revision.
- `census_budget.py` uses one short `flock` transaction around a versioned,
  atomically replaced registry. There is no persistent daemon to restart.
  Ready runs rotate at scenario boundaries. Full, targeted, claims, combined,
  internal attempts and serial infrastructure rechecks use the same worker
  lease wrapper before fixture creation. The classifier remains unchanged.
- The canonical namespace is `/run/user/<uid>/dp-census/`, falling back to
  `/tmp/dp-census-<uid>/`. XDG_RUNTIME_DIR/TMPDIR do not create independent
  budgets. Explicit isolated fixture mode requires both `CENSUS_TESTING=1`
  and `CENSUS_TEST_RUNTIME`; it is not normal oracle proof configuration.
- Admission stops below 4 GiB MemAvailable, resumes at 4.5 GiB, and does not
  preempt running work. Pending boots have a conservative temporary reserve.
  A separate single build slot reserves 1 GiB plus dispatch-wrapper headroom;
  builds use `go build -p 4`. Available memory and transition events are retained.
  This is an admission guard, not a hard memory limit for arbitrary scenarios.
- Each granted build/worker runs in its own process group under a Linux
  subreaper. A pipe prevents exec until that group is recorded. TERM, INT,
  SIGKILL and orphan recovery are reconciled using process start identities;
  a permit is not reclaimed while its engine group remains alive. Any registry
  operation also reconciles dead runners; recovery never depends on an agent
  repeatedly checking a growing log. Killing a whole old run is conservative
  when its supervisor dies; it never becomes CLEAN merely because time passed.
- During migration, shared guards on both standard legacy lock paths keep older
  single-census tools from entering while new runs execute. New registrations
  queue behind an older exclusive holder. Upgrade all participating worktrees
  before expecting concurrent starts through their wrappers.

Custom `CENSUS_RUNNER` remains a trusted diagnostic/test seam. A custom runner
which delegates to the standard build and worker helpers gets ordinary leases;
an arbitrary custom program must not bypass those helpers and claim budgeted
proof. Production proof below uses the standard runner, with no lock bypass,
custom runner, timing override or Go overlay.

## Listener identity before login

The harness reserves all five ports: C game/WHOD, Go telnet/WHOD, and Go HTTP.
Ports are selected outside the kernel's automatic outgoing-client range. A
host-wide boot lease covers each close/start/bind handoff. Linux `/proc/net`
and the expected child's socket FD inodes establish ownership of every listener
before any fixture login. Go's nonfatal telnet-bind log cannot pass readiness
by redirecting the harness to another worker. TCP dials verify ownership both
before and after connecting; browser connections are checked before login as
well. Restart/crash recovery uses the same lease and verification.

Boot admission can wait up to two minutes separately from the unchanged
per-engine readiness deadline. The existing 240-second worker watchdog still
bounds the entire attempt. First-byte and trailing-silence windows are unchanged.
No C source/oracle binary or game/server behavior changes.

The foreign-listener fixture rejects an actual unrelated live endpoint before
connecting. Go WHOD reservation and socket-inode recognition have positive
checks too. Budget tests cover caps, fair selection, memory hysteresis,
release draining/FIFO, immutable queued inputs, duplicate names, legacy guards,
rechecks, run-bound waits, cancellation and runner SIGKILL recovery.

Every completed attempt retains normalized C/Go blocks, raw draw logs and engine
logs beside its attempt log. `check_census_equivalence.py` rejects missing/later
captures, retries/rechecks, surviving listener failures, changed transcripts,
changed draw rows or changed classifications. A final green cannot erase a red.

## Oracle proof and controls

On 2026-10-25 the same frozen 200-scenario selection from #2149 ran alone,
then in two simultaneous 36-worker runs. All used clean implementation commit
`5e00778e60745f891445297f50366e5a115319d0`, seed 1, `--draw-parity`, the
reference oracle, and ordinary shared admission. No lock bypass, test namespace,
custom runner, Go overlay, timing change or pin change was used. The only commit
after this proof adds this report; executable inputs remain the tested revision.

Evidence root:
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-25/sol-parallel-census-implementation/`.
`PROTOCOL.md`, `protocol.json`, `scenarios.txt`, `scenario-hashes.tsv`, start
receipts, scripts, samples, comparisons and controls are retained there. The
three sibling run directories each retain `MANIFEST.md`, immutable `source/`,
input hashes, admission events, summaries and every attempt's native captures:

| Run | Result | Driver elapsed |
|---|---|---:|
| `sol-parallel-impl-1x-1` | CLEAN, 200 PASS, 0 draw divergences | 159.263 s |
| `sol-parallel-impl-2x-1` | CLEAN, 200 PASS, 0 draw divergences | 224.426 s |
| `sol-parallel-impl-2x-2` | CLEAN, 200 PASS, 0 draw divergences | 221.289 s |

`check_census_equivalence.py` compared all classifications, both normalized
block arrays (including command/audience order), each side's raw DRAW rows and
draw MATCH reports. All three produce the same digest:
`39f1d2580071ca249b11953a34e36845730c00ced79c4e106d1c277c27c31211`.
Each contains 3,317 blocks per engine, including the same 1,497 whitespace-only
blocks. Reset (`Reset world.`), named-peer and audience witnesses were explicitly
checked for nonempty output. No attempt retried; no infrastructure recheck,
boot/bind failure or new flake was observed. All three input-hash files are
identical: SHA-256
`045efae8e5c003604b329f1aba08c04a9dcf5d7ea385a3edb9d4278e303107f0`.

| Measured stage | Solo | Two copies |
|---|---:|---:|
| End-to-end wall, registration through finalization | 170.18 s | 237.15 s |
| Scenario executions | 200 | 400 |
| Aggregate throughput versus solo | 1.00× | 1.44× |
| Replayed maximum scenario permits | 36 | 72 (36 per run) |
| Peak sampled process-tree RSS | 4.37 GiB | 8.68 GiB |
| Minimum host MemAvailable | 7.33 GiB | 4.84 GiB |
| Average / peak sampled CPU cores | 1.61 / 2.68 | 2.93 / 9.69 |

The implemented ownership checks and supervision cost throughput relative to
#2149's earlier diagnostic experiment; this proof supports a smaller measured
speedup, not its earlier 1.94× claim. Memory admission did not pause during this
sample. The paired stage saw host-wide swap counters rise by 3,854 input pages
and 41,751 output pages (about 15 MiB / 163 MiB at 4 KiB/page); these are host
counters, not attribution to census processes. RSS sums include shared pages;
short-lived peaks and CPU can be missed by the approximately 0.75-second sampler.
The floor remains admission protection, not a guarantee against host pressure.

Kernel socket/FD samples observed up to 180 solo and 360 concurrent engine
listeners, including WHOD, without an engine port shared by different PIDs.
The 200 solo / 400 paired disposable roots had no cross-run owner. No sampled
process identity survived completion. Snapshot/lease evidence and endpoint
analysis are in `analysis.json`, `endpoints-analysis.json`, `equivalence.json`
and the stage sample files. These are sampled observations, complemented by the
mandatory ownership check before every login; they do not prove every instant.

The deterministic suites passed 50 Python cases and 40 shell classification
cases, alongside the Go suite and listener fixtures. Four retained mutation
controls went green → assertion failure → green: bypass ownership, raise the
host ceiling to 73, remove the memory floor, and remove release draining.
`revert-controls.json` and each before/broken/restored log retain those results.
The comparison fixture also rejects a changed Go response, changed draw row,
retry and foreign-listener log. The initial live smoke failed before executing
any scenario because the build helper placed Go's global `-C` flag too late;
that NOT_CLEAN archive is retained as `sol-parallel-ownership-smoke`. After the
ordering fix, `sol-parallel-ownership-smoke-fixed` completed CLEAN.

This is the requested 200-case solo-then-pair proof, not a universal no-flake
claim. The design's broader proposed ten-trial, asymmetric, memory-pressure and
simultaneous full/combined matrix was not run. Release draining, memory admission,
queue fairness and crash/cancellation paths have isolated fixtures; no live
release gate or final full/combined census is claimed by this PR. Roll back to
exclusive admission and retain the red evidence if new concurrent differences
appear.

## Exact proposed AGENTS.md replacement (not applied)

Replace only the section beginning `### Oracle Evidence Retention` and ending
immediately before `### Reference Oracle Binary` with the following text. This
is proposed text, not an instruction override until Zach approves and lands it.

````markdown
### Oracle Evidence Retention

Oracle transcripts, census logs, summaries and manifests are retained research
evidence. Run every census through `scripts/census.sh`; it writes them to
`$ORACLE_RUNS_ROOT/YYYY-MM-DD/<run-name>/` (default
`~/Archives/darkpawns/oracle-runs/`), captures Go HEAD and dirty status, freezes
source inputs before queueing, checks the reference oracle's sha256, and writes
`MANIFEST.md` itself. Every attempt retains both engines' normalized blocks,
draw logs when enabled, and engine logs; retries never overwrite a red attempt.

```bash
scripts/census.sh start --name <issue-or-run>            # full census, returns after registration
scripts/census.sh start --name <run> --scenarios a,b,c   # targeted
scripts/census.sh start --combined --name <run>          # full seed-1 corpus plus claimed pairs
scripts/census.sh start --combined --release --name <run> # exclusive release gate after draining
scripts/census.sh wait --run <returned-directory>        # bounded wait; ONE line
```

The host/user shares at most **72 scenario workers**, at most **36 per run**.
`--jobs N` sets the run's maximum across all seeds; it does not allocate another
host budget. Registration queues by default. Memory admission stops below
4 GiB MemAvailable and resumes at 4.5 GiB; a separate serialized build slot
reserves build headroom. Running scenarios finish without preemption.
`--release` requires `--combined`, drains active ordinary scenarios/builds,
then gives the release exclusive use. Ordinary combined runs are not releases.
`--no-queue` refuses with exit 5 when immediate admission is unavailable.
`--exclusive` is available for exclusive ordinary diagnosis/rollback.

When a brief requires both a full census and a claims census, use `--combined`.
It executes their union once, retains the full and claims result files and
separate verdicts, and is clean only when both verdicts are CLEAN or
CLEAN_AFTER_RECHECK. Separate full and `--claims` runs remain available for
diagnosis and comparisons. A release ships only from its exact-commit clean
combined gate.

Save the directory printed by `start` and use `--run` for every wait/status or
cancellation. Bare reports refuse to choose among multiple unfinished runs.
`wait` exits 0 (clean), 1 (not clean), 3 (queued/running: call `wait` again),
or 4 (dead/missing/ambiguous). Its one line includes any draw tally.

**Waiting costs tokens; the census does not.** While a census runs:
- Do not poll in a loop, `sleep`-and-check, or re-run `status` repeatedly. Call
  `wait --run` (each call blocks up to 9 minutes), or end your turn and say the
  census is queued/running; Zach gets a desktop notification when it finishes.
- Never `cat`, `tail` or `grep` a running `census.log`. When it has finished,
  read `summary.txt`, and read `results.tsv` only for the rows you need.
- Concurrent registrations are allowed through the updated wrapper. Do not
  change runtime variables, use test namespaces, launch duplicate jobs, or
  call the underlying runner directly to evade admission. A queued line with
  `eta=unknown` is not an instruction to retry start.
- Cancel only your own run, with `census.sh cancel --run <returned-directory>`.
  Never delete runtime locks, kill another run, or reclaim a live worker's
  permit. Upgrade older worktrees before expecting concurrent starts; their
  legacy exclusive lock remains respected during migration.

**Choose the smallest census that can fail on your change** (R5h):
- **Full census**: before every release, on the exact commit being tagged; a
  release ships only from a CLEAN or CLEAN_AFTER_RECHECK combined census, and
  any fix after a red census reruns it in full. Between releases, PRs changing
  player-facing output or game behavior run targeted scenarios covering it.
- **Targeted** (`--scenarios`): tests, comments, docs, or one statement whose
  effect a named set covers. Name the scenarios and why. A brief can require more.
- **None**: docs-only or website-only changes.

**The final full census is batched, and Claude runs it.** Coding agents run
necessary targeted censuses and open their PRs without a final full/combined
census. Claude stacks reviewed PRs over current `origin/main` and runs one
`--combined --release` census for the batch. Attribute a FAIL row by rerunning
only that scenario against each PR. Do not start a separate full census per PR.
Start the required target, then continue independent work in another worktree;
retain the exact run directory instead of waiting idle or following another
agent's result.

**Infrastructure rows.** The tool re-runs INFRA/TIMEOUT rows alone once;
CLEAN_AFTER_RECHECK counts as clean. Do not rerun the whole census for them.
FAIL, STALE and UNPINNABLE are never retried by the census wrapper: investigate.
EXPECTED scenarios remain ledger-backed known divergences. A concurrency
acceptance proof additionally compares first attempts, both transcripts and
draw rows: matching final verdicts after new flakes are not sufficient (R3/R5h).
````

## Decisions for Zach

The three operating decisions above are approved and implemented. The remaining
approval is the exact AGENTS.md replacement quoted here and in the PR. This
stop-tier PR does not edit AGENTS.md or authorize a reference-binary promotion.
