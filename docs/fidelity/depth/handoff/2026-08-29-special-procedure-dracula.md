# Depth-fidelity handoff — 2026-08-29 — `dracula`

## Session and frontier

- Started from `main` at `852498b01` after `git pull --ff-only`, reran `make fidelity-depth`, and reread `docs/fidelity/DEPTH_TESTING.md` plus the newest jailguard handoff.
- Main baseline frontier: 904 total cases; 882 proven/delegated; 6 blocked; 16 excluded; actionable completion 882/888 (99.3%).
- The submitted Dracula branch adds six manifest cases, for 910 total and 888 proven/delegated on the branch. Main remains at the baseline because its PR is intentionally still open after a non-green CI result.
- The source inventory remains 113 `SPECIAL` definitions, 233 active `ASSIGNMOB` registrations, 228 unique active mob vnums, and 66 final assigned procedure names after later registrations win.

## C call path and branch coverage

`SPECIAL(dracula)` at `src/spec_procs.c:1798-1834` was audited from the `look` command entry at `src/interpreter.c:532` through special dispatch at `src/interpreter.c:1407-1456`. The player-facing room messages use `act()` at `src/comm.c:2392-2555`; the embedded `do_say()` call follows `src/act.comm.c:759-821`. Dracula is registered for mob vnums 7903 and 14110 at `src/spec_assign.c:262,432`.

The valid vehicle uses scriptless vnum 14110 in room 8105 with a co-located peer. It covers non-look and missing-player rejection, idle and fighting commandless entry (the latter delegates to `magic_user`), `PRF_NOHASSLE`, C keyword-list abbreviation (`lothar` against `Lothar Vampire Lord`), actor-only mesmerism and bite lines, three TO_ROOM emotes with C's embedded-CRLF quirk, canonical `do_say` exclamation, the mutually exclusive vampire/werewolf state gate, and the TRUE return that suppresses ordinary `look` output.

## RED and GREEN evidence

- The first valid RED on the native-room vehicle reached the C special but showed Go's ordinary `look` response: Go matched against the full short description instead of C's keyword list and omitted the special transcript.
- After switching to a deterministic room-8105 audience vehicle, the corrected implementation first exposed the exact C blank-line behavior from the three embedded-CRLF `act()` formats; preserving those source bytes closed the audience diff.
- GREEN: `spec-proc-dracula` reported no normalized divergence for seeds 1, 2, 3, 5, and 8; `--show-oracle` confirmed the actor and peer blocks.
- Focused tests cover the entry gates/delegation, actor and peer output, keyword abbreviation, actor exclusion, and existing vampire/werewolf transformation suppression.

## Port and manifest result

- Updated `pkg/game/spec_procs4.go` to use the canonical `isnameWithAbbrevs`/mob keyword path, `PRF_NOHASSLE` and PLR state gates, `Act()` room routing, embedded C line endings, and `World.DoSay()`.
- Added `pkg/game/spec_dracula_test.go`, `cmd/dp-oracle-diff/scenarios/spec-proc-dracula.txt`, and six rows to `docs/fidelity/depth/spec-procs.tsv`.
- No files under `src/` or `darkpawns-c-oracle/` were edited.

## Verification and integration

All required local gates passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test ./...`, `golangci-lint run ./...`, and `gofumpt -l .` clean. The five-seed oracle matrix also passed.

PR #740 (`glm/spec-dracula`, commit `83cb64a71`) required the one permitted workflow retry because no checks initially appeared. The retry produced green lint and security jobs, but the test job failed outside this slice in the pre-existing `pkg/telnet` `TestListenAcceptNotBlockedBySlowReverseDNS` data race (`listener_test.go:988` versus `listener.go:187`). The PR is left open and unmerged; do not merge it without green checks, and do not repick Dracula.

The pre-existing untracked `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved and uncommitted.

## Next queue item

Continue the source-order special-procedure inventory with `pet_shops` (`SPECIAL` at `src/spec_procs.c:1844`, room registration `ASSIGNROOM(21235, pet_shops)` at `src/spec_assign.c:618`). This is the next room-special definition after Dracula; audit its room call path and available pet-room vehicle before claiming proof. After active special procedures are exhausted, attempt `objmagic.sleep-entry-gates` once through the cast-sleep outlaw/reagent vehicle, then sweep unmanifested command families in `src/interpreter.c` table order. Do not repick Dracula, jailguard, or outofjailguard.
