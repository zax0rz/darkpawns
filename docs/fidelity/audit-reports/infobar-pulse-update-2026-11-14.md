# InfoBarUpdate pulse output — ruling and DP-1539 fix (2026-11-14)

Finding from the DP-1535 switched-body infobar work: the C source has
`InfoBarUpdate` (`src/act.display.c:226-285`) called from `make_prompt`
(`src/comm.c:1158-1193`) when a body's mana, move, hit, gold or exp changed
since `InfoBarOn` recorded the last-known values. The oracle does not emit
those bytes on a normal command.

## Player observation

`infobar on` then `holler test` (holler spends 20 moves, `src/act.comm.c:1254`):

- C oracle: `You holler, 'test'`; **no** `VT_CURSAVE` / `VT_CURREST` move
  repaint anywhere in the following blocks.
- Go before DP-1539: the same line plus `\e7          62(82)\e8`.

Retained log: `docs/fidelity/evidence/infobar-pulse-2026-11-14/player-holler.txt`.

## Why

On the normal path, `process_output` has already copied `t->output`
(`src/comm.c:1630-1635`) before it calls `make_prompt`; `InfoBarUpdate`
appends to that buffer and the clear at `src/comm.c:1664-1665` throws the
bytes away.

## The exception (bare-prompt path)

`src/comm.c:644-649`: when an input printed nothing, `process_output` does not
run. The prompt loop calls `make_prompt`, whose `InfoBarUpdate` writes the
repaint into `d->output`; the **next** pass flushes it inside
`process_output`'s CRLF frame. So the bar only reaches the wire after an input
that printed nothing — a bare Enter, for example.

## Ruling

Zach ruled (2026-11-14): **Go matches C; no approved divergence.** This is
DP-1539.

## Fix

- Removed the per-pulse `cmdInfoBarUpdate` call from the prompt path.
- Every prompt pass now records the current vitals as the comparison baseline
  (C updates the last-known values in `make_prompt` even when the bytes are
  discarded), for players and for a switched body's shared dummy record.
- The empty-input branch defers the repaint: the prompt pass skips the
  baseline update, and the repaint is emitted after the prompt with C's
  CRLF frame.
- Witnesses: `infobar-pulse-empty-player-depth` (holler → no repaint; one-hour
  pulse; bare Enter → repaint on the next pass) and
  `infobar-pulse-empty-switched-depth` (shared dummy record; holler → no
  repaint; a peer lowers the body's mana; bare Enter → repaint).

## Separate finding

While building the switched-body witness, `set <mob> mana` plus a pumped hour
showed C regenerating a mob's mana (2 → 10) and Go leaving it at 2. The port's
mob `point_update` regeneration is missing; that is not an infobar bug and is
reported separately.
