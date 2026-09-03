# Depth handoff — retsu

Date: 2026-09-02  
Branch: `glm/depth-retsu`  
Feature PR: #1235 (merged green)  
Feature commit: `c89cced0b`  
Main merge: `ece9c0c14`

## Queue position

The special-procedure inventory is exhausted. The one blocked row,
`objmagic.sleep-entry-gates`, was attempted once through the cast-sleep
outlaw/reagent vehicle and remains blocked for the unreachable entry gates.
After refreshing `main`, pulling with `--ff-only`, running `make fidelity-depth`,
reading `DEPTH_TESTING.md`, and reviewing the newest dated handoff, the next
unclaimed interpreter-table family in source order was `retsu` at
`src/interpreter.c:654`. Existing Jin/Kai kuji manifests own the shared
kuji-kiri gates and catalog mapping; they were delegated rather than repicked.

Pre-slice frontier: 3,638 total, 3,537 proven/delegated, 48 blocked, 53
excluded.

Post-slice frontier: 3,655 total, 3,554 proven/delegated, 48 blocked, 53
excluded; actionable completion is 3,554/3,602 = 98.7%.

The next session must refresh `main`, rerun the frontier, reread the guide and
newest handoff, then continue the interpreter-table sweep after `retsu` in
table order. Do not repick the shared claims or any stale handoff claim.

## C call path and observable contract

The command table registers:

```c
{ "retsu"    , POS_STANDING, do_kuji_kiri, 0, SKILL_KK_RETSU }
```

at `src/interpreter.c:654`. `do_kuji_kiri` in `src/new_cmds.c:1552-1739`
rejects a non-Ninja mortal, active fighting, an existing `AFF_KUJI_KIRI`
lockout, and mounted callers before drawing the success roll. It consumes
`check_kk_success()` before the seal mastery branch. An unmastered Retsu emits
`You have not mastered this art yet!` after that draw.

For a learned Retsu, success calls
`call_magic(ch, ch, NULL, SPELL_TELEPORT, GET_LEVEL(ch), CAST_SPELL)` and
clears the room act. `spell_teleport` in `src/spells.c:168-217` refuses a
peaceful source room, requires a PC self-target, selects a non-private room by
the shared RNUM draw, emits the black-screen victim text, origin-room fade-out,
transfer, destination fade-in, and `look_at_room(victim, 0)`. Failure sets the
generic concentration-failure actor text, zeroes both modifiers, changes the
second record to `AFF_NOTHING`, and still joins the populated records.

Because C's `affect_join` replaces matching type/location records, failed Retsu
does not retain the aggregate lockout; a later learned retry is allowed.
Successful Retsu and a peaceful spell refusal both retain the five-tick
`AFF_KUJI_KIRI` lockout through the joined default record.

The C source also exposed a shared visibility divergence during the destination
look: `CAN_SEE_IN_DARK(ch)` is only `AFF_INFRAVISION || PRF_HOLYLIGHT`
(`src/utils.h:451-452`). Go had incorrectly treated immortal level as dark
vision. Seed 3 produced C `Darkness` but Go's room description for the same
RNUM. The Go helper now matches C, and the coupled look test requires explicit
holy light for an immortal to see a dark room. No `src/` or oracle-tree file was
edited.

## Proof artifacts

Scenarios:

- `cmd/dp-oracle-diff/scenarios/retsu-depth.txt` — successful teleport,
  argument ignored, actor/origin audiences, destination look, and success
  lockout.
- `cmd/dp-oracle-diff/scenarios/retsu-failure-depth.txt` — generic failure,
  no room audience, and retry after the inert failed record.
- `cmd/dp-oracle-diff/scenarios/retsu-peaceful-depth.txt` — peaceful
  call-magic refusal and retained lockout.

Manifest: `docs/fidelity/depth/retsu.tsv` (17 rows), with shared kuji cases
delegated to the existing Jin/Kai claims.

Focused tests:

- `pkg/session/retsu_depth_test.go`
  (`TestRetsuRegistrationUsesCEntryGate`).
- `pkg/game/look_depth_test.go`
  (`TestImmortalWithoutHolyLightCannotSeeDarkRoom`).
- `pkg/session/cmd_look_test.go`
  (`TestLook_DarkRoomWithHolyLight`) keeps the explicit C holy-light arm.

With `DP_ORACLE_BIN=/home/zach/darkpawns-c-oracle/bin/circle`, all three
scenarios produced `result: no normalized divergence` at seeds 1, 2, 3, 5,
and 8. The first successful seed-3 run was RED before the visibility fix;
the draw logs were identical, proving the divergence was the Go dark-vision
predicate rather than RNG or RNUM selection.

## Gates and review

All local gates passed on the final feature commit:

- `make fidelity-depth`
- `go build ./...`
- `go vet ./...`
- `go test ./...`
- `golangci-lint run ./...`
- `gofumpt -l .` (clean)
- `git diff --check`

PR #1235's hosted lint, security, and test checks were green; build/deploy
were skipped by the workflow. The PR was self-merged only after the required
checks were green.

This slice follows R1 (player-facing bytes), R2 (the registered command
surface), R3 (draw and affect ordering), R4 (no invented behavior), R5 (the
actual C call path), R5e (the C source wins), and R5b/R5c (shared kuji and
visibility behavior is audited and delegated at the class level).
