# 2026-08-29 — `bank` depth slice

## Frontier and queue

- Started from `main`, ran `git checkout main && git pull --ff-only`, reread
  `docs/fidelity/DEPTH_TESTING.md` and the newest prior handoff, and confirmed
  the post-integration frontier with `make fidelity-depth`.
- Current main reports 959 total cases; 932 proven/delegated; 6 blocked; 21
  excluded; actionable completion 932/938 (99.4%). This slice added ten
  proven cases. The earlier oro-quarters PR, previously open in the prior
  handoff, was integrated on main as `4da680d83` with green lint, security, and
  test checks; it was not re-picked.
- The unrelated untracked brief
  `docs/briefs/BRIEF-2026-08-28-economy-specproc-cluster.md` remains preserved.

## C call path and branch census

- `SPECIAL(bank)` is defined at `src/spec_procs.c:2345-2399` and registered
  for object vnums 8034 and 18224 at `src/spec_assign.c:535` and `:561`.
  `special()` reaches room-object, equipment-object, and inventory-object
  specials from `src/interpreter.c:1407-1475`, before ordinary command
  dispatch; a `FALSE` result reaches the command table handler.
- `balance` reports either the positive `GET_BANK_GOLD` balance or the exact
  empty-account line. `deposit` and `withdraw` first use C `atoi()` and reject
  non-positive amounts, then reject insufficient carried/banked coins, or
  mutate carried gold and bank gold and emit the direct transaction line.
  Successful transactions also call `act(..., TO_ROOM)`, excluding the actor.
  Other commands return `FALSE`.
- The live vehicles cover both registrations, empty and positive balance,
  zero/nonnumeric amount gates, successful deposit/withdraw state, both
  insufficient-funds branches, the actor/peer transaction audience, and
  `gold` fallthrough.

## RED/GREEN evidence and port result

- RED on pre-fix main: the valid #8034 vehicle reached the Go object-special
  path, but `deposit 50` terminated the Go server because the object call path
  supplies no `MobInstance` receiver while the old implementation dereferenced
  `me.GetRoom()`. The old path also used embedded CRLFs and broadcast the
  transaction through the wrong audience helper. The first two fixture attempts
  were discarded as invalid vehicles: one placed the ATM in the wrong room and
  one was suppressed by the existing global object maximum; the corrected
  vehicle was then shown to reach C's bank special with `--show-oracle`.
- GREEN on `glm/spec-bank`:
  `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle /usr/local/go/bin/go run ./cmd/dp-oracle-diff --scenario spec-proc-bank --show-oracle`
  and the corresponding `spec-proc-bank-kir-oshi` vehicle both report
  `result: no normalized divergence`.
- `TestSpecBankObjectCallContract` proves a nil object receiver, exact direct
  and room messages, and carried/bank state. `TestAtoiC` pins leading-space,
  sign, leading-digit, and no-digit behavior.
- The Go fix uses the actor's room as the `Act` origin and leaves `src/` and
  `darkpawns-c-oracle/` untouched.

## Verification and integration

- Local gates passed on the feature branch: `make fidelity-depth`,
  `go build ./...`, `go vet ./...`, `go test ./...`,
  `go test ./pkg/game/...`, `golangci-lint run ./...`, and clean
  `gofumpt -l .`.
- PR #752 (`glm/spec-bank`) had green lint, security, and test checks;
  build/deploy were skipped by workflow policy. It was squash-merged as
  `16f0758a7`. No CI retry was needed because checks fired normally.

This slice applies R1 (player-facing bytes), R2 (registered object command
surface), R3 (deterministic coin state), R4 (no invented output), and R5/R5e
(whole-class audit of both registrations and the actual C dispatch path).

## Next queue item

Continue the `src/spec_procs.c` special-procedure inventory with `horn`, defined
at `src/spec_procs.c:2401` and registered for object vnum 14415 at
`src/spec_assign.c:558`. Do not re-pick `bank` or any earlier claimed
procedure. After the remaining special-procedure inventory, attempt the single
blocked `objmagic.sleep-entry-gates` vehicle, then sweep the remaining
un-manifested command families in `src/interpreter.c` table order.
