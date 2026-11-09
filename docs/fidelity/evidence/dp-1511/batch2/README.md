# DP-1511 — batch 2

Historical report for merged PR #2431. [Batch 3](../batch3/README.md) follows
this handoff; the current pending ledger is maintained one directory above.

This batch clears 17 of the 52 allowances handed off by PR #2421. All 17 have
verdict **Repaired**, real C event witnesses, and before/after runs at every
original manifest seed. The current pending ledger contains **35** scenarios.
No pending allowance was replaced with a broader permanent allowance; no game
code, matcher inventory, governing document, or expected-divergence pin changed.

## Repairs and actual reach

C's set table uses `hit` and `align`, not `hp` or `alignment`
(`src/act.wizard.c:2547-2553`). Its maxhit assignment clamps to 5000, and hit
clamps to the resulting maximum (`src/act.wizard.c:2736,2748`). Remove refused
hp lines where a valid hit assignment already follows; supply actual maxhit/hit
for the short werewolf actor; use explicit 5000 bounds instead of misleading
50000/100000/500000 fixtures. Fix the align field. Remove redundant level-40
set/advance commands on the first player, who is already LVL_IMPL
(`src/db.c:3016-3019`); both C commands refuse their former inputs
(`src/act.wizard.c:1527,2897-2902`).

`audit-set-fields.py` freezes the batch base at
`4b617576753dc078fadba1efcda290bc6ace3238` and checks every warmup set field
against all 59 C field prefixes, including forced and named-sender commands.
It finds 21 invalid field sites: 14 hp, six alignment, one wimpy. This batch
clears 19. The two remaining sites are already pending in
`spell-points-spoken-depth` and `fighter-wimpy-window-depth`.
`invalid-set-fields-before.json` and `invalid-set-fields-after.json` retain the
class audit, reproducible with the script's default and `--working-tree` modes.

A clean comparison alone was not accepted as proof. Reading C output exposed
four additional missing events in the first repaired runs:

- The short werewolf vehicle emitted no seed-1 howl. It now uses level-1,
  5000-HP combatants and forty violence rounds. Every original seed
  (1,2,3,5,8) witnesses both the local howl and the distant-zone howl, plus
  the bite audiences (`src/spec_procs3.c:427-448`).
- The charm vehicle charmed its low-level trainee successfully. Set the NPC
  to level 31, which reaches the warrior-table paralysis save at this frozen
  draw (`src/magic.c:408-425`). C now prints `Your victim resists!` and the
  real retaliation's punch message (`src/spells.c:428-432`). The caster is a
  God, so this is the God-message branch and zero damage, not an ordinary
  miss or proof of mortal damage (`src/fight.c:1039-1043,1480-1481`). The
  manifest's old ordinary-miss description was corrected.
- The troll regen vehicle emitted combat glows but no idle glow in its first
  50 mobile calls. Its idle phase now spans 300 mobile calls. The retained C
  dump has 15 idle and eight combat glow lines, attributed on opposite sides
  of the explicit `hit targ` block (`src/spec_procs3.c:347-363`).
- The magic vehicle's cure-light room recipient was asleep, and smokescreen
  blinded bystanders before the cure. Wake the observer temporarily and cast
  cure before smokescreen, then return it to sleep. The same scenario still
  proves the sleeping recipient's earlier Create Food suppression and now
  visibly proves `Casttarget's wounds glow and mend themselves.` to the awake
  room observer (`src/magic.c:1764-1771`).

The other native-procedure witnesses include the mindflayer's tentacle acts,
the nightbreed hunter's growl, snake bite, both guard intervention acts, and the
elements guardian's self-harm mumble. Admin witnesses show the five wizutil log
producers and the higher-freeze-level thaw refusal. Summon still reaches its
actual victim-level cap (`src/spells.c:243-253`).

## Evidence and triage table

`triage.tsv` is the scenario → verdict → C witness table for all 17;
`original-seeds.json` records the original manifest seed sets. `witnesses.json`
retains exact C dump paths, line numbers, audience/command blocks, and additional
seed-specific magic/werewolf witnesses. `collect-witnesses.py` reproduces both
artifacts and refuses to read a run without a completed CLEAN summary.

All run names below are directories under
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-09/`. The wrapper retains the
exact frozen source inputs, dirty status, binary hash, both engines' blocks,
draw logs when enabled, and all attempts. `runs.tsv` records their final summaries.

| Run | Evidence |
|---|---|
| `sol-dp1511-b2-before-seed{1,2,3,5,8}` | All original pairs red after removing only pending allowances: 17/4/4/3/3 setup-refused failures. |
| `sol-dp1511-b2-after-seed{1,2,3,5,8}` | All 31 pairs compare clean after basic setup repairs. The absent events above mean these greens alone do not establish those claims. |
| `sol-dp1511-b2-witness-repair-seed{1,2,3,5,8}` | Charm and werewolf seed 1 clean; werewolf also clean at 2,3,5,8 with actual local/distant howls at each. |
| `sol-dp1511-b2-final-draws` | All 17 seed-1 scenarios CLEAN, zero draw divergences. Predates the final troll idle window and magic audience adjustments. |
| `sol-dp1511-b2-troll-idle-witness` | Final troll phases CLEAN, zero draw divergences, both idle and fighting regeneration witnessed. |
| `sol-dp1511-b2-magic-audience-seed{1,2,3,5,8}` | Final magic audience choreography CLEAN with zero draw divergences at all five seeds; both healing audiences witnessed at each seed. |

There are **no new Go divergences** in this batch's retained runs. Empty or
wrong-branch greens above are recorded as inadequate fixture evidence, not Go
parity defects. Batch-1 live differences remain documented separately; this
batch does not reclassify them or claim their fixes.

## Carried gate proof and checks

The command inventory and matcher are unchanged: the initial origin/main
inventory has 10,696 direct warmup inputs, 223 direct raw tokens, five force-only
command names, and 76 command-name candidates; the added matcher inventory is
312 exact entries plus 81 patterns, all 393 with actual C witnesses.

The retained full census is still
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-09/sol-dp1511-gate-batch1-r2`
(CLEAN). It predates this batch. R5h is retained in
`sol-dp1511-r5h-old-water-retained` (historical pre-#2389 setup red) and
`sol-dp1511-r5h-current-water` (real fill clean), in the same archive date.
This batch runs targeted scenarios only, per the repository's batched-release
process. Claude owns the final exact-commit combined release gate.

All eight required checks pass independently: build, vet, full Go tests, lint,
formatting, deadcode ratchet, fidelity depth, and expected-divergence checks.
No new pin or unresolved-proof-baseline expansion was introduced.

## Decisions for Zach

Have Claude review this second intermediate PR in your session, then merge.
The pending ledger falls from 52 to 35; DP-1511 stays open. After this batch
lands, continue the remaining scenarios. The Go 1.27 brief has not started.
