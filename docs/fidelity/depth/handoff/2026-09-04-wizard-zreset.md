# Wizard `zreset` audit — 2026-09-04

## Scope

This slice continues the source-order wizard surface at
`src/act.wizard.c:2035-2076`, covering `do_zreset` and its command-table entry
at `src/interpreter.c:848`. The existing manifest proves only the empty-
argument response through `wizard-residual-depth`; this audit expands the
reachable selector, reset, output, and parser branches without re-counting the
shared `reset_zone()` implementation owned by the world-reset surface.

The C handler has four observable branch families: missing argument, `*` world
reset, `.` current-zone reset, and a numeric zone-number lookup using C
`atoi`/`one_argument`. It also has invalid-zone output and exact zone index,
number, and name formatting. The reset itself can mutate mobs, objects, doors,
and RNG state, so state/draw parity belongs at the verified `reset_zone()` call
path and must be delegated or separately blocked rather than inferred from the
handler's acknowledgement (R1/R3/R5b/R5c/R5e).

## Required evidence

- Read and cite `src/act.wizard.c:2035-2076`, `src/interpreter.c:848`, and the
  actual `src/db.c:2074-2195` reset call path before changing Go.
- Create a C-first vehicle that names every selector and parser boundary that
  can be safely observed, including `*`, `.`, numeric prefixes, invalid input,
  and trailing words. Use `--show-oracle` at seeds 1 and 2.
- Record the pre-fix transcript before implementation changes; distinguish
  acknowledgement-byte drift from shared reset state/RNG drift.
- Fix only confirmed divergences, preserve the shared reset ownership boundary,
  and record any reset-state or logging gap as delegated or blocked with a
  source-backed reason.
- Run `make fidelity-depth`, the focused oracle matrix, all repository build,
  vet, test, lint, formatting, and security gates before the implementation
  commit.

No C or oracle-tree files may be changed. The pre-existing untracked
`website/static/images/` directory remains outside this slice.

## C authority

`do_zreset` calls `one_argument`, refuses an empty first token with
`You must specify a zone.\r\n`, resets every table entry for `*` and reports
`Reset world.\r\n`, maps `.` through the actor's current room zone index, and
compares `atoi`'s result against each zone table number. A valid selected zone
reports `Reset zone %d (#%d): %s.\r\n`, where the first value is the zone-table
index. Any unmatched selector reports `Invalid zone number.\r\n`. The C handler
logs successful resets but does not expose the log to the issuing player.

