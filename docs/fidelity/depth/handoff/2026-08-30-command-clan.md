# Depth handoff — clan command

Date: 2026-08-30  
Queue slice: `src/interpreter.c:383`, `clan` / `do_clan`  
Starting main: `4407479e8`

## Queue decision

The special-procedure inventory across `src/spec_procs.c`, `src/spec_procs2.c`,
and `src/spec_procs3.c`, including the active registration tables, remains
exhausted. The blocked `objmagic.sleep-entry-gates` row was already attempted
through the cast-sleep outlaw/reagent vehicle and remains blocked; it was not
repicked. After the merged `checkload` slice at `src/interpreter.c:381`, the
interpreter sweep selected `clan` at line 383 in command-table order.

## C path and proof

The command table registers `clan` at `src/interpreter.c:383`. The top-level
dispatch and subcommand abbreviations are in `src/clan.c:1541-1573`; the
format surface is `src/clan.c:63-121`. The audited subpaths were rename
(`127-153`), create (`156-236`), destroy (`240-283`), membership
(`287-543`), who/members (`545-609`), quit (`611-648`), status/apply
(`651-720`), info (`723-785`), bank/money (`880-1057`), ranks/titles
(`1060-1220`), private (`1222-1269`), application level (`1272-1329`), and
privilege (`1442-1495`). The plan editor is `src/clan.c:1391-1438`, using
`string_write`/`string_add` in `src/modify.c:97-216`; `CAP` is the first-byte
uppercase macro in `src/utils.h:166`.

The RED probes on main confirmed case-sensitive player lookup during
create/enroll, invented or missing info bytes, the absent immortal bank path,
incorrect rank/clan parsing, non-C capitalization, an applicant lock-order
deadlock, and the missing clan-plan editor. The C call paths and
player-visible branches were mapped before changing Go. The editor vehicle
also proves C's first-byte `@` termination, including `@ trailing`. Neither
`src/` nor `darkpawns-c-oracle/` was edited.

## Changes

- Align clan target resolution, numeric parsing, level gates, capitalization,
  bank/rank paths, info rendering, offline member bytes, and applicant state in
  `pkg/game/clan_*.go` and `pkg/game/clans.go`.
- Add the C-compatible clan-plan editor state and session interception in
  `pkg/game/player.go` and `pkg/session/session_login.go`.
- Add six durable oracle vehicles: `clan-depth`, `clan-member-depth`,
  `clan-applicant-depth`, `clan-plan-depth`, `clan-plan-mortal-depth`, and
  `clan-rename-depth`.
- Add `docs/fidelity/depth/clan.tsv` with 50 cases covering the command
  format, admin/member/applicant gates, state changes, audience output, plan
  editing, and rename semantics.

All six vehicles are GREEN at seed 1; `--show-oracle` was used for the plan
editor bytes. The updated manifest reports `clan: 50/50` proven.

## Gates and frontier

The following all pass on the final formatted tree:

```
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
test -z "$(gofumpt -l .)"
```

The refreshed frontier is 1,422 total cases: 1,368 proven/delegated, 14
blocked, and 40 excluded; actionable completion is 1,368/1,382 = 99.0%.

This work follows R1/R2/R3/R4 and R5/R5e; the C call-path, audience/state,
and proof audit follow R5b/R5c. The command surface and player-facing bytes
remain grounded in the C oracle.

## Next queue item

After this slice's PR merges with every hosted check green, return to clean
`main`, pull, refresh the frontier, reread the testing guide and newest
handoff, and take `clap` at `src/interpreter.c:384` in table order.
