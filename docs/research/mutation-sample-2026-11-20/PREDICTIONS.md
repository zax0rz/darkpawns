# Mutation sample 3: predictions, frozen before the draw

Written by Claude and accepted by Zach on 2026-11-19. This was committed before
DP-1596 merged and before sample 3's population was collected or its seed
(`20261120`) was drawn. Nobody has seen any sample 3 result. Don't edit this
file after the draw. The results note scores each prediction as held, partly
held or failed.

The basis is two earlier samples:
- PF-123: 31/60 caught over all mutants;
- PF-129: 42/60 caught over all mutants, and 42/47 on lines C can print.

These are judgments, not measurements.

Definitions are from `~/PROMPT-2026-11-19-mutation-sample-3.md`:
- **Headline:** killed / (c-reachable + go-only-in-game + ask-zach), with a
  95% Wilson interval.
- **The bar:** the lower bound at or above 80%. With n = 100 and no
  exclusions, that needs 88 kills.

## Predictions

1. **The bar is not met on this sample.** The lower bound comes in under
   80%. Claude's confidence is about 2 in 3.
2. **The headline point estimate is between 80% and 92%.**
3. **15 to 25 of the 100 are excluded**, as c-unreachable, go-only-surface or
   harness-blind. Sample 2 had 13 of 60 that could never be killed.
4. **Most catchable misses are on the mob side.** More than half of the
   survivors in the denominator are lines printed by or about mob actions:
   mob item use, mob movement, specials, mid-combat messages. All four of
   sample 2's repaired misses were like this.
5. **At least one survivor is a real Go divergence**, filed as a DP- issue,
   not just a missing scenario. Sample 2 found DP-1585 and the force-all draw
   fix.
6. **Claude's spot check finds at least one class verdict wrong.**
7. **At most 2 are harness-blind.**
