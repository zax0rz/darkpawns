# Frozen lists, 2026-11-11 (roadmap Phase 0)

Zach accepted the Phase 0 decisions on 2026-11-11, which freezes the bar in
[`docs/plans/ROADMAP-2026-11-10-port-completion.md`](../../plans/ROADMAP-2026-11-10-port-completion.md)
(section 6 records the targets). These files are the lists as they stood that
day. From here, every port PR names the list item it removes, and an item
leaves a list only by being settled: proven (a green that can fail, R5h),
fixed, C-unreachable with the C cite, or an approved exception Zach signed.

A newly found bug joins its list with a note on how it was found. A new *kind*
of finding waits for the next release's bar, except a FAIL or draw divergence
in the final release census, which always blocks.

## The lists

| List | File | Count | Measured from |
|---|---|---|---|
| L1. Uncovered player-facing C sites | `L1-uncovered-sites.tsv` | 739 | see below |
| L2. Known divergences | `L2-L4-L5-linear.tsv` (list `L2`) | 58 | open Linear issues, 2026-11-11 |
| L3. Blocked proof units | `L3-blocked-units.tsv` | 62 | 39 depth rows, plus 23 surface-inventory rows (7 activity call-site families, 4 spell-matrix vectors, 2 fight-skill, 7 lifecycle, 3 shop), on `122615de3` |
| L4. Harness blind spots | `L2-L4-L5-linear.tsv` (list `L4`) | 11 | open Linear issues |
| L5. Duplicate and invented Go code | `L2-L4-L5-linear.tsv` (list `L5`) | 5 known | open Linear issues; the function inventory (Phase 2) will add to it |
| L6. Unverified research claims | `L6-unverified-claims.tsv` | 14 | `EVIDENCE_LEDGER.tsv` rows in state `needs-verification` |

### How L1 was measured

`dp-census-coverage` on the newest full census with a dump,
`~/Archives/darkpawns/oracle-runs/2026-11-10/go127-batch5-reds-combined`
(Go `3d14ccab6`, combined verdict CLEAN): 1,247 C output segments uncovered,
3,650 covered, 1,336 unverifiable. From the 1,247, any site whose line falls
inside a `c_site` or `notes` citation in a depth manifest on `122615de3` was
removed as already classified, leaving 739.

Limits, stated so nobody reads more into the number than it holds:

- Witness batches merged on 11-10 and 11-11 aren't in that dump, so some of
  the 739 are already witnessed. The next full census settles them.
- The 1,336 unverifiable sites (format strings the tool can't match) aren't in
  L1. The completion report must list them as not measured.
- 130 rows are in `constants.c` (string tables). Each still needs a class.
- The citation match is by line number, so a site cited by a neighbouring
  range is counted as classified. DeepSeek's per-file audit can disagree by a
  row or two (it reports `clan.c` at 0; this method finds 1).

### How L2, L4 and L5 were split

Claude classified the open Linear issues by title on 2026-11-11. Flaky tests,
site, client, agent-layer and product issues aren't on any list. Zach can move
an issue between lists. Four L2 rows are rulings that only Zach can settle
(DP-1409, DP-1399, DP-1379, DP-1199). DP-1495 may already be fixed by #2474,
and DP-1348/DP-1361 and DP-1376/DP-1394 are pairs describing one defect each.

## Decisions taken in Phase 0 (2026-11-11)

1. **Definition of done:** the bounded claim in roadmap section 1.
2. **Targets:** the section 6 table as suggested. Mutation bar: the lower end
   of the 95% interval at or above 80%, and every catchable miss repaired.
   Random play: 24 hours with no new divergence.
3. **Unit tests as proof:** an oracle witness is the default. A unit test
   counts only where the oracle cannot reach the site (mid-combat states,
   DP-1202, and long-shot random rolls), and it still needs a C cite and a
   revert proof. The completion report shows unit-proven sites in their own
   column.
4. **Long-shot random rows** (such as `mob.roach-starvation-site-blocked`,
   about one in 10^8 per tick) are settled by one standing exception for the
   class, each row naming its odds and the C line.
5. **New kinds of findings** join the next release's bar.
6. **Tag:** `v1.0.0`.
