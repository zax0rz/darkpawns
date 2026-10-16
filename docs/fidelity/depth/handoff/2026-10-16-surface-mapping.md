# Surface mapping handoff — 2026-10-16

## Measured baseline and projection

Current `origin/main` is `d48aec241a8e4ba61d8f61ab651095e9dbe45305`. Its surface inventory sums to **3,360 units**: **2,462 blocked**, **897 proven-already**, and **1 excluded-with-C-reason**. The `activity-call-sites` rows account for **1,385 units**: **491 blocked** and **894 proven-already**.

The five draft PRs recorded below are based on that same main revision. Their per-file residual reductions total 57 units. If all five changes apply, the conditional projection is **3,303 total units / 2,405 blocked** overall and **1,328 total / 434 blocked** for `activity-call-sites`. The projection is not a measurement of main; none of the five PRs has merged.

| PR | C file | Inventory units before → after | Site classifications before → after |
|---|---|---:|---|
| [#2050](https://github.com/zax0rz/darkpawns/pull/2050) | `src/shop.c` | 22 → 0 | Not mapped → 24 proven, 0 gap-reachable, 0 gap-blocked, 0 unreachable |
| [#2053](https://github.com/zax0rz/darkpawns/pull/2053) | `src/spells.c` | 104 → 91 | 25 proven, 104 gap-reachable, 0 gap-blocked, 4 unreachable → 38 proven, 91 gap-reachable, 0 gap-blocked, 4 unreachable |
| [#2056](https://github.com/zax0rz/darkpawns/pull/2056) | `src/act.wizard.c` | 31 → 27 | 320 proven, 25 gap-reachable, 6 stop-tier gap-blocked, 6 unreachable → 324 proven, 21 gap-reachable, 6 stop-tier gap-blocked, 6 unreachable |
| [#2058](https://github.com/zax0rz/darkpawns/pull/2058) | `src/clan.c` | 38 → 26 | 134 proven, 36 gap-reachable, 2 gap-blocked, 6 unreachable → 144 proven, 24 gap-reachable, 2 gap-blocked, 8 unreachable |
| [#2059](https://github.com/zax0rz/darkpawns/pull/2059) | `src/house.c` | 7 → 1 | 32 proven, 7 gap-reachable, 0 gap-blocked, 2 unreachable → 38 proven, 1 gap-reachable, 0 gap-blocked, 2 unreachable |

For `shop.c`, the prior row had 22 residual inventory units but no per-site map; PR #2050 enumerates all 24 live sends and maps them as proven. For `house.c`, the remaining gap is the `MAX_HOUSES` capacity refusal.

## Oracle and gate evidence

- **#2050 shop:** the initial social-target mismatch is retained; `dp1371-shop-steal-room-order` is CLEAN (1/1) at `e0a12e5b7`. The corrected capacity and stack-list probes also pass. All ten repository gates passed individually.
- **#2053 spells:** `dp1371-spell-summon-rebased` is CLEAN (4/4) at `cf320445d`; `dp1371-spell-identify-character-corrected` and `dp1371-spell-identify-scroll` are CLEAN (1/1 each) at `3aa6fe6c9`; `dp1371-spell-teleport-self-only` is CLEAN (1/1) at `c572921f1`. The first identify-character run exposed Go's extra “You feel informed” character preamble; its retained show-oracle diff motivated the correction. Mindsight's immortal-victim recoil is pinned by the exact-output D3 unit case `spell.mindsight-immortal-recoil`. All ten gates passed individually after the exact mindsight-output assertion; `make fidelity-depth` passed after the map and manifest updates.
- **#2056 wizard:** the first stat-report run exposed room-list order and a nondefault attack-label mismatch; both are documented and retained. Corrected `dp1371-wizard-stat-nondefault` and `dp1371-wizard-transfer-refusals` are CLEAN (1/1 each) at `eda7e8185`. All ten gates passed individually.
- **#2058 clan:** `dp1371-clan-bank-title` is CLEAN (3/3) at `a5be90b28` for `clan-callsite-gaps`, `clan-depth`, and `clan-member-depth`. All ten gates passed individually.
- **#2059 house:** `dp1371-house-control-gaps` is CLEAN (1/1) at `415aed140` for `hcontrol-depth`; `make fidelity-depth` passed with 36 maps and 0 errors.

All five PRs remain draft and open. Nothing was merged or deployed.

## Remaining activity-call-site work

After the five draft changes, the largest activity-call-site residual rows are `src/spells.c` 91, `src/spell_parser.c` 49, `src/magic.c` 39, `src/zedit.c` 31, `src/dream.c` 29, `src/act.wizard.c` 27, `src/clan.c` 26, and `src/spec_procs.c` 20. The four `src/new_cmds.c` residual sites are mapped as stop-tier blocked. The `src/spec_procs.c` residual sites are gap-blocked by the approved protected-name divergence. Continue with reachable output gaps and retain the same source-order/C-call-path evidence (R1/R2/R4/R5e/R5g/R5h).
