# DP-1419 — Crash_load's per-item check_for_bad_stats

Stop tier: entry lifecycle and persistence. Zach merges.

## Scope

C's `Crash_load` equips every worn item it restores and runs
`check_for_bad_stats` at each equip. The port restored the equipment from the
player record but never ran the checks, so a returning character with saved
zero-stat gear saw nothing and kept full HP. This round ports the replay.

## C ground truth (re-read while writing, R5g)

* Entry order: `reset_char` → `Crash_load` → `save_char(ch, NOWHERE)` →
  `send_to_char(WELC_MESSG)` → `character_list` → `char_to_room`
  (`src/interpreter.c:2174-2210`). Everything `Crash_load` sends reaches the
  player before the welcome message, and `ch->in_room` is `NOWHERE`
  throughout — `reset_char` sets it (`src/db.c:2945`).
* Load order: `Crash_load` reads the file and calls `auto_equip`
  (`src/objsave.c:557`) per object; `Crash_crashsave` writes worn items in
  ascending wear position (`for j = 0..NUM_WEARS`, `src/objsave.c:807-812`), so
  the equip order is ascending C wear position.
* `auto_equip` → `equip_char`, whose last two statements are
  `affect_total(ch); check_for_bad_stats(ch);` (`src/handler.c:748-749`).
* `affect_total` rebuilds `aff_abils` from `real_abils` plus every slot
  currently filled (`src/handler.c:337-372`), so each equip sees only the items
  equipped SO FAR.
* `check_for_bad_stats` selects the LAST zero in STR/INT/WIS/CHA/DEX order and
  sends one LFCR-terminated line (`src/handler.c:640-675`); CON is not checked.

## What the port does

`World.ReplayCrashLoadBadStats` (`pkg/game/bad_stats.go`) walks
`COrderedWorn(p)` — ascending C wear position — computes `aff_abils` over that
prefix with the new `Player.affectTotalForItemsLocked`
(`pkg/game/effective_attributes.go`), and runs the existing `badStat` +
`applyBadStat` pair. `applyBadStat` gained a delivery seam
(`applyBadStatDeliver`): the entry caller passes the session's own transport,
because C's `send_to_char` writes `ch->desc` and the character has no world
reference, no session-registry entry and no `character_list` entry yet.

The replay holds the character's room at `NOWHERE` for its duration and
restores it afterwards, matching `reset_char`. Without that, `damage()`'s room
`act()` and the combat broadcast would treat the restored record's load room as
an audience.

`enterReturningPlayer` runs the replay after `crashLoadEntry()` and before the
menu-entry save (`pkg/session/menu.go`), so the save records the reduced HP as
C's `save_char` does.

### The per-equip arithmetic (correction to DP-1419)

DP-1419's expected "HP 200 → 160" holds for exactly one item. Because the check
runs at every equip and `aff_abils` is recomputed from the items equipped so
far, a stat that reaches zero keeps firing:

| firing | items equipped so far | DEX | HP |
|---|---|---|---|
| 1 | light (-10 DEX) | 10-10 = 0 | 160 |
| 2 | light + body | 0 | 120 |
| 3 | light + body + head | 0 | 80 |
| 4 | light + body + head + legs | 0 | 40 |

A -10 DEX item in an early slot followed by three more worn items costs
4 x 40 = 160 HP, not 40. The tests cover both ends: the zeroing item first
(4 firings, HP 40) and last (1 firing, HP 160).

### Path trace: which entries fire

* **Fresh login** (password, then menu `1`): `RecordToPlayer` restores the
  equipment at the password step (`pkg/session/session_login.go`), then
  `enterReturningPlayer` runs the replay once.
* **Rented re-entry** (`s.player.RentedOut`, `reloadCharacterFromStore`): the
  store record rebuilds the character and its equipment, then the same
  `enterReturningPlayer` runs the replay once.
* Neither fires twice for one entry: only `enterReturningPlayer` calls the
  replay, and it is called once per entry.
* The gate is the crash file, not the record: `crashLoadEntry` now reports
  whether `object_saves` held a row (C's `fopen` success). A missing file is
  C's `:474-489` arm — nothing is equipped, nothing fires. Known
  simplification: the record and the object-save blob are separate stores; the
  replay walks the record's restored worn items and treats the file's existence
  as the gate, which holds because every C object-save boundary writes both.

### Defined vs undefined at NOWHERE

* **`act()` at NOWHERE is guarded, not an out-of-bounds read.**
  `src/comm.c:2519-2526` checks `ch->in_room != NOWHERE` before reading
  `world[ch->in_room].people` and otherwise logs
  `SYSERR: no valid target to act()!`. DP-1419's "`world[-1]`" premise is
  wrong; the room part of the DEX arm is defined and silent. The Go `actDeliver`
  already mirrors it (`roomVNum < 0` → the same MudLog), and the replay's
  NOWHERE hold makes the two agree. `src/fight.c:1337` (`ROOM_FLAGGED`) is
  short-circuited by `ch != victim` for self-damage, and the room loop at
  `:1425` is inside `if (victim != ch)`; neither is reached.
* **The lethal sub-case is the real blocked one.** A zero-DEX replay that takes
  HP to <= 0 runs C's `damage()` death path at NOWHERE: `death_cry`
  (`src/fight.c:507-514`) explicitly `char_to_room(ch, 0)` — world index 0 — and
  a corpse is made there. The port does not reproduce that branch; the guard
  test only proves the replay survives it without moving the character. Row
  `equipment.saved-zero-stats-lethal` is `blocked`; **Zach's ruling needed**.
* **Finding, not fixed: the DEX arm's skill_message bytes.** `TYPE_SUFFERING`
  is `399` and `IS_WEAPON` is `[TYPE_HIT, TYPE_SUFFERING)`, so `damage()` routes
  the self-damage through `skill_message` (`src/fight.c:1534-1535`).
  `lib/misc/messages:868-882` gives set 399 a hit message whose victim line
  ("You lie on the ground suffering from your terrible wounds...", or the
  bleeding variant — the set has two records, so `dice(1,2)` picks one) goes to
  the character with `TO_VICT`. C therefore sends those bytes at entry; Go's
  combat delivery (`Manager.combatSession` → `attachedBodyLocked`) requires a
  registered session that is neither `menuActive` nor `charCreating`, so at
  entry they are dropped. Closing that gap means a design decision about
  pre-registration combat delivery; it is outside this round's scope and is
  named in the PR for Zach.

### Not in scope (checked, reported)

`auto_equip`'s alignment and doubled-slot refusals are **not** ported: the
restore path places a saved item straight into its recorded slot with
`Equipment.SetSlot`, so an anti-alignment or mismatched-wear item that C would
send to the inventory is worn here. Pre-existing, unrelated to this round, and
left alone.

## Proof

* `pkg/session/crash_load_bad_stats_test.go` drives the real menu entry
  (`MsgCharInput "1"` → `handleMenuChoice` → `enterReturningPlayer`) on a real
  SQLite store: per-item DEX firings and HP arithmetic, the zeroing item last,
  STR/INT/CHA arms, CHA Pestilence hunting plus its mudlog, the rented re-entry,
  and the no-file gate.
* `pkg/game/bad_stats_test.go` adds `TestReplayCrashLoadBadStatsFiresPerEquippedItem`
  (four firings, HP 40), `TestReplayCrashLoadBadStatsHoldsRoomAtNowhere` (a peer
  in the record's room sees nothing) and
  `TestReplayCrashLoadBadStatsLethalLeavesRoomAlone` (the blocked guard).
* R5h controls: `2026-10-13-dp-1419-bad-stats-controls.py` applies two build
  overlays — no check on restore, and a single check on the final stats — and
  requires the tests to fail under each, then pass on the restored tree. The
  single-check mutation is the one that proves the per-equip replay: it prints
  one line and leaves HP 160 instead of 40.
* Depth rows: `docs/fidelity/depth/combat-entry.tsv`
  (`equipment.saved-zero-stats` and four siblings, one `blocked`).

