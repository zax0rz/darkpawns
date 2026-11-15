# InfoBarUpdate pulse output is not observable — 2026-11-14

Finding from the DP-1535 switched-body infobar work. The C source has
`InfoBarUpdate` (`src/act.display.c:226-285`) called from `make_prompt`
(`src/comm.c:1158-1193`) when a body's mana, move, hit, gold or exp changed
since `InfoBarOn` recorded the last-known values. The oracle does not emit
those bytes.

## Player witness

`infobar on` then `holler test` (holler spends 20 moves, `src/act.comm.c:1254`):

- C oracle: `You holler, 'test'` and the room look; **no** `VT_CURSAVE` /
  `VT_CURREST` move repaint anywhere in the following blocks.
- Go port: the same line plus `\e7          80(100)\e8` (the move repaint).

Run: `tmp-infobar-player-pulse` (temporary scenario, log at
`docs/fidelity/evidence/infobar-pulse-2026-11-14/player-holler.txt`);
`--draw-parity` MATCH, normalized divergence on the holler block.

## Switched-body witness

`switched-body-infobar-depth` with `holler infobar pulse` after `infobar on`
as the level-32 Death Dealer: C emits the holler line only; Go emitted the
body repaint. Log: `switched-holler.txt`. The scenario shipped in the infobar
PR uses `north`/`look` instead, so it stays clean.

## Reading

`InfoBarUpdate` writes with `write_to_output(ch->desc, ...)` while
`process_output` is already copying `t->output` into its send buffer
(`src/comm.c:1613-1640`); the queued bytes are not part of that flush and the
next prompt produces none, so the oracle never puts them on the wire. The
source wins only after the call path is verified (R5e); the bytes are the law
(R1).

## Needed ruling

The port's **player** `cmdInfoBarUpdate` predates this work and emits bytes C
does not; the switched-body variant was removed before shipping this PR. Zach
should choose:

1. remove the player prompt repaint output (R1: match the oracle), or
2. approve the repaint as a divergence and record it with `divergent-approved`.

Until then no `infobar.pulse-update` row claims oracle-green.
