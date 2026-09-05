# Depth-fidelity handoff: `home`

Date: 2026-08-31

## Queue position and frontier

This session began at the clean command-depth frontier after the preceding
`hold` slice. The pre-`home` frontier was 2,167 total cases: 2,107
proven/delegated, 16 blocked, and 44 excluded. After the `home` manifest was
added, the frontier is 2,180 total: 2,120 proven/delegated, 16 blocked, and 44
excluded (2,120 of 2,136 actionable cases, 99.3%).

The special-procedure inventory is exhausted. The one explicitly blocked
`objmagic.sleep-entry-gates` row remains blocked and was not repicked. The
interpreter-table command queue has now completed through `home`; its next
unclaimed source-order family is `hop` at `src/interpreter.c:501`.

The feature slice is PR #946, `glm/depth-home`, merged to `main` as
`098af1592`. The handoff also records that the earlier hold handoff PR (#944)
was initially left open after its security job failed on an external module
checksum/network error; after the separate CI retry fix made that check green,
the handoff PR merged. No non-green PR was merged by this slice.

## C call path and branch inventory

The registration is:

```text
src/interpreter.c:500: { "home", POS_RESTING, do_home, LVL_IMMORT, 0 }
```

The handler is `src/act.wizard.c:3238-3297`. Its numeric path calls the
shared `find_target_room` helper at `src/act.wizard.c:184-239`; its no-argument
path transfers the player and calls `do_look(ch, "", 15, 0)`. The visible
room-acts are implemented by `src/comm.c:2397-2558`, and load-room persistence
uses the existing player `room_vnum` save field in `src/db.c:2386-2409`.

The audited player-visible branches are:

- immortal entry gate and the shared one-argument target boundary;
- bare `home`, including invalid saved-home repair to Limbo;
- invalid numeric room diagnostics, including the extra `do_home` line;
- numeric setter state, trailing-argument behavior, and nonnumeric usage;
- shared target lookup restrictions (including god-room/private handling);
- default and custom poof-out room audiences;
- actor relocation and the post-transfer full-room look;
- persisted load-room state and the `PLR_LOADROOM` save boundary.

R1/R5e mattered in two places. The C handler's overlapping `sprintf` calls
produce the observed `" pulled into a different reality."` prefix, so the Go
port preserves the oracle's actual bytes rather than the apparent intended
prefix. Also, the observed custom poof-out path sends a literal `$n` to the
room; Go escapes custom dollar markers before `game.Act` to preserve that
player-facing result. The existing save field was reused; no save-file field
or format was invented (R4).

## RED finding and implementation

Clean `main` initially diverged on numeric setting, nonnumeric usage, invalid
rooms, trailing numeric arguments, bare-home relocation, room audience text,
and the post-transfer look. The old Go command invented a fixed room 3001,
an arrival message, and did not preserve C's load-room state.

The confirmed fixes are:

- `pkg/session/wiz_movement.go` now follows the C setter and bare-transfer
  branches, reuses the shared target resolver, preserves the C diagnostics and
  poof audiences, and performs the equivalent full-room look.
- `pkg/game/player.go` and `pkg/game/player_stats.go` add runtime load-room
  state; `pkg/game/save.go` and `pkg/db/convert.go` map it through the existing
  `room_vnum` field only when `PLR_LOADROOM` is set.
- `pkg/game/home_depth_test.go` and `pkg/session/home_depth_test.go` pin the
  save boundary and numeric setter behavior.
- The four `home` scenarios and `docs/fidelity/depth/home.tsv` add the
  oracle/manifest proof. No file under `src/` or `darkpawns-c-oracle/` was
  edited.

The implementation cites R1, R2, R4, and R5e. Reuse of the shared target
resolver follows R5b/R5c: its call path was audited before delegating the
common restriction branches.

## Verification

The following scenario vehicles were run against the C oracle and Go port:

- `home-depth.txt` with `--show-oracle`, plus seeds 1, 2, 3, 5, and 8;
- `home-default-depth.txt` with `--show-oracle`, plus seeds 1, 2, 3, 5, and 8;
- `home-audience-depth.txt` with `--show-oracle`, plus seeds 1, 2, 3, 5, and 8;
- `home-entry-gate-depth.txt` with seed 1;
- the existing delegated `home-entry-gate` coverage.

All listed oracle runs reported no normalized divergence. The local gates all
passed: `make fidelity-depth`, `go build ./...`, `go vet ./...`, `go test
./...`, `golangci-lint run ./...`, and a clean `gofumpt -l .` check. PR #946's
hosted test, lint, and security checks were green before merge; release build
and deploy jobs were skipped by repository policy.

## Next session

Return to clean `main`, pull, rerun `make fidelity-depth`, reread
`docs/fidelity/DEPTH_TESTING.md` and this newest handoff, then take only the
unclaimed `hop` family at `src/interpreter.c:501`. Continue the command-table
sweep in source order, preserving the one-slice/one-PR rule and the
non-green-check safety rule.
