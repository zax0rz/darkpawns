# 2026-08-29 — `start_room` depth slice

## Frontier and queue

- Started from a clean `main` boundary after the `fearface` slice, pulled
  `f0183ad88`, ran `make fidelity-depth`, and reread
  `docs/fidelity/DEPTH_TESTING.md` plus the newest `fearface` handoff.
- Baseline for this slice: 914 total cases; 888 proven/delegated; 6 blocked;
  20 excluded; actionable completion 888/894 (99.3%).
- This slice adds four cases. Current `main` after integration is 924 total
  cases; 897 proven/delegated; 6 blocked; 21 excluded; actionable completion
  897/903 (99.3%). The six pray cases are also now present on `main` because
  the pending PR merged remotely during this slice's integration window.

## C call path and branch census

- `SPECIAL(start_room)` is defined at `src/spec_procs.c:2204-2243` and
  assigned to room 8099 at `src/spec_assign.c:606`. `special()` invokes the
  registered room pointer before ordinary command dispatch at
  `src/interpreter.c:1407-1415`; a `FALSE` result then allows the command
  handler at `src/interpreter.c:947-948`.
- The C procedure has no command-name gate. It walks room occupants; an
  immortal occupant causes an immediate `FALSE` return with no output.
  Otherwise each mortal occupant receives the birth transition, is moved by
  hometown (`HOME_KD` 8162, `HOME_KO` 18201, `HOME_AZ` 21202, default 8004),
  and receives `do_look` output in the destination room. The procedure then
  returns `TRUE`, suppressing the original command's ordinary output.
- The direct command vehicle uses `goto 8099`, then lowers the first-player
  God to level 1. The next `look` reaches the real room-special path. The
  oracle's live bytes show C's overlapping-`sprintf` behavior drops the
  first three built birth lines; the Go port preserves those observed bytes,
  not the latent intended text.

## RED/GREEN evidence and port result

- RED on main: Go's room handler scanned `me` as though this were a mob
  special; room-special dispatch passes `me=nil`, so the vehicle stayed in
  the Burning Hut and missed the C birth transition.
- GREEN after the fix: `spec-proc-start-room --show-oracle` reports no
  normalized divergence. It proves the birth text, hometown-K move to 8162,
  destination `do_look`, no auto-exit invention, and TRUE return interception.
- Focused `TestSpecStartRoom_BirthTransitionAndImmortalGate` covers the
  immortal early return and mortal routing. No `src/` or
  `darkpawns-c-oracle/` file was edited.

## Verification and integration

- Local gates passed on the feature branch: `make fidelity-depth`,
  `go build ./...`, `go vet ./...`, `go test ./...`,
  `golangci-lint run ./...`, and clean `gofumpt -l .`.
- PR #747 (`glm/spec-start-room`) received green lint, security, and test
  checks (build/deploy skipped by workflow policy) and was squash-merged as
  `6c342fadf`.
- The earlier PR #745 (`glm/spec-pray-for-items`) that this session initially
  observed without checks was subsequently merged into `main` as
  `3599a4021`; its six rows and implementation are therefore now integrated.

This slice applies R1 (player-facing bytes), R2 (registered room command
surface), R4 (no invented output), and R5/R5e (actual C call path and oracle
behavior).

## Next queue item

Continue source order with the active room `newbie_zone_entrance` at room
16300. Begin the next round from a clean `main`/pull/frontier boundary and do
not repick `start_room`, `pray_for_items`, `fearface`, or earlier claimed
procedures. After active specials, attempt the single blocked
`objmagic.sleep-entry-gates` vehicle, then sweep un-manifested command families
in `src/interpreter.c` table order.
