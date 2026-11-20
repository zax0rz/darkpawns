# Go-only player text, and why the string census misses it (2026-11-20)

PF-142. Job 1 of the go-only sweep: the mutation loop (`class go-only-in-game`)
keeps finding Go-only in-game text that `make string-census` does not list in
`docs/fidelity/strings/go-only.tsv`. This note traces every row through
`cmd/dp-string-census` and `internal/stringcensus` (read-only; no census code
changed — that decision waits for Zach after GLM's census PRs) and groups the
misses into cause classes (R5c).

## How the census decides "Go-only"

`internal/stringcensus/census.go`:

- The Go side scans only `DefaultGoDirs = pkg/game, pkg/session, pkg/combat,
  pkg/spells`.
- `gosink.go` hard-codes 18 sink names (`Send`, `SendMessage`, `Act`,
  `sendText`, `SendToRoom`, `BroadcastToRoom`, `actToRoom`, `SendToAll`,
  `sendPromptText`, `SendToZone`, `SendToOutdoor`, `channelAct`,
  `channelSend`, `SendToChar`, `SendToVict`, `sendToChar`,
  `movementSendToChar`, `communicationSend`). A call is player-facing only if
  its callee name is in that table. `fmt.Fprintf` is not a sink.
- `Normalize` (normalize.go) splits literals at `%`-conversions, `$`-act
  codes, colour escapes and line breaks, and **drops segments shorter than
  8 runes** (`MinSegmentLen`, "the brief fixes the floor at 8 characters").
- A Go segment is Go-only only if no C segment contains it and it contains no
  C segment (`censusbuild.go` direction 1, both containment directions), and
  it is not found, case-insensitively, in the world-data trees.
- Extraction resolves only single-assignment locals (`goresolve.go`); anything
  assigned twice, built in a `strings.Builder`, or passed as a computed call
  lands in `go-unresolved.tsv`, which is reported but **never gates the
  ratchet**.

## The rows and the cause of each

Current `go-only-in-game` rows (8), plus the 6 dead `Huh?!?` guards named in
the sweep brief (Round 1 rows 55/60/88/97, Round 2 rows 70/84; deleted in
#2890 and reclassed `c-reachable` in the round files since).

| row | literal (site) | in go-only.tsv? | cause |
|---|---|---|---|
| r1-28 | `║  Available Skills  ║` box header (`pkg/command/skill_commands.go:2262`, CmdListSkills) | no | C4 buffer assembly (strings.Builder + WriteString/Fprintf(&output)); the final send carries `output.String()`, filed unresolved. Package skip (C1) co-causes in the default census. Demonstrated: scanning `pkg/command` alone still does not list it. |
| r1-34 | `%s looks better than %s.` (`pkg/game/skill_actor_utility.go:247`, characterCompare) | no | C2 sink not in table: `sendBodyMessage` (`pkg/game/flee.go:14`, forwards to `Player.SendMessage`). Demonstrated: scanning `pkg/game` alone does not list it. |
| r1-55/60/88/97, r2-70/84 | `Huh?!?` dead guards (deleted in #2890) | no | C3 below the floor: `Huh?!?` is 6 runes < `MinSegmentLen` 8, so it never becomes a segment on either side. (The brief guessed containment; the floor fires first. Containment would also suppress it — C prints `Huh?!?` at src/interpreter.c:916 — but extraction never gets that far.) |
| r2-9 | `The sound of a death cry is heard as %s enters the room!` (`pkg/game/death.go:900`, deathTrapMount) | no | C2 sink not in table: `w.roomMessage` (`pkg/game/spec_procs.go:43`). Luna's file (DP-1609 territory). |
| r2-35 | `Parsed world data not available.` (`pkg/session/wiz_stats.go:1252`, cmdVnum) | **yes** (since DP-1581, 2026-11-18) | Not missed. The mutation row re-sampled an already-queued segment; the census extracted it (`.Send` literal) and the ratchet already carries it. |
| r2-39 | ` on %s` (`pkg/command/skill_commands.go:182`, CmdUseSkill) | no | C4 buffer assembly (`fmt.Fprintf(&output, " on %s", ...)`, builder sent at :196, filed unresolved at go-unresolved.tsv). C3 secondary: normalized `on` is 2 runes < 8, so even direct extraction would drop the fragment. C1 tertiary (pkg/command unscanned). |
| r2-43 | `You create an arrow from it.` (`pkg/spells/affect_spells.go:2152`, castSilkenMissile) | no | C2 sink not in table: `sendToCaster`. C5 secondary: if it were extracted, C's segment `You create an arrow from` (normalized from `act("You create an arrow from $p.", ...)` at src/spells.c:906) sits inside the Go segment and would suppress it anyway (verified by hand-applying Normalize to both literals). Luna is deleting the line under DP-1609. |
| r2-57 | `Are you sure you want to forget %s (Level %d)?` (`pkg/command/skill_commands.go:85`, CmdForget) | no | C4 multi-assignment local: `output := fmt.Sprintf(...)` then `output += ...` marks `output` multi (`goresolve.go recordAssign`), so `s.SendMessage(output)` is filed unresolved (go-unresolved.tsv :89) instead of extracted. C1 co-cause (pkg/command unscanned). |
| r2-82 | `The board could not be saved.` (`pkg/boards/boards.go:1019`, FinalizeBoardWrite) | no | C1 package not scanned: `pkg/boards` is not in `DefaultGoDirs`. Demonstrated: scanning `pkg/boards` alone lists it immediately. |

## Cause classes (R5c)

- **C1 — unscanned package.** `DefaultGoDirs` covers 4 of the player-facing
  packages. `pkg/command`, `pkg/boards` (and `pkg/admin`, `pkg/mail`,
  `pkg/scripting`, `pkg/db`, `pkg/telnet`, …) are never read, so Go-only text
  there is invisible by construction. Operative for r2-82; co-cause for
  r1-28, r2-39, r2-57.
- **C2 — sink not in the 18-name table.** Thin forwarding wrappers added
  after the table was built (`sendBodyMessage`, `w.roomMessage`,
  `sendToCaster`) forward straight into listed sinks but are not themselves
  listed, so their literals are never extracted. Operative for r1-34, r2-9,
  r2-43. This is the class most likely to keep growing: every new wrapper
  silently widens the blind spot.
- **C3 — below the 8-rune floor.** `Huh?!?` (6) and short fragments like
  `on` (2) never become segments. Operative for the 6 dead guards; secondary
  for r2-39.
- **C4 — unresolved flow: buffer assembly and multi-assignment.** Literals
  written into a `strings.Builder` (or a `+=`-rebuilt local) reach the sink
  through a computed value; the extractor refuses to guess and files the site
  in `go-unresolved.tsv`, which does not gate the ratchet. Operative for
  r1-28, r2-39, r2-57.
- **C5 — containment false-negative (documented, not operative today).** A Go
  segment that *contains* any C segment is declared matched
  (`censusbuild.go` direction 1). Demonstrated by hand for r2-43. No current
  row is suppressed *only* by C5; it becomes operative the moment C2/C4 fixes
  land for lines like the arrow text.
- **C6 — C-side variable-assignment gap (false positives, found in Job 2).**
  `cextract.go` takes only the literals inside a sink call's argument list.
  C text assigned to a variable first — `to_char = "You point around the
  room."` (src/new_cmds.c:2534) or `const char *usage = "Usage: checkload …"`
  (src/act.wizard.c:3825) — never becomes a C candidate, so the Go literal
  that mirrors it is reported Go-only. Both rows currently in `go-only.tsv`
  ("points around the room.", "Usage: checkload { obj | mob } <number>") are
  this class: the bytes match C exactly and need no code change. Fix belongs
  with the C extractor (track `to_char`/`to_room`/`usage`-style locals), same
  owner decision as C2.
- **C0 — already listed.** r2-35 shows the census working as designed when the
  text is extractable: listed, queued, ratcheted.

## Recall

Mutation-loop `go-only-in-game` finds vs the census Go-only list:

- Census list today: 14 rows in `go-only.tsv`, 9 unique segments — all in the
  four scanned packages, all via listed sinks, all ≥ 8 runes, none
  C-contained, none data-sourced. (The ratchet reports "9 known go-only
  segments, 0 unreviewed".) At least 2 of the 14 ("points around the room.",
  "Usage: checkload { obj | mob } <number>") are C6 false positives: C prints
  the identical bytes via variable-assigned literals.
- Mutation finds in code today: 8 rows. The census lists 1 of them (r2-35,
  already queued before the loop sampled it) and misses 7.
- Historical finds (6 dead `Huh?!?` guards) were missed by C3 and are now
  deleted (#2890).
- Empirical check, run 2026-11-20 on merged main (`cmd/dp-string-census
  -go-dirs pkg/{command,boards,game,spells} -out-dir /tmp/... -update`):
  scanning `pkg/command` alone surfaces 14 Go-only segments (including the
  forget/useskill usage lines but *not* the builder-assembled text);
  `pkg/boards` alone surfaces `The board could not be saved.`; `pkg/game` and
  `pkg/spells` alone still do *not* surface the wrapper-sink rows. The
  builder/multi-assign sites appear in `go-unresolved.tsv` (e.g.
  `output pkg/command/skill_commands.go:89 .SendMessage`) and never gate.

## Proposed fixes (for Zach, after the census PRs settle)

- C1: derive the scanned set from the packages that can reach a sink (or list
  every `pkg/*` with player output) and regenerate with
  `make string-census-update`. Cheap; expect a one-time jump in the review
  queue from `pkg/command`/`pkg/boards`.
- C2: audit one-line forwarding wrappers mechanically (each must forward into
  a listed sink — the table's own inclusion rule) and add them to `goSinks`;
  make "new forwarding wrapper" a review checklist item so the class can't
  regrow silently.
- C3: either carve out an exact-match exemption for segments that equal a
  known short C line (R2: the command surface is part of the game), or emit a
  separate sub-floor report for review. The floor exists to drown noise; a
  curated allowlist review is the middle path.
- C4: teach `classify` to follow `strings.Builder` (`WriteString` /
  `Fprintf(&b)`) and bounded `+=` chains within one function body, and/or
  promote `go-unresolved.tsv` to a ratchet-visible list so new unresolved
  sites get reviewed instead of only reported.
- C5: when containment matching is next touched, consider requiring the
  contained C segment to be a maximal fixed-text run (or within a length
  ratio of the Go segment) so `You create an arrow from it.` no longer
  launders through `You create an arrow from`. Lowest urgency: no row today is
  suppressed only by C5.

Related: DP-1581 (the go-only review queue this list feeds), DP-1609 (Luna's
deletion of the silken-missile arrow line), PF-140 (mutation round 2).
