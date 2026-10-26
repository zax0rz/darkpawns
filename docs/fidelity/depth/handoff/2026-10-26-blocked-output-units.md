# Blocked output-unit closeout — 2026-10-26

## Count and triage

The goal note's pre-rebase inventory was 897 proven and 98 blocked. After the
requested rebase for #2171, `origin/main` at `a5cddbc31` had 897 proven and 114
blocked, including 16 additional residual units. The updated inventory now has
897 proven and 50 blocked. The map validator reports 42 maps with zero errors.

The 85 rows in the re-triage table now resolve to 29 reachable and proven, 10
with stale blockers removed, 14 still blocked, 20 approved divergences, 9
`excluded-with-C-reason` R1a cases, 1 #2152 rent exclusion, and 2 lifecycle rows
delegated to GLM. The four positive mail outputs originally counted among the
33 reachable sites remain blocked at `src/mail.c:593-594` and
`src/interpreter.c:1638,2225`: source inspection and the oracle show that
reference C's `store_mail` returns before indexing a new note on this x86_64
build, so a live send cannot make `has_mail` true. The shared check output at
`src/mail.c:557` is proven for the no-mail arm; its positive branch remains
blocked in `mail.tsv` pending the same seeded-record fixture.

## Proof trains

- After #2171, the five-seed player-ID run proves the first new mortal has
  IDNum 5 in both engines; stat, set, whois, and finger use that aligned
  identity. The targeted runs are `luna-blocked-idnum-rebased-s1/s2/s3/s5/s8`.
- The non-stop scenarios cover the refreshed track, REDIT null-description,
  ZEDIT duplicate gate, wizard file-output, finger, clan applicant/capacity,
  mail prechecks and empty mailbox, Call Lightning concentration miss, and Lua
  mob-wait paths. The clan-capacity reset-leader run and the new M@0045 miss
  run are CLEAN on seeds 1, 2, 3, 5, and 8 with draw parity.
- The switched-combat, Teleport, reconnect, and usurp scenarios use actual
  command/session paths. `entry-usurp-saved-body-depth` is CLEAN for each of
  seeds 1, 2, 3, 5, and 8. The shared `luna-blocked-rebased-stoptier-core`
  attempt had a red violence scenario; that was isolated and the final
  `luna-blocked-rebased-violence-fixture` is CLEAN.
- The postmaster send/prompt scenario is CLEAN on five seeds. For the live
  completion path, `luna-blocked-mail-positive-recipient` is the retained
  pre-fix revert control: C leaves the mailbox empty while Go reported mail.
  With the C size guard mirrored, `luna-blocked-postmaster-mail-store-guard`
  is CLEAN for saved-recipient completion, login notices, and empty mail, with
  zero draw divergence. The focused `TestHandleMailInputRespectsReferenceMailStoreLayoutGuard`
  also asserts the exact completion text and that no note is indexed or written.

## Remaining blockers and decisions for Zach

- `src/mail.c:553-557`, `:572-594`, and the two login notices at
  `src/interpreter.c:1638,2225` need a fixture that seeds a valid C mail-file
  record and index. `src/mail.c:285-289` rejects the normal send on the
  reference ABI: each mail block struct is 104 bytes after alignment while
  `BLOCK_SIZE` is 100. GLM owns the fixture design. The Go live send path now
  matches C's no-store behavior and still emits `Message sent!`.
- The `src/clan.c:779` at-war output remains blocked: the shipped
  `lib/etc/clans` is a four-byte zero-count header, with no nonzero `at_war`
  record loaded at boot.
- Fault-only mail-store failures, `idlist` open failure, non-ENOENT
  `Crash_load`, and corrupt-board guards remain blocked pending the fault-
  injection fixture GLM will design.
- The `full_moon` descriptor outputs remain with GLM's weather/time lifecycle
  work; their map rows remain blocked and are not included in this handoff's
  completed proof count.
- Rent loss at `src/interpreter.c:2227` is excluded per #2152 decision 4
  (`free_rent = YES`, `src/config.c:106`). The nine C undefined-behavior rows
  remain `excluded-with-C-reason` under R1a.

## Gates and delivery

On the non-stop branch, all ten gates pass: `make fmt`, `go build ./...`,
`go vet ./...`, `go test ./...`, `go test ./pkg/game/...`,
`golangci-lint run ./...`, `make deadcode-ratchet`, `make fidelity-depth`,
`make fidelity-units`, and `make string-census`. The proof change is in draft
PR #2177. A separate `[stop tier]` PR will carry the live mail-store correction
and its regression control. Neither PR is to be merged or deployed before
Claude's review.
