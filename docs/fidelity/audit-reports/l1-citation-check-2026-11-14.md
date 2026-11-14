# L1 citation check: the rule, and the six rows that were not uncited

2026-11-14, DeepSeek. This records the citation check behind the L1 audit so
the completion report can use one rule.

## The check

For every row in `docs/research/frozen-lists-2026-11-11/L1-uncovered-sites.tsv`,
look for the site in every `c_site` and `notes` cell of
`docs/fidelity/depth/*.tsv`:

1. match a `FILE.c:` prefix;
2. split the remainder on `,` and `;`;
3. each piece is `N` or `N-M` (inclusive) and inherits the most recent file
   prefix;
4. the site is cited if its line is inside any expanded range.

That is the frozen-list method, but comma-aware. The earlier run I reported
used a single regex that read only the first item of a comma list
(`FILE:1267,1278` registered only 1267), which made six cited rows look
uncited.

## The six, and where they are cited

| Site | Covering citation |
|---|---|
| `src/fight.c:1278` | `death.reward-milestone-output` — `src/fight.c:1267,1278` (unit-green, exact actor bytes) |
| `src/interpreter.c:2324` | `entry.menu-actions.controls` — `src/interpreter.c:1721,1942-1988,2165-2350` (unit-green; `TestEntryMenuMatrix` asserts the frozen deletion line) |
| `src/new_cmds.c:1029` | `new-cmds.smackheads-empty-args` — `new_cmds.c:975-989,1027-1031` |
| `src/oedit.c:822` | `oedit.val3-menu-lines` — `oedit.c:805,822` (line-exact pin for the frozen sites) |
| `src/shop.c:1330` | `show.shops-detail` — `shop.c:1297-1444` |
| `src/shop.c:1334` | `show.shops-detail` — `shop.c:1297-1444` |

Consequence: with the comma-aware rule, **zero non-`constants.c` rows are
uncited**. My earlier PR (#2630) reported these six as uncited; they were not,
and its "six rows outside this slice" list was the artifact. No manifest change
was needed for them.

The same rule leaves three `constants.c` rows outside every expanded range.
They are string-table entries and still need their class audit; they are not
part of this result.

## Recommendation

Use the comma-aware rule for the completion count. If the final report prefers a
stricter rule (for example, an exact line or a range endpoint rather than any
line inside a range), it should say so explicitly and re-run all lists under
that rule; a mixed rule will keep producing one-off discrepancies like this
one.
