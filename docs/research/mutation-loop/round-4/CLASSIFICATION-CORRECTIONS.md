# Round 4 classification corrections

- **Row 64 — frozen `ask-zach`; corrected to `c-reachable` before any mutant
  census verdict.** Zach's 2026-11-25 ruling in
  `docs/research/mutation-loop/DECISIONS.md` says overlapping C
  `sprintf(buf, "%s...", buf)` appends are classified by the pinned reference
  build's output. C `do_auto` appends this exact `Exits ` fragment at
  `src/act.other.c:1308-1312`; the CLEAN combined run
  `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/nightly-20261125`
  records it in `census-dump/player-state-toggles.txt`. The one-byte mutation
  and full-run commit are unchanged. The row was already in the denominator as
  `ask-zach`, so the denominator remains 93; the effective classes are 93
  `c-reachable`, 7 `c-unreachable`, and no `ask-zach` rows.
- **Row 92 — `c-reachable` affirmed under the NULL `%s` ruling.** C
  `do_clan_plan` formats the old-plan response at `src/clan.c:1427-1432`.
  Zach's 2026-11-25 ruling establishes that a NULL `%s` in the reference
  prints `(null)`; no class or denominator change is needed for this row.

These corrections were recorded before the Round 4 mutant census verdict was
read. The `sample.tsv` checksum in `SHA256SUMS` remains the pre-classification
draw checksum; the original classified commit is preserved in Git history.
