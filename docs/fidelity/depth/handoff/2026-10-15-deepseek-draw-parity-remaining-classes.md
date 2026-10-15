# DeepSeek — the DRAW rows that remain, by class

Date: 2026-10-15 · Branch: `deepseek/draw-parity-remaining-classes` (off `origin/main`
`f1bf62126`) · Rules: R3a, R3b, R3d, R5c, R5e, R5g

Baseline: `finish-baseline-draw-parity` (seed 1, reference oracle
`bd6c548c…`) — 1,171 scenarios, byte CLEAN, **32 DRAW**. Two sibling PRs clear 14
of them (#1987, the setup-symmetry fixture class; #1988, the bare-RETURN draw).
This note records the other 18 with the evidence gathered for each, so the next
pass starts from a boundary table rather than from the first-mismatch line.

Every row below was reproduced serially with
`DP_ORACLE_BIN=$HOME/darkpawns-c-oracle/bin/circle go run ./cmd/dp-oracle-diff
--scenario <name> -draw-parity`, with a temporary local print of the phase/step
draw boundaries (`oraclediff.DrawBoundary`). No harness file is changed by this
branch; the boundary print was reverted before committing.

## 1. Login-flow asymmetry — 3 rows (needs DP-1427 v2)

`house-depth`, `spec-proc-key-seller`, `spec-proc-key-seller-non-owner`.

The oracle reattaches to a pre-seeded character (`Testor / testpassword / <ENTER> /
1` — zero creation draws) while the port creates one through the full creation
dance. Both setups hold **zero** post-entry commands, so there is no stray command
to mirror and no corpus fix: the offset is the creation path itself. Needs the
probe-boundary comparison (compare the streams from the probe boundary rather than
from process start), or a fixture in which both engines create the character.

## 2. Combat round order, God-vs-mob — 6 rows

`shop-live-inventory-control`, `shop-live-inventory-filters`,
`shop-live-inventory-gate`, `shop-stack-list-live`, `show-shops-detail`,
`mudlog-crash-load-entry`.

First mismatch: `c=Dice(1,1) = 1  go=Number(0,20) = 9` at stream 62467 (the five
shop rows) and `stream=62382` for `-filters` / `-gate`; `mudlog-crash-load-entry`
shows the same pair at stream 81529 inside probe step 6
(`send:primary ~dpclock pulse 20`).

`Dice(1,1)` is `skill_message`'s message selector,
`nr = dice(1, fight_messages[i].number_of_attacks)` (`src/fight.c:1035`, reached
from `damage()` at `src/fight.c:1534-1541`); `Number(0,20)` is the Go side's
bare-hand roll. GLM's DP-1438 handoff already filed this as the **God-vs-mob round
order** divergence (finding 1): C draws the position-degraded mob's swing where Go
draws the level-60 player's roll. The enrolment-point change in DP-1438 moved the
first mismatch *past* the mobile-`hit()` class to here, so this is the next engine
work, not a fixture problem.

## 3. One missing saving throw inside a long pulse — 2 rows

`mudlog-attitude-loot-depth`, `spec-proc-dragon-breath-combat`. Both probes are a
single `~dpclock pulse 40`.

| Scenario | C boundaries (setup / warmup / pulse) | Go boundaries | Missing draw |
|---|---|---|---|
| `mudlog-attitude-loot-depth` | 4120 / 4201 / 4213 | 4120 / 4201 / 4212 | `Number(0,99) = 0` |
| `spec-proc-dragon-breath-combat` | 4090 / 4094 / 4106 | 4090 / 4094 / 4105 | `Number(0,99) = 31` |

Setup and warmup agree exactly; Go is one entry short inside the 40-pulse window.
`Number(0,99)` is `mag_savingthrow`'s roll (`MAX(1, save) < number(0,99)`,
`src/magic.c:408-426`). So C reaches one more saving throw than Go in the window —
either a breath/affect path that C runs twice, or a per-pulse effect tick Go does
not run. Both rows are byte-clean.

## 4. Mobile-activity pass lost inside a `pulse 20` — 1 row

`backstab-aware-trio`, probe step 4 (`~dpclock pulse 20`):

```
cbound  ... probe step 3: ~dpclock pulse 20  4135   probe step 4: ~dpclock pulse 20  4137
gobound ... probe step 3: ~dpclock pulse 20  4135   probe step 4: ~dpclock pulse 20  4135
```

C drew two entries, `Number(0,18)` then `Number(0,15)` — exactly one
`mobile_activity` pass for one mobile: the wander door (`src/mobact.c:120`) and the
sound gate (`src/mobact.c:146`). Go drew none. `PULSE_MOBILE` is 4 RL_SEC = 40
pulses (`src/structs.h:633`), so a 20-pulse advance covers half a mobile period and
the pass lands in a given step only when the two sides' absolute pulse counters are
in phase. Needs a look at the DP_CLOCK advance phase, not at mobile_activity.

## 5. Warmup ends without the message-selector draw — 2 rows

`shoot-mob-miss-depth`, `shoot-pc-miss-depth`. C's warmup boundary is 4135 (12
entries), Go's is 4134 (11); the missing entry is `Dice(1,2)` at C's entry 4134,
i.e. `skill_message`'s selector (`src/fight.c:1035`) with
`number_of_attacks == 2`. Neither warmup contains combat — they end with
`set mob instructor dex 25` — so the last warmup command reaches the selector on C
only. Needs a per-command trace of the warmup tail.

## 6. `checkload` draws one short — 1 row

`checkload-depth`, probe step 7 (`checkload mob 16303`): C 31599 → 31603 (4
entries), Go one fewer; the missing entry is `Dice(1,5) = 3`. Needs the C
`do_checkload` path read against the Go handler.

## 7. `carve-depth` warmup swing — 1 row

First mismatch `c=Number(1,20) = 9  go=Number(0,3) = 1` at stream 4098, C in
warmup. Already recorded by GLM's DP-1438 handoff (finding 2): C draws the
`kill fish` / `hit fish` swing (`Number(1,20)` + `Dice(1,6)`) during warmup where
Go's warmup ends the fight path without the swing draws.

## 8. GLM-owned — 2 rows

`mortal-obj-batch` (first mismatch at probe step 6, `palm`) and `peek-depth`
(probe step 3, `peek trainee`) are DP-1436 and DP-1437, GLM's extras.

## Scoreboard

| Class | Rows | Owner |
|---|---|---|
| setup symmetry (fixture) | 11 | **#1987** |
| bare RETURN draws | 3 | **#1988** |
| login-flow asymmetry | 3 | DP-1427 v2 (harness) |
| God-vs-mob round order | 6 | engine (DP-1438 follow-up) |
| missing saving throw in a long pulse | 2 | engine |
| mobile pass inside `pulse 20` | 1 | engine / DP_CLOCK |
| warmup message selector | 2 | engine |
| `checkload` | 1 | engine |
| `carve-depth` warmup | 1 | engine (DP-1438 follow-up 2) |
| `palm` / `peek` | 2 | DP-1436 / DP-1437 |
