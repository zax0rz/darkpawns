# DeepSeek combat findings — 2026-10-29

Four recorded findings from the Egon rounds, re-verified against `src/` (R5e)
before any change. Two PRs: [#2224](https://github.com/zax0rz/darkpawns/pull/2224)
(findings 1-3, one oracle vehicle) and
[#2225](https://github.com/zax0rz/darkpawns/pull/2225) (finding 4, cherry-picked
onto `origin/main`). Evidence:
`~/Archives/darkpawns/oracle-runs/2026-10-29/deepseek-combat-findings/`.

## Fixed

### 1. A mob's attitude loot didn't show the gold

`attitude_loot` runs `do_get(ch, "all corpse")` (`src/fight.c:1120`), and
`create_money` sets `ITEM_WEAR_TAKE` on every pile (`src/handler.c:1484-1486`),
so `can_take_obj` passes and the room sees `$n gets $p from $P.`
(`src/act.item.c:197-212`) before `get_check_money` credits the coins.

The port's money is a prototype-less `ObjectInstance` with no wear flags, so
`IsTakeable()` was false and `npcGetDirect` skipped the pile. `ObjectInstance`
gained `WearFlagsOverride` + `GetWearFlags()` (C stores wear flags per object),
written by `createMoneyObject`; every wear-flag reader now uses it, and
`canTakeObj`'s `type == ITEM_MONEY` shortcut is gone (C reads the object's own
flags, so prototype money without the bit stays untakeable).

### 2. A claw kill lost its corpse description

`hit()` derives `w_type = TYPE_HIT + offset` and hands it to `damage()` ->
`die_with_killer` -> `raw_kill` -> `make_corpse`, whose switch selects the
corpse long description from that `TYPE_*` value (`src/fight.c:1792-1806`,
`283-370`). The engine stored `int(AttackNormal)` (0) in `pair.LastAttackType`,
which is not a `TYPE_*` constant, so every weapon and claw kill built the
generic wording. It now stores `TYPE_HIT + msgAttackType`.

### 3. "Claw damage is 47 in C and 50 in Go" — the brief's hypothesis is wrong

The damage arithmetic is identical; the divergence is an RNG draw-order shift
with two causes:

- `performOneHit` returned straight after the miss message. C's miss still
  calls `damage(ch, victim, 0, attacktype)`, whose tail runs `update_pos`, the
  position-state bytes, the `!AWAKE` `stop_fighting`, and the default arm's
  bleed check where a `MOB_WIMPY` victim flees (`src/fight.c:1568-1614`).
- The session layer's `cb.DoFlee` hook returned early for every NPC, shadowing
  the game layer's correct NPC lane. C calls `do_flee(victim)` directly
  (`src/fight.c:1599`), and `do_flee` burns six `number(0, NUM_OF_DIRS-1)`
  draws (`src/act.offensive.c:392-393`). It now routes mobiles through
  `World.NPCCommandDirect`. `DoRetreat`'s NPC lane stays an early return,
  recorded as unreachable: the retreat arm is gated on `GET_SKILL`, and a
  mobile's skills live in the shared `dummy_mob` record (`src/utils.h:343`).

Reverting the miss tail reproduces the six-draw drift exactly (`Number(0,5) x6`
where Go had `Number(0,900)`) and the 13-vs-16 claw damage behind the
`very hard` / `extremely hard` split.

### 4. `set hitroll`/`damroll` sbyte storage

C's `points.hitroll`/`points.damroll` are `sbyte` (`src/structs.h:916-917`), so
`RANGE(-20,200)` then truncates on assignment: 200 stores -56 while the ack
keeps the clamped 200 (`src/act.wizard.c:2836-2840`). Fixed at the assignment
plus the readers that must honour the width (mob `20 - THAC0`, both getters for
mobs and players).

## Recorded, not fixed

- **The rest of the `sbyte`/`ubyte` class**: ability scores and `stradd`
  (`sbyte`), `weight`/`height` (`ubyte`), `conditions[3]` (`sbyte`),
  `orig_con`/`clan_rank` (`ubyte`), object/timed-affect `modifier` (`sbyte`,
  `src/structs.h:738,885-898,978-989,1050`). `do_set` clamps most of them into
  ranges that fit the byte; the reachable remainder is the player affect fold
  and the parser accepting object modifiers outside -128..127.
- **`death-mixed-group-npc-kill`'s draw row** is the divergence GLM disclosed
  as pre-existing (`stream=4217`, the kill-pulse violence/mobact ordering). It
  stays red and disclosed; out of scope here.
- **The engine's default `TakeDamage` tail** and the special-death arms
  inventoried in `2026-10-29-glm-egon-specdeath.md` remain as recorded there.

## Proof

Every fix has a unit test that fails when the fix is reverted (checked by
reverting each one), and the werewolf diagnostic
`spec-proc-werewolf-mob-victim` — red on `origin/main` for findings 1, 2 and 3
— is now byte-clean with `draw-parity MATCH` at seeds 1, 2 and 3. The new
`attitude-loot-gold-depth` (finding 1) and `set-sbyte-wrap-depth` (finding 4)
scenarios are green with draw parity at seeds 1-3. The 30-scenario targeted
census with `--draw-parity` (named in PR #2224) is CLEAN; its one draw row is
the disclosed `death-mixed-group-npc-kill` baseline above.
