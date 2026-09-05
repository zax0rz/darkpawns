# Queue correction — 2026-09-02 — after `scrounge`

This note supersedes only the next-frontier sentence in
`2026-09-02-command-scrounge.md`. The source-order sweep rechecked the
existing manifests before taking the next slice and found that `sell` is
already claimed by the combined `do_not_here` row
`docs/fidelity/depth/do-not-here.tsv:4` (`list/buy/sell`, proof
`shop-do-not-here`). It must not be re-picked.

The next genuinely un-manifested command family is `send` at
`src/interpreter.c:681`, registered as `POS_SLEEPING` with `do_send` and
`LVL_GOD`. `scream` is also already accounted for by shared social evidence,
as the scrounge handoff states. No implementation slice is claimed by this
correction; begin `send` only after a fresh main checkout/pull, frontier
validation, depth-guide read, newest-handoff read, and C-table/manifest audit.

The scrounge feature and its original handoff remain complete at the counts
recorded there: 3147 total, 3069 proven/delegated, 26 blocked, and 52
excluded. This correction preserves the same queue and fidelity rules:
R1/R2/R3/R4/R5e, with shared ownership bounded by R5b/R5c.
