# BRIEF (mimo) — do_gen_ps: the 2010 static text, verbatim

**Owner:** mimo-v2.5-pro. **Gate:** Claude runs the differential oracle red→green.
**Git:** NONE — isolated worktree, no git commands. Operator commits.
**Closes:** the credits/wizlist/immlist slice of the missing bucket. Sized: M.
**Cite:** `src/act.informative.c:2120-2170` (do_gen_ps — read the whole switch),
`src/interpreter.c` gen_ps rows (:385-:833), `src/db.c:131-190`
(file_to_string_alloc boot caching); rules **R1**, **R4**; Zach's directive:
static text is frozen at the 2010 shutdown state.

## The data (step 1, before any code)

Copy these files from `/Users/zach/.openclaw/workspace/darkpawns-c-oracle/lib/text/`
into the repo's `lib/text/` **byte-for-byte** (read-only source — never edit the
oracle checkout): `credits`, `news`, `info`, `wizlist`, `immlist`, `handbook`,
`policies` (check exact filename in db.c's *_FILE defines), `motd`, `imotd`,
`background`, `future`. Skip the `.bak`/`.old`/`.save`/dated variants. If a
define points at a file the oracle lib lacks, note it and skip.

## The C truth

- Boot: `db.c` caches each file into a string via `file_to_string_alloc`.
- `do_gen_ps` dispatches by subcmd: most cases `page_string` the cached text
  (VERIFY per case — read the switch; `SCMD_CLEAR` sends a clear-screen escape,
  `SCMD_VERSION` prints the version string, `SCMD_WHOAMI` prints the name —
  port each case exactly as written).
- Gates come from the table: `handbook`/`imotd` LVL_IMMORT, `players` LVL_GRGOD
  — the resolver + command_gates.tsv already carry these; just register the
  missing names.

## The Go work

1. Boot-load the text files wherever other lib data loads (follow the existing
   pattern; cache once like C).
2. One faithful gen-ps implementation; per-command handlers or a subcmd map —
   match the existing Go registry idiom. Long texts go through `PageString`
   (the pager port — merged; grep PageString for usage).
3. Register the missing names (`wizlist`, `immlist`, `future`, `handbook`,
   `imotd`, `players`, `whoami`, `clear`/`cls` — check `make reachability`
   output for which are already registered; `credits`/`news`/`info`/`policy`/
   `motd`/`version` exist but may serve invented content — repoint them at the
   2010 files/cases and DELETE the invented strings (R4).
4. R5e discipline: before "fixing" any existing handler, confirm what it
   actually serves today and say so in your report.

## Tests

- Loader: each file loads, cache non-empty, byte-identical to lib/text source
- credits/wizlist/immlist handlers serve the cached text (spot-assert a known
  line from each 2010 file, e.g. wizlist's "Wizards" header)
- clear/version/whoami per their C cases

## Verification

`go build ./... && go vet ./... && go test ./...` green; `gofumpt -w` touched
files; `make reachability` — expect wizlist/immlist/etc. to move to registered,
ZERO regressions. No git. End with: files copied (with byte counts), files
modified, and any do_gen_ps case you could not port faithfully (and why).
