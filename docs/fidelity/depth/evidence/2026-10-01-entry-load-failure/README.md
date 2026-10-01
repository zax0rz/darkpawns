# DP-1371 E1: lookup/load failure

Fresh base: origin/main 1f7e32940, after PR #1749. The real SQLite conversion
boundary already rejects invalid character-data JSON and closes entry. However,
RecordToPlayer restored inventory/equipment through World.SpawnObject before
ApplyCharacterData decoded that field. A rejected restore left two fixture
objects in the live registry. before-fix.log retains that assertion failure on
the base source; it is not a guessed pre-existing failure.

RecordToPlayer now validates character data before allocating any player or
world-owned objects. The validator and ApplyCharacterData share one decoder.
Successful restores still apply character data after equipment, preserving the
existing attribute/affect ordering (R3b), legacy forms, schema and error text.
Decoding twice keeps validation free of game-state changes and avoids moving the
successful application boundary. No new saved-data format or player text.

C authority read for the call path: src/interpreter.c:1762-1795 loads the saved
character before Password; src/db.c:2342-2357 finds and reads its slot and
src/db.c:2412-2458 restores the character fields. Object restoration is later,
src/interpreter.c:2184. C load_char does not check fread's result. These unit
proofs certify the Go SQLite/JSON failure boundary; they do not claim defined
C bytes for a corrupt/truncated binary playerfile (R5f/R5g).

Class audit: every production RecordToPlayer call was read. Login and returning
menu reload propagate conversion errors; persistence.go's creationRecord and
storedPlayer do likewise. Manager's offline metadata reader strips objects and
passes nil world; its stored-player edit callback returns conversion errors.
All now use the same preflight. ApplyCharacterData has only this production
caller. Successful object restoration and final attribute application remain in
the same order. Inventory/equipment malformed JSON and missing prototypes have
their existing skip/fallback semantics; this batch does not change or certify
that separate object-recovery matrix.

Proofs: real SQLite syntax and typed-field errors through supplied-password JSON,
interactive JSON, terminal routing and name retry; no candidate, authentication,
registration, world admission or durable row rewrite. A saved record removed at
the password prompt closes without recreating it; an initially missing name
starts creation. Legacy absent/empty-object/null forms still authenticate. Both
registered inventory and equipment fixtures prove the rejected restore leaves
no objects; valid controls restore both. Real WebSocket frames and TCP telnet
prove the existing restore-error response and close before MOTD/world entry.
Browser rendering remains separate.

Six assertion-only 0/1/0 triples: object preflight, typed schema, restore abort,
missing password record, legacy acceptance, and transport error response. Replay
revert_proofs.py in a disposable checkout; all mutant failures are assertions,
with no build failures, panics or I/O timeouts.

The targeted lifecycle scenarios cover valid returning-record reconstruction,
preferences/load room, menu deletion and sole-slot replacement. Their clean
result is a successful-load regression control, not a corrupt C-file oracle
proof. A full census and claims census are required on completed production
source. No harness, oracle, tooling or governing-document changes.

Logs/manifests: ~/Archives/darkpawns/oracle-runs/2026-10-01/
dp-1371-p4-entry-load-failure-*. Final source hashes and exact census summaries
are retained alongside this note. This changes game code: open one PR and stop
for Zach. Next batch is entry.new-password, then entry.password-retries.

Completed-source gates pass: fmt, build, vet, all tests, cache-clean/lint (0 issues), diff check, fidelity-depth (5233 total; 5113 proven/delegated; 63 blocked), fidelity-units (1189/1189; 1177 rows, 968 symbols, 11 packages), string census, and focused race tests.
