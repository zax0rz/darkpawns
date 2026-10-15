# DeepSeek — the engine-side DRAW classes (draw parity, round 2)

Date: 2026-10-15 · Branch: `deepseek/draw-parity-engine` (off `origin/main`
`d62179115`, which carries #1987–#1990) · Rules: R3a, R3b, R5c, R5e, R5g, R5h

Seed-1 DRAW rows at the start of this round: 18. This branch clears **1** (§6,
`checkload-depth`, a complete R3a restoration). The other five classes are
diagnosed below with the draw-boundary and draw-log evidence gathered this
session; §4 is stop tier (DP_CLOCK pulse accounting).

Method: serial reproduction per scenario with
`DP_ORACLE_BIN=$HOME/darkpawns-c-oracle/bin/circle go run ./cmd/dp-oracle-diff
--scenario <name> -draw-parity`, plus (a) a temporary local print of the
phase/step draw boundaries at the top of `reportDrawParity` and (b) for the
warmup class, a generated-scenario bisect that replays the warmup one command
longer each run. Both were local edits, reverted before committing — the tree
carries no harness change.

## §6 `checkload` — FIXED (1 row: `checkload-depth`)

**The finding.** C's `check_load` does not read prototypes; it reads *instances*
and keeps two fields:

```c
/* do_checkload (src/act.wizard.c:3819-3863) */
    mob = read_mobile(which, REAL);            /* :3841 */
    char_to_room(mob, 0);
    strcpy(buf3, GET_NAME(mob));               /* GET_NAME is short_descr for an NPC, src/utils.h:275-276 */
    extract_char(mob);                         /* :3844 */
...
    obj = read_object(which, REAL);            /* :3853 */
    strcpy(buf3, obj->short_description);
    extract_obj(obj);
```

and `check_load` (`src/act.wizard.c:3681-3817`) reads a throwaway object for
every zone `O`/`E`/`G` command (`:3729-3732`) and a throwaway mobile for every
zone `M` command (`:3750-3753`) — inside the `else if (type == 'o' || type == 'O')`
block, which is why the `mob` arm draws nothing per command and the `obj` arm
draws thousands of times.

Those reads are not free. `read_mobile` rolls the hit dice and the gold variance
(`src/db.c:1748-1773`); `read_object` applies the `ITEM_RARE` variance, one
`dice(1,100)` per affected slot (`src/db.c:1899-1925`). Go reported from
prototypes (`GetMobPrototype` / `GetObjPrototype` / `obj.LoadPercent`), so it drew
none of them.

Measured on the row: `checkload mob 16303` added **4** entries in C and **1** in
Go (the hide roll only) — `Dice(1,5)` (mob 16303 is `1d5+20`, `lib/world/mob/163.mob`)
plus `Number(0,1)` and `Number(1,20)` for its 100 gold. The 12 `checkload obj`
probe steps were ~5,566 draws short each.

**The fix.** `pkg/game/world_zone.go` gains `ReadDisposableMob` and
`ReadDisposableObject` — C's `read_mobile(nr, REAL)` / `read_object(nr, REAL)`
draw sequences, returning an unregistered instance exactly as C's
`extract_char` / `extract_obj` discard theirs. `pkg/session/wiz_info.go` calls
them from `cmdCheckload` (both arms) and from `checkloadObjectReport`'s per-command
`O`/`E`/`G` and `M` arms, in C's zone-command order. The values that reach the
player are unchanged (`GET_NAME` is the short description; `GET_OBJ_LOAD` is a
prototype field), so the bytes stay clean.

**Proof.** `checkload-depth`: before
`draw-parity MISMATCH stream=65101 c=Dice(1,5) = 3 go=Number(0,3) = 1 c-step=probe step 7: checkload mob 16303 go-step=probe step 8: checkload M 16303 trailing`;
after `draw-parity MATCH entries=98410 stream=540708`, `result: no normalized
divergence`.

R5h: `pkg/session/wiz_info_checkload_test.go` →
`TestCheckloadCommandReadsThrowawayInstances` asserts the exact draws of both
arms; `pkg/game/world_zone_read_test.go` asserts the helpers' draw counts and
values (including that a missing vnum and a non-rare object draw nothing). With
`pkg/session/wiz_info.go` reverted to `origin/main` (copy-based revert, no
`git stash`) the command test fails:

```
stream after `checkload mob 200` = 3031993394, want 1004773200 (one mobile read)
stream after `checkload obj 300` = 3031993394, want 2697711130 (three reads)
```

## §2 God-vs-mob round order — 6 rows, needs engine work

`shop-live-inventory-control`, `shop-live-inventory-filters`,
`shop-live-inventory-gate`, `shop-stack-list-live`, `show-shops-detail`,
`mudlog-crash-load-entry`.

First mismatch `c=Dice(1,1) = 1  go=Number(0,20) = 9` at stream 62467 (62382 for
`-filters`/`-gate`, 81529 for `mudlog-crash-load-entry`).
`shop-live-inventory-control` boundaries: **C setup 30334, Go setup 30332** — Go
is two draws short by the end of setup, and the first divergence is inside it:

```
c   30244 Number(0,20)=14  30245 =4  30246 Dice(1,1)=1  30247 Number(0,20)=7  > 30248 Dice(1,1)=1
go  30244 Number(0,20)=14  30245 =4  30246 Dice(1,1)=1  30247 Number(0,20)=7  > 30248 Number(0,20)=9
```

Both sides are inside the same long setup fight and every value agrees through
entry 30247. `Number(0,20)` is a level-60 PC's bare-hand roll
(`number(0, GET_LEVEL(ch)/3)`); `Dice(1,1)` is `skill_message`'s selector,
`nr = dice(1, fight_messages[i].number_of_attacks)` (`src/fight.c:1035`), for a
message set with `number_of_attacks == 1` — the position-degraded mob's swing. So
C's round processes a different combatant at that point than Go's. GLM's DP-1438
handoff filed this as the God-vs-mob round order and confirmed the enrolment-point
change does not move it; `PerformRound` already walks `combatOrder` newest-first
(matching `set_fighting`'s prepend, `src/fight.c:207-226`), so the remaining
difference is in what is *on* that list and in what order at this point in the
fight. GLM is changing the per-round `MOB_SPEC` call
(`pkg/combat/engine.go:624`) this round — coordinate before touching
`PerformRound`.

## §3 one missing saving throw inside a long pulse — 2 rows

`mudlog-attitude-loot-depth`, `spec-proc-dragon-breath-combat`. Both are the
`SPECIAL(dragon_breath)` vehicle: mob 4209 (SPELL_FROST_BREATH,
`src/spec_procs.c:928-989`) spawned in 8105 with a mortal peer teleported in,
probe `~dpclock pulse 40`.

| Scenario | C boundaries (setup / warmup / pulse) | Go boundaries |
|---|---|---|
| `mudlog-attitude-loot-depth` | 4120 / 4201 / 4213 | 4120 / 4201 / 4212 |
| `spec-proc-dragon-breath-combat` | 4090 / 4094 / 4106 | 4090 / 4094 / 4105 |

Setup and warmup agree exactly; Go's log is a strict **prefix** of C's, one entry
short, and the missing entry is a trailing `Number(0,99)`. The window
(`spec-proc-dragon-breath-combat`, indices):

```
4126 Number(0,18)      mobile_activity door      (src/mobact.c:120)
4127 Number(0,15)      mobile_activity sound     (src/mobact.c:146)
4128-4131 Number(0,3)  x4
4132 Number(0,99)      a saving throw            (src/magic.c:422)
4133 Dice(1,1)         skill_message selector    (src/fight.c:1035)
4134 Number(0,500)     PC attacks count          (src/fight.c:1947)
4135 Number(0,10000)   PC parry pre-check        (src/fight.c:1949-1973)
4136 Number(1,20)      PC attack roll            (src/fight.c:1815)
4137 Dice(1,4)         PC damage dice
4138 Number(0,900)     NPC attacks count         (src/fight.c:1917)
4139 Number(1,20)      NPC attack roll
4140 Dice(30,4)        NPC damage dice (the dragon)
4170 Number(0,80)      hit()'s high-level switcheroo scan (src/fight.c:1419-1440)
4171 Dice(1,2)         the swing's damage dice
4172 Number(0,99)      C ONLY
```

`Number(0,99)` immediately after a damage roll is `damage()`'s mounted-dismount
roll — `if (!IS_NPC(victim) && IS_MOUNTED(victim) && dam > 0 && number(0,99) < 10)`
(`src/fight.c:1548`) — whose Go twin is `pkg/combat/fight_core.go:469`. Two
candidate causes, both narrow: (a) `IS_MOUNTED(ch)` is
`IS_AFFECTED(ch, AFF_MOUNT)` (`src/utils.h:362`) while `Player.IsMounted()` also
accepts `MountName != ""` (`pkg/game/deferred_fight_fns.go:50`) — check the
`affMounted` bit against C's `AFF_MOUNT` and whether the engines agree on the
victim's mount state; or (b) C's last attack of the window has a different victim
than Go's, so only C reaches the roll. Neither is confirmed: the C call site is
identified but the divergence condition is not yet proven (R5e).


## §4 a `mobile_activity` pass lost inside `~dpclock pulse 20` — 1 row, STOP TIER

`backstab-aware-trio`, probe step 4:

```
cbound  probe step 3: ~dpclock pulse 20  4135   probe step 4: ~dpclock pulse 20  4137
gobound probe step 3: ~dpclock pulse 20  4135   probe step 4: ~dpclock pulse 20  4135
```

C drew two entries, `Number(0,18)` then `Number(0,15)` — exactly one
`mobile_activity` pass for one mobile (the wander door `src/mobact.c:120` and the
sound gate `src/mobact.c:146`). Go drew none. `PULSE_MOBILE` is 4 RL_SEC = 40
pulses (`src/structs.h:633`), so a 20-pulse advance covers *half* a mobile
period: the pass falls inside a given window only when the two engines' absolute
pulse counters are in phase. The phase depends on how many pulses elapsed before
the window (boot, setup, warmup), which is a DP_CLOCK/harness accounting question
rather than a port behavior — **stop tier**, written up not fixed. If it is pulse
accounting it likely also explains the residual `Number(0,3)`-shaped divergences
in §5 and §7.

## §5 warmup ends without the extra draw — 2 rows

`shoot-mob-miss-depth`, `shoot-pc-miss-depth`. First mismatch
`c=Dice(1,2) = 1  go=Number(0,3) = 1` at stream 4136, C in warmup.

A warmup bisect (replay the warmup one command longer per run) puts the extra
draw exactly at warmup command **10**:

| n (warmup commands) | C warmup delta | Go warmup delta |
|---|---|---|
| 1–7, 9 | +n | +n |
| 10 (`… skillset Shootwindow 'shoot' 1`, `set player Shootwindow dex 0`) | **+11** | **+10** |
| 11 | +12 | +11 |

and the draw log shows the extra entry sitting between command 10's hide roll and
the next command's:

```
C  4126-4135 Number(0,3) x10   > 4136 Dice(1,2)=1   4137 Number(0,3)=3  4138 Number(1,101)=7  4139 Number(0,3)=2  4140 Dice(1,2)=1
Go 4126-4136 Number(0,3) x11                        4137 Number(1,101)=88 4138 Number(0,3)=0
```

So C draws `dice(1,2)` once for `set player Shootwindow dex 0` and once more
inside the probe step, and Go never draws it. `do_set` itself has no draws —
`grep -n 'dice(\|number(' src/act.wizard.c` returns only lines 1228, 1937, 2304,
2467, 2874, 2950, 3486, none inside `do_set` (`:2523-…`) — so the call site is
still unidentified. The `Dice(1,2)` shape fits
`dice(GET_OBJ_VAL(w, 1), GET_OBJ_VAL(w, 2))` for a wielded object whose values
are 1 and 2 (the hoopak is the only wielded item in this vehicle) or
`skill_message`'s selector with `number_of_attacks == 2` (`src/fight.c:1035`).
Next step: grep those two classes for reachability from a `set`/`shoot` command.

## §7 `carve-depth` warmup swing — 1 row

First mismatch `c=Number(1,20) = 9  go=Number(0,3) = 1` at stream 4098. C's
warmup adds **8** entries, Go's **5**:

```
C  4094-4097 Number(0,3) x4   4098 Number(1,20)=9   4099 Dice(1,6)=2
go 4094-4097 Number(0,3) x4   4098 Number(0,3)=1
```

The warmup is `goto 8162 / teleport Carveobs 8162 / get sword / wield sword /
kill fish / hit fish` then ten `~dpclock pulse 20` controls. For a mortal, C's
`do_kill` delegates to `do_hit` (`src/act.offensive.c:138-142`) and both engines'
"already fighting / not standing" gate is the same (`src/act.offensive.c:122`
vs `pkg/session/combat_cmds.go`), so `kill fish` enrols and `hit fish` answers
"You do the best you can!". C's `hit()` attack roll is `number(1,20)`
(`src/fight.c:1815`) and the swing's damage dice is the `Dice(1,6)`.

Go is three entries short: the swing's two draws plus one command that drew
*nothing*. The likely cause is the wait/drain path rather than combat: Go's
`cmdHit` sets `PULSE_VIOLENCE + 2` before the swing, so the next warmup line is
queued and drains during the `~dpclock pulse 20` pumps — if Go's queue or wait
handling differs, that line's hide roll lands outside the window. Confirm with a
wait/drain comparison before touching combat.

## Scoreboard

| Class | Rows | State |
|---|---|---|
| §6 `checkload` reads | 1 | **fixed on this branch** |
| §2 God-vs-mob round order | 6 | diagnosed, needs engine work |
| §3 missing saving throw in a long pulse | 2 | diagnosed, call site found, condition unproven |
| §4 `mobile_activity` pass in `pulse 20` | 1 | stop tier (DP_CLOCK accounting) |
| §5 extra `Dice(1,2)` in the warmup | 2 | bisected to the exact command, call site unidentified |
| §7 `carve-depth` warmup swing | 1 | diagnosed, wait/drain hypothesis |
| §1 login-flow asymmetry | 3 | Claude (harness) |
| §8 `palm` / `peek` | 2 | GLM (#1993) |

