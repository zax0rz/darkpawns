# #2179 ruling follow-up — 2026-10-26

This note supersedes the earlier mail-store interpretation in
`2026-10-26-blocked-output-units.md` for #2179. #2177 is merged and this branch
has been rebased onto `main`.

## Mail ruling

Mail remains functional in Go. The `referenceMailStoreLayoutValid` predicate
and its `HandleMailInput` guard were removed; composition completion always
calls `storeMail`, and the C editor completion bytes remain `Message sent!`.
The 64-bit oracle's `sizeof(long) == 8` makes `header_block_type` 104 bytes,
which fails `BLOCK_SIZE == 100` in `src/mail.h:50,84-104`. This is reference
platform drift: the original 32-bit build stored mail. Do not treat the
reference host's inert mail file as intended gameplay.

The production mail lifecycle and recipient-save tests again cover sending,
restart, receipt, and persisted inventory, and assert `Message sent!`. Oracle
mail-store scenarios that depend on C writing a note are held until GLM-fixes'
EXPECTED pin in #2178 lands; this follow-up does not census those scenarios.

## CloseAfterSend lock audit

All four `CloseAfterSend` call sites are in `pkg/session/reconnect.go`. The
menu path unlocks `Manager.mu` before closing the old session; the three
playing/editor paths unlock it before entering the close-mode switch. None of
the paths holds a world lock. They do retain `playerLifecycleMu` to serialize
body ownership, but the potentially slow transport drain never holds
`Manager.mu` or a world lock.

`TestDupeCheckDrainsNonReadingClientWithoutManagerOrWorldLock` uses a `net.Pipe`
whose client end is deliberately never read. The writer times out at its
deadline, while another goroutine can acquire `Manager.mu` and the world player
count can acquire its world lock during the blocked drain.

## `stat room` exit bytes

`src/act.wizard.c:499-507` emits `"...Type: %s\r\n "`, then sends the exit
description (or the two-space fallback). `internal/oraclediff/normalize.go:99-105`
trims trailing spaces from each line. It hides a trailing space at the end of
an output line, but preserves this byte because the following nonempty
description makes it the first byte of that line. The
`wizard-stat-room-exit-description-depth` scenario exercises room 1204's down
exit (`lib/world/wld/12.wld:20-24`); the raw terminal-frame assertion
`TestStatRoomExitPreservesCPostNewlineSpace` pins the complete sequence. The
Go stat output sends the exit prefix literally so its renderer cannot insert an
extra line ending between the C prefix and the description. With the literal
space included, the empty-description fallback still has its two C spaces.

## Filed diagnostic finding: switched-NPC Call Lightning attack messages

- **Scenario:** `call-lightning-npc-miss-switched-depth`, the 600-pulse
  diagnostic (`luna-blocked-final-calllightning-diagnostic-s1`; also checked
  with a level-1 victim in `luna-blocked-final-calllightning-lowlevel-s1`).
- **Finding:** the target M@0045 Call Lightning miss bytes matched, but the
  full combat transcript also had additional C/Go attack-message differences.
  Draw divergence was zero. This is a real wider combat-message finding, not a
  clean proof of the switched Call Lightning miss.
- **C path:** `src/comm.c:815-818` runs `mobile_activity`; `src/spec_assign.c:197`
  assigns the cleric special to mob 1305; `src/spec_procs.c:1425,1490-1496`
  reaches the Call Lightning cast; `src/magic.c:827` passes spell damage to
  `damage`; `src/fight.c:1023-1095,1534-1545` selects and emits the attack
  message.

The finding remains separate from the clean concentration-miss proof and is
outside this PR's requested fixes. It should be attributed to the shared
combat attack/message owner when that class is revisited.

## Verification

The first targeted attempt, `luna-pr2179-stat-room-exit-space`, was NOT_CLEAN:
its `goto 13101` did not move the immortal out of room 1204, and the initial
capture exposed the extra line break inserted by the Go text renderer. It had
zero draw divergence. The scenario now exercises room 1204 directly, where the
down exit has a nonempty description, and its test renders terminal frames.

The corrected targeted attempt, `luna-pr2179-stat-room-exit-space-rerun`, is
clean with matching draws:

```text
luna-pr2179-stat-room-exit-space-rerun @30d15bbbf: oracle-regression: scenarios=1 passed=1 expected=0 unpinnable=0 stale=0 failed=0 infra=0 timed_out=0 unstable=0 elapsed=20.118s started=2026-10-26T18:48:31-0400 finished=2026-10-26T18:48:51-0400 verdict=CLEAN | oracle-draw-parity: scenarios=1 draw_divergent=0
```

The corrected scenario is `wizard-stat-room-exit-description-depth`; its run
directory is
`/home/zach/Archives/darkpawns/oracle-runs/2026-10-26/luna-pr2179-stat-room-exit-space-rerun`.
`make fidelity-depth`, `make fmt`, `go build ./...`, `go vet ./...`,
`go test ./...`, `go test ./pkg/game/...`, `golangci-lint run ./...`,
`make deadcode-ratchet`, the focused stat-room/non-reading-client tests, and
`git diff --check` passed individually. No mail-store census is included before
the #2178 EXPECTED pin.
