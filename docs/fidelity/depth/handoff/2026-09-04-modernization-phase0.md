# Modernization queue Phase 0 handoff — 2026-09-04

This round starts from `origin/main` at `d61ce0c6a`, after the terminal depth
audit. The checkout retains the pre-existing untracked `website/static/images/`
directory; it is outside this queue and must not be touched.

The authoritative modeled depth corpus is 4,758 cases: 4,653
proven/delegated, 54 blocked, and 51 excluded. The separate
`docs/fidelity/depth/surface-inventory.tsv` has 70 rows and 4,926 weighted
units: 8 `proven-already`, 61 `blocked`, and 1 `excluded-with-C-reason`.
Those are separate denominators under R1/R2/R4/R5e.

Phase 0 scope for this branch:

- add `make oracle-regression` with a 240-second per-scenario budget and
  wall-time reporting for the full scenario corpus;
- verify the 15 social scenarios are already present and green on current main;
- adjudicate the shadow-shop-stack wiring before any shop refactor;
- correct stale modernization baseline documentation to the 4,758-case
  snapshot; and
- preserve the 61 surface rows as explicit blocked/excluded/proven outcomes,
  never as a blanket proof.

The first commit in this round is this handoff, per the depth-loop process.
No C oracle source or `src/` file is editable. Any behavior change remains
oracle-gated under R1–R5e; RED-set files remain human-only for modernization.
