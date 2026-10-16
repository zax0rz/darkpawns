# DP-1371 surface-mapping handoff — 2026-10-15

## Measured baseline

The finish-line field note measured the then-current inventory at
`fd327ae1a`: 4,929 C surface units, of which 4,031 were `blocked` (82%). That
was a historical snapshot, not today's main branch. Recounting
`docs/fidelity/depth/surface-inventory.tsv` at current `origin/main`
`e3645cd5b` gives **3,604 total units: 2,706 blocked, 897 proven-already,
and 1 excluded-with-C-reason** (75.1% blocked). The counts sum from the
inventory's `units` column, grouped by `status`.

Five DP-1371 PRs from the earlier `05aaa2f3d` base remain open. Their individual
inventory deltas total 65 units. If all five changes apply cleanly to current
main, the arithmetic would be **3,539 total / 2,641 blocked**. This is a
conditional projection only: the PRs are unmerged, based on an older main
revision, and must be reconciled with current main before the projection can
become a measured result.

The before/after figures below are the blocked residual `units` on each PR's
inventory row. Classification counts describe all enumerated C output sites in
that file; `unreachable` and `proven` sites account for reductions in the
residual row.

| PR | C file | Inventory units before → after | Site classification after mapping | Oracle result |
|---|---|---:|---|---|
| [#2041](https://github.com/zax0rz/darkpawns/pull/2041) | `src/limits.c` | 22 → 3 | 19 proven, 3 gap-reachable | No census required; mapping and tests only |
| [#2042](https://github.com/zax0rz/darkpawns/pull/2042) | `src/interpreter.c` | 27 → 7 | 18 proven, 7 gap-blocked, 1 unreachable; 26 executable sites mapped | No census required; mapping and tests only |
| [#2043](https://github.com/zax0rz/darkpawns/pull/2043) | `src/clan.c` | 47 → 38 | 134 proven, 36 gap-reachable, 2 stop-tier gap-blocked, 6 unreachable | `clan-callsite-gaps`: CLEAN |
| [#2044](https://github.com/zax0rz/darkpawns/pull/2044) | `src/spell_parser.c` | 57 → 49 | 39 proven, 49 gap-reachable | Experimental target NOT_CLEAN; see evidence below |
| [#2045](https://github.com/zax0rz/darkpawns/pull/2045) | `src/act.wizard.c` | 40 → 31 | 320 proven, 25 gap-reachable, 6 stop-tier gap-blocked, 6 unreachable | `set-extended-depth`: CLEAN; show-oracle had no normalized divergence |

For #2044, the retained failed `cast-power-surface-gaps` run exposed the
independent `do_set` level-refusal punctuation difference at
`src/act.wizard.c:2899`; the experimental scenario was removed from that PR and
the census was not retried. The evidence is in
`~/Archives/darkpawns/oracle-runs/2026-10-15/dp1371-spell-parser/`, including
`summary.txt`, `results.tsv`, and `show-oracle.txt`. The final #2044 change is
test/documentation-only. The separately authorized `do_set` change and its
clean targeted census are in #2045.

The combined shared `batch-m` census is also a separate result:
`oracle-combined` ran 3,435 pairs (1,040 deduplicated), with full `NOT_CLEAN`
and claims `CLEAN`. Its retained summary is
`~/Archives/darkpawns/oracle-runs/2026-10-15/batch-m/summary.txt`. It was not
rerun.

## Largest blocked families on current main

These are the largest remaining inventory rows at `e3645cd5b`, before applying
the five open PRs. The five 299-unit use-vector rows are the 1,495-unit
spell-matrix family, which this goal defers. Other large rows include:

| Units | Family |
|---:|---|
| 299 each | `CAST_WAND`, `CAST_STAFF`, `CAST_SPELL player cast`, `CAST_SCROLL`, `CAST_POTION` use vectors |
| 204 | `new_cmds.c` skill/action output |
| 105 | `src/spells.c` act/send family |
| 97 | `fight.c` combat-round and death output |
| 82 | `lib/misc/messages` skill-message corpus |
| 80 | `new_cmds2.c` skill/action output |
| 61 | `src/magic.c` act/send family |
| 57 | `src/spell_parser.c` act/send family |
| 47 | `src/clan.c` act/send family |
| 46 | `src/mobprog.c` act/send family |
| 40 | `src/modify.c` act/send family |
| 40 | `src/act.wizard.c` act/send family |
| 34 | `src/house.c` act/send family |
| 31 | `src/zedit.c` act/send family |
| 29 | `src/dream.c` act/send family |
| 27 | `src/interpreter.c` act/send family |
| 22 | `src/shop.c` act/send family |
| 22 | `src/oedit.c` act/send family |
| 22 | `src/limits.c` act/send family |

The failed shop attempt is excluded from the PR set. Its targeted census at
`~/Archives/darkpawns/oracle-runs/2026-10-15/dp1371-shop-surface/` had FAIL rows
for `shop-buy-count-capacity` and `shop-buy-weight-sell-steal`; the latter
transcript did not reach the intended successful buy/sell branches. No PR was
opened and those failed rows were not rerun.

## Handoff state

The five mapping PRs reduce the measured units on their own base by 65 in total,
but substantial blocked work remains. The activity-call-site families and the
deferred spell-matrix and fight-skill families are not complete. No PR was
merged and nothing was deployed. Use a fresh inventory measurement after the
open PRs are reconciled and merged; do not report the conditional 3,539 / 2,641
projection as a main-branch measurement.
