# PROPOSAL: the lifetime of a live extra-description edit (DP-1292)

**For:** Zach. **Written by:** DeepSeek, 2026-11-14. **Status:** proposal only — no
code, no manifest change, no scenario. **Scope:** the command the issue cites,
`string obj <name> description <keyword>` (`src/modify.c:594-772`), not the OLC
editors. An `oedit` extra-description edit is written to the zone file by
`oedit_save_to_disk` and is not affected by this document.

## The claim under test

> An OLC edit to an existing extra description is lost at save and reload.
> C changes the description node shared with the prototype (`src/modify.c:703-707`;
> `read_object` makes a shallow copy at `src/db.c:1951`; `free_obj` skips
> prototype-shared lists).
> Go snapshots the description per instance and never persists it
> (`pkg/game/live_strings.go`, `pkg/game/object.go` `GetSaveState`).

The C half is right, and its mechanism is narrower than "save and reload": the
edit survives *everything except a reboot*. The Go half is right, and its
mechanism is wider than "not persisted": the edit dies with the instance, so it
also disappears for every other session and for the actor after a relog. The
sections below separate those two windows, because they need different fixes and
only one of them is about the save file.

## What C does, step by step

1. **`read_object` aliases the prototype's list.** `CREATE(obj); clear_object(obj);
   *obj = obj_proto[i];` (`src/db.c:1947-1951`; the struct copy is line 1949).
   The copy is shallow, so `obj->ex_description` is the *same pointer* as
   `obj_proto[i].ex_description`. Every instance of a prototype shares one list.
   `free_obj` knows this: for a prototype instance it frees strings only when they
   differ from the prototype's pointers (`src/db.c:2845-2866`).
2. **Field 4 finds the node before the editor opens** (`src/modify.c:683-709`):
   - *Existing keyword* (`str_cmp` equal, case-insensitive, `src/modify.c:701-707`):
     `FREE(ed->description); ed->description = 0; ch->desc->str = &ed->description;`
     — the editor's write target is the **shared prototype node's field**.
   - *New keyword* (`src/modify.c:690-700`): a fresh node is allocated, linked in
     front of `obj->ex_description`, and only the *instance's* head pointer moves.
     `obj_proto[i].ex_description` still points at the old chain.
3. **The editor writes through that pointer.** Every accepted line lands in
   `*d->str` (`src/improved-edit.c:183-209,355-384`), so the existing-keyword edit
   changes the prototype's node in place — for the whole process.
4. **Nothing on disk carries it.** The player object file is
   `struct obj_file_elem` (`src/structs.h:770-786`); its only strings are
   `name[128]`, `shortd[128]`, `desc[256]`. `Obj_to_store_from`
   (`src/objsave.c:102-145`) writes those three and no extra descriptions;
   `Obj_from_store_to` (`src/objsave.c:58-92`) rebuilds the object with
   `read_object(object.item_number, VIRTUAL)` and restores only those three
   strings. So a relog re-reads the *live in-memory prototype*, which still holds
   the edit. A reboot reloads the world files and loses it.

### Which players see it, and when

`string` is registered `POS_RESTING, LVL_IMMORT+1` (`src/interpreter.c:745`) —

| situation (existing keyword) | C | Go today |
|---|---|---|
| the editing builder, same session | sees the new text | sees it |
| any other session, `look <kw>` on any instance of that prototype, same boot | sees it (shared node) | **old text** (instance snapshot) |
| the editing builder after quit + login, same boot | sees it (prototype re-read) | **old text** |
| everybody after a reboot | old text (world file wins) | old text |
| new keyword, any other instance | never sees it (instance-local head) | never sees it |
| new keyword, editing builder after relog | **old text** — the node died with the instance | old text |

Visible-once-it-lands paths: `look <keyword>` resolves room extra descriptions at
`src/act.informative.c:1033`, equipment at `:1041`, carried objects at `:1050`,
and room objects at `:1061`. Nothing gates those on level, so the audience is every
player who looks.

### C's memory bugs on the same path — do not port these

- **Instance free after a new-keyword add frees prototype-shared nodes.** The head
  pointer now differs from the prototype's, so `free_obj`'s guard at
  `src/db.c:2855-2857` passes and the loop frees the *entire* chain, including
  nodes `obj_proto` still points at. The prototype's list dangles (undefined
  behaviour, R1a).
- **Field 6 deletes only the head, and otherwise crashes.** `do_string` resets
  `ed = obj->ex_description` (`src/modify.c:717`) after `find_exdesc` returns,
  then compares only the head (`:722`); a non-head keyword falls through to the
  common tail with `field == 6`, reads `length[5]` past the five-element array and
  writes through a `d->str` that was never set. The port already records this as a
  forced divergence (`string.delete-non-head-extradesc` in
  `docs/fidelity/depth/string.tsv`, `TestStringDeleteNonHeadExtraDescRefusesInsteadOfCrashing`).
- A non-head delete that *did* take the intended branch would relink shared nodes
  (`tmp->next = ed->next` on a prototype node), mutating the prototype.

The consequence for this proposal: "match C" must mean *match the bytes C's
players saw*, not *copy C's list surgery*. The existing forced divergence is the
precedent, and it is the reason option A below is scoped to the one case that is
not undefined behaviour.

### One more C fact the options depend on: the edit is not always ephemeral

`string` writes only memory — but that memory *is* `obj_proto[]`, and C's OLC
object writer serialises `obj_proto[i].ex_description` (`src/oedit.c:409-448`,
"Yep, save them too"). So a later `oedit save <zone>` in C writes the builder's
`string` edit into the `.obj` file: it becomes durable as a side effect. In the
port the writer reads `world.SnapshotObjs()` (`pkg/session/oedit.go:1554`), i.e.
`World.objs`, so a live-prototype write-through reaches the same file at the same
moment. Both engines lose the edit at a reboot, and both keep it if an OLC save
of that zone happens before the reboot.

## What the port does now

- `string obj … description <kw>` is `pkg/session/string_cmd.go:291-341`.
  An existing keyword clears and rewrites the **instance's** copy
  (`SetLiveExtraDescs`, `pkg/game/live_strings.go:85-97`); a new keyword prepends
  to that copy. The prototype is never touched.
- An instance owns a snapshot from its first `do_string` touch
  (`Runtime.ExtraDescsLive`, `pkg/game/runtime_state.go:46-48`); until then
  `GetExtraDescs` reads `Prototype.ExtraDescs` (`pkg/game/object.go:330-334`).
- The save state carries no extra descriptions: `GetSaveState`
  (`pkg/game/object.go:431-470`) persists name/short/long/mold/mail/script and
  `CustomData`, none of which hold `Runtime.ExtraDescs` — deliberately, since
  `obj_file_elem` has no such field either.
- Prototypes are shared pointers: `GetObjPrototype` returns the live
  `*parser.Obj` under `RLock` (`pkg/game/world_zone.go:19-23`). The machinery for
  an atomic prototype swap already exists — `CloneObj`, `SnapshotObj`,
  `CommitEditedObj` (`pkg/game/world_oedit.go:12-90`), `SetObjScript`
  (`:129-150`), `RefreshLiveObjInstances` (`:108-123`) — so a write-through needs
  no new data model, only a new, narrower operation.


## Options

### A. Prototype write-through for keywords the prototype already has (recommended)

Rule: when field 4 names a keyword that exists on the prototype, write the text
to the **live prototype** (and to live snapshots that carry the same keyword);
when it does not, keep today's instance-local behaviour (C's new node is
instance-local too).

- **Mechanism:** a new `World.SetLiveObjExtraDesc(vnum int, keyword, text string) bool`
  that clones `w.objs[vnum]`, rewrites the matching entry in `ExtraDescs`, stores
  the clone in `w.objs`, and repoints live instances of that vnum at it without
  clearing `AffectsOverride`/`ValuesOverride`/`TypeFlagOverride`/`WeightOverride`
  (unlike `RefreshLiveObjInstances`, whose override clearing mirrors a whole-struct
  OLC replace that C does not perform here). Live snapshots holding that keyword
  get the same text so the shared-node visibility C has is preserved.
- **Durability:** `parsedData` is left alone, exactly as C leaves the zone file
  alone; `SnapshotObjs` (`World.objs`) is what `saveOeditZone` writes, so the C
  side effect described above is reproduced.
- **Fidelity:** matches C for every case that is not undefined behaviour, and
  matches the existing forced-divergence precedent for the cases that are.
- **Cost:** ~1 world method + 1 call-site change + unit tests + one oracle
  vehicle. No save-format change. Locking is the only real risk: `w.objs` is read
  under `RLock` everywhere, and the write must happen under `Lock` with a clone,
  never by mutating the pointed-at struct (readers hold escaped pointers).
- **Residual difference:** an instance whose snapshot was taken *before* the edit
  and whose keyword the prototype does not have keeps its own text; C would too
  (that node is instance-owned). An instance whose snapshot holds a
  prototype-derived keyword is patched by the rule above.

### B. Keep the instance-local overlay and record the divergence

No code. Add a `divergent-approved` row naming the visible difference ("other
sessions and relogin revert to the prototype text") and have Zach approve it. This
is honest and cheap, but it accepts a builder-visible difference on a path the
issue already calls a "mystery bug waiting to happen", and it leaves the C
side-effect durability (`oedit save`) unmatched.

### C. Persist the instance snapshot in the player save (the issue's original option)

**Reject.** It is not what C does: `obj_file_elem` has no extra-description field
(`src/structs.h:770-786`), so a reboot wipes the text in C while this option would
keep it. It changes the save format (a step `AGENTS.md` reserves for Zach), and it
fixes only the actor's own instance: the other players' view, which is the larger
half of the divergence, would still be wrong until the actor re-logs.

## Recommendation

**Option A**, scoped to existing prototype keywords, with option B's ledger row kept
for the new-keyword/instance lifecycle difference (which is not a divergence —
both engines lose those at extraction). Concretely:

1. World API + snapshot patching + the `string` call site (one PR).
2. Proof (R5h): an oracle vehicle with two immortals in one room —
   `string obj sword description blade` from actor A, `look blade` from actor B in
   the same boot (C: new text, port today: old text), then `<RELOGIN>` for B and
   `look blade` again (C: still new; port today: old), then a `<RESTART>` vehicle
   to show both engines reverting. Plus unit tests for the world API and for
   "snapshot taken before the edit still sees it".
3. A depth row for the cross-session visibility case, citing `src/modify.c:701-707`
   and `src/db.c:1947-1951`; the in-session bytes stay with
   `string.extradesc-modify`.

If Zach prefers not to add prototype mutability at all, **option B** is the
fallback; I would want the ledger note to say plainly that C keeps the text until
reboot and the port does not.

## Open questions for Zach

1. **Scope of the write-through.** Only `string obj … description <existing kw>`,
   or also the object `name`/`short`/`long` fields? Those are instance-scoped in C
   (`d->str = &obj->name` replaces the instance's pointer; `free_obj`'s pointer
   comparison proves it), so I recommend leaving them alone. The `string` *delete*
   path is already a forced divergence and stays one.
2. **Is a live prototype mutation acceptable at all?** C has it; the Go model is
   immutable prototypes with atomic swaps. Option A preserves the atomic-swap
   discipline (clone + publish), but it does make an OLC-visible prototype change
   reachable from a non-OLC command, exactly as C does.
3. **Does anything persist the prototype to disk in the port besides
   `saveOeditZone`?** I found only that writer. If a future world export also
   serialises `SnapshotObjs`, the C side-effect durability would need a decision.

## What this proposal does not do

No code, no manifest row, no scenario, no rulebook edit. If approved, the
implementation is a separate PR with its own red/green and gates.
