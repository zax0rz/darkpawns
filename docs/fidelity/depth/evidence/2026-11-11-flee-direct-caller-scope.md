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

## Approved scope and witness

Zach approved including switched-body-only output in this D1 row on
2026-11-11. The C-first scenario `flee-get-on-feet-switched-turn` loads indexed
Race-9 zombie #10, switches the implementor descriptor into its body, sits it,
and has a skilled, good-aligned peer with ample mana issue `turn`. The retained
transcript shows the switched player receiving `You are suddenly terrified!`
and `Get on your feet first!` after C's direct `do_flee` call. The clean targeted
census is
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-flee-switched-turn-loadmob`
(seed 1, draw parity matched).

R5h revert proof: for one retained run, the Go `DoFlee` refusal changed one
byte from `!` to `?` in `pkg/game/flee.go`; the same scenario failed only on
that switched-body line and the draw stream still matched. The NOT_CLEAN run is
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-flee-switched-turn-revert-proof`.
The source was restored immediately after the census snapshot. The row now
claims the switched-body scenario at seed 1 (R1/R2/R5e/R5g/R5h).
