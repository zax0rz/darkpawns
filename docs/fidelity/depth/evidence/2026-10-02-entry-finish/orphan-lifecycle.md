# Reachable descriptorless teardown: narrow lifecycle repair

The third topology is **reachable**, so it is repaired and unit-green, not excluded. No registry redesign or invented world-body fallback is needed.

## Production reachability

Every Go gameplay body admission originally called Register and then World.AddPlayer separately (guest login, returning menu entry, first creation). Unexpected transport loss keeps a retained Session and marks its body linkless; an ordinary linkdead body is therefore not the unsupported Session-less fixture.

However, Unregister, UnregisterSession and UnregisterAndClose deleted the retained descriptor before cleanupSession saved the body to SQLite and removed it from the world. A slow save exposes a real world body with no retained session. A new password-accepted login can then run performDupeCheck, see no session and miss the body. World admission can also race retirement between Register and AddPlayer. These are operation-interleaving windows; C's game_loop (src/comm.c:458-656) executes input/nanny and close_socket sequentially. perform_dupe_check is src/interpreter.c:1528-1659; close_socket is src/comm.c:2086-2156.

`TestEntryDuplicateTeardownSerialization` enters a real SQLite-backed character via login/menu. A GameStore wrapper pauses the actual teardown SavePlayer without changing registries or IDs. It proves the body is still in the world after its descriptor was removed. Duplicate selection and new admission must wait until that teardown completes; afterwards selection finds no retiring target, or admission retains exactly its new session and body. Before the repair all three unregister variants fail the explicit premature-lookup assertion. The final matrix includes shutdown retirement and concurrent admission too.

## Repair / whole-class audit

A small Manager lifecycle mutex serializes duplicate adoption, registration, complete registration-plus-world-admission, transport-loss save/detach and retirement. Manager/world locks still never nest. Internal register/unregister helpers avoid self-deadlock for menu duplicate close and admission rollback. All three production AddPlayer call paths now use enterWorld; a repository-wide search leaves only that helper's production AddPlayer call. No world or session indexing changes.

All gameplay descriptor deletion/cleanup sites are covered: Unregister, UnregisterSession, UnregisterAndClose, quit's duplicate sweep and shutdown. Shutdown now snapshots without deleting ahead of cleanup; each captured descriptor is retired by its captured key and concrete pointer under the same lifecycle lock. This also preserves stale-owner safety. Existing shutdown notices, timeout and retained cleanup remain. CheckIdlePasswords deletes only unauthenticated sessions, which have no admitted body. Deferred extraction removes world bodies before descriptor retirement; it cannot create a body with no descriptor. Unexpected EOF retains the Session, verified by existing reconnect/transport and DP-1381 live-rename/EOF tests.

The transient world-only state still physically exists inside cleanup, but duplicate selection/admission cannot observe it as a target or race past it. There is no gameplay producer of a persistent arbitrary Session-less world body: all additions register first under the coupled operation, and all destructive removals complete before new lifecycle selection. Fabricated AddPlayer-only unit fixtures are outside that supported gameplay invariant. The reachable topology is nevertheless kept as a repaired row, not quietly relabeled excluded.

## R5h and evidence

Archive: ~/Archives/darkpawns/oracle-runs/2026-10-02/dp-1371-entry-finish-proofs/.

- orphan-before/assertions.log: pre-fix actual teardown assertions fail in all three unregister modes.
- orphan-selector: bypass adoption's lifecycle lock; 0/1/0.
- orphan-retirement: bypass transport retirement's lifecycle lock; 0/1/0.
- orphan-admission: bypass complete admission's lifecycle lock; 0/1/0, explicit admission/ownership assertions. An initial replay exposed a repeated-save fixture channel panic; the wrapper now signals once and those panic bytes are not used as proof.

Each triple retains fixed/reverted/restored logs, mutation and exit codes in triple.json. Shutdown and new admission extend the final selector replay matrix.

## Other readers and limits

Affected readers: WebSocket/telnet disconnects, linkdead reaper, login/password duplicate selection, menu/new/guest admission, quit anti-dupe cleanup, shutdown, SQLite saves and concrete world/session ownership. Room/GMCP/logging readers retain existing bytes and map layouts. A slow save now delays lifecycle transitions instead of letting them overtake retirement, as C's sequential loop does. No lock is acquired while holding the manager or world lock; broadcasts use their existing separate snapshots. Editor cancellation and asynchronous transport closure do not synchronously call a public lifecycle method.

Live name editing's stale-key consumers remain the separately blocked entry.identity-consumers work, rather than an unapproved registry redesign in this case. No production access, save format, oracle or governing-document changes.
