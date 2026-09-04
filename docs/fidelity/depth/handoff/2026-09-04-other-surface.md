# Other activity-surface audit — 2026-09-04

## Scope

This slice audits the 162 literal `act()`/`send_to_char()` call sites in
`src/act.other.c`, in source-file order. The file contains communication and
session commands plus save, practice, visibility, title, grouping, utility,
mount, theft, recall, stealth, appraise, inactivity, scouting, and roll paths.

## Existing ownership

Existing depth manifests already own substantial portions of this file:
`comm.tsv`, `quit.tsv`, `reallyquit.tsv`, `save.tsv`, `do-not-here.tsv`,
`sneak.tsv`, `hide.tsv`, `practice`-related guild rows in `spec-procs.tsv`,
`group.tsv`, `report.tsv`, `split.tsv`, `use.tsv`, `display.tsv`, `gen-write.tsv`,
`gen-tog.tsv`, `afk.tsv`, `auto.tsv`, `eat.tsv`, `mount.tsv`, `dismount.tsv`,
`peek.tsv`, `recall.tsv`, `appraise.tsv`, `inactive.tsv`, `scout.tsv`, and
`roll.tsv`. This inventory is a call-site ownership audit, so each literal
must still map to a focused proof row or to an explicit C-reason exclusion.

## Protocol and open work

The slice begins with the standard two-seed depth protocol and a 300-second
per-scenario timeout. No C or oracle source may be modified. The remaining
direct families requiring evidence or sharper classification include theft,
practice and visible gates, title, ungroup, wimpy, transform, stealth, yank,
and any residual branches in ride/recall/utility handlers. Existing focused
rows will be rechecked in source order; reachable reds remain blocked rather
than being absorbed into a broad green claim. The inventory row will be
promoted only after all 162 literals have explicit proof ownership, a blocked
classification, or a documented C-reason exclusion.
