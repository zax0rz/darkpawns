# Depth-fidelity handoff — `string` — 2026-09-21

## Queue position

This session ports `string`, the live world string editor, on branch
`port/string-cmd` from `origin/main` at `55264c66c` (refreshed — the `luaedit`
port merged after the branch was cut, and its changes to `pkg/session/tedit.go`
and `pkg/telnet` are baked into this work).

C registration: `{ "string", POS_RESTING, do_string, LVL_IMMORT+1, 0 }` at
`src/interpreter.c:745`, so the gate is level 32 / POS_RESTING. The gate row was
already waiting in `pkg/session/command_gates.tsv:459`; `registerCommand`
auto-gates from it.

`string` edits strings on LIVE entities — an NPC's name/short/long/description/
title, and one carried object's name/short/long plus extra descriptions — and
writes nothing to the world files. It is the last command-surface gap.

## Implementation summary

New files:

- `pkg/session/string_cmd.go`: `cmdString`/`cmdStringText`, the `quad_arg` port
  (with `old_search_block`'s mode-0 prefix scan inlined), the field/length
  tables, every message literal, the three object side paths (name/short/long,
  extra-desc create-or-modify, extra-desc delete), and the live-string editor
  entry.
- `pkg/game/live_strings.go`: instance-local writes for the live fields. Mobs
  clone-and-swap their prototype snapshot (the atomic pattern
  `RefreshLiveMobStrings` already uses); objects use the typed `Runtime` string
  overrides corpses/drinks/money already use; extra descriptions take a
  per-instance snapshot because C edits one object's `ex_description` list.
- `pkg/session/string_cmd_test.go`: 20 tests, all literal-level.
- `cmd/dp-oracle-diff/scenarios/string-depth.txt`: 18 annotated cases.
- `docs/fidelity/depth/string.tsv`: 28 manifested cases (26 proven, 2 excluded).

Modified:

- `pkg/session/commands.go`: registered `string`; routed it through
  `cmdStringText` so the inline string keeps the raw line remainder.
- `pkg/session/tedit.go`: `textEditState.liveString` (C's `d->str` write-through)
  and `textEditState.playingEditor` (the CON_PLAYING marker); the commit hook
  writes through instead of caching.
- `pkg/game/object.go`, `pkg/game/runtime_state.go`: `GetExtraDescs` serves the
  per-instance snapshot once `do_string` has touched the list.
- `pkg/session/session_send.go`: the prompt now keys on the descriptor string
  editor, not `PLR_WRITING`, and frames CON_PLAYING flushes (see findings).
- `pkg/telnet/listener.go`: `ensureLineEnded` (see findings).

## C behavior implemented (and the traps verified)

- `quad_arg` (modify.c:562-592): `one_argument` for type/name/field (lowercased,
  fill words skipped) — `game.OneArgument`; `is_abbrev` on the type; then the
  inline string is a verbatim copy of the rest of the line.
- `old_search_block` (whod.c:496-527), mode 0: a prefix match over
  `string_fields[]`, returning the 1-based index, `0` for an empty token and
  `-1` for no match. That split is the trap: an omitted field token is the only
  way to reach "No field by that name. Try 'help string'." — an unknown token
  reaches the switch default instead ("That field is undefined for
  monsters/objects."). "de" resolves to description; only "del" reaches
  delete-description.
- Inline writes: `length[] = {15,60,256,240,60}`, with the lowercase-LFCR
  `"String too long - truncated.\n\r"` (modify.c:757) — not `string_add`'s
  capital-T CRLF notice, which the improved editor still emits.
- Mob `long` (field 3) appends a real CRLF **inside the switch** before the
  inline/editor test (modify.c:648-651). Consequence, verified by the vehicle:
  `string mob x long` with no text writes `"\r\n"` and reports `Ok.` instead of
  opening the editor, and a 255-byte value plus the append is truncated at 256
  bytes leaving a bare `\r`.
- Player targets: name needs `LEVEL_IMPL-1` (`<` so 38 is refused, 39 allowed)
  and emits the WARNING line; short/long are monsters-only; title is
  players-only; description applies to both.
- Object field 4 create-or-modify: `str_cmp` (utils.c:107, case-insensitive
  exact) either resets the existing node in place or head-inserts a new one, then
  the editor opens with `MAX_STRING_LENGTH` and the case returns past the
  standard tail — so the inline string is always a keyword, never a value.
- Object field 6: head-node delete with "Field deleted.\n\r".
- Editor shape: `do_string` never calls `string_write`, so there is no
  PLR_WRITING, no `d->backstr`, and `STATE(d)` stays CON_PLAYING. Every line and
  every `/action` writes through into the live field, entry already cleared it
  (`*ch->desc->str = 0`), save is silent (`playing_string_cleanup` has no
  non-mail branch), and abort restores nothing while logging
  `SYSERR: string_add: Aborting write from unknown origin.`
- Transport: C's `process_output` frames a CON_PLAYING flush with an extra CRLF
  before the prompt, so the `string` editor's output carries a blank line that
  the CON_* editors' output does not (comm.c:1633-1636).

## Findings reported, not silently fixed

1. **`get_obj` is dead code (not ported).** `get_obj` (modify.c:538-558) has zero
   callers anywhere under `src/`; the command's call path is `get_char_vis`
   (:620) and `get_obj_in_list_vis` (:668). Porting it would be invention (R4),
   so it is manifested `excluded`.
2. **Field 6 on a non-head extra description is a reachable C segfault.**
   `find_exdesc` walks the whole list (act.informative.c:987-996) but the guard
   at modify.c:722 compares only the head node, so deleting a non-head field
   falls out of the switch into the standard tail with `field == 6`: `length[5]`
   is read past the five-element array and `CREATE(*d->str, …)` writes through
   the NULL pointer `do_string` never set for that field. Forced divergence: the
   port reproduces C's observable output (nothing), keeps the list, and refuses
   in the SYSERR shape (`SYSERR: do_string: delete-description fell through the
   head-node guard; C dereferences NULL here`). It cannot be an oracle case —
   the C process would die instead of emitting a transcript.
3. **Field 4's NULL-description case.** `find_exdesc` returns the matching
   entry's *description pointer*, so a field created by field 4 but never written
   reads as `No field with that keyword.` even though the keyword exists. Go
   cannot distinguish C's NULL from an empty string; the same message results for
   a world-file field whose description is literally empty — the one byte-level
   corner this port does not model. Unit-proven, manifested.
4. **The prompt path did not model `d->str`.** `make_prompt` returns `"] "`
   whenever the descriptor string pointer is set (comm.c:1038-1039). Go keyed
   that on `PLR_WRITING`, which every previous editor sets because they all route
   through `string_write` — `do_string` does not, so the first `string` editor
   showed the ordinary vitals prompt instead of `"] "`, and dropped
   `process_output`'s CON_PLAYING frame. Both are fixed in `session_send.go`;
   `luaedit`/`tedit`/OLC keep their existing bytes.
5. **LFCR messages got a second line ending.** The telnet event writer appended
   CRLF to any text not ending in `"\n"`, but C's historical `"\n\r"` ends most
   handler output — so a command emitting two messages gained a blank line
   between them (reproduced by the WARNING/`Ok.` pair). `pkg/telnet` now treats a
   trailing `"\n"` or `"\r"` as terminated. This is a shared transport class
   rather than a `string`-specific defect; the vehicle is the first to expose it.
6. **`docs/port-reachability-map.md` contradicts this work.** Its Bucket D
   (line 293) lists `string` among the commands "superseded" by the web admin and
   says to drop them. The port proceeds on the explicit instruction in the brief;
   `tedit` and `luaedit` were similarly ported after that ruling. The map should
   be reconciled in a follow-up rather than edited here.
7. **Live extra descriptions are instance-lifetime only.** C's live
   `obj_data.ex_description` list is written into the player's saved objects with
   the rest of the instance; the Go overlay (`Runtime.ExtraDescs`) is not part of
   `GetSaveState`, so a `string`-created extra description on a *carried* object
   does not survive a save/reload. Runtime semantics inside a session (and for
   room/mob instances) are exact, which is what the command's contract covers;
   persisting the overlay would change the object save format, so it is reported,
   not done.

## Validation

- `string-depth` oracle vehicle: **green** (`--seed 1`, no normalized
  divergence), 18 annotated cases across the entry gates, both field-error
  variants, every inline field, the truncation and blank-append quirks, the
  extra-desc create/modify/delete lifecycle with `look` proofs of the live
  instance, the editor write-through/abort/clear paths, and the player-target
  refusals with the level-39 rename gate.
- Unit tests: 20 in `pkg/session/string_cmd_test.go` (byte-exact literals, both
  truncation variants, write-through, abort-no-restore, silent save, disconnect,
  prompt/make_prompt, the forced-divergence refusal) plus
  `TestEnsureLineEndedKeepsCLineEndingsIntact` in `pkg/telnet`.
- `make fidelity-depth`: `do_string: 26/28` (the two excluded rows are `get_obj`
  and the unreachable `IS_NPC(ch)` guard).
- `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`,
  `gofumpt -l .` — all clean.
- Focused oracle regression subset (25 scenarios: every editor/OLC family plus
  ban/god-toggles/advance/snoop/gecho/purge/boards/help/info-pulse) run as a
  regression check for the two shared-code changes: 24/25 passed, with
  `string-depth` reported through the worker's
  `missing or malformed divergence fingerprints` path. That classification is
  partly an artifact: this scenario repeats probe lines (`@` three times, and
  `look crust` after create/modify/delete) and the worker's fingerprint extractor
  rejects duplicate command keys, so a divergence landing on any repeated probe
  cannot be attributed. The underlying divergence was not captured and the
  scenario is green in 18 subsequent runs, including 4-way and 6-way concurrent
  runs with added CPU load, and a solo worker run. Recorded here as an
  unexplained load-time observation rather than a pass.
- The full corpus census runs at review (`make oracle-regression`), which is also
  where the 517-command surface count is taken.

## Remaining / next

- Full oracle census at review, including the 517-command surface count.
- Live review drive: a scratch instance exercising the level-39 rename and a
  carried-object edit, plus the non-head delete repro (where C crashes and the
  port must refuse).
- Optional follow-ups: reconcile `docs/port-reachability-map.md` Bucket D; decide
  whether the live extra-description overlay should persist with carried objects.

