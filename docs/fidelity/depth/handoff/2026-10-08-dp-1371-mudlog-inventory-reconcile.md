# DP-1371 D7: inventory reconciliation of landed producers (PR 0)

**Tier: proof-only inventory.** Docs only — no game, harness, scenario, manifest,
oracle or census change; no census was run. Base `origin/main` `1b4b83b3f`
(merge #1854). Companion to the producer work in the follow-up trains.

## What was wrong

[`2026-10-06-dp-1371-mudlog-sites.tsv`](2026-10-06-dp-1371-mudlog-sites.tsv) was
authored before #1820, #1821, #1822, #1847 and #1854 landed. Seventeen of its
`missing` rows already have a proven producer on `main`, so the table
over-stated the remaining C sites and the "missing" label no longer named the
frontier. This train relabels exactly those rows; it adds no producer and
claims no new boundary.

## The seventeen rows

Each row keeps its `group` (the train map entry) and its `notes`; only `state`
and `proof` change. `state` moves from `missing` to `implemented-boundary`, and
`proof` names the existing `mudlog.tsv` case, proving test or oracle vehicle —
the same evidence the landing PR already carries, not a new one.

| Group | C site | Landed in | `proof` now points at |
|---|---|---|---|
| local-01 | `src/act.informative.c:1610` | #1820, #1821 | `mudlog.local-help-miss`, `mudlog.help-acting-body`; `TestMudlogLocalHelp`, `TestHelpMudlogActingBody`; `mudlog-local-depth@1,2,3,5,8`, `mudlog-help-body-depth@1,2,3,5,8` |
| local-01 | `src/clan.c:966` | #1820 | `mudlog.local-clan-withdraw`; `TestMudlogLocalClanWithdraw`; `mudlog-local-depth@1,2,3,5,8` |
| local-01 | `src/clan.c:977` | #1820 | `mudlog.local-clan-deposit`; `TestMudlogLocalClanDeposit`; `mudlog-local-depth@1,2,3,5,8` |
| local-01 | `src/oedit.c:1545` | #1820 | `mudlog.local-oedit-default`; `TestMudlogLocalOeditDefault` |
| local-01 | `src/sedit.c:1167` | #1820 | `mudlog.local-sedit-default`; `TestMudlogLocalSeditDefault` |
| game-milestones | `src/class.c:715` | #1821 | `mudlog.milestone-advance-level`; `TestMilestoneMudlogAdvance`; `mudlog-assassin-milestones-depth@1,2,3,5,8` |
| game-milestones | `src/spec_procs2.c:456` | #1821 | `mudlog.milestone-stable-failure`; `TestMilestoneMudlogStable` |
| game-milestones | `src/spec_procs2.c:833` | #1821 | `mudlog.milestone-remort`; `TestMilestoneMudlogRemort` |
| game-milestones | `src/spec_procs2.c:880` | #1821 | `mudlog.milestone-assassin-pc`; `TestMilestoneMudlogAssassinPC`; `mudlog-assassin-milestones-depth@1,2,3,5,8` |
| game-milestones | `src/spec_procs2.c:913` | #1821 | `mudlog.milestone-assassin-hire`; `TestMilestoneMudlogAssassinHire`; `mudlog-assassin-milestones-depth@1,2,3,5,8` |
| game-milestones | `src/spec_procs2.c:1573` | #1821 | `mudlog.milestone-medusa`; `TestMilestoneMudlogMedusa`; `mudlog-medusa-depth@1,2,3,5,8` |
| game-milestones | `src/fight.c:1158` | #1821 | `mudlog.milestone-attitude-loot`; `TestMilestoneMudlogLoot`; `mudlog-attitude-loot-depth@1,2,3,5,8` |
| steal | `src/act.other.c:434` | #1822 | `mudlog.steal-equipment`; `TestStealZoneMudlogEquipment`; `mudlog-steal-success-depth@1,2,3,5,8` |
| steal | `src/act.other.c:477` | #1822 | `mudlog.steal-inventory`; `TestStealZoneMudlogInventory`; `mudlog-steal-success-depth@1,2,3,5,8` |
| steal | `src/act.other.c:535` | #1822 | `mudlog.steal-failure`; `TestStealZoneMudlogFailure`; `mudlog-steal-failure-depth@1,2,3,5,8` |
| olc-zone-tail | `src/zedit.c:1213` | #1822 | `mudlog.zone-arg3-default`; `TestStealZoneMudlogZoneArg3` |
| olc-zone-tail | `src/zedit.c:1272` | #1822 | `mudlog.zone-parser-default`; `TestStealZoneMudlogZoneDefault` |

The three unit-only rows (`oedit.c:1545`, `sedit.c:1167`, `zedit.c:1213`,
`zedit.c:1272`) say so in their `proof` text: their tests inject impossible
editor/parser modes, so they are defensive unit proofs with no valid-play
reachability claim. Naming them `implemented-boundary` records that a producer
exists and is proven at its boundary — it is not a valid-play or whole-handler
claim, matching the scope the landing handoffs state.

## State counts

| State | Before | After |
|---|---|---|
| `implemented-boundary` | 72 | 89 |
| `missing` | 63 | 46 |

Deliberately untouched: the nine rows of the held-back groups
(`olc-error-seams` 2, `oracle-index` 2, `oracle-stable` 1, `whod-lifecycle` 4)
and `spec_procs2.c:634` (`kender_steal`), which is still `missing` and belongs
to the next train. 46 − 9 = **37** rows remain in the implementation frontier.

## Reproduce and control

```sh
python3 docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-sites-check.py
```

Passes on this tip: `208 retained sites; 207 active; eight LOG invocations; all
calls reconciled`.

The green can fail. Removing any row (control: the `src/act.other.c:434` line)
makes the checker exit 1 with `AssertionError: C sites and inventory differ`;
restoring the row returns exit 0. The relabel itself was applied by field
replacement and verified by diff: exactly 17 lines changed, and for each one
fields 1-8 and 11-12 are byte-identical to `HEAD` — only `state` and `proof`
differ. The inventory checker validates call expressions, site uniqueness and
field presence, so this train's `state`/`proof` text is reviewed by hand
(the table above) rather than by the script.

## Deliberate choice for review

The `group` column is left at `local-01`, `game-milestones`, `steal` and
`olc-zone-tail` rather than moved to the catch-all `landed` group. The brief
asked for `state` and `proof`; keeping the group preserves the train map and
the checker's per-group counts. If Claude prefers these rows counted with the
other 53 landed boundaries, that is a one-column follow-up.
