# DeepSeek — the engine-side DRAW classes, round 3

Date: 2026-10-15 · Branch: `deepseek/draw-round-order` (off `origin/main`
`dc9ec4da0`) · Rules: R1, R3a, R3b, R5c, R5e, R5g, R5h

Re-ran every assigned scenario on fresh `origin/main` (with #2003's combat-round
specials, the visibility `LIGHT_OK` change, and the NPC interpreter merged) before
touching anything. Only `checkload-depth` moved — it is `draw-parity MATCH` now
(round 2's #2001 landed). All six §2 rows, both §3 rows, both §5 rows, §7, and
§4 reproduce their round-2 first mismatches exactly.

## §5 — FIXED (2 rows: `shoot-mob-miss-depth`, `shoot-pc-miss-depth`)

**The chain, all cited.** C's `do_set` ends by saving a live player target:

```c
/* src/act.wizard.c:3064-3069 */
  if (!is_file && !IS_NPC(vict))
    save_char(vict, NOWHERE);

  if (is_file) {
    char_to_store(vict, &tmp_store);
```

`save_char` → `char_to_store` (src/db.c:2373), which **unequips every worn item
and equips them all again**:

```c
/* src/db.c:2634-2637 */
  for (i = 0; i < NUM_WEARS; i++)
    {
      if (char_eq[i])
    equip_char(ch, char_eq[i], i);
    }
```

and `equip_char` ends with `check_for_bad_stats(ch)` (src/handler.c:749). With
DEX zeroed — which is exactly what the set just did — the DEX arm fires:

```c
/* src/handler.c:667-671 */
  case 'd':
    send_to_char("You trip over your own feet and hit your "
        "head!\n\r",ch);
    damage(ch, ch, 40, TYPE_SUFFERING);
```

and `damage()` reaches `skill_message`, whose selector is
`nr = dice(1, fight_messages[i].number_of_attacks)` (src/fight.c:1035) — the
`Dice(1,2)` the census sees. The row's `set player Shootwindow dex 0` is a
player-target set, so the whole chain runs once per worn item.

Measured, on the warmup bisect from round 2 (replay the warmup one command
longer each run, with a temporary local draw-boundary print — reverted):

| Variant | C warmup delta | Go warmup delta | Verdict |
|---|---|---|---|
| `dex 0` (baseline) | +12 | +11 | MISMATCH |
| `dex 5` instead of `dex 0` | +11 | +11 | **MATCH** |
| command 10 removed | +10 | +10 | **MATCH** |
| `wield hoopak` / `skillset` swapped | +12 | +11 | MISMATCH |

So the extra draw needs the `set … dex 0` line **and** DEX actually reaching 0 —
the bad-stats arm, not the command's own draws (`do_set`'s field switch and
`affect_total`, src/handler.c:314-372, contain none).

**The fix.** `pkg/game/bad_stats.go` gains `ReplayEquipmentBadStats`, the
`char_to_store` re-equip walk (per worn item, in C wear order, rebuilding
`aff_abils` over each prefix) — the same walk `ReplayCrashLoadBadStats` already
used for the Crash_load path, now shared. `pkg/session/wiz_set.go` calls it
right after the acknowledgement line, before the file write, matching C's tail.

| Scenario | Before | After |
|---|---|---|
| `shoot-mob-miss-depth` | `MISMATCH stream=4136 c=Dice(1,2) = 1 go=Number(0,3) = 1 c-step=warmup go-step=warmup` | `MATCH entries=4138 stream=4141` |
| `shoot-pc-miss-depth` | `MISMATCH stream=4158 c=Dice(1,2) = 1 go=Number(0,3) = 1 c-step=warmup go-step=warmup` | `MATCH entries=4163 stream=4163` |

Both keep `result: no normalized divergence`.

**R5h.** `pkg/game/bad_stats_test.go` →
`TestReplayEquipmentBadStatsFiresPerWornItem` asserts the walk fires once per
worn item (200 → 40 HP over four items). With `pkg/session/wiz_set.go` reverted
to `origin/main` (copy-based, no `git stash`) `shoot-mob-miss-depth` returns to
its recorded first mismatch:

```
draw-parity  MISMATCH  stream=4136  c=Dice(1,2) = 1  go=Number(0,3) = 1  c-step=warmup  go-step=warmup
```

Note: a session-level unit test was attempted first and abandoned — the
`pkg/session` fixture does not reproduce the arm's damage (`DamageRefused` /
`worldRef` wiring), so the helper-level test plus the oracle row carry the proof.

## §2 — re-diagnosed: it is not the combat round order (6 rows)

All six rows still reproduce their round-2 first mismatch. The evidence now says
the first mismatch is in **`room_activity`, not combat**:

`shop-live-inventory-control` (fixture `empty-players`, `quiet-zone 80`,
`set-room-flag 8162 4 off`; warmup `goto 12133`) — the setup's draw stream ends
in one long run of `Number(0,20)`:

```
30037-30246 x210  Number(0,20)     <- one room_activity pass: loud_mobs per noise-mob
30247 x1  Dice(1,1)                <- Ananta's water arm  (C and Go agree)
30248 x1  Number(0,20)
30249 x1  Dice(1,1)                <- C ONLY
30250-30294 x45  Number(0,20)
30295 x1  Dice(1,1)                <- grannit's water arm (C and Go agree)
```

`Number(0,20)` there is `loud_mobs`' noise gate (`src/spec_procs2.c:1536`, called
from `room_activity`'s NPC arm, `src/comm.c:744`), and each `Dice(1,1)` is
`damage()`'s `skill_message` selector for a water arm. So C fires **three** arms
in that pass and Go fires **two** — the round-2 handoff's "God-vs-mob round
order" framing (GLM's DP-1438 note) describes a *later* mismatch, not this one.

What was ruled out this round:

- **The arm predicates are faithful.** `src/comm.c:710-720` vs
  `pkg/game/room_activity.go:179-209`; the affect indices match
  `src/structs.h` exactly (WATERWALK 6, FLY 26, FLAMING 31, WATERBREATHE 36).
- **`has_boat` is not the cause.** C's `has_boat` (src/act.movement.c:58-85) only
  counts a NON-wearable boat in inventory (`find_eq_pos(ch, obj, NULL) < 0`) plus
  any boat actually worn; Go counted any inventory boat. Tightening Go to C's
  rule was implemented and **changed no row** (all six §2 rows and §7 reproduce
  identically), so it is not on this branch — it is a real C-fidelity difference
  worth landing separately, but it is not this bug.
- **No character in Go's world is AFF_FLAMING** (no mob file carries the flag),
  so the extra arm is an underwater/noswim arm.
- **`quiet-mobs` bisect:** adding `quiet-mobs` to the fixture makes the scenario
  `MATCH` (`entries=4062 stream=4062`). The divergent character is reset-loaded.

Instrumenting Go's pass (`DP_ZZARM`-gated prints, reverted) localised the list
neighbourhood: Go's order is `… the naga Vasuki (4614) · the naga Ananta (4611,
sector 7) · Spandaramet (4621, sector 2) · a black bat (4546) …`, and Go fires
arms only for Ananta and grannit. C damages one more character in that
neighbourhood. So a reset-loaded mobile is *eligible* in C and not in Go — its
room/sector or its water-affect state differs, or it sits at a different list
position. Next step: dump both engines' `character_list` order and the per-mob
room/affect state around Ananta, or bisect the resets with `quiet-zone`.

## §3 — the `fight.c:1548` mount roll cannot be the source (2 rows)

The brief asked for the condition to be proven. It fails on the first term:

```c
/* src/utils.h:362 */  #define IS_MOUNTED(ch)  (IS_AFFECTED(ch, AFF_MOUNT))
/* src/fight.c:1548 */ if (!IS_NPC(victim) && IS_MOUNTED(victim) && dam > 0 && number(0,99) < 10)
```

`AFF_MOUNT` (bit 29, `src/structs.h:339`) is set in exactly one place —
`do_ride`, on both the rider and the mount (`src/act.other.c:1584-1585`) — and
removed by `unmount` (`src/utils.c:381-383`). Neither `mudlog-attitude-loot-depth`
nor `spec-proc-dragon-breath-combat` issues any ride/mount command (grep: zero
lines), and a *player* cannot be born with the bit. The victim of the extra
`damage()` is therefore not mounted, so that roll is skipped in C too.

Damage is not the blocker: the swing's `Dice(1,2)` gives `dam > 0`. The remaining
`Number(0,99)` site on that path is `mag_savingthrow` (`src/magic.c:422`), whose
`mag_damage` caller draws in the order the log shows — damage dice, then the
saving throw, then `damage()`'s `skill_message` selector. Both rows stay DRAW
rows; the next pass should trace `mag_areas` → `mag_damage` for the breath spell
and count the saving throws per victim per round.

## §7 `carve-depth` — unchanged (1 row)

`MISMATCH stream=4098 c=Number(1,20) = 9 go=Number(0,3) = 1 c-step=warmup
go-step=probe step 1: carve` — identical to round 2. C's warmup adds 8 entries
(6 hide rolls + the `hit fish` swing's `Number(1,20)` and `Dice(1,6)`), Go's 5.
The wait/drain hypothesis from round 2 is still untested; §4 (Claude's DP_CLOCK
work) may cover part of it.

