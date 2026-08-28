# Depth-fidelity handoff — 2026-08-28 — special procedure `fighter`

## Frontier and queue position

- Started from `main` at `5890281f4`, refreshed the frontier with
  `make fidelity-depth`, and merged the slice as PR #695 at `30385252f`.
- The special-procedure census remains 113 `SPECIAL` definitions across
  `src/spec_procs.c`, `src/spec_procs2.c`, and `src/spec_procs3.c`; there are
  233 active `ASSIGNMOB` registrations, 228 unique active mob VNUMs, and 66
  final assigned procedure names after later-registration wins.
- Before this slice: 544 total cases, 532 proven/delegated, 1 blocked, and 11
  excluded. After this slice: 550 total, 538 proven/delegated, 1 blocked, and
  11 excluded; actionable completion remains 538/539 (99.8%).
- This slice claims `fighter` at `src/spec_procs.c:509`, assigned to VNUMs
  4914, 5200, 7901, 7902, 12111, 14407, 12850, 20002, 20011, 20019, 20020,
  20036, and 20042. The next unclaimed source-ordered active procedure is
  `paladin` at `src/spec_procs.c:537`, assigned to VNUMs 71 and 7915.

## C path and reachability

R5e was verified from the actual call path before changing Go:
`src/mobact.c:68-93` skips mobs already fighting, player-command dispatch
passes a nonzero command, and `src/fight.c:1898-2032` invokes `MOB_SPEC` after
the NPC's ordinary attack loop. `src/spec_procs.c:509-535` then gates
`cmd==0`, `POS_FIGHTING`, nonnegative HP, `FIGHTING(ch)`, and
`GET_MOB_WAIT==0`, selects `number(0,10)`, and dispatches subcmd=1 to
headbutt, parry, bash, or berserk. The native callees were checked through
`new_cmds.c:368-460`, `act.offensive.c:419-507`, `new_cmds.c:2134-2187`, and
`new_cmds.c:2340-2389`.

The C `WAIT_STATE(ch, cycle)` macro writes the ordinary `ch->wait` field for
non-null NPCs; it does not write `GET_MOB_WAIT`. The Go port therefore keeps
the native skill cooldown separate from the special's `GET_MOB_WAIT` gate and
only applies the player victim's bash wait state.

## Proof and confirmed divergences

- The pre-fix Go special emitted invented generic room text and selected the
  wrong target source/RNG helper; it did not execute the native skill paths.
- GREEN on `spec-proc-fighter` with assigned mob 20020, its Lua fight script
  stripped, and seeds 1 through 6. Seed 1 was also run with `--show-oracle`;
  the vehicle reaches the combat-time special seam and matches the surrounding
  C transcript. The live proof deliberately claims the stable vehicle, while
  focused tests claim the selected native arms.
- `TestSpecFighter_Golden` proves the entry gates. `TestSpecFighter_NativeSkills`
  proves headbutt recoil/damage/knockdown, bash movement/damage/knockdown and
  player wait, parry audience messages and the one-round marker, and berserk's
  three native affect records. It also proves that these native calls do not
  mutate the mob's `GET_MOB_WAIT` field.

## Change and gates

- Go-only changes add the native fighter skill damage/audience path, resolve
  the actual combat target, expose the combat engine's parry marker, and add
  the scenario, focused tests, and six manifest rows. Neither `src/` nor
  `darkpawns-c-oracle/` was edited.
- PR #695 (`glm/spec-fighter`) was self-merged only after hosted lint,
  security, and test checks all passed; build/deploy jobs were skipped as
  expected for this fidelity-only change.
- Passed `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test
  ./...`, `golangci-lint run ./...` (0 issues), `gofumpt -l .` (empty), and
  `git diff --check`.
- Fidelity rules exercised: R1, R3, R4, R5c, and R5e.

## Next action

Checkout and pull `main`, rerun `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this handoff, then map and prove
`paladin` in source and registration order. The blocked
`objmagic.sleep-entry-gates` row is untouched.
