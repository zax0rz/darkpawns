# DP-1371 surface-mapping handoff — 2026-10-15

## Latest measured continuation

The current `origin/main` snapshot is `d48aec241` (2026-10-15): **3,360
inventory units, 2,462 blocked, 897 proven-already, and 1 excluded with a C
reason**. The activity-call-site subtotal is 1,385 units, 491 blocked. This is
the measured main-branch state; the PR deltas below are still drafts.

| Open PR | C file | Inventory units before → after | Site classifications after |
|---|---|---:|---|
| [#2050](https://github.com/zax0rz/darkpawns/pull/2050) | `src/shop.c` | 22 → 0 | 24 proven, 0 gaps |
| [#2051](https://github.com/zax0rz/darkpawns/pull/2051) | `src/whod.c` | 10 → 0 | 10 proven, 0 gaps |
| [#2053](https://github.com/zax0rz/darkpawns/pull/2053) | `src/spells.c` | 104 → 97 | 32 proven, 97 gap-reachable, 4 unreachable |
| [#2054](https://github.com/zax0rz/darkpawns/pull/2054) | `src/magic.c` | 39 → 38 | 42 proven, 38 gap-reachable, 3 unreachable |
| [#2055](https://github.com/zax0rz/darkpawns/pull/2055) | `src/gate.c` | 7 → 0 | 8 proven, 0 gaps |
| [#2056](https://github.com/zax0rz/darkpawns/pull/2056) | `src/act.wizard.c` | 31 → 31 | 320 proven, 25 gap-reachable, 6 stop-tier blocked, 6 unreachable |
| [#2057](https://github.com/zax0rz/darkpawns/pull/2057) | `src/oc.c`, `src/scripts.c` | 2 → 0 each | `oc.c`: 2 proven; `scripts.c`: 1 live site proven, line 87 is a declaration |

Applying these seven draft changes together would project **3,309 total units,
2,411 blocked**, including **1,334 activity-call-site units, 440 blocked**.
That projection is not a measurement of main; the PRs must be reconciled and
merged before the inventory is recounted. The wizard change adds clean
five-seed oracle transcripts for eight branches that already had unit proofs,
so it strengthens evidence without reducing the measured residual. The
`scripts.c` audit removes one stale count that treated the local `act`
declaration as a call site.

The historical finish-line baseline was 4,929 total units and 4,031 blocked at
`fd327ae1a` (2026-09-04). Compared with that snapshot, current `origin/main`
has 1,569 fewer units and 1,569 fewer blocked units. The seven open PRs
would bring that cumulative reduction to 1,620 if all land.

Targeted census results for this continuation: `dp1371-gate-collision-spacing-rebased`
and `dp1371-gate-portal-final` are CLEAN; `dp1371-set-extended-output-gaps`
is CLEAN across five seeds; `dp1371-spell-summon-rebased` is CLEAN (4/4);
`dp1371-whod-rebased` is CLEAN (1/1). The `oc.c` and `scripts.c` changes are
documentation-only mappings and require no census.
The old `gate-spell-depth` divergence is retained at `e3645cd5b`, investigated
as the pre-fix extra CRLF from `roomMessage`, and not retried. Claude owns the
final full census; this work uses targeted censuses only.

## Historical measured baseline (e364 checkpoint)

The finish-line field note measured the then-current inventory at
`fd327ae1a`: 4,929 C surface units, of which 4,031 were `blocked` (82%). That
was a historical snapshot, not today's main branch. Recounting
`docs/fidelity/depth/surface-inventory.tsv` at current `origin/main`
`e3645cd5b` gives **3,604 total units: 2,706 blocked, 897 proven-already,
and 1 excluded-with-C-reason** (75.1% blocked). The counts sum from the
inventory's `units` column, grouped by `status`.

Five earlier DP-1371 PRs (#2041–#2045) from `05aaa2f3d` reduce their inventory
rows by 65 units in aggregate. Two more PRs from measured current main
`e3645cd5b` reduce the `spells.c` row by one (#2047) and the `oedit.c` row by
ten and the `olc.c` row by twenty (#2048). If all seven changes apply cleanly,
the arithmetic would be **3,508 total / 2,610 blocked**. This is a conditional
projection only: none
has merged, the first five use an older main revision, and all changes must be
reconciled before the projection can become a measured result.

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
| [#2047](https://github.com/zax0rz/darkpawns/pull/2047) | `src/spells.c` | 105 → 104 | 25 proven, 104 gap-reachable, 4 unreachable | No census required; proof and inventory update only |
| [#2048](https://github.com/zax0rz/darkpawns/pull/2048) | `src/oedit.c`, `src/olc.c` | OEDIT 22 → 12; OLC 21 → 1 | OEDIT: 60 proven, 12 gap-reachable; OLC: 20 proven, 1 gap-blocked | No census required; tests, maps, and inventory updates only |

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

## Largest blocked families at the e364 checkpoint

These are the largest remaining inventory rows at `e3645cd5b`, before applying
the seven open PRs. The five 299-unit use-vector rows are the 1,495-unit
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

PR #2047 proves one of the 105 `spells.c` gaps from current main. It leaves the
other 104 reachable sites open; its updated row is not part of the measured
current-main count until merge.

PR #2048 proves ten of the 22 `oedit.c` gaps and maps all 21 `olc.c` output
sites from current main. OEDIT retains 12 reachable sites; OLC retains one
session-lifecycle/transport gap. Neither updated row is part of the measured
current-main count until merge.

## Historical handoff state at e364

The seven PRs account for 96 fewer blocked units after applying their per-row
changes, but substantial blocked work remains. The activity-call-site families
and the deferred spell-matrix and fight-skill families are not complete. No PR
was merged and nothing was deployed. Use a fresh inventory measurement after
the open PRs are reconciled and merged; do not report the conditional 3,508 /
2,610 projection as a main-branch measurement.
