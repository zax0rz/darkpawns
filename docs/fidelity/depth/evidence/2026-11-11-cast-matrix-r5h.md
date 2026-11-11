# CAST matrix R5h output-byte controls — 2026-11-11

These retained controls demonstrate that each promoted OLC matrix family can fail
when one player-facing byte is wrong. The old `*-revert-proof` runs only captured
refused setup from the demotion attempt; they are historical setup evidence, not
R5h controls.

## Potion family — PR #2486

- Clean candidate: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-cast-potion-promoted-final` (`PASS`, seed 1).
- Mutated run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-cast-potion-lay-hands-revert-proof` (`NOT_CLEAN`, `spellmatrix-potion-olc-03-depth@1`, failed block 177, `draw_divergent=0`).
- The run captured Go HEAD `d5eb503376146a87842c5655d1705f7343037d56` with a dirty worktree. I changed only the final `!` to `?` in Go's `SpellLayHands` output at `pkg/spells/affect_spells.go:758`. C emits `Your wounds mend beneath your hands!` (`src/magic.c:1817-1823`). The failed block shows the promoted row can detect that byte change. The source was restored immediately after the retained census completed.

## Staff family — PR #2487

- Clean candidate: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-cast-staff-classified-final-v2` (`CLEAN`, seed 1).
- Mutated run: `/home/zach/Archives/darkpawns/oracle-runs/2026-11-11/luna-l3-cast-staff-detect-align-revert-proof` (`NOT_CLEAN`, `spellmatrix-staff-olc-03-depth@1`, failed block 235, `draw_divergent=0`).
- The run captured Go HEAD `4b09e96fc82fa8e2ba246439666c9e8c8dc7e9e` with a dirty worktree. I changed only the final `.` to `?` in Go's `SpellDetectAlign` recipient output at `pkg/spells/affect_spells.go:361`. C emits `Your eyes tingle.` for Detect Alignment (`src/magic.c:1006-1012`). The failed block shows the promoted staff row can detect that byte change. The source was restored immediately after the retained census completed.

Both runs used the scenario's default seed 1 and enabled draw parity. Each retained manifest records the mutation worktree state, and both verdicts show no draw divergence. The candidate census paths above are the original clean, accepted proofs; the NOT_CLEAN runs are the new R5h mutation controls.
