# Shoot Train 2: retaliation identity boundary

**Historical stop, resolved:** Zach approved #1796; the complete body-identity
stack merged after review, combined census and live playtest. Train 2 resumes
on `dcea4e13b`; this note preserves the original finding and decision boundary.

The implementation and six case proofs remain uncommitted in
`/home/zach/dp-p4-shoot-outcomes`, branch `sol/p4-shoot-outcomes`, base
`e33be515d`. No case is marked complete; no combined census or implementation
PR has been opened. Normal gates and targeted race checks passed before this
additional reachability control.

## Executable finding

Two actual mobs spawned from the same prototype through `World.SpawnMobQuiet`
have distinct instance IDs and equal `GetName()` values. Start combat between
the first and a real player, then invoke ranged retaliation between the second
and another real player. The new direct entry emits zero swings instead of C's
one. This is not a synthetic duplicate-player topology. Repeated prototype
loads also occur in shipped zone 163 (for example two M loads of 16303 in room
16304, `lib/world/zon/163.zon:81-83`).

Retained compiling assertion failure and executable fixture:
`~/Archives/darkpawns/oracle-runs/2026-10-05/dp-1371-shoot-outcomes-proofs/duplicate-real-body-red.txt`
and `shoot_duplicate_real_diagnostic_test.go`. To reproduce, copy that fixture
into `pkg/game/shoot_duplicate_diagnostic_test.go` and run
`go test ./pkg/game -run TestShootRealDuplicateMobRetaliationDiagnostic -count=1`.
Remove the temporary copy afterward. Earlier fixture compilation errors are
not evidence; the retained final result is an assertion failure.

## Cause and C contract

`MobInstance.GetName` returns the short description (`pkg/game/mob.go:904`).
The combat engine keys pairs by attacker/defender names, rejects any existing
attacker name in startCombat, deduplicates combatOrder by name, and performs
round/name callback resolution by name. The new after-jail enrollment attempt
therefore aborts the immediate swing on collision. Replacing the map entry or
calling StopCombat by name would damage the first mob's unrelated combat.
Continuing only the immediate swing would still fail the approved requirement
that both ensuing fighters be retained.

C `src/fight.c:208-223` enrolls actual char pointers; equal short descriptions
are irrelevant. Shoot calls the surviving actual victim's hit at
`src/act.offensive.c:955`. R1/R3/R5b/R5e/R5h require preserving the body identity
through enrollment, the immediate attack, and subsequent rounds.

## Proposed prerequisite

Recommend a separate design for making the existing combat engine's identity
consumers body-aware: pair keys and combatOrder membership, target resolution,
cleanup/StopCombat, and callbacks resolving NPC weapons, affects and protections.
Do not create a duplicate combat registry or fix only the first refusal. Audit
all readers before choosing the minimal implementation; preserve ordinary
player lookup/output and prove distinct same-description mobs fighting separate
players independently, including one killed/extracted without clearing the other.
Then finish this train with the resulting proven retaliation boundary.

This is a design decision, not an approved divergence. #1792's design says
“No synthetic players, new registry”; its narrow hit entry must also retain
both ensuing fighter memberships. The goal's 2026-10-02 train rule 5 requires
a stop for a design-first decision. Existing single-mob oracle greens do not
justify certifying this reachable topology. The completed work is preserved;
all six outcome rows remain blocked pending that decision.
