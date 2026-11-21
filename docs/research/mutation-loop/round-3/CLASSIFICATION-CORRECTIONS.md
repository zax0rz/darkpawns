# Round 3 classification corrections

- **Rows 30 and 40 — C evidence citations corrected before scoring.** Their frozen class remains `c-reachable`; no census results informed these edits. Row 30’s exact sampled wording is the `death_cry` literal at `src/fight.c:524`, not the preceding `$n` act at `:517`. Row 40’s sampled wording is in `do_transform` at `src/act.other.c:1475`; `:1439` is the separate “into a werewolf” message. The citations in `sample.tsv` and `probes.tsv` now point to the matching C lines.
