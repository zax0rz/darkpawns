# DP-1417 design: oedit extra-description presence (loaded-NULL vs editor-emptied)

**Status: approved (Claude, with Zach's ruling) and implemented on this branch.**
Approved as written: Option A; the presence flags are `json:"-"` (no
save-format change); the §5 look fix is in scope. See the port handoff
`2026-10-10-dp-1417-oedit-extra-presence.md`.
Base `origin/main` `e985ab07a`. Tier: stop (OLC persistence). Zach merges.
Author: DeepSeek. Order per `~/BRIEF-2026-10-09-mudlog-finish-deepseek.md`
(DP-1417 → DP-1415 → PR 4 → the `mudlog.unported-sites` flip → DP-1419).

This document answers the ruling on Linear DP-1417 and the recon in
`2026-10-07-dp-1371-mudlog-zone-seams.md:85-112`. It chooses nothing that the
issue reserved; §3 and §7 list the decisions the reviewer must make. If they
are answered as recommended, the coding PR follows on this branch.

R1/R2/R4/R5e/R5g/R5h apply. No census is planned for this design commit
(docs only). A census is running in another agent's slot; no gates were run
here, by census etiquette.

## 1. C ground truth (verified with grep, R5g)

An extra description is `struct extra_descr_data` with **two independent
pointers** (`keyword`, `description`); NULL and allocated-empty are distinct
states for each.

| Site | Behaviour |
|---|---|
| `src/oedit.c:417-426` | save guard: `if (!ex_desc->keyword \|\| !ex_desc->description)` → `mudlog("SYSERR: OLC: oedit_save_to_disk: " "Corrupt ex_desc!", BRF, LVL_BUILDER, TRUE)` and `continue` (record skipped) |
| `src/db.c:1475-1479` (obj), `:845-849` (room) | load: `new_descr->keyword = fread_string(...)`, `new_descr->description = fread_string(...)` |
| `src/db.c:2736-2740` | `fread_string` returns **NULL iff the accumulated field is empty** (`strlen(buf) > 0 ? CREATE... : rslt = NULL`). So at load, NULL ⟺ empty text — an exact bijection |
| `src/oedit.c:1466-1472` | `OEDIT_EXTRADESC_KEY`: `keyword = str_udup(arg)` |
| `src/olc.c:520-523` | `str_udup(txt) = str_dup((txt && *txt) ? txt : "undefined")` — the keyword is **never NULL** from the editor; an empty line becomes the literal `undefined` |
| `src/oedit.c:1506-1520` | `OEDIT_EXTRADESC_DESCRIPTION`: opens the improved string editor on `&OLC_DESC(d)->description` |
| `src/improved-edit.c:193-211` | deleting the last line leaves `*d->str` **allocated empty** (`*t = '\0'; RECREATE(*d->str, char, strlen(*d->str) + 3)`), i.e. non-NULL |
| `src/oedit.c:1522-1538` | `OEDIT_EXTRADESC_MENU` case 3 "goto next": when the current node is complete and has no `next`, `CREATE(new_extra, ...)` yields a **zeroed node (NULL/NULL)** and `OLC_DESC(d)` points at it |
| `src/oedit.c:163-335` | `oedit_save_internally` copies the whole struct (incl. the `ex_description` list) into `obj_proto[]`; it never normalises presence |

So after a builder commits, a prototype can hold a node that is (NULL, set),
(set, NULL), (NULL, NULL) or (set, allocated-empty). The writer's guard tests
two fields, but the decision it makes — log and skip — is per **node**, so a
single per-node flag cannot also drive the per-field `<NONE>`/`Set.` rendering
(§3).

The ruling's two named states are a subset of this:

1. **allocated-empty (valid):** edit → delete the last description line →
   `str_udup`/string editor keep both pointers non-NULL → save writes
   `E\n<kw>~\n~\n` with no diagnostic.
2. **loaded NULL (corrupt):** `fread_string` turns the empty tilde field back
   into NULL on the next boot, so the *next* zone save logs and drops the
   record. (`E\n~\n<text>~\n` — NULL keyword — hits the same guard.)

## 2. Go today (the two defects)

The parser's `ExtraDesc` is `{Keywords, Description string}` only
(`pkg/parser/obj.go:50-54`); it has no way to say "this text is empty because
C's pointer is NULL".

The oedit **descriptor** already models C's two pointers correctly and
independently: `oeditState.extraMeta []oeditExtraMeta{keywordSet,
descriptionSet}` (`pkg/session/oedit.go:129-147`), seeded by text
(`:271-277`), set true by the key editor (`:1272`) and the description editor
(`:1457`), and consumed by the menu's `<NONE>`/`Set.` rendering and the
"goto next"/"unlink incomplete" gate (`:723-734`, `:1281-1301`). That is the
transient editor state the 10-07 note warns not to depend on.

Presence is lost at **publication**: `saveOeditInternallyLocked` builds
`state.obj` and calls `world.CommitEditedObj` (`pkg/session/oedit.go:1474-1506`),
and `CloneObj` copies only the `[]parser.ExtraDesc`
(`pkg/game/world_oedit.go:12-17`). The prototype keeps text alone, so a later
`oedit` re-derives presence from text (`:271-277`).

The disk writer then drops every node with empty text, **without logging**:

```go
// pkg/session/oedit.go:1592-1599 (writeOeditObj)
for _, exDesc := range obj.ExtraDescs {
    if exDesc.Keywords == "" || exDesc.Description == "" {
        continue                       // C writes the allocated-empty case; C logs here
    }
    fmt.Fprintf(sb, "E\n%s~\n%s~\n", exDesc.Keywords, stripOeditString(exDesc.Description))
}
```

That is exactly the two defects of the issue: the valid editor-emptied record
is dropped on the **first** save (C writes it), and the corrupt record is
dropped **silently** (C logs BRF/LVL_BUILDER/TRUE).

`writeOeditObj` has one caller, `saveOeditZoneLocked`
(`pkg/session/oedit.go:1544`), which is shared by the telnet `oedit save`
(`:206`) and the admin/OLC save-all writer (`pkg/session/olc_control.go:106`,
`pkg/session/olc_registry.go:181`). So one writer change covers both.

## 3. The decision: where presence lives in memory

The issue requires presence to be **explicit**, carried through every writer
and reader, and never inferred from text or a transient descriptor. Three
placements are possible.

**Option A (recommended) — two presence fields on `parser.ExtraDesc`:**

```go
// pkg/parser/obj.go
type ExtraDesc struct {
    Keywords    string
    Description string
    // NullKeyword / NullDescription mirror C's two extra_descr_data pointers:
    // fread_string returns NULL for an empty tilde field at load, while the OLC
    // string editor can leave an allocated empty string. The zero value
    // (false/false) is C's valid allocated case, so every existing producer
    // (Lua extra(), do_string, the web OLC) keeps its current behaviour.
    NullKeyword     bool `json:",omitempty"`
    NullDescription bool `json:",omitempty"`
}
```

* Loaders set `NullKeyword = (text == "")`, `NullDescription = (text == "")`
  (`fread_string` is exactly that predicate, so this is definitional, not
  inference).
* The editor folds its `extraMeta` into the flags at commit
  (`saveOeditInternallyLocked`), so a committed allocated-empty node keeps
  `NullDescription = false` while a "goto next" node becomes `(true, true)`.
* `startOedit` seeds `extraMeta` from the flags instead of from text.
* The writer mirrors C's guard exactly (log + `continue` when either flag is
  set); the valid allocated-empty node is written.
* Per-field flags (rather than one node flag) are what keep the **menu** and
  the "goto next"/"unlink" gate correct on a *re-edit* of a committed
  allocated-empty node, which C renders as an empty-but-set description.

**Option B — one node flag (`Null`/`Corrupt`).** Smaller, and enough for the
writer's node-level guard, but the description editor is per-field: after a
commit of an allocated-empty description, a re-edit would infer
`descriptionSet = false` from the empty text and render `<NONE>`/`Not set`
where C renders the empty description/`Set.`, and "quit" would unlink the
node. That is a reachable player-facing divergence (R1). Not recommended.

**Option C — a parallel presence slice beside `[]ExtraDesc`.** Rejected by the
10-07 note's own warning: a second structure that must be kept in lockstep
through clone, apply and commit is exactly the transient/desynced
representation the ruling forbids.

Recommendation: **Option A**, with the two fields carrying `json:",omitempty"`
so that every pre-existing serialised value (present nodes) is byte-identical
— see §7 for the save-file question this raises.

## 4. Carry-through inventory (Option A)

Every place an `ExtraDesc` is created, copied, mutated, read or written:

| Surface | File:line | Change |
|---|---|---|
| Object load | `pkg/parser/obj.go:208-226` | set both flags from empty text |
| Room load | `pkg/parser/wld.go:241-247`, `:266-278` | set flags too (harmless; room writer ignores them) |
| Object clone | `pkg/game/world_oedit.go:12-17` | none — struct copy carries the flags |
| Prototype publish | `pkg/game/world_oedit.go:61-90` (`CommitEditedObj`), `:108-123` (`RefreshLiveObjInstances`) | none — `CloneObj` carries them |
| Editor seed | `pkg/session/oedit.go:271-277` | derive `extraMeta` from the flags, not text |
| Editor key/description edits | `pkg/session/oedit.go:1264-1275`, `:1432-1465` | clear the corresponding flag (allocated non-NULL) |
| Editor "goto next" / unlink | `pkg/session/oedit.go:1276-1316` | new node keeps `(true,true)`; unlink unchanged |
| Editor commit | `pkg/session/oedit.go:1474-1506` | fold `extraMeta` → flags before `CommitEditedObj` |
| Disk writer | `pkg/session/oedit.go:1562-1605` | C's guard: log BRF/`LVL_IMMORT`/TRUE and `continue` |
| Instance effective list | `pkg/game/object.go:291-308`, `pkg/game/runtime_state.go:38-45`, `pkg/game/live_strings.go:84-97` | flags ride along in the value |
| `look <kw>` | `pkg/game/look.go:1230-1237` (+ call sites `:495-529`) | see §5 |
| do_string field 4/6 | `pkg/session/string_cmd.go:290-330`, `:343-367` | field 6's empty-description test becomes the flag (`:351`) |
| Lua `extra()` | `pkg/game/world_bridge.go:941-956` | zero value (present) — matches C's `str_dup` |
| Lua `SetObjectExtraDesc` | `pkg/game/world.go:425-448` | zero value (present) |
| Web OLC add/set | `pkg/admin/olc_write.go:359-371`, `pkg/admin/olc_entity_write.go:545-553`, `pkg/olc/apply.go:547-632` | zero value (present) |
| Draft clone/diff | `pkg/olc/entity_draft.go:243-245`, `pkg/olc/draft.go:90-107` | value copy carries flags; optionally diff them |
| Note writer | `pkg/session/comm_cmds.go:236-241` | zero value (present) |
| `wiz_stats` display | `pkg/session/wiz_stats.go:562` | read-only; leave |

The `apply.go` / admin `parser.ExtraDesc{...}` constructors keep working
unchanged because the zero value means "allocated/present", which is what C's
`str_dup`-based writers produce.

## 5. R5e finding: the look path also loses the distinction

`find_exdesc` (`src/act.informative.c:987-996`) returns the **first keyword
match's `description` pointer**, and every caller tests it against NULL
(`:1033-1066`). So:

* a node whose keyword matches but whose description is NULL is "not found":
  the caller falls through to the next list and eventually to
  `show_obj_to_char(found_obj, ch, 5)` — it does **not** send nothing-and-return;
* a node whose description is allocated-empty (`""`) **is** found
  (`send_to_char("")` then return), which prints nothing but short-circuits;
* a NULL keyword never matches at all — `isname_with_abbrevs(word, NULL)`
  returns 0 (`src/handler.c:115-134`), and Go's `isnameWithAbbrevs(str, "")`
  already returns false, so the keyword side needs no change.

Go's `findExtraDescription` (`pkg/game/look.go:1230-1237`) returns
`(Description, true)` for any keyword match, including a NULL-description node,
so that case short-circuits where C falls through. The fix returns not-found at
the first keyword match whose description is NULL (C returns on the first
match; it does not continue scanning later nodes):

```go
for _, d := range descriptions {
    if isnameWithAbbrevs(name, d.Keywords) {
        if d.NullDescription {
            return "", false // find_exdesc returned NULL; callers fall through
        }
        return d.Description, true
    }
}
```

`pkg/game/live_strings.go:99-109` already documents this NULL-pointer nuance
for do_string; `stringObjDeleteExtraDesc` currently implements it as
`descs[index].Description == ""` (`pkg/session/string_cmd.go:351`), which
should switch to the flag for the same reason.

## 6. Save / abort / reload semantics

* **On-disk format is unchanged.** The writer's bytes for a valid node are the
  same `E\n<kw>~\n<desc>~\n`; an editor-emptied description still serialises as
  `E\n<kw>~\n~\n`.
* **Reload re-creates the NULL state**, exactly as C does: the parser sets
  `NullDescription = true` for the empty field, so the next zone save logs and
  drops the record. The ruling's "survives the first save, warns and deletes it
  after a reboot" is therefore reproduced without any file-format change.
* **Abort** (quit without saving) never publishes the working copy, so no
  flags reach a prototype; nothing changes there.
* **Position of the producer** is inside the per-object scan, in C's exact
  place: after the object's `S` script line and before its `A` affect lines
  (`src/oedit.c:409-444`), and it can fire once per corrupt node, per object,
  per zone save. `writeOeditObj` runs with only the per-zone save lock held —
  `world.SnapshotObjs()` has already released `world.mu` before the render
  loop — so calling `MudLog` there is safe under the #1825 lock audit (no
  world/player/manager/zone lock held).
* The Go call mirrors C's `LVL_BUILDER`, which `src/olc.h:54` aliases to
  `LVL_IMMORT` (31): `game.MudLog("SYSERR: OLC: oedit_save_to_disk: Corrupt
  ex_desc!", game.MudlogBrief, game.LVL_IMMORT, true)`.

## 7. Decision 2 — the player save file (stop-and-report)

Adding fields to `parser.ExtraDesc` is not confined to world data:
`ObjectRuntimeState.ExtraDescs []parser.ExtraDesc`
(`pkg/game/runtime_state.go:44`) is JSON-serialised into player saves through
`SaveItemData.State` (`pkg/game/save.go:97-104`, `pkg/db/object_save.go:47-56`).
That is the "do not change save file format" surface in AGENTS.md, and the
brief lists a save-format question beyond DP-1404 as a stop-and-report item.

With `json:",omitempty"` on the new booleans:

* every pre-existing save and every "present" node serialises **byte for
  byte** as today (false omits the key); old saves unmarshal with false, i.e.
  present, which is the correct default;
* a key appears only for a node whose C pointer is NULL that has reached a
  `Runtime.ExtraDescs` snapshot (i.e. do_string edited the extra descriptions
  of an instance whose prototype holds a corrupt node). No pre-existing data
  takes that path.

Options for review:

1. **Recommended** — accept the two `omitempty` booleans (backward-compatible;
   the narrow new key is a new state, not a change to any existing value).
2. Give the fields `json:"-"` so they never serialise. Simplest guarantee, but
   an instance-level `Runtime.ExtraDescs` snapshot loses presence across a
   relogin, so `look`/do_string on such an instance would treat a corrupt node
   as present after a reboot. World prototypes (what the writer reads) are
   unaffected either way.
3. Keep the flags off the shared type entirely and store presence in an
   object-only structure — rejected in §3/§4 (lockstep/desync risk).

Recommendation: option 1. Either way this is the item that needs Zach's
explicit nod, because it touches the saved-object schema.

**Ruling (Zach, DP-1417 review): option 2 — `json:"-"`.** There is no
save-format change. Presence is derived from empty text wherever an extra
description is reconstructed from stored data (the `fread_string` load rule),
and never has to survive a player save, because C's `objsave.c` stores no extra
descriptions. Implemented; `TestExtraDescPresenceNeverChangesSaveBytes` proves a
pre-existing payload round-trips byte for byte.

## 8. Proof plan (R5h — each piece must be able to fail)

Unit (real dispatch, real parser, real writer; an independent BRF syslog
observer at level 31 with `log` on, a level-30 negative, and the file flag):

1. `TestOeditSaveLoadedNullExtraLogsAndSkips` — zone `.obj` seeded with
   `E\n~\n<text>~\n` (NULL keyword) and `E\n<kw>~\n~\n` (NULL description):
   `oedit save <zone>` emits **exactly** the payload once per corrupt node at
   BRF/31/file-TRUE, and the record is absent from the saved file.
2. `TestOeditSaveEditedEmptyExtraWrites` — drive the real editor
   (`f` → `1` → keyword → `2` → `/c` → line → `/d1` → `/s` → `0` → `q` → `y`,
   as in the retained 10-07 C reproduction), then `oedit save`: the file
   contains `E\n<kw>~\n~\n` and no producer fires.
3. `TestOeditReeditAllocatedEmptyIsSetNotNone` — after (2), re-open `oedit`:
   the extra-desc menu shows the empty description (set), not `<NONE>`; the
   per-field presence proof.
4. `TestLookNullDescriptionFallsThrough` — keyword matches, description NULL:
   `look <kw>` falls through to the generic object view exactly as C's
   `find_exdesc` returning NULL does. Paired with the allocated-empty case,
   which is found and prints nothing.

Controls (`*-controls.py`, compiling assertion-red, then restored; R5h):

* remove `NullDescription` at load → tests 1/2 fail on their assertions;
* remove the writer's `continue` → the corrupt record is written;
* remove the writer's `MudLog` → the log assertion fails;
* revert `startOedit` to text inference → test 3 fails;
* remove the look guard → test 4 fails.

Census/vehicle: the site is reachable through builder `oedit` (valid play
under the ruling), so try an oracle vehicle that drives a level-31 builder
through the corrupt-then-save sequence with an independent BRF observer; if the
harness cannot drive the interactive editor, classify the row **unit-green
only** with the retained unchanged-C reproduction
`2026-10-07-dp-1371-oedit-null-C.py` as the paired bytes, and say so in the PR.
The issue requires a **full combined census** at the implementation tip either
way.

## 9. Inventory / manifest updates (in the coding PR)

* `docs/fidelity/depth/handoff/2026-10-06-dp-1371-mudlog-sites.tsv:84`:
  `blocked-representation` → `implemented-boundary`, tests named in `proof`.
* `docs/fidelity/depth/mudlog.tsv:108` `mudlog.olc-oedit-corrupt-extra`:
  `blocked` → `unit-green` (or `oracle-green`) with the proof pointer; drop the
  "Pending port DP-1417" note.
* `make fidelity-depth`, the inventory checker, `make string-census`; then the
  standard gates, a focused `-race` run on `pkg/session`, `pkg/game`,
  `pkg/parser`, `pkg/olc`; then the census.

## 10. Decisions requested from Claude / Zach

1. **Representation:** Option A (two per-field null flags on
   `parser.ExtraDesc`) vs Option B (one node flag). Recommended: A.
2. **Save schema:** may the two `json:",omitempty"` booleans ride on
   `parser.ExtraDesc` (§7 option 1), or must they be `json:"-"` (§7 option 2)?
   This is the save-format nod.
3. **Look fix in scope:** confirm the `findExtraDescription` NULL-description
   fix (§5) belongs in this PR rather than a separate one (same defect class;
   the issue names the look path).

Nothing is coded until 1–3 are answered. No gates and no census were run for
this docs-only commit (a census was running in another agent's slot).

**Resolved (Claude, with Zach's ruling, 2026-10-10):** (1) Option A approved;
(2) `json:"-"` — no save-format change, derive presence from empty text on load;
(3) the look fix is in scope, and `src/modify.c:718` was checked (the fifth
`find_exdesc` caller — see the port handoff). Implemented on this branch.
