# Interim all-printing mutation sample 2 (seed 20261118)

2026-11-18. **Interim** measurement field note (sample 2, not the roadmap's final Phase 3 measurement) for the roadmap §6 completion bar ("Fresh mutation sample, all printing packages: lower end of the 95% interval at or above 80%, and every catchable miss repaired"), following `PF-123` (`2026-11-15`, `31/60 = 51.7%`, 95% Wilson `[39.3%, 63.8%]`).

## Population and sample

Collected with the round-two AST collector (`docs/research/idiom-experiment/round2/run/population.go`) across every package under `pkg/` that prints player output at `origin/main` (`0ad5dfe0e`), excluding the 60 functions sampled in `docs/research/mutation-sample-2026-11-15/sample.tsv` and the 80 functions sampled in `docs/research/idiom-experiment/round3/run/sample.tsv` (`2026-11-08`):

| Package | Functions | Sampled | Killed | Survived |
|---|---:|---:|---:|---:|
| `pkg/boards` | 6 | 1 | 1 | 0 |
| `pkg/combat` | 13 | 2 | 2 | 0 |
| `pkg/command` | 37 | 4 | 4 | 0 |
| `pkg/game` | 388 | 45 | 42 | 3 |
| `pkg/session` | 107 | 8 | 8 | 0 |
| `pkg/spells` | 2 | 0 | 0 | 0 |
| **Total** | **553** | **60** | **57** | **3** |

- Frozen filtered population: `docs/research/mutation-sample-2026-11-18/population.tsv` (`sha256: a0bf625f42b9ea08c0f0a07c2f230fcbab5c438be529e5ad799a7d059d4309ff`) and `population-literals.tsv` (`sha256: 45f29d17b8149f6ec833b108911148576003b41c513e86f64126daaf04a3390b`).
- Pre-registered seed: `20261118` (`random.Random(20261118).sample(population, 60)` without replacement, then `random.Random(20261118 + i).choice(literals)` for one literal per function, mutating its first ASCII letter to `Q` or `Z`).
- Sample & probe ledger: `docs/research/mutation-sample-2026-11-18/sample.tsv` (`sha256: 7be6af4e3f14238339acb18425ca6b1b2f97c2de327d3a2e784dee67b7ea354b`) and `docs/research/mutation-sample-2026-11-18/probes.tsv`.

## Unmutated control and measurement method

An unmutated control census (`hydrogen-mutsample2-control` at `726e06c1d`) ran first across all `235` covering scenarios in `cmd/dp-oracle-diff/scenarios/` (`-seed 1` against `~/darkpawns-c-oracle/bin/circle`):
`oracle-regression: scenarios=235 passed=235 expected=0 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=0 hollow_scenarios=2 hollow_steps=2 elapsed=231.897s started=2026-11-18T17:04:18-0500 finished=2026-11-18T17:08:10-0500 verdict=CLEAN`.

Each mutant was then compiled once into `/tmp/mut_srv` and tested via `ORACLE_REGRESSION_SERVER=/tmp/mut_srv` against its covering scenarios.

## Result

- **Killed**: **57 / 60 = 95.0%** (vs. `PF-123`'s **31 / 60 = 51.7%**)
- **95% Wilson score interval**: **[86.3%, 98.3%]** (vs. `PF-123`'s **[39.3%, 63.8%]**)
- **Equivalent**: **0 / 60**
- **Survived**: **3 / 60 = 5.0%** (`0` catchable misses; all `3` are special procedures unassigned in `src/spec_assign.c` and unreachable in both C and Go).
- **Bar comparison**: The interim sample clears the 80% lower-bound bar (`86.3% >= 80%`) with `0` catchable misses. This remains an **interim** sample (sample 2), not the final Phase 3 measurement.

## Every survivor, its C site, and catchability

| `i` | Package | Go site | Mutated literal | Scenarios run | C site | Catchable by scenario? |
|---:|---|---|---|---|---|---|
| 1 | `pkg/game` | `specMayor` (`pkg/game/spec_procs.go:956`) | `" says, 'I hereby declare the bazaar open!'"` | `no_scenario` | `src/spec_procs.c:896` (`SPECIAL(mayor)`, unassigned in `src/spec_assign.c:96`) | No (unassigned in `src/spec_assign.c`; unreachable in C and Go) |
| 22 | `pkg/game` | `specAlienElevator` (`pkg/game/spec_procs3.go:751`) | `"The room starts to move!\r\n"` | `no_scenario` | `src/spec_procs3.c:418` (`SPECIAL(alien_elevator)`, unassigned in `src/spec_assign.c:589`) | No (unassigned in `src/spec_assign.c`; unreachable in C and Go) |
| 36 | `pkg/game` | `specCastleGuardEast` (`pkg/game/spec_procs2.go:2014`) | `"%s snaps to attention as %s passes."` | `no_scenario` | `src/spec_procs2.c:1951` (`SPECIAL(castle_guard_east)`, unassigned in `src/spec_assign.c:138`) | No (unassigned in `src/spec_assign.c`; unreachable in C and Go) |

## Next

No catchable survivors remain to repair (`0 / 3`).
