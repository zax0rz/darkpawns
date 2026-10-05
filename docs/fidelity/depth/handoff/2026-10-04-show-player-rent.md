# D3 durable reports train

Base: 84592b3b5, after #1785. Zach approved both metadata proposals and the rent legacy policy: preserve existing inventory, establish object-save metadata only at the next real object save. Governing documents and save JSON stay unchanged.

## Player

C reads char_file_u even for an online target (`src/act.wizard.c:2327-2347`). SQLite last_logon records the character save boundary; migration backfills existing updated_at once, as approved. Scalar report reads clear object payloads before conversion and never create world objects. Go sex encoding is translated to C's gender labels.

`src/db.c:2589-2595` advances played and the accounting timestamp on each save. A separate player accounting clock retains descriptor connection age. Store restoration starts a fresh accounting interval (`src/db.c:2436`). Session saves, creation saves, password confirmation, menu deletion, offline set-file and PlayerStoreEdit account before conversion. Creation phase projections reuse the same accounting snapshot instead of accounting twice. Playing-time readers (score, stat and veteran) use the accounting interval. Description-only SQL updates retain last_logon. Offline compare-and-save includes last_logon so a newer save cannot be overwritten.

Clock audit: DP_FIXED_TIME in the reference freezes calendar initialization, not process time(0) in char_to_store. Character accounting therefore uses an independently injectable wall-clock seam; the unit proof supplies exact times. The raw oracle verifies matching current-minute dates; unit proofs certify distinct persisted dates and exact hour/minute fields. The existing wall-clock normalization is extended only to the two anchored Started/Last date fields in this report; Played and all other bytes remain comparable. A compiling removal control catches minute rollover; unit proofs retain exact date semantics.

Proofs retained under `~/Archives/darkpawns/oracle-runs/2026-10-04/dp-1371-show-player-rent-proofs/`: empty-report-arm, removed elapsed accumulation and disabled legacy backfill controls each compile and fail assertions, then restored pass. The report unit proves live-vs-saved disagreement, exact scalar/date/played bytes and repeated reads without world allocations. SQLite test proves restart idempotence, description edits and stale-save rejection.

Retained date-reader follow-up: live stat and stat-file still format ConnectedAt. C live stat reads the runtime save-accounting logon (`src/act.wizard.c:755-758`); stat file substitutes saved last_logon (`src/act.wizard.c:1035`). The new metadata now permits that separate report repair. This train migrates their played-time reader only; it does not claim these pre-existing date fields complete.
