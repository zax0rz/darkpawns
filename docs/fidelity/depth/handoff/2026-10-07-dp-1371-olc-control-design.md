# D7 OLC control seams: design and decision boundary

Stop tier: missing command routing and shared dirty-list ordering change persistence and session behavior. This note changes no implementation or manifest verdict.

## Verified problem

C exposes `olc` at builder level (`src/interpreter.c:590`). In `do_olc`, NPCs return silently; a first argument whose first four bytes are `save` selects save-all, and other arguments select save-info (`src/olc.c:80–100`). This dispatch happens before the editor-specific authorization path. Go currently has no corresponding session command.

C's dirty list prepends a new `(zone, kind)` entry, leaves an existing entry in place when marked again, and removes an entry on successful save (`src/olc.c:382–413`). Go's one shared `pkg/olc.SaveList` uses a map and exposes a sorted value snapshot. Its sorted `List` contract already has callers and a test. Save order is observable: `olc_saveinfo` enumerates the list, and `olc_saveall` prints the per-entry acknowledgement before invoking that entry's writer (`src/olc.c:313–365`). The final producer is `OLC: <acting name> saves all`, CMP, builder level, file TRUE, after the list is empty. An initially empty list prints `The database is up to date.\r\n` and emits no producer.

The existing admin `Manager.SaveOLCZone` saves all five file kinds and rejects an active claim in the zone (`pkg/session/olc_registry.go:153–187`). It is not the C save-all command, which visits only dirty entries in list order. Reusing it would change the command's contract.

## Proposed implementation after approval

1. Extend the existing SaveList to retain insertion order under its existing mutex. Keep sorted `List()` for existing admin consumers and add a C-facing newest-first snapshot. Repeated marks do not move an entry; remove then mark creates a new head. There is one registry, not a parallel dirty list.
2. Add the actual builder-level `olc` dispatch, preserving C argument selection and NPC refusal behavior. Save-info prints the exact C heading, labels and list order.
3. Save each dirty entry through its existing per-kind writer, acknowledging before the write, removing only on success, and emitting the final MudLog only after completion. Keep the admin endpoint's existing contract. Audit concurrent edits and marker removal so a save cannot erase a newer edit's dirty state.
4. Audit the other seven control sites separately. Description states and invalid-action defaults need call-path reachability proofs before classification. Do not equate the text editor's ordinary unknown slash command with the inner `parse_action` invalid-action producer: the outer dispatcher handles unknown slash commands itself (`src/improved-edit.c:50–87`), while the inner default is at 473–476. MEDIT cleanup followed by subsequent C state access requires its own reachability/undefined-behavior assessment; it is not license to introduce a Go crash.

## Decision required: failed save-all

C's save-all loop rereads the list head until the writer removes it (`src/olc.c:332–339`). A failed room-file open logs and returns without removing that head (`src/redit.c:289–295`; removal is at 393). Therefore a persistent obstruction causes repeated acknowledgements and repeated attempts without command completion. This follows from the source; no claim of a completed oracle failure transcript is made here.

Two choices are reviewable:

- Preserve the retry behavior. This needs a separately reviewed execution/cancellation design, because copying an unbounded synchronous loop into Go can monopolize the command path and produce unbounded output. No cancellation semantics are assumed approved by this note.
- Approve a bounded failure divergence: attempt the head once, preserve its dirty marker, retain the existing per-kind diagnostic, return without the final success producer, and leave subsequent entries untouched. This requires Zach's explicit approval and a tracked divergence. No new player-facing error string is proposed.

Until that decision, save-all remains blocked. Successful saves and save-info may be implemented as a separate bounded train only if their failure boundary is expressly agreed; this note does not silently classify failed saves as excluded.

## Proof and lock plan

Each case gets a separate commit and an assertion-failing, compiling revert triple. Cover newest-first order across kinds and zones, repeated mark, remove/re-mark, exact argument dispatch, empty-list silence, acknowledgement-before-write, dirty marker retention on failure, and final producer timing/type/threshold/file flag. Use an independent syslog observer and paired C output for successful save-all. Stage file failures only in disposable fixtures.

List every acquired lock at the implementation tip. The intended shape is a brief SaveList snapshot acquisition, released before any writer; a single zone save lock around the existing per-kind snapshot/write path; then release all save/world/dirty-list locks before final MudLog delivery. Audit the writer's actual world and dirty-marker acquisitions and concurrent edit boundaries before claiming that order safe. Do not acquire player lifecycle locks for this command. Run race coverage over dirty marks versus save completion and all normal gates, then one combined census at the implementation tip.

## Inventory continuity

The earlier `local-01` report producer remains missing: `do_gen_write` logs `<acting name> <command>: <normalized argument>` before opening its report file, CMP, immortal level, file FALSE (`src/act.other.c:1118–1120`). Go's `doGenWrite` already normalizes the argument but lacks that MudLog call. Keep it as an independent bounded case; this design neither resolves it nor folds it into OLC. Previously landed producer proofs must be reconciled individually rather than trusting stale inventory state labels. The MEDIT/SEDIT checked-write investigation remains blocked under the earlier approved boundary.
