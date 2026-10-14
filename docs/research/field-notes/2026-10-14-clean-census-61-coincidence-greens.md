# 2026-10-14: A CLEAN census with 61 coincidence greens

## Scope

- **Tracks:** legacy-port fidelity (what a passing differential test proves)
- **Window:** 2026-10-14
- **Primary artifacts:** `~/Archives/darkpawns/oracle-runs/2026-10-14/draw-parity-corpus-1/` (`summary.txt`, `results.tsv`, `census.log` with every `draw-parity` line, `draw-scenarios.txt`, `mismatch-clusters.txt`, `TRIAGE.md`); PR #1967 (DP-1427 draw-parity mode); oracle seam PR zax0rz/darkpawns-c-oracle#5; reference oracle `bd6c548c…`
- **Status:** verified. Counts come straight from the retained census results.

## What was observed

The first corpus-wide run of the new draw-parity mode compared not only each scenario's transcript bytes but the full sequence of RNG draws the C oracle and the Go port make. Same corpus (1,169 scenarios), same seed (1), same reference binary family as every census before it.

| Measure | Result |
|---|---|
| Byte verdict | **CLEAN**: 1,169 scenarios, 0 failed, 0 infrastructure |
| Draw parity | **61 scenarios (5.2%) diverge**: identical transcripts, different dice |

So about one in twenty "proven" scenarios was green on a stream that had already drifted. The bytes matched because the drift hadn't yet reached a roll whose outcome showed, or because both sides' different rolls happened to produce the same text (as with PF-067's seed 1).

The mismatches cluster. **22 of 61 (36%) are one cause.** In setup, one side draws the scavenger gate `number(0,10)` (`src/mobact.c:104`) where the other draws the wander door `number(0,18)` (`:120`). Within a single mob, C always draws the two in that fixed order. A swap therefore means **C and Go process mobs in a different order**: C walks the prepend-ordered `character_list`, and Go walks a sorted slice. That was already filed as DP-1364 but never had evidence of its reach. Smaller clusters point at missing combat- or skill-sized draws (`number(1,20)`, `number(1,101)`) late in probe steps.

## Why it matters for the fidelity paper

1. **A CLEAN differential verdict was an upper bound on fidelity, not a measurement of it.** The project's "a green must be able to fail" rule (R5h) was enforced one proof at a time; this is the first corpus-scale measurement of how often green could not fail. For an RNG-heavy legacy system, comparing outputs leaves a measurable blind spot. Comparing the hidden state that determines future outputs (here, the draw stream) closes much of it.
2. **The blind spot is structured, not random.** One ordering bug accounts for over a third of it, which suggests coincidence greens are cheap to remove at scale once you can see them.
3. **Combined with the `quiet-mobs` finding** (98% of scenarios run in an empty world), this reframes "done": the manifest measured fidelity mostly in conditions where whole subsystems never ran, and even there 5% of greens were coincidental.

## Open

- Fix DP-1364 and re-run. The prediction to test: the 22-scenario cluster disappears and some other clusters shrink, because they were later views of the same drift.
- Populated-world soak scenarios (Luna's living-world goal) will measure the blind spot where it is largest.
