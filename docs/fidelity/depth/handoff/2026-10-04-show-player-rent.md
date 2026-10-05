# D3 durable reports train

Base: 84592b3b5, after #1785. Zach approved both metadata proposals and the rent legacy policy: preserve existing inventory, establish object-save metadata only at the next real object save. Governing documents and save JSON stay unchanged.

## Player

C reads char_file_u even for an online target (`src/act.wizard.c:2327-2347`). SQLite last_logon records the character save boundary; migration backfills existing updated_at once, as approved. Scalar report reads clear object payloads before conversion and never create world objects. Go sex encoding is translated to C's gender labels.

`src/db.c:2589-2595` advances played and the accounting timestamp on each save. A separate player accounting clock retains descriptor connection age. Store restoration starts a fresh accounting interval (`src/db.c:2436`). Session saves, creation saves, password confirmation, menu deletion, offline set-file and PlayerStoreEdit account before conversion. Creation phase projections reuse the same accounting snapshot instead of accounting twice. Playing-time readers (score, stat and veteran) use the accounting interval. Description-only SQL updates retain last_logon. Offline compare-and-save includes last_logon so a newer save cannot be overwritten.

Clock audit: DP_FIXED_TIME in the reference freezes calendar initialization, not process time(0) in char_to_store. Character accounting therefore uses an independently injectable wall-clock seam; the unit proof supplies exact times. The raw oracle verifies matching current-minute dates; unit proofs certify distinct persisted dates and exact hour/minute fields. The existing wall-clock normalization is extended only to the two anchored Started/Last date fields in this report; Played and all other bytes remain comparable. A compiling removal control catches minute rollover; unit proofs retain exact date semantics.

Proofs retained under `~/Archives/darkpawns/oracle-runs/2026-10-04/dp-1371-show-player-rent-proofs/`: empty-report-arm, removed elapsed accumulation and disabled legacy backfill controls each compile and fail assertions, then restored pass. The report unit proves live-vs-saved disagreement, exact scalar/date/played bytes and repeated reads without world allocations. SQLite test proves restart idempotence, description edits and stale-save rejection.

## Rent

Independent `object_saves` SQLite rows record C's logical filename identity, kind and flat object snapshot. Existing players.inventory/equipment/character_data JSON and their restore paths are unchanged. No backfill guesses old object-save presence or kind. First crash/quit/free-idle save establishes metadata. Character-only saves, description edits and offline file edits leave it untouched. Identity is the lower-case C filename grouping, not character ID, so renaming does not silently move an old logical file. Menu entry rewrites an existing header to Crash without creating a missing file; self-delete removes the identity (`src/interpreter.c:2339`).

The projection reverses carried siblings, orders equipment by C wear index, then emits each sibling/child/object recursively, preserving signed container depth (`src/objsave.c:673-680,782-823`). New flat snapshot records encode ancestry by locate only. No JSON save-version bump. Report reads prototype load percent/description without read_object or world allocation, skips deleted prototypes, prints all recorded rent-code labels and uses the existing pager (`src/objsave.c:275-327`). The shipped free_rent=YES means legal quit and idle rent use RENT_RENTED (`src/config.c:106; src/limits.c:445-448`); the report's Cryo/TimedOut/Undef unit fixtures certify formatting, not an unimplemented paid-rent lifecycle.

Actual save hooks: World.SavePlayerRecord calls ObjectSaver after successful SaveCrash character saves; failure retains PLR_CRASH for retry. Legal quit and idle call it after NORENT filtering and before rented objects are extracted. Store loading rewrites only the object header. Snapshot storage does not become a second gameplay restore backend.

Other readers: login/menu inventory restoration, admin player JSON and existing save/edit fields retain their existing formats and sources. Game score/stat/veteran readers use the separate save-accounting clock. The new metadata is consumed only by these durable reports and object header lifecycle operations. Character-only writes remain distinct from object saves. Existing shop detailed-report gap remains separately retained.

Lock acquisitions: AccountCharacterSave takes/releases Player.mu before conversion. Conversion snapshots player/inventory/equipment under their existing separate locks; no new nested locks. Metadata queries/writes use SQLite with no held player/world lock. Report prototype lookup takes/releases World.mu per lookup before pager delivery. SaveCrash adds no lifecycle lock; quit/menu/idle preserve caller lifecycle behavior. Idle descriptor close finishes before the object snapshot. Pager/output delivery paths are unchanged. No new manager/lifecycle lock is acquired by any callback.

Revert triples: report arm, signed depth, equipment order, absent/empty distinction, crash producer, character-only gate, quit producer, idle producer, load header rewrite, returning-entry call, and delete identity each compile and fail assertions, then restored pass. Additional unit coverage pins repeated VNums, every label, float formatting, deleted prototypes, paging, save failure/retry and no world allocations. Raw oracle proves absent/Crash, equipped/nested/inventory records, snapshot persistence after live removal and replacement only on the next save. Five seeds are claimed in the combined census at the train tip.

Retained date-reader follow-up: live stat and stat-file still format ConnectedAt. C live stat reads the runtime save-accounting logon (`src/act.wizard.c:755-758`); stat file substitutes saved last_logon (`src/act.wizard.c:1035`). The new metadata now permits that separate report repair. This train migrates their played-time reader only; it does not claim these pre-existing date fields complete.

## #1791 review: rename cleanup

R3b/R5g: `src/act.wizard.c:2888-2895` changes the name, deletes the old
object-save file, deletes the old alias file, then saves. The name field now
performs the same cleanup order, retaining the DP-1381 guard and existing save
path. Failures are logged internally, as on menu deletion; no new player bytes.
No new locks: cleanup runs under the existing offline-name reservation.

`TestOfflineRenameDeletesOldRent` checks the old name's `show rent` bytes.
`TestOfflineRenamePreventsAliasInheritance` creates and logs in a later character
with the old name and checks its loaded aliases. Both failed on the original
code, pass with the repair, fail with each cleanup independently disabled, and
pass again on restore (R5h). Evidence is retained in
`~/Archives/darkpawns/oracle-runs/2026-10-05/dp-1791-rename-proofs/`.
Targeted census: `show-rent-stored`, `set-name-cleanup`, `set-depth`,
`set-extended-depth`, `set-gate-depth`. The new scenario saves the peer's objects
and aliases, disconnects it, and checks old/new rent identities around rename;
the unit proof covers later alias inheritance at the storage/login boundary.
