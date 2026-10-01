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
verification and the complete rejection/frozen/delete-confirm
matrix. They are not claimed complete here. Guest capabilities and approved
security overlays are untouched. Full and claims censuses certify this entry
code batch; exact summaries, source hashes and seed inspections are retained
beside this note after completion.

The quit-to-menu password finding reproduces on untouched origin/main 9fa21c7e8 with the reference oracle (dp-1371-p4-entry-deleted-menu-baseline). Go remains on the old Password route at relogin after the menu verification sequence; retained attempt logs show three Wrong password outcomes and EOF. The runner classifies the early close as INFRA, so this is a retained pre-existing finding, not a clean vehicle or revert proof. The final no-settle vehicle and the full census are clean.

The sole deleted-slot replacement retains C's index-zero God initialization (`src/db.c:3016-3024`), even after the fresh-mud crown has been consumed. Its unit gate has a revert triple; entry-deleted-sole-record@1,2,3,5,8 drives the real wizard flag, quit/relogin, new world entry and score exposing God level/health. General bootstrap/RNG matrix ownership remains separate. The original full run and stopped claims run are preliminary; full-final/claims-final certify the completed source.

Final full census on completed game source (matches 2a7da8aad):

```
oracle-regression: scenarios=1058 passed=1052 expected=5 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=1 elapsed=500.512s started=2026-10-01T17:42:38-0400 finished=2026-10-01T17:50:58-0400 verdict=CLEAN
```

The full run captured the sole-record vehicle before its test-only world/score refinement. The final targeted sole-record revert/restoration and claims run exercise the refined vehicle. Game code is unchanged; no general bootstrap/RNG claim is inferred from the precursor menu-only block.

The final refined sole-record live proof is PASS → content FAIL → PASS: dp-1371-p4-entry-deleted-sole-restored, sole-mutant-2, sole-restored-2. Removing the sole-slot God predicate changes the replacement world/score; C shows 500 hit points and rank level 40. Both mutation/restoration runs have zero infrastructure errors/timeouts. The earlier sole-mutant run was discovery evidence, not the before leg of this triple.

Final claims census on Go source 2a7da8aadc0cb99114e24dc39623ef06511f88a6:

```
oracle-claims: seed=all pairs=3020 expected=11 expected_unstable=0 fail=0 infra=0 pass=3009 stale=0 timeout=0 unpinnable=0 elapsed=1696.527s verdict=CLEAN
```

Both final live vehicles pass at seeds 1,2,3,5,8. Retained C captures were inspected for deletion, folded fresh-name confirmation/password and replacement menu; the sole-record vehicle also exposes world entry, health 500 and rank level 40. See live-seed-inspection.tsv. Production source is unchanged since the completed-source full census.

PR #1749 review repair: deleted records now take the existing missing-record path in admin login (identical 401 body, bcrypt timing decoy) and all OLC/schema/new-zone/file-edit saved-record checks. Clan membership and alias cleanup are implemented in this batch: src/interpreter.c:2331-2340, clan.c:789-796, objsave.c:1227-1251. C excludes clan table index zero; tests preserve that boundary, and test both ordinary deleted players and LVL_GRGOD. Durable character save precedes live clan total adjustment; failed save restores the in-memory clan fields and marker. Alias removal treats missing files as harmless and logs real failures as C does. Three additional assertion-only revert triples are retained by review_revert_proofs.py and review-revert-triples.tsv. Earlier census summaries certify the original PR; new review-full/review-claims runs certify these changes.

Review repair gates: make fmt, go build ./..., go vet ./..., go test ./..., cache-clean/lint (0 issues), diff check, fidelity-depth (5230 total / 5109 proven / 64 blocked), fidelity-units (1183/1183; 1173 rows, 962 symbols, 11 packages), string census all pass. review-source-sha256.tsv captures completed repair source; source-sha256.tsv retains original PR evidence.

Review full census on completed production source f7f1f4e10:

```
oracle-regression: scenarios=1058 passed=1052 expected=5 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=1 elapsed=500.307s started=2026-10-01T18:47:54-0400 finished=2026-10-01T18:56:15-0400 verdict=CLEAN
```

Review claims census (Go HEAD 2acd706ff9148de7885c7e8eabfd0dc435906d5a; completed production source f7f1f4e10):

```
oracle-claims: seed=all pairs=3020 expected=11 expected_unstable=0 fail=0 infra=0 pass=3009 stale=0 timeout=0 unpinnable=0 elapsed=1692.392s verdict=CLEAN
```

All ten new vehicle/seed pairs pass. C captures confirm the deletion/fresh-name/password/menu paths and sole-slot world/score health 500/rank level 40. See review-live-seed-inspection.tsv. Final production and test hashes match review-source-sha256.tsv; the full and claims runs certify the same production source. Focused review race tests also pass.
