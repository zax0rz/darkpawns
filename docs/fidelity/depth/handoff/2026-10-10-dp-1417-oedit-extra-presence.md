# DP-1417 port: oedit extra-description presence (loaded-NULL vs editor-emptied)

Base `origin/main` `e985ab07a`, worktree `/home/zach/dp-mudlog-1417`, branch
`deepseek/mudlog-1417`. Tier: stop (OLC persistence). Zach merges.
Design: `2026-10-10-dp-1417-oedit-extra-presence-design.md` (approved by Claude
with Zach's ruling: Option A; the presence flags are `json:"-"`; the look fix is
in scope). R1/R2/R4/R5e/R5g/R5h.

## The C site

`src/oedit.c:417-426`:

```c
if (!ex_desc->keyword || !ex_desc->description) {
    mudlog("SYSERR: OLC: oedit_save_to_disk: " "Corrupt ex_desc!",
           BRF, LVL_BUILDER, TRUE);
    continue;
}
```

* **Payload** `SYSERR: OLC: oedit_save_to_disk: Corrupt ex_desc!` — one space
  (two adjacent string literals), one log per corrupt node.
* **Type** BRF, **level** `LVL_BUILDER` (`src/olc.h:54` aliases it to
  `LVL_IMMORT`, 31), **file** TRUE.
* **Position** inside the per-object scan, after the object's `S` script line
  and before its `A` affect lines (`src/oedit.c:409-444`); the object itself is
  still written. The writer is shared by telnet `oedit save` and the admin/OLC
  save-all path (`pkg/session/olc_control.go:106`,
  `pkg/session/olc_registry.go:181`).
* **Reachability** builder OLC, valid play under the 2026-10-09 ruling. Two
  live states: a load whose tilde field was empty (`fread_string` → NULL,
  `src/db.c:2736-2740`), and a `goto next` node C creates zeroed
  (`src/oedit.c:1522-1538`). The valid counterpart is the editor-allocated empty
  string (`src/improved-edit.c:193-211`), which must still be written.

## The port

Presence is now explicit in memory (`parser.ExtraDesc`, Option A):

* `NullKeyword` / `NullDescription` mirror C's two pointers. Zero value =
  allocated/present, so every existing producer (Lua `extra()`, do_string, the
  web OLC, note text) is unchanged.
* `json:"-"` on both (Zach's ruling): no save-format change. C's `objsave.c`
  stores no extra descriptions, so presence never has to survive a player save.
* `parser.NewLoadedExtraDesc` encodes the `fread_string` rule and both loaders
  use it: `pkg/parser/obj.go` (object `E` blocks) and `pkg/parser/wld.go`
  (room `E` blocks).

What changed:

| Surface | Change |
|---|---|
| `pkg/parser/obj.go` | flags on `ExtraDesc`; `NewLoadedExtraDesc`; the object `E` loader derives them |
| `pkg/parser/wld.go` | the room `E` loader derives them |
| `pkg/session/oedit.go` | `startOedit` seeds `extraMeta` from the flags, not text; `saveOeditInternallyLocked` folds the descriptor's per-field presence into the committed prototype; `writeOeditObj` mirrors C's guard (log + skip) |
| `pkg/game/look.go` | `findExtraDescription` returns not-found at the first keyword match whose description is NULL, and does not scan later nodes |
| `pkg/session/string_cmd.go` | do_string field 4 nulls the description pointer when the editor opens (`src/modify.c:686-709`) and clears it when the editor writes; field 6's `!find_exdesc` test (`src/modify.c:718`) uses the flag instead of empty text |
| `pkg/game/live_strings.go` | comment: a NULL description is "not found" |

## The fifth `find_exdesc` caller (R5e)

`src/modify.c:718` (`do_string` field 6, delete-description) calls
`find_exdesc(string, obj->ex_description)` and tests `!find_exdesc(...)`. Since
`find_exdesc` returns the matched node's description pointer, a node whose
keyword matches but whose description is NULL yields C's "No field with that
keyword." — the same rule as the `do_look` callers. Go's
`stringObjDeleteExtraDesc` already had that behaviour but keyed it on empty
text; it now keys on `NullDescription`, so an editor-allocated empty
description (a live field) is still found and deleted.

Field 4 of the same command (`src/modify.c:686-709`) is why that flag is not
always false: both its arms `ed->description = 0` before opening the editor
(field 4 also passes no `backstr`, so an abandoned editor leaves the pointer
NULL). `stringObjExtraDescEditor` now mirrors that — the node is NULL on open
and becomes present when the editor writes a buffer (C allocates even an empty
one) — which is what makes the field-6 test agree with C in the open-editor
state. `TestStringObjExtraDescNullsPointerUntilWritten` pins it.

## Lock audit (#1825)

`writeOeditObj` is called from `saveOeditZoneLocked`, which holds only the
per-zone save lock and has already released `world.mu` (the object snapshot is
taken before the render loop). The new `MudLog` call therefore holds no world,
player, manager or lifecycle lock — the same audit as the zedit post-open
renderer. Delivery takes its own short manager/player reads and never the zone
save lock.

## Proof

Unit (real parser, real command dispatch, real writer):

* `TestOeditSaveLoadedNullExtraLogsAndSkips` (pkg/session) — a seeded zone file
  with `E\n~\nBroken description~\n`; `oedit save 30` emits exactly the payload
  once at BRF/31/file-TRUE to an independent observer, nothing to a level-30
  observer, and skips the record while still writing the object.
* `TestOeditSaveEditedEmptyExtraWrites` (pkg/session) — the retained editor
  sequence reaches an allocated-empty description; the save writes
  `E\nallocated~\n~\n` and emits no diagnostic.
* `TestOeditReeditAllocatedEmptyIsSetNotNone` (pkg/session) — after the commit,
  re-entering the editor shows the empty description as set, not `<NONE>`.
* `TestLookNullDescriptionFallsThroughToObjectView` (pkg/game) — `find_exdesc`'s
  NULL-description semantics, and the scan does not continue to a later node.
* `TestExtraDescPresenceNeverChangesSaveBytes` (pkg/game) — a pre-existing save
  payload round-trips byte for byte and a NULL node serialises without the flags
  (the no-save-format-change proof).
* `TestStringObjExtraDescNullsPointerUntilWritten` (pkg/session) — do_string
  field 4's NULL-on-open rule and its field-6 consequence.

R5h controls, five cases, each 1 → 0 → 1 (`go test`, assertion red, never a
build failure):
`docs/fidelity/depth/handoff/2026-10-10-dp-1417-oedit-presence-controls.py`
(`presence-flag`, `writer-skip`, `producer`, `reedit-text-inference`,
`look-guard`).

Census: unit-green only — the harness does not drive the interactive OLC
editor, so there is no oracle vehicle with an independent BRF observer. The
retained unchanged-C reproduction
(`2026-10-07-dp-1371-oedit-null-C.py`) supplies the paired C bytes. A full
combined census is still required at the implementation tip and runs in Zach's
slot.

## Inventory and manifest

* `2026-10-06-dp-1371-mudlog-sites.tsv` `src/oedit.c:422`:
  `blocked-representation` → `implemented-boundary`, with the test list in
  `proof`.
* `mudlog.tsv` `mudlog.olc-oedit-corrupt-extra`: `blocked` → `unit-green`, with
  the test list in `proof`.
* `mudlog.unported-sites` stays `blocked` until the remaining ports and PR 4.

No `.obj` format change, no migration, no change to any player save byte.
