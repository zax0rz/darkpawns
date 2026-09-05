# Depth-fidelity handoff: `cityguard`

Date: 2026-08-28  
Branch: `glm/spec-cityguard`  
Starting main: `fc2ad0c58` (`fix: deepen janitor special procedure (#702)`)

## Frontier

Before this slice, `make fidelity-depth` reported 579 total cases, 565
proven/delegated, 3 blocked, and 11 excluded: 565/568 actionable, or 99.5%.

This slice adds seven manifest rows. The current frontier is 586 total, 569
proven/delegated, 5 blocked, and 12 excluded: 569/574 actionable, or 99.1%.

## Queue position and C inventory

`cityguard` is the next unproven special procedure after `janitor` in
`src/spec_procs.c:771-821`. A fresh inventory pass over
`src/spec_procs.c`, `src/spec_procs2.c`, and `src/spec_procs3.c`, cross-checked
against the `ASSIGNMOB` table in `src/spec_assign.c`, found these relevant
registrations in table order:

- mob 2747 at `src/spec_assign.c:212`; its prototype action flags do not carry
  `MOB_SPEC`, so `mobile_activity()` cannot dispatch cityguard for it;
- mob 18215 at `:408`; this is the first active scriptless vehicle used here;
- mobs 21200-21203 at `:493-496`, and mobs 21227-21228 at `:503-504`.

The actual C path is:

- `src/mobact.c:68-93` filters fighting or asleep mobs and invokes a registered
  special with `cmd == 0`;
- `src/interpreter.c:1407-1456` is the command-side special path, which gives
  cityguard a nonzero command and therefore trips its first `cmd` gate;
- `src/spec_procs.c:771-821` checks `cmd`/`AWAKE`, delegates a fighting mob to
  the shared `fighter()`, scans visible players for the legacy
  `PLR_FLAGS`/`PLR_OUTLAW` bit, calls `hit()`, then calls shared
  `breed_killer()`, and finally selects the lowest-aligned visible combatant
  attacking a nonnegative-alignment target with NPC topology on either side.

The Go change mirrors the reachable cityguard gates, uses the C `PLR_FLAGS`
store and `CAN_SEE` boundary, emits the exact `act()` templates, scans both
players and mobs for the protection topology, and routes `hit()` through the
canonical synchronous combat entry (`StartCombat` plus
`PerformInitialAttack`). It does not duplicate `fighter()` or
`breed_killer()`; the latter is a separate shared owner that remains queued.
No file under `src/` or `darkpawns-c-oracle/` was edited.

## Proof and blocked branches

Focused GREEN tests are:

- `TestSpecCityguard_EntryGates` — nonzero command and sleeping early returns;
- `TestSpecCityguard_OutlawVisibility` — blindness, legacy outlaw bit, exact
  room warning, and no-message fallthrough;
- `TestSpecCityguard_ProtectionSelectionAndHitBoundary` — lowest-alignment
  selection, NPC topology, exact protection warning, and the synchronous hit
  seam;
- the existing `TestSpecCityguard_Golden` now uses an NPC victim for the C
  `IS_NPC(tch) || IS_NPC(FIGHTING(tch))` condition and expects C's exact
  `You just pissed me off` wording.

The C-first vehicle is `cmd/dp-oracle-diff/scenarios/spec-proc-cityguard.txt`.
On clean main, the assigned 18215 scriptless/no-exit vehicle was RED: Go used
the old lowercase pre-rendered warning and placeholder `Attack`, while C
emitted the canonical warning and synchronous punch. After the fix, the
warning and punch opener match, but the already-shared player death/respawn
bytes still differ (`raw_kill`/death extraction wording and menu/temple
output). Two honest runs reached this same downstream seam, so
`mob.cityguard-pulse-dispatch` is blocked; this slice does not fix unrelated
combat/death behavior (R5b/R5c).

`cmd/dp-oracle-diff/scenarios/spec-proc-cityguard-breed.txt` patches an innate
`AFF_VAMPIRE` bit onto a high-HP, non-special NPC in the same disposable room.
Seeds 1 and 2 both prove the C line
`A Kir-Oshi guard exclaims, 'Die, nightbreed!!'`; Go remains silent because
cityguard does not yet reach the non-faithful shared `specBreedKiller` path.
After two honest attempts, `mob.cityguard-breed-killer` is blocked with the
future `breed_killer` owner named explicitly. The target's own aggressive
special was avoided; a separate disposable `load mob` attempt reset the C
connection and was not used as proof.

The direct `fighter()` branch is marked `excluded`: `mobile_activity()` skips
fighting mobs before special dispatch, and command dispatch fails the earlier
nonzero-`cmd` gate. This is an unreachable registered-surface branch, not a
vacuous oracle claim (R2/R4/R5e).

## Manifest rows

Added `mob.cityguard-entry-gates`, `mob.cityguard-outlaw-warning`,
`mob.cityguard-hit-boundary`, `mob.cityguard-protection-intervention`,
`mob.cityguard-breed-killer`, `mob.cityguard-pulse-dispatch`, and
`mob.cityguard-fighting-branch` to `docs/fidelity/depth/spec-procs.tsv`.

This slice follows R1 (player-facing bytes), R2 (command/autonomous dispatch
surface), R4 (no invention), R5b/R5c (audit the whole shared behavior class
and delegate rather than duplicate), and R5e (verify the actual C call path).

## Verification and next handoff

All repository gates passed on this branch:

- `make fidelity-depth` — 586 total, 569 proven/delegated, 5 blocked, 12
  excluded;
- `go build ./...`;
- `go vet ./...`;
- `go test ./...`;
- `golangci-lint run ./...` — 0 issues;
- `gofumpt -l .` — clean.

The cityguard oracle vehicles were run with
`DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle` and `--show-oracle`.
The two live RED results are retained as sharp evidence in the blocked rows;
the cityguard opener fix is limited to the confirmed divergence.

After this PR is merged only when all GitHub checks are green, refresh `main`,
rerun `make fidelity-depth`, reread `docs/fidelity/DEPTH_TESTING.md` and this
handoff, then take the next unproven special procedure in source order:
`mayor` at `src/spec_procs.c:823`. The blocked
`objmagic.sleep-entry-gates` row remains after the special-procedure inventory
and before the interpreter command-family sweep.
