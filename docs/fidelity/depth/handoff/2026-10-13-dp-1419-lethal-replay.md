# DP-1419 follow-up — the lethal Crash_load replay

Stop tier: entry lifecycle. Zach's ruling (Linear DP-1419, 2026-10-13) is the
spec; this round ports it. Closes `equipment.saved-zero-stats-lethal`.

## The C path

A zero-DEX trip during `Crash_load` runs C's death pipeline while
`ch->in_room` is still `NOWHERE` (`reset_char`, `src/db.c:2945`):

1. `damage()` (`src/fight.c:1314`): HP drops by 40, `update_pos` sets `POS_DEAD`
   (`:1483-1489`), the room `act("$n is dead!  R.I.P.")` finds no audience and
   logs the guarded `SYSERR: no valid target to act()!` (`src/comm.c:2519-2526`),
   and the victim gets `"You are dead!  Sorry...\r\n"` (`:1586-1588`).
2. The PC death branch (`:1670-1691`) logs `"%s killed by %s at %s"` with
   `world[victim->in_room].name`, increments `GET_DEATHS`/`GET_LAST_DEATH`, then
   `GET_KILLS(ch)++` (ch == victim) and `counter_procs(ch)`.
3. `die_with_killer` (`:586`): `gain_exp(-(GET_EXP/37))`, then, for HP < 0 at
   level > 5, `!number(0,3)` → CON−1 (and at level > 20 `!number(0,5)` → another
   CON−1), then `affect_total`, `GET_POS = POS_DEAD`, and `raw_kill`.
4. **`raw_kill` returns at its first statement** — `if (ch->in_room == NOWHERE)
   return;` (`src/fight.c:538-539`) — so `death_cry`, `make_corpse`,
   `make_dust`, the affect/tattoo stripping, the unmount and `extract_char`
   never run. There is no corpse and no extraction.
5. Later worn items still run `check_for_bad_stats`: the trip line is sent
   again, but `damage()` refuses at the corpse gate (`:1319-1325`, server log
   `SYSERR: Attempt to damage a corpse.`), so there is no further HP loss and no
   second death message.
6. Login continues: entry save, `WELC_MESSG`, `char_to_room(load_room)`. The
   character is admitted at `POS_DEAD` with HP <= -11.

## What the port does

* `World.ReplayCrashLoadBadStats` installs the entering session's descriptor as
  the player's text sink for the replay (`Player.SetEntryTextSink`), so every
  line C's `send_to_char` writes to `ch->desc` reaches that descriptor: the
  bad-stat lines, `"You are dead!  Sorry..."`, the wounded-band lines, and the
  `Bleeding` line. The live path is untouched (the sink is nil outside the
  replay).
* `handlePlayerDeath` gained `raw_kill`'s NOWHERE guard after the CON block:
  `StopFighting`, `SetPosition(PosDead)`, `return` — skipping
  `unmountRawKillBody`, `DeathCry`, the corpse, the AFF_WEREWOLF removal and
  `QueuePlayerExtraction`. That is steps 4 and 6.
* The exp loss now calls the port's `gain_exp` (`World.GainExp`) instead of an
  inline `GET_EXP - GET_EXP/37`, so C's level gate and `-max_exp_loss` clamp
  apply (`src/limits.c:287-322`; `src/config.c:82`).
* `selfDamage`'s corpse gate now emits C's server log
  `SYSERR: Attempt to damage a corpse.` (`log()` is `basic_mud_log`,
  `src/utils.h:39` → `BasicMudLog`) unless the victim is already queued for
  extraction (`:1321-1322`).

## Step 5 trace: what the first tick does

The character is in a room at `POS_DEAD` with HP <= -11. `point_update`
(`src/limits.c:493-513`) branches on `POS_STUNNED` and above, then `POS_INCAP`
and `POS_MORTALLYW` — `POS_DEAD` takes no branch, so no regen, no wound damage
and no extraction. Every command answers `"Lie still; you are DEAD!!! :-(\r\n"`
(`src/interpreter.c:925-927`). The character stays dead in the world; nothing in
C respawns them. The port matches both.

## R1a: two reads at NOWHERE, and the decisions

Both are `world[-1]` reads. `world` is a heap pointer (`src/act.comm.c:42`), so
`world[-1]` reads the allocator's chunk header, not a room; the reference has no
bytes to reproduce, and no oracle scenario can reach the path (a saved
zero-stat character whose gear kills them during `Crash_load`).

1. **The death mudlog's room name** (`src/fight.c:1673-1680`,
   `world[victim->in_room].name`). **Decision:** the port prints the empty
   segment its room lookup yields for an unresolvable vnum, so the line reads
   `"<name> killed by <name> at "`. The rest of the line — names, the
   `BRF`/`LVL_IMMORT`/file-TRUE flags — is C's. Pinned by
   `TestCrashLoadLethalEntryLeavesCharacterDead`.
2. **The neutral-room test** (`src/fight.c:1493`,
   `ROOM_FLAGGED(victim->in_room, ROOM_NEUTRAL)`). This one is load-bearing: if
   the garbage flags happen to include `ROOM_NEUTRAL`, C takes the rescue branch
   (`GET_HIT = 1`, `"You are saved by the gods!"`, `char_to_room(8004)`) and the
   character does **not** die. **Decision:** the port's lookup finds no room for
   an unresolvable vnum, so no flags, so the death path always runs. Recorded in
   the row `equipment.saved-zero-stats-lethal-nowhere-reads` (blocked: C's
   branch is garbage-dependent and unprovable).

## R3: the draws

`die_with_killer` makes exactly one `number(0,3)` at level 6+ and one
`number(0,5)` more only when the first came back 0 at level 21+. Both tests pin
the CON total *and* the stream position after the replay on seeds 1, 2, 3, 5
and 8 (a reference run consumes the expected draws; the replay's next draw must
match). `gain_exp` itself draws nothing.

## Found deviations (recorded, not changed)

* **DP-1021's HP floor.** `Player.TakeDamage` floors HP at -11
  (`pkg/game/player_affects.go`), so a lethal 40 from HP 10 leaves -11, not C's
  -30 (`src/fight.c:1483` has no floor; `update_pos` derives `POS_DEAD` from
  HP <= -11). Both are `POS_DEAD`, but the saved record and the prompt differ.
  The floor is a tested, documented decision (`TestTakeDamage_FloorsAtNegative11`,
  DP-1021), so it is left alone.
* **The CON floor and `LoginConstitution`.** `handlePlayerDeath` floors
  `Stats.Con` at 1, and `LoginConstitution` reads the unclamped real CON unless
  affects exist. C decrements `real_abils.con` with no floor and `affect_total`
  clamps `GET_CON` to 0..18, so C can reach `GET_CON 0` and the
  "You are too unhealthy to play!" entry gate; the port cannot. Pre-existing
  (an early migration commit, no ruling), left alone.
* **`equipment.saved-zero-stats-dex-suffering-message` stays blocked.** The DEX
  arm's `skill_message` set-399 victim line reaches the player through the
  combat `SendToChar` callback, which needs a registered non-menu session; the
  entry descriptor sink covers `SendMessage`-routed bytes only. Closing it is
  the pre-registration combat-delivery decision already named for Zach.

## Proof

* `pkg/game/bad_stats_test.go` — `TestReplayCrashLoadBadStatsLethalStopsAtRawKill`
  (HP/position, counters, exp, no corpse, no extraction, and the exact five-line
  stream), `…LethalConLossDraw`, `…LethalConLossSecondDraw`,
  `…LethalExpLossClamp`.
* `pkg/session/crash_load_lethal_test.go` — the real menu entry
  (`MsgCharInput "1"` → `handleMenuChoice` → `enterReturningPlayer`) on a real
  SQLite store: the death bytes in C's order before the welcome text, the
  character in the world at `POS_DEAD` with no corpse and no extraction, the
  death mudlog at an immortal (including the empty room segment), the first
  tick leaving them alone, and the command refusal.
* R5h controls: `2026-10-13-dp-1419-lethal-controls.py` applies four build
  overlays — drop the raw_kill guard, drop the CON block, restore the inline exp
  computation, drop the descriptor sink — and requires the matching tests to
  fail under each, then pass on the restored tree.
* Depth rows: `docs/fidelity/depth/combat-entry.tsv`
  (`equipment.saved-zero-stats-lethal` closed to `unit-green`, plus
  `…-lethal-penalties`, `…-lethal-tick` and the blocked `…-lethal-nowhere-reads`).
