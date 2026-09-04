# Depth-fidelity terminal audit — 2026-09-04

## Initial handoff

This terminal-audit slice starts from `origin/main` at `2a4e13b86`, after the
final `yuball` social slice. The checked-out repository has no tracked
changes; the pre-existing untracked `website/static/images/` directory is
preserved and is outside this fidelity scope.

The per-case depth generator reports 4,758 total cases: 4,653
proven/delegated, 54 blocked, and 51 excluded. The separate
`surface-inventory.tsv` enumerates 70 rows and 4,926 weighted units; 8 rows
are `proven-already`, 61 are still `unproven`, and 1 is
`excluded-with-C-reason`. It is intentionally outside the generated case
count, so this slice audits it as a second, explicit denominator.

The terminal pass will record current-main reruns for the Phase 2 reds,
recheck the exclusion decisions and blocked clinic, then process the surface
inventory in source order. Each residual surface row will end as proven,
blocked with a named owner/evidence boundary, or excluded with an explicit C
reachability reason. No real player-visible gap will be relabeled as an
exclusion (R1/R2/R4/R5e).

No C oracle or `src/` file is editable in this slice. This file is committed
before the evidence and inventory edits, as required by the depth-loop
handoff protocol.
