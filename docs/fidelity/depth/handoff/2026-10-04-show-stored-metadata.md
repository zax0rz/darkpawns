# D3 stored-report metadata audit

Audit base: f89b4a441. `show.shops-list` proceeds as a report over existing parsed/live records. `show.player-valid` and `show.rent-valid` remain blocked; the next implementation needs a metadata decision under the goal's unchanged-save-format constraint. No persistence format or schema changes are made in this train.

## Player

C `do_show` reads the durable `char_file_u`, even when the character is online (`src/act.wizard.c:2327-2347`). It prints sex/class/level, gold/bank/exp/alignment/practices, birth, last_logon and accumulated played time. `char_to_store` sets last_logon to the save time, accumulates time since the previous accounting boundary, and advances the runtime accounting timestamp (`src/db.c:2589-2595`). The label Last therefore describes the persisted C save boundary, not simply the start of this connection.

Go preserves birth and played, and the other displayed scalars, in existing columns plus CharacterData (`pkg/game/save.go`, `pkg/game/character_data.go`, `pkg/db/convert.go`). It does not preserve an explicit last_logon. ConnectedAt is runtime-only. Loading a character constructs a new ConnectedAt, so using the restored value would report the read time. SQLite updated_at exists, but is general mutation metadata written with CURRENT_TIMESTAMP: SavePlayer, SavePlayerIfCurrent and UpdateDescription all change it (`pkg/db/player.go`). It is not exposed by PlayerRecord, is not driven by DP_FIXED_TIME, and has no certified C save-boundary contract. Reusing it as last_logon requires a writer audit and explicit semantics, not just a formatter.

Recommendation for a separate stop-tier design/implementation: retain explicit C last_logon metadata in SQLite, using the shared real-time seam at the char_to_store-equivalent boundary; audit all save/edit/entry/disconnect callers and played-time accounting, including the distinction between connection age and C's save accounting clock. Read the stored record through an object-free report projection: restoring inventory just to print scalar fields repeats the loaded-object ownership class. Prove online-versus-durable disagreement, two different birth/last dates, multi-hour/minute played time, edits and save boundaries, deterministic time, and no object allocations.

Decision needed: authorize SQLite-only metadata and choose the old-record migration policy. Historical last_logon cannot be reconstructed exactly from a save that never stored it. Prefer an explicitly approved migration from the existing updated_at as the best retained legacy timestamp, then exact C boundaries for all new saves. Do not silently use current time, Birth, or a connected descriptor as the missing field. No new divergence is claimed or approved here.

## Rent

C `Crash_listrent` opens the independent object-save file, prints its actual filename, prints its recorded rentcode (Rent, Crash, Cryo, TimedOut or Undef), then walks its saved object records and prints prototype load percent, C locate and short description; it ignores records whose prototypes no longer exist (`src/objsave.c:275-327`). `Crash_save` emits sibling/child/object order, with negative locate for container depth (`src/objsave.c:673-680`). Filename grouping is defined by `src/utils.c:527-586`; rentcode constants are `src/structs.h:586-591`.

Go's SQLite inventory/equipment JSON retains VNum, Count, Locate and parent ContainerIndex/ContainerVNum (`pkg/game/save.go:97-103`, `pkg/db/convert.go:221-265`). This is sufficient to investigate a faithful order/depth projection without changing JSON. It does not retain whether an independent C-equivalent object save exists, its save reason, or the distinctions between crash/rent/cryo/forced/timedout. A player record's existence alone does not prove an object-save file exists. House saves are unrelated and cannot fill the gap.

Recommendation: retain explicit object-save presence and kind in SQLite at the existing object persistence boundaries, separate from character-only saves. Derive C record order/locates from the retained object topology with failing-first tests; report only the recorded object-save snapshot, not live inventory. Require a reviewed compatibility mapping for the filename header: a display string alone would invent a C file that the port does not store. Prove each save kind, empty versus absent snapshot, equipment/containers/depth, deleted prototypes, repeated VNums, restore/restart, and no allocation or mutation from the report.

Decision needed: choose the compatible metadata/storage mapping under the unchanged-save-format constraint. Prefer SQLite-only metadata and an explicitly recorded logical object-save identity, with a documented legacy migration policy. Keep the row blocked until approved; do not label old unknown saves Crash or Undef just because one yields convenient output.

## Retained shop follow-up

The current row certifies `list_all_shops`, not `list_detailed_shop`. Arguments selecting detailed shops remain unported (`src/shop.c:1312-1444`). That report includes runtime keeper gold, SHOP_BANK, preserved native special assignment and wrapped products/buy-word lists. Those state readers need their own scope/audit; the complete listing is not proof of them. This train retains that gap instead of claiming the whole shops command complete.
