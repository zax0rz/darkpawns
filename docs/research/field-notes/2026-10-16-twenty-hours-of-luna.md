# Twenty hours of one agent on the surface map

*Field note, 2026-10-16. Claim PF-075. Measurements taken on the batch-o merge (`claude/batch-2026-10-16o`), whose census was still running when this was written; they become current-main facts only if batch-o merges.*

On 2026-10-15 Luna (Codex) started one long goal: map every `act()`/`send_to_char()`/`stc()` call site of the C source, prove each one through real command dispatch, and fix the Go where the proof failed (DP-1371). Twenty hours later it was still running.

## What it produced

- **48 pull requests**, #1991 to #2060, opened between 15:54Z on 10-15 and 05:10Z on 10-16. 38 merged through Claude's batched combined censuses; the other 10 are in batch-o.
- **Blocked surface units: 4,031 → 2,280.** Same definition as the PF-070 baseline (`status = blocked` in `surface-inventory.tsv`). The total fell too, 4,929 → 3,178, because per-site maps replace family-level estimates with exact counts.
- **41 per-site maps covering 2,228 C output calls**: 1,731 proven (78%), 244 gap-reachable, 65 gap-blocked, 188 unreachable with a C reason.

## What it found

The maps are bookkeeping; the point is what proving each site turned up. Examples, all player-visible:

- Go text with no C source, removed (R4): "That only works on players." (vampirism), a "You feel informed" preamble (identify), a Go-only bribe path.
- Inverted logic: the cityguard bribe roll refused on the wrong outcome of `number(0, 2)`.
- Swapped tables: `set class` mapped `a`/`v` to Avatar/Magus where C maps them to Magus/Avatar.
- Missing audiences: shop buy/sell room lines and keeper tells were Go summaries ("You buy X for N gold pieces.") instead of C's three messages.

It also withdrew its own false greens: an earlier clean meteor-swarm run stopped at `do_cast`'s peaceful gate and never reached the claimed C site; shift-reality and call-of-chaos fixtures used the wrong spell names and stopped at "Will what?!?". It recorded those runs as excluded rather than keeping the green. That is R5h applied by the agent to its own evidence.

## What review still caught

Review was not a formality. Of the 48 PRs, at least five needed a correction before merge, all found by checking the C call path rather than the transcript:

- #2017: a scenario that never ran (unsupported fixture flag; its mob lives in an unindexed `.mob` file, so it could not exist in either server).
- #2036: a zero-coin bribe skipped `perform_give_gold`'s checks, so a cityguard attacked for 0 coins.
- #2054: `act()` appends its own CRLF, so "There is no corpse!" ends in a blank line. The oracle trims trailing blank lines, so its scenario could not see this.
- #2056: colour gated on both colour bits; C's `C_NRM` needs only `PRF_COLOR_2`.
- #2007's real fix (smokescreen now blinds) exposed three neighbouring Go paths that ignored visibility (DP-1446).

## What made it possible

This run stands on an earlier one. In late August and early September, a single long Luna session (per Zach, more than five days) walked the port from breadth to depth:

- Breadth first: every registered command matched C at least once (PF-004, 2026-08-23).
- Then depth, in source and command-table order: 43 PRs (#687 to #731) took the modeled depth ledger to 855 of 861 actionable cases (PF-006, MA-005, [2026-08-28 milestone](2026-08-28-fidelity-depth-milestone.md)).
- It ended in the 2026-09-04 terminal audit: 4,758 per-case rows, 98.9% of actionable cases proven or delegated (DP-008). That audit also introduced `surface-inventory.tsv` as a second, explicit denominator: the C output surface that per-case depth did not cover.

The 20-hour run is the continuation of that work against the second denominator. It moved fast because the first one had already built what it needed: per-case manifests to cite, the scenario and fixture vocabulary, real-dispatch proof habits, and an inventory of exactly what was left. Without the five-day walk there would have been no map to fill in.

## Why it matters

The throughput is the obvious number, but the more useful observation is the shape: the agent produces proofs with failing controls, and the reviewer's job narrows to the places the oracle cannot see: line endings it normalizes, fixtures that silently no-op, and call paths the scenario never reaches. Both halves were needed. This is the harness-not-model point of the slop-squash thesis again: the same class of model that wrote the invented messages removed them, once the process demanded a failing control for every claim.

## Open action items

- **Recover the five-day session's own record.** The repository fixes the late-August work by PR range and dates, but not that it was one Luna session lasting more than five days; that comes from Zach. The session has not been deleted. Find it, then cite its start and end times, turn count, and any compaction or handoff points here and in PF-075's evidence, replacing "per Zach".
- **Amend PF-075 when the current run ends.** Final PR count and hours, final blocked surface units against the 4,031 baseline, and how many PRs needed a review correction.
