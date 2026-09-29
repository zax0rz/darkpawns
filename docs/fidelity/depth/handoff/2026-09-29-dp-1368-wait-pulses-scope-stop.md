# DP-1368 — addendum proof complete; stop on newly exposed Go differences

This supersedes the pacing blocker in
[the initial stop report](2026-09-29-dp-1368-wait-pulses-stop.md). The user
explicitly authorized resuming under ADDENDUM 2026-09-29 in
`~/BRIEF-2026-09-29-wait-state-pulses.md`. The reviewer traced 150ms to C's
structural output latency: output flush precedes heartbeat, whose output reaches
the following real pass. The 150ms captures are retained, not counted as a wait
state defect. No C change was needed for this diagnosis.

Main worktree: `/home/zach/dp-wait-pulses`, `sol/wait-pulses`, implementation
commit `44f159953`. Oracle worktree: `/home/zach/dp-oracle-wait`, same branch,
commit `d2ace874868f34315280bb3a33b68a53d322de4b`. No PR is ready. PR 2 has not
started, and the reference binary/pin and divergence ledgers/pins are untouched.

## Revised pacing proof — complete

All nine scenarios pass all three required arms, with identical normalized C
blocks (R3c/R5h):

| Scenario | A: main 300ms | B: main 600ms | C: fast 300ms | C blocks |
|---|---|---|---|---|
| aid-failure-depth | PASS | PASS | PASS | identical |
| flesh-alter-gates | PASS | PASS | PASS | identical |
| order-charmed-depth | PASS | PASS | PASS | identical |
| parry-depth | PASS | PASS | PASS | identical |
| parry-failure-depth | PASS | PASS | PASS | identical |
| position-fighting | PASS | PASS | PASS | identical |
| serpent-depth | PASS | PASS | PASS | identical |
| sleep-spell-depth | PASS | PASS | PASS | identical |
| dpclock-wait-counts-pulses | PASS | PASS | PASS | identical |

Arm C was built in the throwaway worktree `/home/zach/dp-wait-pulses-fast` off
`44f159953`, with only the four Go harness files from `e9dd1b833` applied as an
uncommitted patch. It was never committed. The binary and exact patch are
archived; the throwaway checkout is removed after proof. Harness SHA-256:
`8584185504b6571c4967869f23ac746d0f734e27fd25eb428ea09eea8616ffe7`.
The two parallel-drain unit test packages also passed.

Reference, Arm C, aid-failure-depth:

```text
result: normalized divergence detected
--- [aid Failvict [actor]] c-oracle
+++ [aid Failvict [actor]] go-port
@@ -1,0 +1,1 @@
+You fumble and ruin the bandages.
```

The second aid actor block is empty in C while Go has the fumble line. The same
scenario passes the reference in main's 300/600ms arms. This is the revised
negative proof (R5h), independent of the discarded 150ms point.

The new nosummon vehicle also retains its old-vs-new proof: reference diverges;
candidate passes, keeps look queued through pulse 39, and releases at pulse 40.
The count comes from `src/act.other.c:1210`, with PULSE_VIOLENCE defined by
`src/structs.h:628-634` (read this round).

## Census scope — five existing C dumps changed, below the >15 stop

Reference `dp-1368-reference`:

```text
scenarios=1036 passed=1029 expected=6 unpinnable=0 stale=0 failed=0
infra=0 timed_out=0 unstable=1 elapsed=878.446s verdict=CLEAN
```

Candidate `dp-1368-candidate` (the full run started before the initial stop):

```text
scenarios=1037 passed=1027 expected=6 unpinnable=0 stale=0 failed=3
infra=0 timed_out=0 unstable=1 elapsed=876.773s verdict=NOT_CLEAN
```

The completed census-dump directories were compared byte-for-byte. Latin-1
was used solely as a lossless display representation for non-UTF-8 bytes; no
replacement decoding or extra normalization was applied. Exact changed list:

| Scenario | One-line change | Disposition |
|---|---|---|
| accuse-noarg-depth | Run-varying garbage bytes change. | Existing EXPECTED_UNSTABLE in both; ledger already names run-varying behavior. |
| bash-failure-depth | Pulse 40 now prints `You drag yourself to your feet.` in C. | PASS → FAIL; real Go state discrepancy below. |
| combat-entry-peaceful-wait | C now prints queued `Hit who?` by the second 20-pulse pump. | PASS → FAIL; Go's wait count is wrong below. |
| informative-residual-depth | User listing's displayed clock changes from 18:20:03 to 18:34:55. | PASS in both; timestamp-only difference, no wait/pump correction applies. |
| lifecycle-reconnect-linkdead | C relogin stops at Password; reconnect text and final look are absent. | PASS → FAIL; saved-character login wait needs an explicit fixture pump. |

No reference-only dumps; the sole candidate-only dump is the added
`dpclock-wait-counts-pulses`. Outcome-kind differences are exactly the three
PASS→FAIL rows plus the added PASS vehicle; elapsed seconds are not outcome
changes. See archived results-comparison.tsv.

## Stop — Go releases on a different pulse; pumps would hide the findings

### combat-entry-peaceful-wait: C 22 pulses, Go 60

The real mortal `kill` call delegates to do_hit (`src/act.offensive.c:138-140`),
which sets `WAIT_STATE(ch, PULSE_VIOLENCE + 2)` after hit returns
(`src/act.offensive.c:126-127`). This is 22 pulses, including when the peaceful
room refuses the swing. The scenario already pumps 20+20 pulses. The candidate
C releases the queued bare hit at pulse 22; its second pump contains `Hit who?`.

The live Go call is cmdKill → cmdHit (`pkg/session/combat_cmds.go:19-23`).
cmdHit calls `SetWaitState(3)` at line 130, and the setter multiplies by
PULSE_VIOLENCE (`pkg/game/player_affects.go:100-103`), storing 60 pulses.
Manager.DrainInputQueues decrements once per pulse and releases only at zero
(`pkg/session/manager.go:957-990`). This is a confirmed reachable count mismatch
(R3c/R5e), not a missing scenario pump. Adding 60 would conceal the wrong release
boundary. It requires a separate authorized Go correction, not D4 pacing.

R5c class inventory: grep of `WAIT_STATE(ch, PULSE_VIOLENCE + N)` finds eight
C sites (six +2, one +3, one +1). The same 3-round storage pattern also appears
in DoKick (`pkg/game/skill_combat.go:279,299`; C `src/act.offensive.c:633`),
DoCircle (Go lines 779,796,820; C `src/new_cmds.c:2466`), DoStrike
(`pkg/game/skill_advanced.go:240,250`; C `src/new_cmds.c:1488`), and cmdRetreat's
failure arm (`pkg/session/combat_cmds.go:361`; C `src/act.offensive.c:1052`).
These are source-level sibling candidates for the follow-up class audit, not
additional oracle-confirmed findings. The complete C grep also includes
`src/act.offensive.c:689`, `src/new_cmds2.c:187,521`; do not assume those are
wrong without tracing their own Go paths. Exact counts should use raw pulse
storage rather than replacing +2 with a whole-round multiple (R3c).

### bash-failure-depth: Go prematurely stands an already-fighting victim

C's failed bash leaves the player sitting and sets the final 40-pulse wait
(`src/act.offensive.c:483-487,496-497`). The existing scenario already pumps
20+20. C damage calls set_fighting(victim) only when !FIGHTING(victim)
(`src/fight.c:1443-1445`). A sitting combatant already in that fight stays sitting
until perform_violence's wait-expiry branch (`src/fight.c:1990-1996`;
CHECK_WAIT is wait>1 at `src/utils.h:468`), which prints the stand-up message.

Go performOneHit's standVictim checks position but omits the !FIGHTING condition
(`pkg/combat/engine.go:757-759`), and calls it on both the miss and hit paths
(lines 772 and 782). At the first mob attack, the already-fighting basher is stood
silently. The later automatic stand branch sees him fighting and cannot emit
C's wait-expiry message. This is a confirmed live path (R3b/R1/R5e).

The read-only investigation ran the unchanged scenario at the prescribed 300ms,
retained both sides, and reproduced:

```diff
 You duck under a guard trainee's fist as he takes a swing at you.
-You drag yourself to your feet.
 A guard trainee ducks under your fist as you try to hit him.
```

No extra pump can correct this state transition. No Go code was changed, and no
pin or divergence ledger entry was added. The brief requires stopping on real
Go findings; existing scenarios were not edited to mask them.

### Remaining D4 fixture correction: saved login wait

The saved-character load path explicitly sets `d->character->wait = 1`
(`src/interpreter.c:1762-1766`). This is a nanny assignment, not the post-command
wait=1 removed under DP_CLOCK. The lifecycle reconnect scenario has no pulse
between the relogin name and password, so C now keeps password queued at that
one-pulse wait. Its next look queues as well. The source-derived correction is
an explicit pulse 1 between the name and password, mirrored across both relogin
sections to keep pulse parity. This edit remains pending because the two real
Go findings trigger the stop before D4 can complete. Its citation must name the
actual wait assignment rather than invent a WAIT_STATE call (R5g).

## Evidence and continuation

Archive:
`/home/zach/Archives/darkpawns/oracle-runs/2026-09-29/dp-1368-development/`.
Key files: revised-pacing.tsv (exact arm captures), scope-comparison.txt,
results-comparison.tsv, reference-fast-aid-300ms.log,
bash-failure-candidate-investigation.log, BRIEF-with-addendum.md, and the
initial 150ms captures. The complete census manifests/summaries/dumps are in the
sibling dp-1368-reference and dp-1368-candidate directories; neither census.log
was read during this resumed work.

Old and candidate binaries remain archived. Candidate and two clean checkout
builds of oracle commit d2ace87 retain identical SHA-256:
`49a0799cd7bb107ea846bdd2768a85c100a75803fa2d9d8ded9337aa427bb76b`.
Shared reference remains:
`3200165e4e82d26a7403dd91ce544dfd8b2a23ad61a14a6220cfdc8119627916`.
Promotion and reference pin update are deferred until a clean final candidate
census is possible. The C implementation itself is unchanged during this resume.

The seam's pulse-vs-wall-clock wait limitation from DP-1202 is proven fixed by
the revised matrix and vehicle. The wider issue is not declared resolved while
these new reachable Go discrepancies block full census proof. Resume with an
explicitly amended scope for the two Go fixes; then make the source-cited login
pump correction, re-run the affected scenarios through census.sh, and complete
the final full candidate census and both PRs for PR 1 review. PR 2 must still
wait for Zach's merge and binary promotion.

Resumed verification gates passed individually: make fmt, go build ./...,
go vet ./..., go test ./... with the candidate DP_ORACLE_BIN,
golangci-lint cache clean, golangci-lint run ./..., git diff --check,
and make fidelity-depth. Only this handoff and the vehicle depth note changed.
