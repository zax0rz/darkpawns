# Phase 7 OLC handoff — bounded `tedit` workflow

Date: 2026-09-16

## Decision and review boundary

This round revisits the earlier OLC disposition under the explicit Phase 7
objective. It selects exactly one coherent editor workflow: the original
in-game text-file editor, `tedit`. The existing web/admin decision remains
untouched; this PR does not change `pkg/admin`, `admin-ui`, the website, world
data, deployment, or production data. The larger object/room/mobile/shop/zone
editors remain out of scope.

The implementation branch is `glm/phase7-olc-tedit`, based on the freshly
fetched `origin/main` at the start of this round. The initial workflow is
committed as `f186df61c` (`feat: restore tedit text editor workflow`); this
follow-up closes the blank-line, shared-live-buffer, replacement-size, and
sibling-parser fidelity gaps described below. The primary checkout was
preserved. This handoff stops at one reviewable, unmerged PR; it does not
merge or deploy.

## Original in-game OLC inventory

The inventory was made from the checked-in C command table and source files,
then checked against the actual call paths. `wc -l` counts are included to
make the dependency and scope decision reviewable.

| C family | Size | Phase 7 disposition |
| --- | ---: | --- |
| `tedit.c` / `do_tedit` | 98 | Selected and implemented end-to-end. |
| `file-edit.c` | 199 | Implemented only as the shared dependency needed by `tedit`; broader directory/file helpers remain out of scope. |
| `improved-edit.c` | 627 | Implemented only for the `tedit` string path and its reachable actions. |
| `modify.c` | 869 | Implemented only for the `CON_TEDIT`/`string_add` behavior reached by this slice. |
| `olc.c` | 524 | Residual shared OasisOLC state machine; not opened by this slice. |
| `medit.c` | 1,126 | Residual interactive mobile editor; no implementation. |
| `oedit.c` | 1,564 | Residual interactive object editor; no implementation. |
| `redit.c` | 1,078 | Residual interactive room editor; no implementation. |
| `sedit.c` | 1,178 | Residual interactive shop editor; no implementation. |
| `zedit.c` | 1,276 | Residual interactive zone editor; no implementation. |
| `luaedit.c` | 58 | Residual in-game Lua directory/editor; no implementation. |
| `cedit.c` | — | No checked-in source file or command-table registration was found; no family was invented. |

The adjacent `dig` builder command is a room-exit action, not one of these
editor state machines, and remains owned by its existing proof. No second
editor was started in this round.

The earlier reachability map and audit reports describe OLC as superseded by
the web admin. That is historical decision context, not authority to ignore
this explicitly authorized Phase 7 slice. The selected implementation is
still bounded to `tedit`; it does not reopen the other families or alter the
admin replacement.

## C call path and implementation

The selected path is:

1. `src/interpreter.c:764` registers `tedit` at `POS_DEAD` and `LVL_GRGOD`.
2. `src/tedit.c:34-98` scans the ordered twelve-entry field table, after
   `one_argument` lowercases the field token, applies each field's own level
   gate, and calls `general_file_edit`.
3. `src/file-edit.c:150-174` prints the instructions and existing buffer,
   stores the target file, and enters `CON_TEDIT`.
4. `src/modify.c:114-205` sanitizes input, appends CRLF lines, and routes
   slash commands through `src/improved-edit.c:27-85`.
5. `src/file-edit.c:28-73` saves or aborts, then runs `cleanup_olc`; direct
   disconnect follows `src/comm.c:2107-2119` and cleans OLC without invoking
   the tedit save/abort callback.

`pkg/session/tedit.go` ports the finite field table and complete improved
editor action set. It keeps the C source-order prefix behavior and uses a
per-session descriptor snapshot only for `backstr`/cleanup; the active buffer
is serialized through the shared static-text cache used by
`news`/`motd`/the other static commands. Each editor input/action refreshes
from and commits to that live authority, so overlapping descriptors and an
observer see the same unsaved bytes as C's `d->str`. On save, C's
`src/olc.c:424-440` `strip_string` mutates that live buffer to LF before
`fputs`; the Go path preserves that post-save cache state, while a later read
from disk reconstructs CRLF. `help/screen` updates the live `World.HelpScreen`
path rather than creating a second cache authority. The editor replacement
path preserves C's unsigned size arithmetic, `strtok` token positions, and
the conservative per-occurrence `/ra` size check.

`RawLine` in `CommandData` and the telnet boundary preserve the complete input
line for active `CON_TEDIT` state. This is necessary for blank/whitespace-only
lines, `/h`, `/s`, `@`, and attached slash actions to remain editor input
instead of becoming ordinary commands. The Go audience gate also mirrors C's
`SENDOK`: room acts omit players carrying `PLR_WRITING`. Manager and telnet
disconnect paths call the C-equivalent cleanup; they do not save unsaved text.

The `pkg/admin` router was inspected as part of the boundary check. Its
mutation endpoints cover world entities and `/admin/save-world`; there is no
route that mutates the `lib/text` tedit files. No admin or frontend file is
changed by this work.

## Proof and evidence

The durable depth manifest is
[`docs/fidelity/depth/tedit.tsv`](../tedit.tsv). It maps the complete selected
workflow to D1, D2, D3, D4, and D5 proof, with exact C locations and explicit
remaining-family boundaries.

Focused unit proof in `pkg/session/tedit_test.go` covers:

- command and field authorization, including mortal hiding and GRGOD-versus-
  Implementor field access;
- C boot CRLF representation;
- abort rollback and disconnect cleanup;
- save, live-cache visibility, and a fresh-manager disk reload; and
- raw-line routing before the ordinary command interpreter;
- blank-line append, overlapping descriptors with independent abort snapshots,
  and the replacement unsigned-size/`strtok` boundary audit, including
  malformed delete/replace arguments and `/ra` overflow behavior.

`cmd/dp-oracle-diff/scenarios/tedit-depth.txt` is the live C-vs-Go vehicle. It
has a named peer for the begin/abort OLC room audience and probes field listing,
invalid selection, uppercase/prefix selection, every improved-editor action
used by this workflow, invalid inputs, append/abort, save, and post-save
`news` visibility. The scenario is intentionally small and does not mutate
checked-in world or production inputs.

`cmd/dp-oracle-diff/scenarios/tedit-boundaries-depth.txt` is the directed
multi-client C-vs-Go boundary vehicle. It proves that a blank line appends,
malformed delete/replace arguments retain C diagnostics, shorter replacements
are accepted under C's unsigned arithmetic, an observer sees unsaved live
`news`, and a second editor's abort restores its own `backstr` snapshot. The
directed `send:<peer>` probe support is unit-tested in
`internal/oraclediff/oraclediff_test.go`.

Both scenarios are normalized-green at seed 1, and the five-seed matrix is
green for seeds 1, 2, 3, 5, and 8 for each vehicle. Its preserved external log
is `/home/zach/dp-phase7-tedit-boundaries-matrix-2026-09-15.log`.

The final repository gates passed on the frozen implementation and scenario
inputs: `make fmt`, `make check-fmt`, `git diff --check`, `go build ./...`,
`go vet ./...`, `go test ./...`, `go test ./pkg/game/...`,
`golangci-lint run ./...` (0 issues), `go test -race ./pkg/session ./pkg/telnet`,
`make fidelity-depth`, and `make expected-divergences-check`. The depth report
is 4,858 total cases with 4,739 proven/delegated, 68 blocked, and 51 excluded;
`do_tedit` is 42/42. Expected-divergence pins remain valid (26 rows across 10
scenarios).

The required full oracle census was run against those frozen inputs. The
preserved external log is
`/home/zach/dp-phase7-olc-full-census-2026-09-15.log`; it reports 943
scenarios, 933 passed, 9 expected, 1 established unpinnable
`accuse-noarg-depth` case requiring human clearance, and zero stale, failed,
infrastructure, or timed-out cases. The command exits 2 for that existing
unpinnable baseline; this is the expected repository behavior. The handoff
update records evidence only and does not change the tested runtime inputs.

## Remaining scope and stop condition

This PR does not claim OasisOLC completion. `olc`, `medit`, `oedit`, `redit`,
`sedit`, `zedit`, and `luaedit` retain their residual interactive state-machine
scope, with their existing manifests and historical proof gaps left explicit.
The shared `file-edit.c`, `improved-edit.c`, and `modify.c` rows remain broader
than this selected `tedit` path. No generic OLC framework is introduced.

After the final validations, this branch will be pushed as one unmerged PR for
human review. Do not merge, deploy, start another editor, or expand the scope
from this handoff.
