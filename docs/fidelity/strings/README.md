# Player-facing string census

The census in `cmd/dp-string-census` compares **every player-facing string in
both trees**, not just the ones a scenario happens to print. `make
oracle-regression` can only diff what a scenario makes both servers emit; on
2026-09-23 a playtest found Go printing `[ GHOST SHIP ] ...` and accepting
`autoexit` — strings and commands with no C source at all (R4) — and no
scenario ever triggered either. This tool flags that class in about a second,
in both directions:

- **go-only** — a Go player-facing segment with no C source and no world-data
  source. An R4 review queue.
- **c-missing** — a C player-facing segment no Go segment contains. A port-gap
  queue.

It changes no game code. `src/` is read-only input (RULEBOOK R5), and so is the
Go tree: the tool only reads.

This closes the `HARNESS_PARITY.md` row "Player-facing strings outside any
scenario" (PR #1585). No string census existed before it, so the two invented
strings above survived a green census.

## Run it

```bash
make string-census         # ratchet: fail on a new go-only segment
make string-census-update  # regenerate the reports and the baseline
go run ./cmd/dp-string-census --top 30   # report only, write nothing
```

Coverage of the C surface by the scenarios that ran is a separate tool and a
separate dump; see [Coverage](#coverage-which-c-strings-the-census-ever-makes-c-print).

The tool runs over the real tree in under a second (257 Go files, 66 C files,
630 world-data files).

## What each side reads

### Go: precise (go/ast)

Every non-test `.go` file in `pkg/game`, `pkg/session`, `pkg/combat`,
`pkg/spells` is parsed and walked for calls to the sink table below. A literal
counts when it is

- passed directly to a sink,
- part of a `+` concatenation of literals passed to a sink (the Go spelling of
  C's adjacent literals),
- the format or an argument of `fmt.Sprintf`/`fmt.Sprint` whose result is the
  sink argument,
- held in a local variable assigned exactly once in the same function — the
  report then says `.SendMessage(via msg)`.

Anything else is either ignored or reported, never guessed:

- **ignored**: parameters, struct fields, globals, non-string literals. Nothing
  in the file set proves they hold player text.
- **reported unresolved** in `go-unresolved.tsv`: an argument that mentions a
  local variable the extractor could not resolve (assigned twice, destructured,
  declared without a value, or built by a call such as `fmt.Fprintf(&sb, ...)` +
  `sb.String()`), or a `strings.Join`-style construction. These are leads for
  tightening the sink table, not ratchet failures.

### The Go sink table

| Sink | What it is |
|---|---|
| `.Send`, `.SendMessage`, `.sendText`, `.sendPromptText` | session and player primitives |
| `Act` | `pkg/game Act()`, the act()-style primitive |
| `.SendToRoom`, `SendToRoom`, `.BroadcastToRoom`, `.actToRoom`, `.SendToAll`, `.SendToZone`, `.SendToOutdoor` | broadcast surfaces |
| `.channelAct`, `.channelSend` | `pkg/game/out_of_band.go` channel lines |
| `SendToChar`, `SendToVict`, `sendToChar`, `movementSendToChar`, `communicationSend` | one-line forwarding wrappers, each verified to call a sink directly |

The table is `goSinks` in `internal/stringcensus/gosink.go`; `summary.txt` lists
it with the literal counts this run found. `goSinks` is the census's whole
declared scope: a literal that never passes through one of these calls is out of
scope, and adding a sink is a deliberate widening of the census.

### C: broad (a lexer over `src/*.c`)

The C side is deliberately permissive, because its job is to say a string
*exists* in C, not to prove it is reachable. A small lexer handles comments,
escapes and adjacent literal concatenation (`"a"` `"b"` is one string). Literals
are kept when they sit in the argument list of:

`send_to_char` (and its `stc` macro), `act`, `SEND_TO_Q`, `write_to_output`,
`page_string`, `send_to_room`, `send_to_zone`, `send_to_outdoor`,
`send_to_all`, plus `sprintf`, `snprintf`, `strcpy`, `strncpy`, `strcat`,
`strncat` tagged `buffered:` (the brief's "simple heuristic is fine": the buffer
is usually flushed by a later `send_to_char`). Every literal in `constants.c` and
`class.c` is kept and tagged `table`, because those are player-facing string
tables rather than call sites.

Each row also records the enclosing C function (the `ACMD(name)` / `SPECIAL(name)`
declaration macros are understood, and a brace that opens anything but a
function definition does not move the tracker).

## Normalization (both sides, identically)

A literal is unescaped, then split into **segments**. A segment ends at:

1. **a line break** — `\r`, `\n`, `\r\n` and `\n\r` alike;
2. **an ANSI/colour escape** — Go's literal `"\x1b[36m"`; C's `CCRED(ch, C_NRM)`
   macros are runtime expressions, not literal concatenation, so they never
   appear inside a C literal to begin with (they are separate `%s` arguments);
3. **a printf conversion** — `%s`, `%-10s`, `%ld`, `%3d`, `%.2f`, `%[1]s`;
4. **an act code** — `$n`, `$N`, `$m`, `$e`, `$s`, `$p`, `$P`, `$o`, `$O`, `$t`,
   `$T`, `$F`, `$a`, `$A`, `$r`, `$R`, `$q`, `$Q` (the union of
   `src/comm.c:2408` and `pkg/game/act.go:356`; Go implements a superset).

`%%` and `$$` are literal `%` and `$`. Whitespace inside a segment collapses to
single spaces. Segments shorter than 8 characters are dropped: "Ok." and "Yes."
collide across unrelated strings, and the floor is what keeps the report
readable. Every rule is in `internal/stringcensus/normalize.go` with its
reasoning, and `normalize_test.go` pins each one.

## Matching

Case-sensitive, on normalized segments:

- a Go segment is **c-sourced** when it is a substring of some C segment *or*
  vice versa;
- a C segment is **ported** when some Go segment contains it. `c-missing.tsv` is
  therefore stricter than its mirror: a Go string that is only a *fragment* of a
  C string does not clear the C row. That is intentional — the list is a review
  queue, not a verdict;
- a Go segment that is neither c-sourced nor c-missing is checked against the
  world data (`lib/world`, `lib/misc`, `lib/text`, whitespace collapsed and
  lower-cased). A hit is **data-sourced**, not invented;
- everything left is **go-only**.

## Outputs

| File | Columns |
|---|---|
| `go-only.tsv` | segment, file:line, sink |
| `c-missing.tsv` | segment, file:line, sink, function |
| `go-unresolved.tsv` | expression, file:line, sink |
| `summary.txt` | totals, both sink tables, counts per Go package / per C file, go-only and c-missing groups by file |
| `go-only-baseline.json` | every known go-only segment with a `reason`, starting `unreviewed` |

The four `.tsv`/`.txt` files are generated (`make string-census-update`) and
deterministic: no dates, no machine paths. The baseline is the ratchet.

## The ratchet

`make string-census` fails when a go-only segment appears that is not in
`go-only-baseline.json`. That is the CI gate in `.github/workflows/ci.yml`.
Warnings, not failures:

- baseline entries no longer produced (fix one, do not have to update the
  baseline to go green);
- reports that are stale on disk (regenerate before merging a PR that changes
  message text, so reviewers see the diff).

A reason starts `unreviewed` and is settled by a human: data-driven text,
transport framing, a deliberate post-C addition, or a bug that removes the
string. The first baseline is entirely `unreviewed` on purpose — triaging 128
strings is fidelity work, not tooling work, and pretending otherwise would hide
the queue.

Every row **has** since been triaged: [`triage.md`](triage.md) holds the
classification and the evidence (`bug:invented`, `bug:paraphrase`,
`census:composed`, `data`, `surface:no-c`, `unsure`), the reason and evidence
travel in the baseline row itself, and
`internal/stringcensus/baseline_vocabulary_test.go` rejects an unreviewed or
evidence-free row. A new go-only string still starts `unreviewed` and needs a
triage row of its own.

### Known limits

- **Literals only.** A format string composed from variables
  (`roomVerb := "says"` in `pkg/game/directed_speech.go:117`) is invisible; the
  call site appears in `go-unresolved.tsv` instead.
- **No type checking.** `.Send` on any receiver is attributed to the session
  sink. Widening the table is a review decision, and the census does not pretend
  to know receiver types.
- **Parameter forwarding is invisible.** `helper(msg string)` called with a
  literal is not followed into `helper`; only the sink table's own forwarders
  are listed, and each is verified by hand.
- **Keyed by segment text.** The ratchet fails on a *new* go-only segment. A
  second occurrence of a known segment in a new file does not fail, and moving
  a string between files does not either; `go-only.tsv` shows those moves.
- **Buffered C literals are a heuristic.** A `sprintf` into a buffer that is
  never sent still enters the C corpus, which can only make a Go string
  *c-sourced* (never go-only). Over-inclusion on the C side is safe; the Go side
  is the strict one.

## Coverage: which C strings the census ever makes C print

A green census says the scenarios that ran matched. It says nothing about which
parts of the game no scenario reaches: `quit.safe-logout` was green for months
while the menu after the goodbye was missing, because the scenario stopped at
the goodbye. Coverage by C output string is a cheap, honest map of what the
judge actually sees.

`cmd/dp-census-coverage` reads a **census dump** — one `<scenario>.txt` per
scenario holding that run's normalized C blocks — and reports, for every C
literal brief 07 extracts, which scenarios printed it.

```bash
# 1. leave a dump behind during a census (default off; results and timing unchanged)
ORACLE_REGRESSION_DUMP=/tmp/dp-dump make oracle-regression

# 2. report coverage of that dump
make census-coverage CENSUS_DUMP=/tmp/dp-dump
# or: ORACLE_REGRESSION_DUMP=/tmp/dp-dump make census-coverage
```

The coverage tool never starts its own census: coverage is a claim about the
scenarios that ran, so the dump has to come from a real run. It errors out
rather than reporting 0% if the directory is missing or empty.

### How a C literal is matched

`dp-oracle-diff --dump-oracle <dir>` writes exactly what `--show-oracle` prints
today (`normalized C oracle blocks:` plus one `--- [label]` section per probe
block), so a dump file is the terminal output on disk.

Matching reuses brief 07's lexer and normalizer, then applies the wildcard rules
the brief names:

- a literal's fixed text is split into **fragments**, one per printed line;
- inside a fragment, the segments must appear **in order on one line** with
  arbitrary text between them — that gap is where printf verbs and act codes were
  substituted (`You have %d gold pieces on hand.` matches
  `You have 1234 gold pieces on hand.`);
- a later fragment must match a **later line**, because a fragment boundary came
  from a `\r\n` inside the C literal;
- a site is covered when some block of some scenario matches it; the report
  records the scenario names and the probe labels that did.

Go output never counts. Coverage is measured against the C blocks in the dump,
so a sentence the port prints and C never does cannot make a C site look
covered (pinned by a unit test whose fixture sentence lives in the Go tree).

### Outputs

| File | Contents |
|---|---|
| `coverage.tsv` | one row per C segment: segment, file:line, sink, function, `covered` (`yes`/`no`/`unverifiable`), the scenarios that printed it, the probe labels that printed it |
| `coverage-summary.txt` | totals, then per C file and per C function: sites covered/total, segments covered/total and percent, unverifiable sites, and the never-seen segment list |

Both are **generated per dump and not committed** (`.gitignore`, like
`docs/reports/scenario-coverage-*.tsv`): the dump is not in the repository, so a
committed `coverage.tsv` would be an untraceable snapshot of one machine's run.
Regenerate on demand; attach the report to a PR when it matters.

### Unverifiable literals

A literal whose fixed text is entirely below the census floor (`MinSegmentLen`,
8 characters) has nothing to compare: `send_to_char("Ok.\r\n", ch)` cannot be
proven printed by string matching. Those sites are reported as `unverifiable`,
listed in `coverage.tsv` with a placeholder segment, and **excluded from the
percentage** rather than counted as misses. On 2026-09-23 they are 1336 of 5454
C sites (24.5%) — the single biggest lever for raising coverage precision is
lowering that floor, which would also change brief 07's corpus, so it is a
separate decision.

### Known limits

- **Coverage is an upper bound, because matching is substring-based.** A short,
  generic segment can coincide with unrelated C output. The first real run gives
  the worked example: `src/new_cmds2.c:896` (`flow_room`) stores `"the south"`
  in a direction buffer, and the probe `look` in `look-start-room` printed a
  room description containing the words "the south", so the site reads covered
  without `flow_room` having run. The report names the probe label per row, so
  such a row is checkable by hand; treat a single-coincidence row as a lead, not
  a fact.
- **Per-block, per-line, not per-print-order.** Multi-fragment sites only need
  their fragments in order somewhere in one block; if a literal prints twice,
  fragment 1 of the first print and fragment 2 of the second would satisfy it.
- **Scenario granularity, not step granularity.** A site is covered by a
  scenario when any of its blocks matched; the report names the probe label but
  does not reconstruct the step sequence.
- **Dumps are per-run artifacts.** Coverage answers "what did *this* dump
  reach", so a filtered run (`ORACLE_REGRESSION_SCENARIOS=...`) produces a
  filtered report. The summary prints the scenario and block counts at the top so
  a partial dump is visible in the numbers themselves.

## First run, 2026-09-23

- 153 go-only rows / 128 distinct segments, 3108 c-missing rows, 30 data-sourced
  segments, 516 unresolved sink arguments (see `summary.txt` for the current
  numbers).
- The checks the brief names all hold: `[ GHOST SHIP ]` / `[ NIGHT GATE ]`
  (`pkg/game/weather.go:596-635`) and `Autoexits enabled.` / `Autoexits
  disabled.` (`pkg/session/informative_cmds.go:26,28`) are go-only, and the
  ported `Goodbye, friend.. Come back soon!` (`src/act.other.c:137`,
  `pkg/game/quit.go:73`) is c-sourced.
- The sink heuristic produced no pure-internal rows: every go-only segment is
  text a player or immortal can read, so no entry is pre-triaged as
  non-player text.

### What it already found (not triaged, not fixed here)

Tooling-only scope: these are reported, not changed. R5c says find one, find the
class — this makes the class visible and rerunnable — and R5e says a finding is
not confirmed until its call path is, which is why every row is a pointer for a
human rather than a filed bug: each row is its own fidelity decision.

- `pkg/combat/fight_core.go:180,494` sends `You are incapacitated and will
  slowly die, if not aided.` while `src/fight.c:1573` sends `You are
  incapacitated an will slowly die, if not aided.` — C's missing `d` is the
  game's byte, so the port corrected a typo it does not own (R1).
- `pkg/game/ai.go:172` and `pkg/game/graph.go:343` announce `has arrived.`;
  `src/handler.c:532` `char_to_room()` prints nothing, and C's only arrivals are
  `$n arrives from the %s.` (`src/act.movement.c:238`) and `$n arrives
  suddenly.` (`src/spells.c:314`).
- `pkg/combat/engine.go:971` and `:974` send `You hit %s for %d damage!` /
  `%s hits you for %d damage!`; no C file contains `damage!`, so the combat
  damage line has no C source.
- Web/OLC-only surfaces with no C counterpart (expected, still R4 decisions):
  `Description is too long; type @ or /s to save, /a to abort.`,
  `Unable to save description.`, `Failed to save new password. Please try again
  later.`

Those four groups are a sample. `go-only.tsv` is the complete list, and
`c-missing.tsv` holds the other direction (3108 rows, e.g. the whole `do_say`
verb family `says, '` / `exclaims, '`, which Go renders from variables instead of
literals).

