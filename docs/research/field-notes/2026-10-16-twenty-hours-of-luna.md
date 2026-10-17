# Twenty-six hours of one agent on the surface map

*Field note, 2026-10-16. Claim PF-075. First written at hour 20 on the batch-o merge; amended when the goal finished, with final measurements taken on the batch-r merge (`claude/batch-2026-10-16r`, Luna's last three PRs), whose census came back CLEAN; these are current-main facts as of `60c15a30f`.*

On 2026-10-15 Luna (Codex) started one long goal: map every `act()`/`send_to_char()`/`stc()` call site of the C source, prove each one through real command dispatch, and fix the Go where the proof failed (DP-1371). It finished after **26 hours 20 minutes**.

## What it produced

Final, at goal completion:

- **66 pull requests**, #1991 to #2082, opened between 15:54Z on 10-15 and 16:38Z on 10-16. 62 merged through Claude's batched combined censuses (batches k through r), 3 were in the final batch, and 1 was closed.
- **Per-site maps: 41 maps covering 2,228 C output calls**, now **1,903 proven (85.4%)**, 58 gap-reachable, 77 gap-blocked and 190 unreachable with a C reason. At hour 20 it was 1,731 proven (78%): the last six hours mostly turned mapped gaps into proofs.
- **Blocked inventory units: 4,031 → 2,106.** Same definition as the PF-070 baseline (`status = blocked` in `surface-inventory.tsv`). The inventory total fell too, 4,929 → 3,004, because per-site maps replace family-level estimates with exact counts. Most of the remaining blocked units sit in families that have no per-site map yet, so they are estimates, not counted sites.

Hour-20 snapshot, kept for comparison: 48 PRs, 2,280 blocked units, 1,731 of 2,228 sites proven.

## What it found

The maps are bookkeeping; the point is what proving each site turned up. Examples, all player-visible:

- Go text with no C source, removed (R4): "That only works on players." (vampirism), a "You feel informed" preamble (identify), a Go-only bribe path.
- Inverted logic: the cityguard bribe roll refused on the wrong outcome of `number(0, 2)`.
- Swapped tables: `set class` mapped `a`/`v` to Avatar/Magus where C maps them to Magus/Avatar.
- Missing audiences: shop buy/sell room lines and keeper tells were Go summaries ("You buy X for N gold pieces.") instead of C's three messages.
- Wrong constants: mail needed level 5 and a 50-coin stamp; C's `MIN_MAIL_LEVEL` is 2 and `STAMP_PRICE` 25 (#2076).
- Dead code that looked alive: held lights never burned out, because Go's `UpdateCharObjects` read the wrong slot and nothing called it (#2061; the worn/carried timer half is DP-1447).
- An invented permission gate: removing a board post required the board's read level, a check C's `Board_remove_msg` does not have (#2062).

It also withdrew its own false greens: an earlier clean meteor-swarm run stopped at `do_cast`'s peaceful gate and never reached the claimed C site; shift-reality and call-of-chaos fixtures used the wrong spell names and stopped at "Will what?!?". It recorded those runs as excluded rather than keeping the green. That is R5h applied by the agent to its own evidence.

## What review still caught

Review was not a formality. Of the 66 PRs, at least seven needed a correction before merge, about one in ten, and the rate held steady through the run. Every one sat at a place the oracle cannot see, and each was found by checking the C call path rather than the transcript:

- #2002 and #2006: surface maps in the wrong format (a missing `src/` prefix; case ids joined with commas), which the map checker now rejects.
- #2017: a scenario that never ran (unsupported fixture flag; its mob lives in an unindexed `.mob` file, so it could not exist in either server).
- #2036: a zero-coin bribe skipped `perform_give_gold`'s checks, so a cityguard attacked for 0 coins.
- #2053: a scenario that pinned C's undefined behaviour (`spell_conjure_elemental` reads a NULL pointer) as an ordinary census row; retired from the corpus under R1a.
- #2054: `act()` appends its own CRLF, so "There is no corpse!" ends in a blank line. The oracle trims trailing blank lines, so its scenario could not see this.
- #2056: colour gated on both colour bits where C's `C_NRM` needs only `PRF_COLOR_2`, plus an unchecked `affected_bits[]` index that could panic the server.

Separately, #2007's correct fix (smokescreen now really blinds) exposed three neighbouring Go paths that ignored visibility (DP-1446). That was not an error in the PR; the fix made existing errors observable.

## What made it possible

This run stands on an earlier one. In late August and early September, a single long Luna session (per Zach, more than five days) walked the port from breadth to depth:

- Breadth first: every registered command matched C at least once (PF-004, 2026-08-23).
- Then depth, in source and command-table order: 43 PRs (#687 to #731) took the modeled depth ledger to 855 of 861 actionable cases (PF-006, MA-005, [2026-08-28 milestone](2026-08-28-fidelity-depth-milestone.md)).
- It ended in the 2026-09-04 terminal audit: 4,758 per-case rows, 98.9% of actionable cases proven or delegated (DP-008). That audit also introduced `surface-inventory.tsv` as a second, explicit denominator: the C output surface that per-case depth did not cover.

The 26-hour run is the continuation of that work against the second denominator. It moved fast because the first one had already built what it needed: per-case manifests to cite, the scenario and fixture vocabulary, real-dispatch proof habits, and an inventory of exactly what was left. Without the five-day walk there would have been no map to fill in.

## Why it matters

The throughput is the obvious number, but the more useful observation is the shape: the agent produces proofs with failing controls, and the reviewer's job narrows to the places the oracle cannot see: line endings it normalizes, fixtures that silently no-op, and call paths the scenario never reaches. Both halves were needed. This is the harness-not-model point of the slop-squash thesis again: the same class of model that wrote the invented messages removed them, once the process demanded a failing control for every claim.

## Addendum, 2026-10-17: the second goal

Luna's follow-up goal (fight-skill reconciliation, `fight.c` and the `lib/misc/messages` corpus, the spell matrix, the last activity gaps) ran 20h 10m 44s by its own handoff (`docs/fidelity/depth/handoff/2026-10-16-fight-and-spell-surface.md`). It ended with blocked inventory units at 494 of 1,392 on main `c2cd1e004`, down from this note's 2,106: 1,612 fewer. The spell matrix (`surface/spell-matrix.tsv`) records 96 proven cells, 70 delegated, 348 reachable gaps, 1 blocked gap and 980 cells C cannot reach. Its last work landed through Claude's batches t to w, each a combined draw-parity census that came back CLEAN_AFTER_RECHECK (PF-076 and PF-077 record the intermediate projections; PF-078 the final one).

Review caught one new class in this goal. #2088's later commits put 43 targeted-census **run names** (`dp1371-groinrip-skill-message-seed5` and similar, 108 tokens across six manifests) in `proof` columns, where the claims census expects scenario names. Every gate passed, because none checks that a proof token names a scenario file. The batch-v combined census died at startup on the first missing file (`~/Archives/darkpawns/oracle-runs/2026-10-17/batch-v/census.log`). The run names moved to the rows' notes (`cdbb7d64f`); batch-v2 ran clean. The goal brief asked for per-seed proof and did not say where run names belong: an instruction gap, not a fidelity error. A depth-gate check that every proof token resolves to a scenario or `Test*` symbol is proposed to Zach.

Two goals also mapped the same C file in parallel PRs twice (#2091 against #2088 for `fight.c`, #2092's skill-message matrix), and the batches dropped the duplicates by hand. The proposed steer is to check open PRs and `origin/main` for an existing map before starting a surface.

## Open action items

- **Recover the five-day session's own record.** The repository fixes the late-August work by PR range and dates, but not that it was one Luna session lasting more than five days; that comes from Zach. The session has not been deleted. Find it, then cite its start and end times, turn count, and any compaction or handoff points here and in PF-075's evidence, replacing "per Zach".
- ~~Amend PF-075 when the current run ends.~~ Done 2026-10-16 (this revision).
