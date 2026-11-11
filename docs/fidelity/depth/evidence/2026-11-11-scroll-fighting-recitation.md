# Fighting scroll-recitation witness — 2026-11-11

## Stale-row correction

`parser.recite.scroll-fighting` was marked blocked because the original
single-pulse scenario left the attacker's C `WAIT_STATE` active, so the recite
command was still queued when capture ended. The scenario was repaired on
2026-11-08 to drain a second pulse burst (`c5822e1d3`); the supporting
`surface/spell-parser.tsv` was updated, but the primary output manifest retained
the old blocked note.

C's registered `recite` command is available at `POS_RESTING`
(`src/interpreter.c:639`). `do_hit` opens the fight and applies the attacker
wait (`src/act.offensive.c:116-128`); the two pulses drain that wait while
leaving `FIGHTING(ch)` set. The real `SCMD_RECITE` path selects the carried
scroll and calls `mag_objectmagic` (`src/act.other.c:907-973`), whose recall
scroll branch prints `You can't concentrate enough!.` and returns before the
sitting gate (`src/spell_parser.c:661-664`). These C lines were grepped in this
session (R5g).

## Live census evidence

Each run selected only `spell-parser-scroll-fighting`, enabled `--draw-parity`,
and passed with the C transcript containing the recite refusal after the hit
and pulse steps. The retained seed-1 C block is
`/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-recitescroll-s1/census-dump/spell-parser-scroll-fighting.txt`.
The wrapper's warmup-refusal gate passed in each run, so neither the object
pickup nor either mob-setup command was refused on C or Go.

- Seed 1: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-recitescroll-s1` — CLEAN, one scenario passed, draw_divergent=0.
- Seed 2: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-recitescroll-s2` — CLEAN, one scenario passed, draw_divergent=0.
- Seed 3: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-recitescroll-s3` — CLEAN, one scenario passed, draw_divergent=0.
- Seed 5: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-recitescroll-s5` — CLEAN, one scenario passed, draw_divergent=0.
- Seed 8: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-recitescroll-s8` — CLEAN, one scenario passed, draw_divergent=0.

## Revert proof

For `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-recitescroll-revert-byte`, Go's fighting-refusal message was temporarily changed to omit the C period. The same targeted seed-1 census failed the transcript row (`passed=0`, `failed=1`, `draw_divergent=0`). The exact message was restored immediately after registration; the proof snapshot retained the mutation (R1/R3/R5h).
