# BRIEF (glm) — port the C help system (DP-1189)

**Owner:** glm-5.2. **Gate:** Claude runs the differential oracle red→green; CI green.
**Git:** branch off `main` as `glm/help-system`, commit, push, open a PR. Do NOT
merge. Sized to one PR (M/L).
**Closes:** DP-1189. **Depends on:** the pager (merged, `PageString`).
**Cite:** `src/act.informative.c:1566-1674` (do_help), `src/db.c` (`load_help`,
`index_boot` help branch — read them), `lib/text/help/` (the 2010 data, already
in-repo: `index`, `index.mini`, `screen`, `*.hlp`); rules **R1**, **R4**
(`docs/fidelity/RULEBOOK.md`).

## The C truth

**Boot:** `index_boot` reads `lib/text/help/index` (one .hlp filename per line,
`$`-terminated) and `load_help` parses each file into `help_table` entries
(keyword line(s) + entry text — read load_help for the exact record format,
including multi-keyword lines and the terminator). The table is keyword-sorted.
The no-arg help screen is a separate whole file (`lib/text/help/screen` — verify
which file C's `help` global loads; grep db.c for it).

**do_help, exactly:**
1. no argument → `page_string` the help screen. Through the pager.
2. no help_table → `No help available.\r\n`
3. binary search on `strn_cmp(argument, keyword, strlen(argument))` — PREFIX
   match — then **backtrack to the FIRST matching entry** (`while mid>0 &&
   still-matches: mid--`, the "Jeff Fink" loop). Order among same-prefix
   keywords matters; sort must reproduce C's (see load_help/sort in db.c).
4. mortal + entry contains "wizonly" → `There is no help on: %s\r\n` (same as
   a miss — the entry's existence is hidden).
5. hit display, byte-exact:
   - green, then `\r\n[ ` cyan TOPIC-UPPERCASED green ` ]\r\n` normal
   - red separator: 75 dashes (two concatenated C string chunks — count them
     in the source, don't trust this brief), then normal
   - entry text AFTER its first line (C skips to past the first `\n` — the
     keyword line), via `PageString`
6. miss → `There is no help on: %s\r\n`. C also mudlogs and appends to a
   `misc/help` usage file — server-side only, NOT player-facing: skip the file
   write, note the skip in the PR.

## The Go replacement

`cmdHelp` (`pkg/session/cmd_info.go:858`) is a wholesale invention: fabricated
topic overview, `helpTopics` map, registry-description fallback, invented
unknown-topic text. **All of it goes.** R4: `help <anything>` serves ONLY what
the help files serve. Delete the helpTopics map and the registry fallback —
if a command has no help entry, C says "There is no help on:" and so do we.
(Keep the `?` registration from #420 — it dispatches here.)

Boot the help table once at world load (wherever lib/text loading lives in Go —
follow how other lib data is loaded), from the SAME in-repo files. `index` vs
`index.mini`: C picks by mini-mud mode; port the normal path, note the mini one.

## Tests

- loader: entry count vs the .hlp files; a known multi-keyword record resolves
  under each keyword; sort order reproduces a known same-prefix pair
- prefix + first-match: `help s` resolves to whatever C's table order says (fix
  the expectation from the parsed table, not from guessing)
- wizonly hidden for mortals; visible for level ≥ LVL_IMMORT
- no-arg → screen content, paginated (>22 lines)
- miss → exact message

## Oracle gate (Claude authors/runs; sketches in the PR)

Probes: `help say` (the [ SAY ] page from the 2026-07-22 sweep transcript —
including the `'` shorthand paragraph), `? cure light`, `help zzqx` (miss line),
bare `help` → first page + pager prompt + `q`, `help commands`. RED today (Go
prints registry one-liners / invented overview); GREEN on the branch. The
`? say` quarantine note in act-informative-sweep.txt gets retired at gate time.

## Guardrails

- **Never** edit `src/` or `darkpawns-c-oracle/` — reference only. The in-repo
  `lib/text/help/` IS the 2010 data: read it, never rewrite it.
- `make reachability` zero regressions; build/vet/test/lint/gofumpt green.
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.

## Deliverable

Loader + faithful do_help + deletions of the invented surface + tests + scenario
sketches + the index.mini note. Claude reconciles + runs the oracle gate.
