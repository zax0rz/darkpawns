# DP-1583 dispositions — checkpoint batch (act.comm.c → boards.c)

Status: **checkpoint (50 of 413 rows)**, awaiting Claude's spot-check before
the remaining C files are settled. Companion tables:
[`dp1583-dispositions.tsv`](dp1583-dispositions.tsv) and
[`dp1583-missing-cites.tsv`](dp1583-missing-cites.tsv). This run is docs-only:
no Go code, no manifests, no rulebook edits.

## Counts per disposition (checkpoint batch)

| disposition | rows |
|---|---:|
| ported-other-name | 44 |
| inlined | 4 |
| c-unreachable | 2 |
| go-native | 0 |
| go-only-surface-replaced | 0 |
| gap | 0 |
| ask-zach | 0 |
| **total** | **50** |

Missing-cites rows: **46** (one per `ported-other-name` Go function whose doc
comment carries no machine-resolvable C citation — `do_qcomm` contributes two,
`cmdQsay` and `cmdQecho`).

## Files covered

`src/act.comm.c` (1), `src/act.display.c` (3), `src/act.informative.c` (9),
`src/act.item.c` (1), `src/act.other.c` (1), `src/act.social.c` (3),
`src/act.wizard.c` (24), `src/ban.c` (1), `src/boards.c` (7).

## Method

- Every C citation in the tables was grepped or read this session from `src/`
  in this worktree (R5g); every Go citation likewise from `pkg/` (worktree at
  origin/main, 2026-11-19).
- The Go side was found by behaviour, not by name: each row's evidence names a
  shared message string (with `act.wizard.c:NNN = wiz_system.go:NN` style line
  pairs) or a shared branch, so a reviewer can check it in one step.
- `ported-other-name` vs `inlined`: a named Go function that does the C
  function's job is `ported-other-name`; C bodies folded into a caller's Go
  equivalent (no dedicated Go function) are `inlined`.
- **Missing-cites criterion:** the fnindex citation grammar
  (`tools/fnindex/main.go:45-51`) accepts `src/<file>.c:<line>`, `C name()`,
  `C: name()`, `<file>.c:name()`, "ported from name()", `` C `name` `` and
  `ACMD/SPECIAL/ASPELL(name)`. A `ported-other-name` row goes to
  `dp1583-missing-cites.tsv` when its Go doc comment cites none of these
  forms — even when it names the C in prose (`act.wizard.c do_zlist()`). That
  prose-only style is exactly why these rows appeared in the 413: the port
  exists, the citation is unresolvable. `inlined` rows are not listed there
  (the brief attaches the missing-cites list to `ported-other-name` only).
- `c-unreachable` rows cite a grep with no live callers, per the brief.

## Findings worth attention

- **`do_skills` is dead on the C side** (`src/act.informative.c:2689`): its
  only references are the definition and the forward declaration
  (`src/interpreter.c:237`); no `cmd_info[]` row registers it. The Go registry
  correctly has no `skills` command either. Not a gap.
- **`show_mult_obj_to_char` is dead on the C side**: its only caller,
  `list_obj_in_heap` (`src/act.informative.c:291`), is itself only referenced
  by a commented-out call (`src/act.informative.c:825`). Go's note-look path
  (`appendObjectLook`, `pkg/game/look.go:1165`) already covers the shared
  strings through the `show_obj_to_char` port; the mult-listing-only bits (the
  `[num]` stacking count and its `\n\r\n\r` terminator) have no live C path.
- **Four stale doc-comment banners** on Go functions whose bodies port other C
  functions (recorded in missing-cites; not edited, per brief):
  - `cmdTeleport` (`pkg/session/wiz_movement.go:289`) banners itself "load";
  - `cmdWizlock` (`pkg/session/wiz_system.go:193`) banners itself "cmdStat";
  - `cmdDate` (`pkg/session/wiz_system.go:267`) banners itself "cmdHome";
  - `cmdHome` (`pkg/session/wiz_movement.go:472`) banners itself "heal".
- **Citation-style drift is the dominant class, not missing ports.** In this
  batch every live C function has a Go implementation; the audit's actionable
  output is 46 citation normalizations, best done mechanically in one later
  pass (e.g. teach the index `do_X` ↔ `cmdX` per DP-1583's triage note, or
  rewrite the comments to `src/<file>.c:<line>` form).

## Candidate gaps

None in this batch.

## Decisions for Zach

None required in this batch: no `ask-zach` rows, no candidate-gap issues, no
rule questions. (If the remaining 363 rows follow this pattern, the one
decision worth making early is whether citation normalization should be a
mechanical bulk pass or folded into each function's next depth pass.)
