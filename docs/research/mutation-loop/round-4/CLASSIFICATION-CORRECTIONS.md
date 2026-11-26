# Round 4 classification corrections

- **Row 64 — frozen `ask-zach`; corrected to `c-reachable` after the census.**
  The all-mutants census finished at 18:43 on 2026-11-25; the ruling reached
  `main` at 19:18 (`a48baa25c`). The frozen `sample.tsv` keeps `ask-zach`. Zach's 2026-11-25 ruling in
  `docs/research/mutation-loop/DECISIONS.md` says overlapping C
  `sprintf(buf, "%s...", buf)` appends are classified by the pinned reference
  build's output. C `do_auto` appends this exact `Exits ` fragment at
  `src/act.other.c:1308-1312`; the CLEAN combined run
  `/home/zach/Archives/darkpawns/oracle-runs/2026-11-25/nightly-20261125`
  records it in `census-dump/player-state-toggles.txt`. The one-byte mutation
  and full-run commit are unchanged. The row was already in the denominator as
  `ask-zach`, so this correction did not change the denominator.
- **Row 92 — `c-reachable` affirmed under the NULL `%s` ruling.** C
  `do_clan_plan` formats the old-plan response at `src/clan.c:1427-1432`.
  Zach's 2026-11-25 ruling establishes that a NULL `%s` in the reference
  prints `(null)`; no class or denominator change is needed for this row.

- **Row 0 — C-site attribution corrected; class remains `c-reachable`.** The
  cited line at `src/spell_parser.c:868` is in `cast_spell`, not `do_cast`.
  The `lua-cast-spell-gates-depth` fixture reaches the default-position branch
  with a stunned peer, so the pinned C path can print the sampled bytes. The
  Go sample is in the session command gate; its isolated check is recorded in
  `probes.tsv`.
- **Row 27 — post-census call-path correction; frozen class remains
  `c-reachable`.** `src/interpreter.c:374` sets carve's minimum position to
  `POS_STANDING`; `command_interpreter` returns for a fighting character at
  `src/interpreter.c:939-948`, before C `do_carve` can reach the literal at
  `src/new_cmds.c:124-127`. The depth manifest records this proof in
  `docs/fidelity/depth/carve.tsv` (`carve.fighting-no-arg`). The post-result
  correction is recorded here, but does not revise the pre-census class or
  denominator. The row remains a non-kill in the frozen measurement.
- **Row 80 — post-census call-path correction; frozen class remains
  `c-reachable`.** C `do_clan_set` dispatches `clan set applev` to
  `do_clan_application` at `src/clan.c:1531-1535`; an empty remainder returns
  through `send_clan_format` at `src/clan.c:1273-1281`, before the sampled
  `Set to which level?` branch at `src/clan.c:1308-1310`. This post-result
  correction does not revise the pre-census class or denominator; the row
  remains a non-kill in the frozen measurement. It also corrects the original
  evidence's function name: `do_clan_application`, not `do_clan_sp`.

Rows 0, 27 and 80 have post-census evidence corrections. They do not revise
any pre-census class or denominator. The `sample.tsv` checksum in `SHA256SUMS`
remains the frozen draw checksum.
