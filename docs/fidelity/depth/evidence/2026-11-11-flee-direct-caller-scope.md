# `do_flee` lower-position direct-caller scope — 2026-11-11

## C call-site audit

`flee` is registered at `POS_FIGHTING` (`src/interpreter.c:445`), while
`do_flee` emits `Get on your feet first!` when the body's position is lower
(`src/act.offensive.c:371-376`). The ordinary typed command therefore cannot
reach this line. A full `src/` search in this session found these live
`do_flee` callers: `src/new_cmds2.c:1021`; `src/spec_procs.c:1343,1707,2194`;
and `src/fight.c:1599,1610`. The other `src/fight.c:1618` occurrence is in a
commented-out block.

- `do_turn` scans the room for `RACE_UNDEAD`/`RACE_VAMPIRE` and calls `do_flee`
  when the level difference exceeds three (`src/new_cmds2.c:999-1021`). Normal
  player creation parses only `h/e/d/k/r/m/s` (`src/utils.c:600-614`), so an
  ordinary player cannot be that target. A switched mobile body can have one
  of those races.
- `mickey` and `conductor` first select a visible non-NPC target, then guard
  the `do_flee` calls with `IS_NPC(victim)` (`src/spec_procs.c:1310-1345,
  1670-1710`); those two calls are unreachable in their actual paths.
- `fearface` selects only non-NPC targets and contains a `do_flee` call, but
  C does not assign the special; the existing `mob.fearface-unassigned` row
  records that dead dispatch (`src/spec_procs.c:2171-2201`,
  `src/spec_assign.c:84-130`).
- `fight.c`'s NPC `MOB_WIMPY` branch can call `do_flee` without a standing
  check (`src/fight.c:1594-1599`). Its player-wimpy branch requires
  `GET_POS(victim) >= POS_FIGHTING` before calling it (`:1601-1610`), so that
  player path cannot produce the lower-position refusal.

The remaining route with a player descriptor is an immortal switched into a
mobile body. C's `do_switch` moves the descriptor onto the NPC (`src/act.wizard.c:1175-1200`),
so a direct caller that selects that mobile can send the line to the switched
player. `do_turn` is one such possible caller if the body is undead/vampire;
the NPC-wimpy combat path is another when its state permits. These C lines
were grepped in this session (R5g).

## Decision requested

Should the D1 row include output seen only by a descriptor switched into an NPC
body? If yes, the witness needs a C-first switched-body scenario and coordination
with Ember's switch bridge work; I did not touch those files. If no, exclude the
row as unreachable from the ordinary player's typed `flee` command and target
paths in the unmodified C world. The row remains blocked until Zach chooses the
scope. No census was run because no accepted witness scope is established.
