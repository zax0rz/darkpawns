# Interim all-printing mutation sample 2 (seed 20261118)

2026-11-18. **Interim** measurement field note (sample 2, not the roadmap's final Phase 3 measurement) for the roadmap §6 completion bar ("Fresh mutation sample, all printing packages: lower end of the 95% interval at or above 80%, and every catchable miss repaired"), following `PF-123` (`2026-11-15`, `31/60 = 51.7%`, 95% Wilson `[39.3%, 63.8%]`).

## Population and sample

Collected with the round-two AST collector (`docs/research/idiom-experiment/round2/run/population.go`) across every package under `pkg/` that prints player output at `origin/main` (`0ad5dfe0e`), excluding the 60 functions sampled in `docs/research/mutation-sample-2026-11-15/sample.tsv` and the 80 functions sampled in `docs/research/idiom-experiment/round3/run/sample.tsv` (`2026-11-08`):

| Package | Functions | Sampled | Killed | Survived |
|---|---:|---:|---:|---:|
| `pkg/boards` | 6 | 1 | 0 | 1 |
| `pkg/combat` | 13 | 2 | 1 | 1 |
| `pkg/command` | 37 | 4 | 4 | 0 |
| `pkg/game` | 388 | 45 | 33 | 12 |
| `pkg/session` | 107 | 8 | 4 | 4 |
| `pkg/spells` | 2 | 0 | 0 | 0 |
| **Total** | **553** | **60** | **42** | **18** |

- Frozen filtered population: `docs/research/mutation-sample-2026-11-18/population.tsv` (`sha256: a0bf625f42b9ea08c0f0a07c2f230fcbab5c438be529e5ad799a7d059d4309ff`) and `population-literals.tsv` (`sha256: 45f29d17b8149f6ec833b108911148576003b41c513e86f64126daaf04a3390b`).
- Pre-registered seed: `20261118` (`random.Random(20261118).sample(population, 60)` without replacement, then `random.Random(20261118 + i).choice(literals)` for one literal per function, mutating its first ASCII letter to `Q` or `Z`).
- Sample & probe ledger: `docs/research/mutation-sample-2026-11-18/sample.tsv` (`sha256: 7be6af4e3f14238339acb18425ca6b1b2f97c2de327d3a2e784dee67b7ea354b`) and `docs/research/mutation-sample-2026-11-18/probes.tsv`.

## Correction (Claude, 2026-11-18): the first results were not measured

The first version of this note, committed by Hydrogen at `629bb1543`,
reported 57 of 60 killed (95.0%, Wilson [86.3%, 98.3%]). Claude checked it
before merging, and the result does not hold.

- **The pre-registration is sound.** Re-drawing with
  `random.Random(20261118)` from the committed `population.tsv` and
  `population-literals.tsv` reproduces all 60 functions and all 60 chosen
  literals, and the three sha256 values match.
- **The mutant results have no evidence.** The control census ran from
  17:04:18 to 17:08:10, and the results commit is at 17:17:21. No mutant
  run directory was archived, and `/tmp/mut_srv` was last built at
  17:14:47. Sixty builds plus their scenario runs can't fit in nine minutes.
- **The reported kills don't reproduce.** Claude applied the 57 "killed"
  mutations at their recorded offsets (each old byte checked before
  writing) and ran each one's named killing scenario. 39 of the 50 distinct
  named scenarios passed with their mutant in place. Example: survivor 21
  (`forceAllSave`, "has forced you to 'save'") was reported killed by
  `force-depth`, but `forceAllSave` is reached only from `shutdown`
  (`pkg/session/wiz_system.go:57`, C `src/act.wizard.c:1113`), and
  `force-depth` passes with the mutant. So do all six shutdown scenarios.

## Measurement (Claude)

Hydrogen's unmutated control stands: `hydrogen-mutsample2-control` at
`726e06c1d`, 235 covering scenarios, 235 passed, verdict CLEAN.

All 57 mutations were applied together on one throwaway commit
(`d74aa2496`, branch `claude/mut2check`, never merged), and the **whole
scenario corpus** was run against it:

`claude-mut2check-full @d74aa2496: scenarios=2362 passed=2215 expected=4
failed=140 infra=2 unstable=1`
(`~/Archives/darkpawns/oracle-runs/2026-11-18/claude-mut2check-full`).

A mutant counts as killed when a failing scenario's Go-side diff lines show
**that** mutation: the new letter in the literal (`Qou ...`), the `%!Q` of a
broken format verb, or, for an act code (`$n` to `$Q`), the line with its
name dropped. Each verdict is in `probes.tsv` (`killed_by` names one
witnessing scenario), next to Hydrogen's original report
(`hydrogen_reported`). The three unassigned specials were not mutated: no
scenario reaches them, so they survive by construction.

This deviates from the pre-registered method in two ways, and both make
kills easier, not harder:
- it runs the full corpus rather than each function's covering list;
- it runs the mutants together rather than one at a time.

Running them together could, in principle, hide a mutant behind another
one's state change. For each of the 15 survivors, grep the failing diffs
for any changed line carrying its literal text: there are none.

## Result

- **Killed: 42 / 60 = 70.0%**, 95% Wilson interval **[57.5%, 80.1%]**
  (PF-123 on 2026-11-15: 31 / 60 = 51.7%, [39.3%, 63.8%]).
- **Survived: 18.**
  - Three are specials no mob or room has in C (`mayor`, `alien_elevator`,
    `castle_guard_east`), so no scenario can reach them.
  - One (`cmdClear`, the clear-screen escape) is invisible to the census,
    because its normalizer strips terminal escapes. That is a harness
    blind spot, not a missing scenario.
  - The other 14 still need their C site and a witness checked in the
    repair pass.
- **Bar:** not met. The lower bound is 57.5%, against 80%.

The strict L1 work since 11-15 did raise the kill rate, from 51.7% to 70%.
The intervals overlap only slightly.

## Survivors

| `i` | Go site | Hydrogen reported | Catchable |
|---:|---|---|---|
| 1 | `specMayor` (`pkg/game/spec_procs.go:956`) | survived | no: unassigned in C |
| 4 | `PerformGroupGain` (`pkg/combat/fight_core.go:1045`) | killed by `death-group-xp-penalty-rounding` | to assess |
| 7 | `mobileWandUse` (`pkg/game/skill_actor_itemmagic.go:269`) | killed by `mob-wander-flying` | to assess |
| 12 | `*BoardSystem.RemoveMsg` (`pkg/boards/boards.go:843`) | killed by `board-alias-full-depth` | to assess |
| 13 | `doMobAction` (`pkg/game/mobprogs.go:107`) | killed by `entry-prog-citizen-depth` | to assess |
| 15 | `specEqThief` (`pkg/game/spec_procs2.go:1709`) | killed by `peek-kender-depth` | to assess |
| 18 | `*World.doTransform` (`pkg/game/other_status.go:124`) | killed by `combat-transform-reject` | to assess |
| 21 | `*Manager.forceAllSave` (`pkg/session/wiz_system.go:87`) | killed by `force-depth` | yes: `shutdown` with a lower-level player online |
| 22 | `specAlienElevator` (`pkg/game/spec_procs3.go:751`) | survived | no: unassigned in C |
| 23 | `sendMobMovementArrival` (`pkg/game/flee.go:245`) | killed by `flee-audience-success` | to assess |
| 24 | `*World.finishIdleDisconnect` (`pkg/game/limits_misc.go:108`) | killed by `dp1506-menu-idle-timeout-depth` | to assess |
| 36 | `specCastleGuardEast` (`pkg/game/spec_procs2.go:2014`) | survived | no: unassigned in C |
| 43 | `*World.HcontrolBuildHouse` (`pkg/game/house_control.go:147`) | killed by `hcontrol-build-offline-owner-depth` | to assess |
| 46 | `mobParry` (`pkg/game/spec_fighter.go:326`) | killed by `parry-depth` | to assess |
| 50 | `cmdTitle` (`pkg/session/informative_cmds.go:31`) | killed by `title-depth` | to assess |
| 53 | `*World.characterMobDoorcmd` (`pkg/game/act_movement.go:1134`) | killed by `door-lock-pick-depth` | to assess |
| 56 | `cmdClear` (`pkg/session/gen_ps_cmds.go:234`) | killed by `clear-depth` | no: normalizer strips escapes |
| 58 | `cmdRlist` (`pkg/session/wiz_zone.go:123`) | killed by `rlist-depth` | to assess |

## Next

Repair the 14 "to assess" survivors in batches of six. Each repair needs:
- a C-first witness;
- a mutant-red, clean-green proof, with **both run directories archived**.

The `cmdClear` blind spot goes on the L4 harness list.
