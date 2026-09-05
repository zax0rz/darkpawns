# Depth-fidelity handoff — `quit`

Date: 2026-09-01

## Queue position

This round began from clean `main` after `git pull --ff-only`, a successful
`make fidelity-depth`, and rereading `docs/fidelity/DEPTH_TESTING.md` plus
the latest `qui` handoff. The special-procedure inventory remains exhausted,
the one blocked row `objmagic.sleep-entry-gates` remains queued after its
single cast-sleep vehicle, and the interpreter sweep advanced from `qui` to
`quit`.

The next unmanifested interpreter family is `qsay` at
`src/interpreter.c:631`. The claimed `qecho` family and its open feature and
handoff PRs must not be repicked. `quaff` and `quest` are already represented
by existing depth manifests.

Frontier before this slice: 2,911 total; 2,839 proven/delegated; 22 blocked;
50 excluded.

Frontier after this slice: 2,926 total; 2,853 proven/delegated; 22 blocked;
51 excluded.

## C call path and behavior surface

The command-table entry is:

```c
/* src/interpreter.c:630 */
{ "quit"     , POS_DEAD    , do_quit     , 0, SCMD_QUIT },
```

`src/act.other.c:72-181` first returns for NPCs or descriptors absent from
the command surface, then computes `isokquit`: temples 8004 and 8008 are
always safe; hometown 2 owns room 18201; hometown 3 owns rooms 21202 and
21258; and the default branch accepts a room only when `is_owner` succeeds.
The POS_FIGHTING branch refuses with the exact fighting message. POS_INCAP
and below emit the death line and call `die(ch)` without logging out. A mortal
using regular `quit` outside a safe room receives the two-line REALLYQUIT
refusal plus the level-five-and-below RECALL hint. The handler ignores its
argument.

On success C emits one visible room leave act, the exact actor goodbye,
turns off an active infobar, closes other descriptors with the same id, saves
equipment in a safe room, calls the shared unmount helper when mounted, and
extracts the player. The safe-room and successful-audience vehicle uses an
immortal to force a normal mortal peer into temple 8004; this reaches the
normal `quit` handler path without changing source or world fixtures.

## Evidence and confirmed divergences

Scenarios:

- `cmd/dp-oracle-diff/scenarios/quit-unsafe-depth.txt`
- `cmd/dp-oracle-diff/scenarios/quit-safe-depth.txt`

Manifest: `docs/fidelity/depth/quit.tsv` (15 rows: 13 proven/delegated and
one explicitly excluded).

Focused test: `pkg/session/quit_depth_test.go`, supplemented by the existing
quit state tests in `pkg/session/quit_test.go`.

Both oracle vehicles were GREEN at seeds 1, 2, 3, 5, and 8. The safe vehicle
was also run with `--show-oracle` at seed 1, confirming the intended forced
safe-room extraction block. No production divergence was confirmed, so this
slice adds only C-first scenarios, the manifest, and the command-gate proof.
The NPC/no-descriptor branch is explicitly excluded from this player
descriptor command surface. No `src/` or C-oracle file was edited.

## Verification and integration

All local gates passed on the feature branch:

```text
make fidelity-depth
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
gofumpt -l .  # clean
```

Feature branch: `glm/depth-quit`

Feature commit: `aafa7dc58` (`test: prove quit depth fidelity (R1/R2/R3/R5e)`)

Feature PR: #1100 — merged as `7a2d8abe8`. Hosted security, lint, race/e2e
test, and build-and-push checks were green in run `33570171941`; deploy was
skipped by workflow conditions. The PR was merged only after the hosted
checks were green.

The `qecho` feature PR #1096 and handoff PR #1097, purge handoff PR #1095,
and plot handoff PR #1064 remain open because their checks did not fire after
their one permitted exact workflow retries; none was merged.

## Fidelity rules

This slice follows R1 (player-facing bytes), R2 (command surface), R3
(determinism and ordering), R4 (no invention), R5 (process discipline), R5b
and R5c (shared behavior ownership), and R5e (verify the actual C call path).
The source-order claim is maintained for the next `qsay` slice.
