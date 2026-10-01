# DP-1371 E1: deleted names

C's saved PLR_DELETED branch was missing from Go name entry, and hard menu
deletion lost the distinction between fresh and reused names. The repair keeps
the marker in the existing character-data record, folds reused names exactly as
C, and starts fresh confirmation before old-password authentication. It retains
the deleted row through abandoned creation, then replaces it atomically at
accepted stats with a new character identity. The highest-god boundary is C's
LVL_GRGOD: those menu deletions do not set the marker.

C: src/interpreter.c:1762-1787,1819-1821,2144-2147,2329-2347;
src/db.c:3057. R1/R4/R5e/R5f/R5g/R5h. No save/schema, reference oracle,
harness/tooling, world, guest capabilities or governing document changes.

entry.deleted-name becomes unit-green, with separate replacement, abandonment,
atomic rollback/stale record, failure-admission, WebSocket and menu-reuse rows.
entry.deleted-name.menu-live owns entry-deleted-menu-reuse@1,2,3,5,8. It starts
at the accepted-character menu through existing no-settle, proves deletion and
identical uppercase-name relogin into fresh creation, ending at the replacement
menu. It does not claim browser rendering, full menu actions or RNG values.

Revert scripts, assertion triples, exact census summaries and final source hashes
are in ../evidence/2026-10-01-entry-deleted/. Complete retained outputs are under
~/Archives/darkpawns/oracle-runs/2026-10-01/dp-1371-p4-entry-deleted-*.

Proposed menu-actions follow-ups: verification after returning from world entry,
the remaining menu
rejection/frozen/confirmation matrix. That row stays blocked. The next batch is
entry.lookup-load-failure. This changes game code: open the PR and stop for Zach.

The quit-to-menu password finding reproduces on untouched origin/main 9fa21c7e8 with the reference oracle (dp-1371-p4-entry-deleted-menu-baseline). Go remains on the old Password route at relogin after the menu verification sequence; retained attempt logs show three Wrong password outcomes and EOF. The runner classifies the early close as INFRA, so this is a retained pre-existing finding, not a clean vehicle or revert proof. The final no-settle vehicle and the full census are clean.

Preliminary full census before the sole-deleted-record bootstrap repair (production/scenario hashes matched 46ff60924; run captured dirty 9fa21c7e8 before the implementation commit):

```
oracle-regression: scenarios=1057 passed=1050 expected=5 unpinnable=0 stale=0 failed=0 infra=1 timed_out=0 unstable=1 elapsed=499.461s started=2026-10-01T17:22:18-0400 finished=2026-10-01T17:30:37-0400 verdict=CLEAN_AFTER_RECHECK rechecked=informative-residual-depth
```

Live folding revert: targeted PASS → content FAIL (Reclaim vs RECLAIM in confirmation/password prompts, zero infrastructure/timeouts) → restored PASS.

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

Review full census on completed production source f7f1f4e10:

```
oracle-regression: scenarios=1058 passed=1052 expected=5 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=1 elapsed=500.307s started=2026-10-01T18:47:54-0400 finished=2026-10-01T18:56:15-0400 verdict=CLEAN
```
