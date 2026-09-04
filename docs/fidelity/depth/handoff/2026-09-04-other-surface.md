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

## Findings and closure

The existing two-seed vehicles for practice, stealth, visible's ordinary
already-visible branch, and the residual ride/recall/utility paths remained
green. New two-seed vehicles for theft, title, ungroup, wimpy, and transform
also matched the C oracle. The focused title vehicle confirmed that the live
direct-send title acknowledgement preserves doubled-dollar bytes; Go no
longer collapses `$$` on this path (R1/R5e).

Three small parser/state gaps were closed in Go: wimpy now applies C's
first-byte `isdigit` gate and `atoi`/`one_argument` behavior; steal parses
separate C object and target operands and resolves the target in the visible
room; and ungroup uses C's room target lookup plus actor/victim/observer
stop-following messages. `visible` now removes both ordinary sneak and ninja
stealth affect records. `transform` now follows the C werewolf-first and
vampire state machines, including weather/moon gates, exact bytes, resource
bonuses, and the 666 HP clamp. Focused unit tests pin the new parser/state
boundaries (R1/R2/R3/R5e).

The 162-call-site inventory row is promoted to `proven-already`. The new
`docs/fidelity/depth/other.tsv` owns 43 focused rows: 31 oracle-green rows
across seeds 1 and 2, 3 delegated shared rows, 9 unit rows, and no new
exclusion. Existing
blocked rows for reachable vampire/werewolf item branches remain blocked in
`drink.tsv` and `eat.tsv`; they are not hidden by this surface classification.
The depth report after this slice is 4,164 total cases, 4,067
proven/delegated, 46 blocked, and 51 excluded; the weighted activity-surface
inventory is 4,925 units, with 2,954 literal activity call sites plus 1,495
spell-matrix cells, 463 fight/skill units, 10 lifecycle units, and 3 shop
units. The remaining unproven inventory rows continue in source-file order
after this handoff.
