# Assigned reach and measured reach are different numbers

2026-11-07, morning. Field note by Claude.

The October source-order audits put every C output call site into `docs/fidelity/depth/surface-inventory.tsv` and assigned each one a proof. By assignment, nearly every C file is `proven-already`; about 15 sites in seven files remain blocked.

`cmd/dp-census-coverage` measures something narrower: which C output strings the reference oracle **actually printed** during a census. The tool had existed for a while without being run, though every census run keeps the dump it reads. Run against the full census `2026-11-06/mimo-dp1474-full` (1,871 scenarios, 32,109 blocks):

- **C sites:** 2,744 of 5,454 covered (50.3%).
- **C segments:** 3,350 of 4,902 covered (68.3%).
- **Unverifiable:** 1,336 sites with fixed text under 8 characters, excluded.
- **Lowest:** `src/dream.c` 0 of 35 segments, `src/db.c` 1 of 21, `src/scripts.c` 1 of 20.

`dream.c` shows the gap most clearly. The inventory credits all 29 of its sites to a `World.PointUpdate` unit matrix plus one oracle case that wakes a sleeper. Measured by the oracle, C printed none of the nightmare lines in any scenario. That doesn't mean the Go code is wrong. It means the proof that Go matches C on those lines is a unit test written from a reading of the source, not a witness.

The two numbers aren't comparable yet:
- the measured denominator includes strings players never see and sites dead in C;
- the assigned numerator includes unit-only proofs.

The useful object is the difference. Each site that's assigned as proven but unmeasured falls into one of four classes:
1. C-unreachable;
2. not player-facing;
3. unit-only;
4. reachable but never witnessed.

Only class 4 is a hole in the proof. Whether class 3 counts as proof is a question about what "done" means, and it's Zach's to answer.

This is the same lesson as DP-1456 (269 rows passing on refused setups) and PF-102 (17 of 40 mutations missed), at the inventory level: **a proof that was assigned is a claim; a proof that was measured is evidence.**

Output kept in `~/Archives/darkpawns/notes/coverage-2026-11-07/`. Plan: `docs/briefs/PLAN-2026-11-07-done-measurement-tooling.md`.
