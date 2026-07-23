# BRIEF (glm) — port the C output pager (DP-1195)

**Owner:** glm-5.2. **Gate:** Claude runs the differential oracle red→green; CI must
pass on the PR.
**Git:** branch off `main` as `glm/pager-port`, commit, push, open a PR against `main`.
Do NOT merge — review is Claude/Daeron's. Sized to one PR (M/L).
**Closes:** DP-1195. **Unblocks:** DP-1189 (help), credits/wizlist/immlist/socials ports.
**Cite:** `src/modify.c:346-530` (Buselli pager: next_page / count_pages /
paginate_string / page_string / show_string), `src/comm.c:610-620` (input routing
while paging), `src/comm.h:63` (`PAGE_LENGTH 22`; read `PAGE_WIDTH` there too);
rules **R1**, **R4** (`docs/fidelity/RULEBOOK.md`).

## The C truth (read the cited code first — it's short and complete)

- Output longer than PAGE_LENGTH (22) rendered lines is not sent whole: `page_string`
  splits it into pages via `next_page`, which counts lines AND wraps columns past
  PAGE_WIDTH, and is **ANSI-aware** — `\x1B...m` color sequences count zero columns
  (modify.c:368-374). Page splits must land exactly where C's counter lands them.
- While paging, the descriptor is in pager mode: **every input line routes to
  `show_string`, not the command interpreter** (comm.c:618). Navigation: RETURN =
  next page, `q`/`Q` = quit pager, `r`/`R` = refresh page, `b`/`B` = back one page,
  a number = jump to that page. Anything else prints:
  `Valid commands while paging are RETURN, Q, R, B, or a numeric value.`
- The page prompt (oracle-captured live, byte truth):
  `[ Return to continue, (q)uit, (r)efresh, (b)ack, or page number (1/2) ]`
  with current/total page numbers. Read show_string for exactly when/how it prints.

## Go implementation

1. **Pager state on the session** — mirror the existing pre-dispatch input modes:
   the telnet listener already routes input to char-creation/menu handlers before
   command dispatch (`pkg/telnet/listener.go` — the `IsCharCreating() || IsMenuActive()`
   fork). Add `IsPaging()` with the same shape: while active, input lines go to the
   pager navigator, never to `ExecuteCommand`.
2. **`PageString(s, text)`** in pkg/session: if the rendered text ≤ one page, send it
   whole (C behaves this way — verify in page_string); else store pages + enter pager
   mode + send page 1 + prompt. Port next_page's counting rules exactly (line count,
   PAGE_WIDTH column wrap, ANSI-skip).
3. **Navigator**: RETURN/Q/R/B/number semantics per show_string, including the
   invalid-input message and what happens after the last page (read the C — does it
   auto-exit or wait for q? Mirror it).
4. **Convert `levels` (cmdLevels or equivalent) to PageString** — the proof-gate
   command. Do NOT convert other commands in this PR (credits/wizlist/etc. also need
   content work — separate briefs); note any obvious candidates you see in the PR
   description instead.
5. **WebSocket/GMCP clients**: pager is a telnet-surface behavior. Gate it so
   structured-data clients (`wantsStructuredData` / agent sessions) are unaffected —
   look at how existing telnet-only behaviors are gated and match; state your choice
   in the PR description.

## Tests (unit)

- next_page parity: a 23-line string → 2 pages, split at line 22; a string with ANSI
  codes crossing the width boundary → codes don't count toward columns; a long
  unbroken line wraps at PAGE_WIDTH and counts as extra lines
- Navigator: RETURN advances, B from page 1, R re-sends, number jump in and out of
  range, Q exits, junk input → the exact Valid-commands line; input routing: a
  command word typed while paging is pager input, NOT a command
- levels: >22-line output enters pager mode; ≤1 page output (if constructible) sends
  whole with no prompt

## Oracle gate (Claude authors/runs — provide sketches in the PR)

New scenario `pager-navigation.txt`: `levels` → `<ENTER>` (page 2) → `r` → `b` → `7`
(if in range) → junk word → `q` → then a normal command to prove clean pager exit.
RED on pre-fix main (Go dumps all 30 levels, then treats nav input as commands);
GREEN on the branch. Claude will also un-quarantine `levels` in
`act-informative-sweep.txt` at gate time.

## Guardrails

- **Never** edit `src/` or `darkpawns-c-oracle/` — reference only.
- The telnet write path has locking discipline (`wmu` etc.) — read neighboring code
  before adding writes; no new goroutines for the pager.
- `make reachability` zero regressions; build/vet/test/lint green.
- Don't stage `website/static/map/world-sphere.json` or `docs/reports/reek/*`.

## Deliverable

Pager engine + levels converted + tests + scenario sketch + the WebSocket-gating
decision documented in the PR description. Claude reconciles + runs the oracle gate.
