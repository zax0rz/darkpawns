# DP-1381: reserve offline name edits

Zach approves the offline rename guard. `set file ... name` refuses while the source is retained online/linkdead or entering, or the proposed destination is held by an online/entering descriptor, case-insensitively. It uses the existing `Sorry, you can't do that.\r\n` from src/act.wizard.c:2661. C's actual offline rename at 2889-2897 allows these edits; the restriction is explicitly divergent-approved under DP-1381, not a C-parity claim.

`reserveOfflineRename` holds the entry-name reservation lock through the existing full-record compare-and-save. It checks retained map keys, descriptor names, character names, switched originals and live world bodies. The original session/world entry key protects a source whose live display name has changed. Guest process-local IDs may overlap offline SQLite IDs, so numeric equality is deliberately not an identity check. Unheld offline edits remain allowed and retain the stored ID. No store schema, save format or registry representation changes.

## Reachability / exclusions

The historical `topology-reachability.go.fixture` and `topology-stop.md` characterize the pre-guard paths. They are not current green proofs. `TestEntryDuplicateRenameTopologyExclusions` executes those same actual file-edit commands against SQLite and proves they cannot change the held identity, then logs in through the ordinary uppercase-name entry path and adopts exactly the original body/ID.

Same-name/different-ID: the folded unique SQLite identity index prevents two stored identities with the same name. Freeing an online identity's name and reusing it for another ID requires a file edit; DP-1381 refuses both the held source and the held destination. IDs cannot be changed for PCs by do_set. Exclusion applies to a stored login identity selecting the wrong retained body, not to arbitrary coincident live display names.

Folded-key: initial entry restores the store's indexed spelling; the offline case-only edit cannot change it while retained. Live do_set/do_string edits change the display field and descriptor `playerName`, but do not rekey the maps or change the indexed store name. `SavePlayer` / `playerUpdate` deliberately omit name, and `ApplyCharacterData` omits its redundant JSON name on restore. `TestEntryLiveRenameKeepsStoredIdentity` uses the actual live command, real EOF save and fresh login to prove this alternative does not create the folded-key miss. Existing stale-key live-rename consumers belong to the separately blocked identity-consumers case; this does not claim they are repaired.

## R5h

`rename-guard`: fixed/reverted/restored 0/1/0. Mutation bypasses the guard; source, destination, entering, case-only and both topology-building assertions then fail. Retained evidence: ~/Archives/darkpawns/oracle-runs/2026-10-02/dp-1371-entry-finish-proofs/rename-guard/{fixed,reverted,restored}.log and triple.json. The pre-fix assertions log is retained separately; a later test-fixture nil transport-channel panic was corrected and is not claimed as proof.

## Other readers / locks

Admin login, OLC/file-edit authority, browser/telnet entry, deleted-name treatment, normal saves and live name editors are unchanged. The reservation lock serializes against claim/release; session and world snapshots are read separately, never with the manager/world locks nested. File writes retain the existing compare-and-save freshness check. The world-body-without-session topology is classified in the next case, independently of these exclusions.

## Guest ID reader audit correction

Two actual guest logins can receive world IDs 0 and 1 while the offline Returner record also has SQLite ID 1. The original guard wrongly refused that unheld offline rename. `TestEntryOfflineRenameGuard/offline-with-guests` fails before correction and proves the actual guest/store overlap. The guard now checks retained names/entry keys and the source's original world key, never blanket numeric equality. It still rejects a live-renamed source and preserves both topology exclusions. Guest login/activity is unchanged. `rename-guest-id` reinstates the wrong blanket numeric check and gives an assertion 0/1/0. The first combined run was deliberately stopped for this correction; it is retained as aborted evidence, not a gate verdict.
