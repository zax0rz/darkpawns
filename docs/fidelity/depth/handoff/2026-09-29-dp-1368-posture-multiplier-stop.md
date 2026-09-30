# DP-1368 — Addendum 2 landed; stop on posture arithmetic

PR 1 is withheld. The authorized four changes are committed in order, with
reversion-tested units. Additional validation of the existing five-seed bash
proof exposes an independent legacy formula change, outside Addendum 2's
allowed Go sites. The brief explicitly limits the Go exception to "these items only"; the damage arithmetic is a separate fix. No damage formula fix, scenario tuning, divergence ledger
or divergence pin was added. The shared reference binary was not promoted.

## Authorized changes and proof

| Item | Commit | Proof |
|---|---|---|
| Hit wait | 69ba6b284 | stores22 pulses; queued command absent through21, executes on22, no duplicate on23; former60-pulse setter fails unit; unedited combat-entry-peaceful-wait CLEAN |
| +N siblings | 3a1d3582a | kick/dragon kick/strike/circle/retreat failure22 raw pulses; first-aid success23; each repaired branch fails on literal reversion; already-correct aid failure23/detect failure21 untouched |
| Victim stand gate | 32b5083e6 | hit/miss preserve existing fighters' posture; pending initial enrollment stands fresh victims once; both gate mutations fail respective unit halves; unedited bash seed1 green with stand-up line |
| Reconnect fixture + D5 SHA | befcb2a57 | nanny wait=1 pumped before password in both sections, citing interpreter.c:1766; fixture unit fails without pump; targeted lifecycle-reconnect-linkdead CLEAN |

The main branch remains sol/wait-pulses at /home/zach/dp-wait-pulses.
Oracle branch remains sol/wait-pulses at /home/zach/dp-oracle-wait,
commit d2ace874868f34315280bb3a33b68a53d322de4b.

The full +N scope census had only the known bash and reconnect content failures;
hit changed FAIL→PASS. A poofin-depth INFRA (divergence did not reproduce) was
rechecked alone through census.sh and passed. The wrapper skips its automatic
INFRA recheck when content FAILs exist; no FAIL row was retried. Comparing
candidate C dumps before/after the +N changes found only existing accuse UB
bytes and informative wall-clock timestamp drift, no additional wait change.

## New confirmed defect: fractional posture damage violates R1/R4

Unedited bash-failure-depth on candidate passes seeds1,2,3,5; seed8 diverges:

```diff
-A guard trainee barely hits you.
+A guard trainee hits you.
 You drag yourself to your feet.
 A guard trainee ducks under your fist as you try to hit him.
```

C src/fight.c:1854-1855 executes integer division:

```c
if (GET_POS(victim) < POS_FIGHTING)
    dam *= 1 + (POS_FIGHTING - GET_POS(victim)) / 3;
```

Go pkg/combat/formulas.go:532-541 deliberately replaced it with float math,
with the explicit comment that this would "restore developer intent":

```go
dam = int(float64(dam) * (1.0 + float64(PosFighting-defPos)/3.0))
```

R1/R4 require the executed C arithmetic, including integer truncation, rather
than the C comments' fractional examples. R5e call path: performOneHit calls
CalculateDamage before the damage-point stand transition, so a sitting victim
is read as sitting. Keeping an already-fighting basher sitting correctly now
exposes this independent arithmetic mismatch on later hits. Git blame traces
the intentional float change and comment to 0210c7ff9a (DP-515,2026-05-27).

Control, same scenario/seed8 and pre-stand-fix Go build at3a1d3582a:

| C binary | Go build | Result |
|---|---|---|
| old reference | pre-stand-fix | PASS |
| candidate | pre-stand-fix | FAIL: damage label matches, C stand-up line missing |
| candidate | corrected stand gate | FAIL: stand-up line matches, damage label differs |

This is a masking interaction, not evidence to revert the correct stand gate.
The source-unit probe in a throwaway worktree isolates the whole posture class:
base damage12, neutral AC100, fixed1d1+11 mob damage, strength10.

| Victim posture | C integer result | Current Go result |
|---|---:|---:|
| standing | 12 | 12 |
| sitting | 12 | 16 |
| resting | 12 | 20 |
| sleeping | 24 | 24 |
| stunned | 24 | 27 |
| incapacitated | 24 | 32 |
| mortally wounded | 36 | 36 |

The existing TestCalculateDamage_VictimPositionMultiplier tests only sleeping,
where both formulas agree. The new source probe fails the four divergent
postures. It is retained in evidence, not added to main as a failing test.
R5c audit found the melee CalculateDamage expression as the floating site;
DoCircle's duplicated position multiplier already uses integer arithmetic
(pkg/game/skill_combat.go:805-806) and was not changed. The temporary control
worktree was removed, with both binaries' transcript controls retained.

A further scope amendment is needed for the posture-arithmetic class, a
reversion-tested exact-integer unit over all reachable postures, seed8 bash
proof, and a new final full scope census. This report does not implement it.

## Completed seam proof and retained evidence

The nine-case revised pacing matrix passes with fixed Go at main300ms,
main600ms and parked fast300ms; normalized C blocks are byte-identical across
all27 runs. Old-binary fast300ms aid fails on its second actor response;
old-binary vehicle fails and candidate vehicle passes. The150ms logs remain
retained under Addendum1's next-loop output-latency explanation. The fast
worktree was removed and no parked harness code was committed.

Both independent clean C builds at d2ace874 use
`-g -O0 -fcommon -ffile-prefix-map=$PWD=.` and match candidate SHA256:
49a0799cd7bb107ea846bdd2768a85c100a75803fa2d9d8ded9337aa427bb76b.
Old shared reference SHA256:
3200165e4e82d26a7403dd91ce544dfd8b2a23ad61a14a6220cfdc8119627916.
The shared checkout remains a4c769213cb6542ee26591975836632860233da3.

Evidence root:
~/Archives/darkpawns/oracle-runs/2026-09-29/dp-1368-development/.
Key artifacts: additive-literal-revert.log, stand-*-reverted.log,
reconnect-fixture-{reverted,green}.log, final-matrix.tsv,
bash-multiseed.tsv, bash-candidate-seed-8.log,
bash-seed-8-pre-stand-{candidate,reference}.log,
posture_source_test.go, posture-source-unit-red.log.
Candidate/old binaries and SHA256SUMS are also retained in final-candidate's
promotion bundle, without promotion. Prior stop reports remain historical.

## Final candidate census (seed1)

```
oracle-regression: scenarios=1037 passed=1030 expected=6 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=1 elapsed=872.577s started=2026-09-29T19:21:21-0400 finished=2026-09-29T19:35:53-0400 verdict=CLEAN
```

Run: ~/Archives/darkpawns/oracle-runs/2026-09-29/dp-1368-final-candidate/.
Captured Go HEAD befcb2a57. Outcome-kind comparison (excluding elapsed-time
columns) against dp-1368-reference differs only by the added PASS vehicle
row, dpclock-wait-counts-pulses. No scenario outside the authorized scope
changed verdict. Final C dumps have four changed common scenarios: bash's
stand-up line, hit's queued "Hit who?", existing accuse-noarg-depth UB bytes,
and informative-residual-depth's wall timestamp. The new vehicle is the
only added dump. Reconnect output is restored to the original reference
shape by the source-exact fixture pump. All these artifacts are retained.

make fmt, go build ./..., go vet ./..., go test ./... (candidate
DP_ORACLE_BIN), golangci-lint cache clean, golangci-lint run ./...,
git diff --check, and make fidelity-depth pass. A CLEAN seed1 run does not
make the known five-seed bash proof green. The historical five-seed manifest
row now explicitly notes this revalidation failure, without changing its
status or inventing a divergence-ledger row.


No PRs are opened. The old-reference full census on final Go remains pending,
as does the review-ready PR1 proof; a seed1 green cannot discharge seed8's
confirmed R1/R4 finding.
