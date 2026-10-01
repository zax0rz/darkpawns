# E1 deleted-name evidence

Fresh main base 9fa21c7e8 after #1748. C was read at
src/interpreter.c:1762-1787,1819-1821,2144-2147,2329-2347 and
src/db.c:3057. R1/R4/R5e/R5f/R5g/R5h govern this batch.

Two reachable gaps were found: saved PLR_DELETED records asked for the old
password, and menu deletion removed the row, losing C's deleted-name folding.
The pre-fix assertion fails on JSON, terminal and get_name retry, retained at
~/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-p4-entry-deleted-proofs/pre-fix-red.log.
Wizard set's deleted flag and offline save path make the saved-flag gate reachable
(pkg/session/wiz_set.go); it is not an invented fixture-only state.

The repair retains the existing saved character-data format. Ordinary menu
deletion saves PLR_DELETED and discards stored objects; LVL_GRGOD and above do
not acquire the marker. Name entry reads the saved flag before password routing,
starts fresh folded confirmation and retains the old row until stats acceptance.
A transaction compares the original ID/name/character-data, deletes that row
and inserts a fresh identity. Failure rolls the deletion back and admits no
candidate. C likewise allocates a fresh idnum (src/db.c:3057); its reused file
position is not Go's character identity. A missing atomic capability fails closed.
Malformed character data is still owned by the existing restore-failure path,
which this batch does not promote.

Proofs use real SQLite, all three session entry routes, and real WebSocket JSON.
They cover no early durable mutation, N/close abandonment, clean state/new ID,
failed insert rollback, stale/undeleted record guard, fail-closed admission,
menu reuse and the three-point LVL_GRGOD boundary. The old mock menu-delete test
now uses SQLite because a mock Exec cannot certify compare-and-save durability.
revert_proofs.py records assertion-only 0/1/0 triples in revert-triples.tsv.

The live entry-deleted-menu-reuse vehicle uses no-settle to remain at the
initial menu, then deletes and relogs with identical uppercase RECLAIM input.
The inspected C blocks contain the deletion warning/close and Reclaim's fresh
confirmation, new password, creation and accepted replacement menu. The manifest
claims seeds 1,2,3,5,8 in standard form. Stats are normalized: this is not a
reroll/RNG-values claim. Browser-authored rendering and full menu actions remain
separate rows.

Rejected development attempts are retained: the first lacked an enter-game
step under the default settle route; the next quit-to-menu attempts hit the
existing post-world-entry menu-password path. The final vehicle uses the existing
no-settle control to test deletion directly at the accepted-character menu.
No harness/oracle/tooling or governing document edits are involved.

Proposed follow-ups for entry.menu-actions: post-world-entry menu password
verification, the C clan removal and alias-file cleanup in self-deletion
(src/interpreter.c:2331-2340), and the complete rejection/frozen/delete-confirm
matrix. They are not claimed complete here. Guest capabilities and approved
security overlays are untouched. Full and claims censuses certify this entry
code batch; exact summaries, source hashes and seed inspections are retained
beside this note after completion.

The quit-to-menu password finding reproduces on untouched origin/main 9fa21c7e8 with the reference oracle (dp-1371-p4-entry-deleted-menu-baseline). Go remains on the old Password route at relogin after the menu verification sequence; retained attempt logs show three Wrong password outcomes and EOF. The runner classifies the early close as INFRA, so this is a retained pre-existing finding, not a clean vehicle or revert proof. The final no-settle vehicle and the full census are clean.

The sole deleted-slot replacement retains C's index-zero God initialization (`src/db.c:3016-3024`), even after the fresh-mud crown has been consumed. Its unit gate has a revert triple; entry-deleted-sole-record@1,2,3,5,8 drives the real wizard flag, quit/relogin, new world entry and score exposing God level/health. General bootstrap/RNG matrix ownership remains separate. The original full run and stopped claims run are preliminary; full-final/claims-final certify the completed source.
