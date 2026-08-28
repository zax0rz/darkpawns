# Depth-fidelity handoff — 2026-08-28 — special procedure `thief`

## Frontier and queue position

- Started from `main` at `25ccbaf6d`, pulled before work, and refreshed the
  frontier with `make fidelity-depth`.
- Special-procedure census remains 113 `SPECIAL` definitions across
  `src/spec_procs.c`, `src/spec_procs2.c`, and `src/spec_procs3.c`; 233 active
  `ASSIGNMOB` registrations, 228 unique active mob VNUMs, and 66 final
  assigned procedure names after later-registration wins.
- Before this slice: 535 total cases, 523 proven/delegated, 1 blocked, and 11
  excluded. After this slice: 539 total, 527 proven/delegated, 1 blocked, and
  11 excluded; actionable completion is 527/528 (99.8%).
- `guild` and `snake` are already claimed by their dated handoffs. This slice
  claims `thief`; the next unclaimed source-definition item is `magic_user`.

## C path and reachability

R5e was verified from the actual call path: `src/mobact.c:68-93` skips fighting
or non-awake mobs, then invokes the registered special with `cmd == 0`;
`src/interpreter.c:1407-1456` is the player-command dispatch path; and
`src/spec_procs.c:300-327,389-406` contains `npc_steal` and `SPECIAL(thief)`.
The live vehicle uses assigned mob 18218, removes its target-room exits to
prevent wander, and exercises the autonomous path after the 40-heartbeat
mobile cadence.

## Proof and confirmed divergences

- RED on the pre-fix implementation: `spec-proc-thief`, seed 1, left the C
  victim at 846 gold while Go stayed at 1000. Seed 4 additionally exposed
  that Go used `number(0,5)`, broadcast the victim-only wallet line to the
  bystander, used the mob's hard-coded `its`, and emitted a lower-case room
  line where C's `act()` emitted `A vagabond ...`.
- GREEN on `spec-proc-thief`, seeds 1–6, including periodic percentage theft,
  victim/bystander audience separation, C capitalization, and sex-aware
  pronouns.
- Added `TestSpecThief_StateAndAudience` for C draw order/ranges, victim
  deduction, thief credit, and exact victim/bystander bytes. Existing
  `TestSpecThief_Golden` covers command, position, level, and no-target gates.

## Change and gates

- Go-only fix in `pkg/game/spec_procs.go`; scenario, manifest, and focused
  tests were added. No C source or oracle tree was edited.
- PR #693 (`glm/spec-thief`) was self-merged only after hosted lint, security,
  and test checks all passed. Merge landed on `main` as `e7d40bc32`.
- Passed `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test
  ./...`, `golangci-lint run ./...`, and clean `gofumpt -l .`; `git diff --check`
  was clean.
- Fidelity rules exercised: R1, R3, R4, R5c, and R5e.

## Next action

Checkout and pull `main`, rerun `make fidelity-depth`, reread
`DEPTH_TESTING.md` and this handoff, then map and prove the next source-ordered
active special, `magic_user`. The blocked `objmagic.sleep-entry-gates` row is
untouched.
