# First-response capture window (R5h)

The failed first combined run at 36 workers took 1627.231s. Full was CLEAN;
claims had one FAIL, wizard-zreset-depth@2. Both content attempts captured an
empty Go block for zreset *, while C emitted Reset world. The fingerprint was
f29e1d5ea2e86ffdcae1b69694f73e9c6b9ec29c6c937221e17670a739f5aeaf.
It remains a retained failed run; no retry overwrites it or relabels it clean.

C prints the line after resetting all zones (`src/act.wizard.c:2046-2051`). Go
also sends it after the reset (`pkg/session/wiz_zone.go:35-48`). The transport
used the 300ms end-of-output window for the first byte too. CPU contention can
therefore return an empty block before the command finishes.

The harness now defaults to a two-second first-byte wait while retaining 300ms
trailing silence. `-first-byte-wait=0` is the explicit legacy timing control.
The allowance is configured on every transport and reconnect, but is armed only
by a successful Send and consumed by that connection's next read. Passive peers,
initial greeting reads, and drains without a command keep plain quiescence.
TCP ignores telnet framing for the first game byte. The browser reader ignores
client chrome and local echo before starting the trailing-silence window.
No output is invented; legitimate silence remains bounded by the first-byte wait.

Empty-output census before this repair: the retained CLEAN full seed-1 dump has
9,364 normalized C probe/audience blocks, of which 2,556 are empty. The retained
clean claims dumps have 31,018 such blocks, of which 8,355 are empty. These are
logical empty normalized outputs, not raw empty socket reads: prompts can still
arrive in those blocks. The complete enumeration is retained at
~/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-combined-benchmark/empty-normalized-probes.json.
New `capture-statistics` diagnostic lines additionally count raw empty open and
closed probe/audience blocks, without changing the report's verdict or pins.

The disposable CPU fixture leaves both engines unchanged. It starts Go on one
available CPU at nice 19, forwards all traffic normally, then starts two busy
processes on that CPU for 950ms immediately before forwarding zreset *. C is
unrestricted. The initial old-window run loses precisely Reset world and
produces the exact original fingerprint on both attempts. The fixed-window run
passes all six probes under the same contention. Runs use census.sh, the reference
oracle, the unchanged classification worker and retained attempts/dumps.
All fixture processes are killed when the disposable server is stopped.

Unit CPU-delay proofs cover TCP, browser-local-echo handling, bounded silence,
and preserving the short trailing window. Three assertion-only 0/1/0 triples
are retained in unit-revert-triples.tsv. The real oracle triple is fixed PASS,
legacy-window FAIL with the original fingerprint, then fixed PASS again.
CPU runner, proxy and replay scripts are included here; original run directories
are dp-1371-first-byte-cpu-before/after/reverted/restored under the same archive.

Why 24 workers was slower: old full + claims consumed 74,553 worker-seconds at
36-way concurrency and took 2228.061s. Combined at 24 consumed 57,901 worker-
seconds and took 2426.657s, nearly the theoretical 57,901/24 = 2412.542s.
Deduplication removes 22.3% of work but 24 workers removes 33.3% of capacity.
Combined at 36 consumed 58,136 worker-seconds and took 1627.231s, close to
58,136/36 = 1614.889s, but its one failed pair invalidates it as a clean speedup.
The scheduler is keeping the pool busy; the reduced worker count dominates.
These are initial worker times, excluding separate census infrastructure rechecks.
The new capture allowance's cost will be reported from the fresh fixed-commit
baseline and combined measurements; the old 37m08s baseline stays documented.

Replay units with `python3 unit_reverts.py <checkout> <retained-log-directory>`.
Replay the CPU oracle fixture with `CPU_PROOF_FIRST_WAIT=0` (legacy) or `2s`
(fixed), `ORACLE_REGRESSION_SEED=2`, `CENSUS_RUNNER=<absolute-path>/cpu_runner.sh`,
and `census.sh start --name <unique-name> --scenarios wizard-zreset-depth --jobs 1`.
Then use census.sh wait. The legacy run must be NOT_CLEAN with the stated content
fingerprint; no infrastructure timeout or build failure counts as this proof.


## Actor-only review revision

RunAudienceProbe resolves the issuing target (including send:peer), sends its
command, completes that target's ReadUntilQuiescent, and only then calls
readAudiencePeers. Passive peers are read concurrently with one another, never
concurrently with the target's capture. Their blocks remain in sorted-name order.
SetFirstByteWait is now a per-command allowance, armed by successful Send and
consumed by the next read. Thus a named peer issuing a command gets two seconds;
the usual primary becomes its passive audience and keeps 300ms. No static
actor/peer designation can misclassify send:peer.

Outside probe capture, drainClients first drains the primary and then drains
peers concurrently. Those drains issue no command and use 300ms. Initial
connection/greeting reads likewise issue no command. During peer setup, each
peer issues its own login/setup commands and gets the allowance for the read
following its own Send; these reads need no preceding primary capture. Warmup
uses the same RunAudienceProbe order. Pulse commands and relogin setup arm the
allowance on the connection that actually issues them.

TestSlowIssuingConnectionCapturesPeerAfterActor uses 650ms of CPU work before
the issuing response and peer output. It proves both normal actor and send:peer
routing with real pipe transports. Moving passive reads before issuing capture
loses peer output and fails an assertion; restoring the order passes. Replay
with peer_reverts.py CHECKOUT LOG-DIRECTORY. TestPassivePeerUsesShortQuiescence
also proves the configured allowance does not extend a silent passive read.
The original zreset CPU-contention oracle triple is retained and rerun for this
revision. The new 36-worker run is captured at a fixed revision; prior measurements
remain historical rather than overwritten.
